package httpapi

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"github.com/observa/observad/internal/ingest"
)

var metrics = struct {
	requests     *prometheus.CounterVec
	latency      *prometheus.HistogramVec
	ingested     prometheus.Counter
	rejected     prometheus.Counter
	queueDepth   prometheus.GaugeFunc
	queuePending prometheus.GaugeFunc
}{
	requests: promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "observa_http_requests_total",
		Help: "Total HTTP requests by route and status.",
	}, []string{"route", "method", "status"}),
	latency: promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "observa_http_request_duration_seconds",
		Help:    "HTTP request latency by route.",
		Buckets: prometheus.DefBuckets,
	}, []string{"route"}),
	ingested: promauto.NewCounter(prometheus.CounterOpts{
		Name: "observa_logs_ingested_total",
		Help: "Total log lines accepted for ingestion.",
	}),
	rejected: promauto.NewCounter(prometheus.CounterOpts{
		Name: "observa_logs_rejected_total",
		Help: "Total log lines rejected during validation.",
	}),
}

func registerIngestMetrics(buf *ingest.Buffer) {
	if buf == nil {
		return
	}
	metrics.queueDepth = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "observa_ingest_queue_depth",
		Help: "Approximate number of entries in the Redis ingest stream.",
	}, func() float64 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		n, err := buf.Depth(ctx)
		if err != nil {
			return 0
		}
		return float64(n)
	})
	metrics.queuePending = promauto.NewGaugeFunc(prometheus.GaugeOpts{
		Name: "observa_ingest_queue_pending",
		Help: "Entries in the Redis consumer group pending entries list.",
	}, func() float64 {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		n, err := buf.PendingCount(ctx)
		if err != nil {
			return 0
		}
		return float64(n)
	})
}

func observability() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == "" {
			route = "unmatched"
		}
		metrics.latency.WithLabelValues(route).Observe(time.Since(start).Seconds())
		metrics.requests.WithLabelValues(
			route,
			c.Request.Method,
			statusClass(c.Writer.Status()),
		).Inc()
	}
}

func statusClass(code int) string {
	switch {
	case code >= 500:
		return "5xx"
	case code >= 400:
		return "4xx"
	case code >= 300:
		return "3xx"
	case code >= 200:
		return "2xx"
	default:
		return "1xx"
	}
}
