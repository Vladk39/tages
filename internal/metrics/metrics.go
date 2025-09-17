package metrics

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

var (
	DbOperationsTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "apps",
		Subsystem: "image",
		Name:      "operation_total_db",
		Help:      "Total operation of DB by type",
	},
		[]string{"status", "operation"})

	DbOperationDuration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "apps",
		Subsystem: "image",
		Name:      "operation_db_duration_seconds",
		Help:      "Duration of DB operations",
		Buckets:   prometheus.DefBuckets,
	},
		[]string{"status", "operation"})

	grpcErrorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "apps",
			Subsystem: "grpc_errors",
			Name:      "grpc_req_errors",
			Help:      "Total errors in grpc req",
		}, []string{"method", "code"},
	)
)

func DBMetricsFunc(status, operation string, start time.Time) {
	DbOperationsTotal.WithLabelValues(status, operation).Inc()
	DbOperationDuration.WithLabelValues(status, operation).Observe(time.Since(start).Seconds())
}

func UnaryErrorMetricsInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		resp, err := handler(ctx, req)
		if err != nil {
			st, _ := status.FromError(err)
			grpcErrorCounter.WithLabelValues(info.FullMethod, st.Code().String()).Inc()
		}
		return resp, err
	}
}

func StreamErrorMetricsInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		err := handler(srv, ss)
		if err != nil {
			st, _ := status.FromError(err)
			grpcErrorCounter.WithLabelValues(info.FullMethod, st.Code().String()).Inc()
		}
		return err
	}
}

func Initalize(reg *prometheus.Registry) {
	reg.MustRegister(grpcErrorCounter, DbOperationsTotal, DbOperationDuration)
}
