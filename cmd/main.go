package main

import (
	"context"
	"net"
	"net/http"
	"os"
	"syscall"
	"time"

	"Tages/internal/cache"
	"Tages/internal/metrics"
	"Tages/internal/ratelimiter"
	"Tages/internal/service"
	"Tages/internal/storage"
	"Tages/pkg"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var (
	reg         = prometheus.NewRegistry()
	grpcMetrics = grpc_prometheus.NewServerMetrics()
)

func main() {
	var g run.Group

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = g
	_ = ctx
	// metrics

	reg.MustRegister(grpcMetrics)
	metrics.Initalize(reg)

	configure()
	logger := getLogger()
	logger.Info("Starting service...")

	logger.Info("Opening storage...")
	store, err := getStorage(ctx, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to init storage")
	}

	logger.Info("Creating cache...")
	CacheDetector := make(chan bool, 1)
	cache := cache.NewCache(logger, CacheDetector)
	go cache.RunWatcher()

	logger.Info("Creating service...")
	srv, err := service.NewServicefile(ctx, logger, cache, store)
	if err != nil {
		logger.WithError(err).Fatal("Failed to init service")
	}

	logger.Info("Create ratelimiter...")

	viper.SetDefault("ratelimiter.tokens", 10)
	viper.SetDefault("ratelimiter.interval_ms", 1000)

	rateCoin := viper.GetInt("ratelimiter.tokens")
	rateRefresh := time.Duration(viper.GetInt("ratelimiter.interval_ms")) * time.Millisecond
	rateLimiter := ratelimiter.New(rateCoin, rateRefresh, logger)
	defer rateLimiter.Stop()

	go func() {
		srv.HeatCache(ctx)
	}()

	var grpcs *grpc.Server

	g.Add(
		run.SignalHandler(
			ctx,
			[]os.Signal{os.Interrupt, syscall.SIGTERM, syscall.SIGINT, syscall.SIGHUP}...),
	)
	g.Add(func() error {
		logger.Infof("Starting server on %s", viper.GetString("listen"))
		listener, err := net.Listen("tcp", viper.GetString("listen"))
		if err != nil {
			return err
		}

		grpcs = grpc.NewServer(
			grpc.MaxRecvMsgSize(20*1024*1024),
			grpc.MaxSendMsgSize(20*1024*1024),
			grpc.ChainStreamInterceptor(
				grpcMetrics.StreamServerInterceptor(),
				metrics.StreamErrorMetricsInterceptor(),
				rateLimiter.StreamInterceptor(),
			),
			grpc.ChainUnaryInterceptor(
				grpcMetrics.UnaryServerInterceptor(),
				metrics.UnaryErrorMetricsInterceptor(),
				rateLimiter.UnaryInterceptor(),
			),
		)

		pkg.RegisterFileServiceServer(grpcs, srv)
		grpcMetrics.InitializeMetrics(grpcs)
		logger.Info("Server started.")

		return grpcs.Serve(listener)
	}, func(err error) {
		logger.WithError(err).Info("Stopping server")
		if grpcs != nil {
			grpcs.GracefulStop()
			logger.Info("Server shut down")
		}

		cancel()
	})
	g.Add(func() error {
		mux := http.NewServeMux()
		mux.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

		listenAddr := viper.GetString("metrics")
		logger.Infof("Metrics listening on %s", listenAddr)

		server := &http.Server{
			Addr:    listenAddr,
			Handler: mux,
		}

		go func() {
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.WithError(err).Error("Metrics server stopped unexpectedly")
			}
		}()

		<-ctx.Done()
		return server.Shutdown(context.Background())
	}, func(err error) {
		logger.WithError(err).Info("Stopping metrics server")
	})
	g.Add(func() error {
		<-ctx.Done()
		logger.Info("Closing storage...")
		if err := store.Close(ctx); err != nil {
			logger.WithError(err).Error("Failed to close storage")
			return err
		}
		logger.Info("Storage closed")
		return nil
	}, func(err error) {
	})

	if err := g.Run(); err != nil {
		logger.WithError(err).Fatal("run group finished with error")
	}

}

func configure() {
	// сервак
	viper.SetDefault("listen", ":8080")
	// logger
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.level", "info")
	// metrics
	viper.SetDefault("metrics", ":9094")
	// dsn postgresql
	viper.SetDefault("db.dsn", "postgres://myuser:mypassword@localhost:5432/mydb?sslmode=disable")
	// ratelimiter
	viper.SetDefault("ratelimiter.tokens", 10)
	viper.SetDefault("upload.dir", "../uploads")
	viper.SetDefault("ratelimiter.interval_ms", 1000)
}

func getLogger() *logrus.Logger {
	logger := logrus.New()
	if viper.GetString("log.format") == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	}

	level, err := logrus.ParseLevel(viper.GetString("log.level"))
	if err != nil {
		level = logrus.InfoLevel
	}

	logger.SetLevel(level)

	if err != nil {
		logger.WithError(err).Warn("Failed to parse log level")
	}

	return logger
}

func getStorage(ctx context.Context, log *logrus.Logger) (*storage.Storage, error) {
	dsn := viper.GetString("db.dsn")
	log.WithField("dsn", dsn).Info("Initializing storage connection")

	store, err := storage.NewStorage(dsn)
	if err != nil {
		log.WithError(err).Error("Failed to create storage instance")
		return nil, err
	}

	log.Info("Storage instance created, initializing...")

	if err := store.Init(ctx); err != nil {
		log.WithError(err).Error("Failed to initialize storage")
		return nil, err
	}
	log.Info("Storage successfully initialized")

	return store, nil
}
