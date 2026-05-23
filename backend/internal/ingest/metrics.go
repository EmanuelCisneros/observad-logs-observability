package ingest

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var ingestLag = promauto.NewHistogram(prometheus.HistogramOpts{
	Name:    "observa_ingest_to_tail_seconds",
	Help:    "Time from log timestamp to WebSocket broadcast after successful store insert.",
	Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1, 2, 5, 10, 30},
})

func recordIngestLag(d time.Duration) {
	if d < 0 {
		d = 0
	}
	ingestLag.Observe(d.Seconds())
}
