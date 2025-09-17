package ratelimiter

import (
	"context"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"
)

type TokenBucket struct {
	tokens      int
	maxTokens   int
	refillEvery time.Duration
	lastRefill  time.Time
	mu          sync.Mutex
}

func NewTokenBucket(maxTokens int, refillEvery time.Duration) *TokenBucket {
	return &TokenBucket{
		tokens:      maxTokens,
		maxTokens:   maxTokens,
		refillEvery: refillEvery,
		lastRefill:  time.Now(),
	}
}

func (tb *TokenBucket) Allow() bool {
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)

	tb.mu.Lock()
	defer tb.mu.Unlock()

	newTokens := int(elapsed / tb.refillEvery)
	if newTokens > 0 {
		tb.tokens += newTokens
		if tb.tokens > tb.maxTokens {
			tb.tokens = tb.maxTokens
		}
		tb.lastRefill = now
	}

	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}

type bucketWrapper struct {
	bucket   *TokenBucket
	lastSeen time.Time
}

type RateLimiter struct {
	buckets   map[string]*bucketWrapper
	bucketsMu sync.RWMutex
	maxTokens int
	interval  time.Duration
	ttl       time.Duration
	logger    *logrus.Logger
	stopCh    chan struct{}
}

func New(maxTokens int, interval time.Duration, logger *logrus.Logger) *RateLimiter {
	rl := &RateLimiter{
		buckets:   make(map[string]*bucketWrapper),
		maxTokens: maxTokens,
		interval:  interval,
		ttl:       5 * time.Minute,
		logger:    logger,
		stopCh:    make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

func (rl *RateLimiter) Stop() {
	close(rl.stopCh)
	rl.bucketsMu.Lock()
	rl.buckets = make(map[string]*bucketWrapper)
	rl.bucketsMu.Unlock()
}

func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-rl.stopCh:
			return
		case <-ticker.C:
			now := time.Now()
			rl.bucketsMu.RLock()
			toDelete := []string{}
			for ip, wrapper := range rl.buckets {
				if now.Sub(wrapper.lastSeen) > rl.ttl {
					toDelete = append(toDelete, ip)
				}
			}
			rl.bucketsMu.RUnlock()

			if len(toDelete) > 0 {
				rl.bucketsMu.Lock()
				for _, ip := range toDelete {
					delete(rl.buckets, ip)
				}
				rl.bucketsMu.Unlock()
			}
		}
	}
}

func (rl *RateLimiter) allowRequest(ip string) error {
	rl.bucketsMu.RLock()
	wrapper, exists := rl.buckets[ip]
	rl.bucketsMu.RUnlock()

	if !exists {
		bucket := NewTokenBucket(rl.maxTokens, rl.interval)
		wrapper = &bucketWrapper{bucket: bucket, lastSeen: time.Now()}
		rl.bucketsMu.Lock()
		rl.buckets[ip] = wrapper
		rl.bucketsMu.Unlock()
		rl.logger.Infof("Created new bucket for IP: %s", ip)
	}

	wrapper.lastSeen = time.Now()

	if !wrapper.bucket.Allow() {
		rl.logger.Infof("Rate limit exceeded for IP: %s", ip)
		return status.Errorf(codes.ResourceExhausted, "too many requests")
	}

	return nil
}

func (rl *RateLimiter) UnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(
		ctx context.Context,
		req interface{},
		info *grpc.UnaryServerInfo,
		handler grpc.UnaryHandler,
	) (interface{}, error) {
		p, _ := peer.FromContext(ctx)
		ip := "unknown"
		if p != nil && p.Addr != nil {
			ip = p.Addr.String()
		}

		if err := rl.allowRequest(ip); err != nil {
			return nil, err
		}

		return handler(ctx, req)
	}
}

func (rl *RateLimiter) StreamInterceptor() grpc.StreamServerInterceptor {
	return func(
		srv interface{},
		ss grpc.ServerStream,
		info *grpc.StreamServerInfo,
		handler grpc.StreamHandler,
	) error {
		p, _ := peer.FromContext(ss.Context())
		ip := "unknown"
		if p != nil && p.Addr != nil {
			ip = p.Addr.String()
		}

		if err := rl.allowRequest(ip); err != nil {
			return err
		}

		return handler(srv, ss)
	}
}
