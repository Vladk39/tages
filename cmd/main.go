package main

import (
	"context"
	"net"
	"os"
	"syscall"
	"time"

	"Tages/internal/cache"
	"Tages/internal/metrics"
	"Tages/internal/service"
	"Tages/internal/storage"
	"Tages/pkg"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/oklog/run"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"google.golang.org/grpc"
)

var (
	reg         = prometheus.NewRegistry()
	grpcMetrics = grpc_prometheus.NewServerMetrics()
)

// grpc.UnaryInterceptor(rateLimiter.UnaryInterceptor()),
// 	grpc.StreamInterceptor(rateLimiter.StreamInterceptor()),

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
	logger.Info("Starting service")

	logger.Info("Opening storage...")
	store, err := getStorage(ctx, logger)
	if err != nil {
		logger.WithError(err).Fatal("Failed to init storage")
	}

	logger.Info("Creating cache...")
	CacheDetector := make(chan bool, 1)
	cache := cache.NewCache(logger, CacheDetector)

	logger.Info("Creating service...")
	srv, err := service.NewServicefile(ctx, logger, cache, store)
	if err != nil {
		logger.WithError(err).Fatal("Failed to init service")
	}

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
			),
			grpc.ChainUnaryInterceptor(
				grpcMetrics.UnaryServerInterceptor(),
				metrics.UnaryErrorMetricsInterceptor(),
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

}

func configure() {
	viper.SetDefault("listen", ":8080")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("metrics.listen", ":9093")
	viper.SetDefault("db.dsn", "postgres://myuser:mypassword@localhost:5432/mydb?sslmode=disable")
	viper.SetDefault("listen", ":8080")
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

	store, err := storage.NewStorage(dsn)
	if err != nil {
		return nil, err
	}

	if err := store.Init(ctx); err != nil {
		return nil, err
	}

	return store, nil
}
