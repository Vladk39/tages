package metrics

import (
	"context"
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/shirou/gopsutil/mem"
	"github.com/sirupsen/logrus"
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

	gcFrequency = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "backend",
			Subsystem: "runtime",
			Name:      "gc_runs_total",
			Help:      "Total number of gc runs",
		})

	gcTotalTime = prometheus.NewCounter(
		prometheus.CounterOpts{
			Namespace: "backend",
			Subsystem: "runtime",
			Name:      "gc_totak_time_seconds",
			Help:      "Total time spent in GC",
		})

	memAlloc = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "backend",
		Subsystem: "runtime",
		Name:      "go_memstats_alloc_bytes",
		Help:      "Current memory allocated and still in use.",
	})

	memHeapInuse = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "backend",
		Subsystem: "runtime",
		Name:      "go_memstats_heap_inuse_bytes",
		Help:      "Heap memory in use.",
	})

	memHeapObjects = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: "backend",
		Subsystem: "runtime",
		Name:      "go_memstats_heap_objects",
		Help:      "Number of allocated heap objects.",
	})

	gcPauseDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
		Namespace: "backend",
		Subsystem: "runtime",
		Name:      "gc_pause_duration_seconds",
		Help:      "GC pause duration in seconds",
		Buckets:   []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
	})
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
	reg.MustRegister(grpcErrorCounter, DbOperationsTotal, DbOperationDuration, gcPauseDuration, gcFrequency, gcTotalTime, memAlloc, memHeapInuse, memHeapObjects)
}

func CollectorGCHeapMetrics(ctx context.Context, log *logrus.Logger, ch chan bool) {
	log.Info("GC metrics started")

	var stats runtime.MemStats
	// количество GC циклов
	var lastnumGC uint32
	// общее время пауз на предидущих измирениях
	var lastPauseTotal time.Duration

	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Info("GC metrics Stoped")
			return
		case <-ticker.C:
			runtime.ReadMemStats(&stats)

			if stats.NumGC > lastnumGC {
				NewGcRuns := stats.NumGC - lastnumGC
				gcFrequency.Add(float64(NewGcRuns))

				lastPauseNs := stats.PauseNs[(stats.NumGC+255)%256]
				gcPauseDuration.Observe(float64(lastPauseNs) / 1e9)

				lastnumGC = stats.NumGC
			}

			if stats.PauseTotalNs > uint64(lastPauseTotal) {
				newPauseTime := time.Duration(stats.PauseTotalNs) - lastPauseTotal
				gcTotalTime.Add(newPauseTime.Seconds())
				lastPauseTotal = time.Duration(stats.PauseTotalNs)
			}

			memAlloc.Set(float64(stats.Alloc))
			memHeapInuse.Set(float64(stats.HeapInuse))
			memHeapObjects.Set(float64(stats.HeapObjects))

			vmStat, err := mem.VirtualMemory()
			if err != nil {
				log.Warnf("failed to get system memory: %v", err)
				continue
			}

			if vmStat.UsedPercent >= 70 {
				log.Warnf("Memory usage high: %.2f%%", vmStat.UsedPercent)
				select {
				case ch <- false:
				default:

				}
			}
		}
	}
}
