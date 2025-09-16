package main

import (
	"context"
	"time"

	grpc_prometheus "github.com/grpc-ecosystem/go-grpc-prometheus"
	"github.com/oklog/run"
	"github.com/sirupsen/logrus"
	"github.com/spf13/viper"
)

var (
	grpcMetrics = grpc_prometheus.NewServerMetrics()
)

func main() {
	var g run.Group

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = g
	_ = ctx
	// metrics
}

func configure() {
	viper.SetDefault("listen", ":8080")
	viper.SetDefault("log.format", "json")
	viper.SetDefault("log.level", "info")
	viper.SetDefault("metrics.listen", ":9093")
	viper.SetDefault("dsnpsg", "postgres://myuser:mypassword@localhost:5432/mydb?sslmode=disable")
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
