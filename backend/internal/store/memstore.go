package store

import (
	"context"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/observa/observad/internal/model"
)

type MemStore struct {
	mu   sync.RWMutex
	logs []model.Log
}

func NewMemStore() *MemStore {
	return &MemStore{logs: make([]model.Log, 0, 1024)}
}

func (m *MemStore) Ping(context.Context) error { return nil }
func (m *MemStore) Close() error               { return nil }

func (m *MemStore) InsertBatch(_ context.Context, logs []model.Log) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.logs = append(m.logs, logs...)
	return nil
}

// Len is for tests.
func (m *MemStore) Len() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.logs)
}

func (m *MemStore) Search(_ context.Context, q model.LogQuery) (model.LogPage, error) {
	q.Clamp()

	m.mu.RLock()
	matched := make([]model.Log, 0)
	for _, l := range m.logs {
		if matches(l, q) {
			matched = append(matched, l)
		}
	}
	m.mu.RUnlock()

	sort.Slice(matched, func(i, j int) bool {
		if q.Order == "asc" {
			return matched[i].Timestamp.Before(matched[j].Timestamp)
		}
		return matched[i].Timestamp.After(matched[j].Timestamp)
	})

	page := model.LogPage{Total: uint64(len(matched))}
	lo := q.Offset
	if lo > len(matched) {
		lo = len(matched)
	}
	hi := lo + q.Limit
	if hi > len(matched) {
		hi = len(matched)
	}
	page.Logs = append([]model.Log(nil), matched[lo:hi]...)
	return page, nil
}

func (m *MemStore) Stats(_ context.Context, q model.LogQuery) (model.Stats, error) {
	q.Clamp()

	stats := model.Stats{
		ByLevel:     map[model.Level]uint64{},
		TopServices: []model.ServiceCount{},
		Histogram:   []model.HistogramBucket{},
	}
	svcCounts := map[string]uint64{}
	histo := map[time.Time]map[model.Level]uint64{}

	m.mu.RLock()
	for _, l := range m.logs {
		if !matches(l, q) {
			continue
		}
		stats.Total++
		stats.ByLevel[l.Level]++
		svcCounts[l.Service]++

		bucket := l.Timestamp.Truncate(time.Minute)
		if histo[bucket] == nil {
			histo[bucket] = map[model.Level]uint64{}
		}
		histo[bucket][l.Level]++
	}
	m.mu.RUnlock()

	for svc, n := range svcCounts {
		stats.TopServices = append(stats.TopServices, model.ServiceCount{Service: svc, Count: n})
	}
	sort.Slice(stats.TopServices, func(i, j int) bool {
		return stats.TopServices[i].Count > stats.TopServices[j].Count
	})
	if len(stats.TopServices) > 10 {
		stats.TopServices = stats.TopServices[:10]
	}

	for bucket, counts := range histo {
		stats.Histogram = append(stats.Histogram, model.HistogramBucket{
			Bucket: bucket,
			Counts: counts,
		})
	}
	sort.Slice(stats.Histogram, func(i, j int) bool {
		return stats.Histogram[i].Bucket.Before(stats.Histogram[j].Bucket)
	})
	return stats, nil
}

// matches mirrors ClickHouse WHERE filtering for MemStore.
func matches(l model.Log, q model.LogQuery) bool {
	if !q.From.IsZero() && l.Timestamp.Before(q.From) {
		return false
	}
	if !q.To.IsZero() && l.Timestamp.After(q.To) {
		return false
	}
	if len(q.Levels) > 0 && !containsLevel(q.Levels, l.Level) {
		return false
	}
	if len(q.Services) > 0 && !containsString(q.Services, l.Service) {
		return false
	}
	if q.Search != "" && !strings.Contains(
		strings.ToLower(l.Message), strings.ToLower(q.Search)) {
		return false
	}
	return true
}

func containsLevel(s []model.Level, v model.Level) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

func containsString(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
