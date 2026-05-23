package store

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/observa/observad/internal/model"
)

// seed is shared test data.
func seed(t *testing.T) (*MemStore, context.Context) {
	t.Helper()
	ctx := context.Background()
	m := NewMemStore()

	base := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	logs := []model.Log{
		{ID: "1", Timestamp: base.Add(0 * time.Minute), Level: model.LevelInfo, Service: "api", Message: "request handled"},
		{ID: "2", Timestamp: base.Add(1 * time.Minute), Level: model.LevelError, Service: "api", Message: "database timeout"},
		{ID: "3", Timestamp: base.Add(2 * time.Minute), Level: model.LevelWarn, Service: "worker", Message: "retry scheduled"},
		{ID: "4", Timestamp: base.Add(3 * time.Minute), Level: model.LevelError, Service: "worker", Message: "connection refused"},
		{ID: "5", Timestamp: base.Add(4 * time.Minute), Level: model.LevelInfo, Service: "cron", Message: "job completed"},
	}
	require.NoError(t, m.InsertBatch(ctx, logs))
	return m, ctx
}

func TestMemStore_InsertAndLen(t *testing.T) {
	m, _ := seed(t)
	assert.Equal(t, 5, m.Len())
}

func TestMemStore_InsertEmptyBatch(t *testing.T) {
	m := NewMemStore()
	require.NoError(t, m.InsertBatch(context.Background(), nil))
	assert.Equal(t, 0, m.Len())
}

func TestMemStore_SearchNoFilter(t *testing.T) {
	m, ctx := seed(t)
	page, err := m.Search(ctx, model.LogQuery{})
	require.NoError(t, err)
	assert.Equal(t, uint64(5), page.Total)
	assert.Len(t, page.Logs, 5)
}

func TestMemStore_SearchByLevel(t *testing.T) {
	m, ctx := seed(t)
	page, err := m.Search(ctx, model.LogQuery{Levels: []model.Level{model.LevelError}})
	require.NoError(t, err)
	assert.Equal(t, uint64(2), page.Total)
	for _, l := range page.Logs {
		assert.Equal(t, model.LevelError, l.Level)
	}
}

func TestMemStore_SearchByService(t *testing.T) {
	m, ctx := seed(t)
	page, err := m.Search(ctx, model.LogQuery{Services: []string{"worker"}})
	require.NoError(t, err)
	assert.Equal(t, uint64(2), page.Total)
}

func TestMemStore_SearchBySubstring(t *testing.T) {
	m, ctx := seed(t)
	page, err := m.Search(ctx, model.LogQuery{Search: "timeout"})
	require.NoError(t, err)
	require.Equal(t, uint64(1), page.Total)
	assert.Equal(t, "2", page.Logs[0].ID)
}

func TestMemStore_SearchSubstringCaseInsensitive(t *testing.T) {
	m, ctx := seed(t)
	page, err := m.Search(ctx, model.LogQuery{Search: "TIMEOUT"})
	require.NoError(t, err)
	assert.Equal(t, uint64(1), page.Total)
}

func TestMemStore_SearchByTimeRange(t *testing.T) {
	m, ctx := seed(t)
	base := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	page, err := m.Search(ctx, model.LogQuery{
		From: base.Add(1 * time.Minute),
		To:   base.Add(3 * time.Minute),
	})
	require.NoError(t, err)
	assert.Equal(t, uint64(3), page.Total, "logs 2,3,4 fall inside the window")
}

func TestMemStore_SearchOrderDesc(t *testing.T) {
	m, ctx := seed(t)
	page, err := m.Search(ctx, model.LogQuery{Order: "desc"})
	require.NoError(t, err)
	require.Len(t, page.Logs, 5)
	// Newest first.
	assert.Equal(t, "5", page.Logs[0].ID)
	assert.Equal(t, "1", page.Logs[4].ID)
}

func TestMemStore_SearchOrderAsc(t *testing.T) {
	m, ctx := seed(t)
	page, err := m.Search(ctx, model.LogQuery{Order: "asc"})
	require.NoError(t, err)
	require.Len(t, page.Logs, 5)
	assert.Equal(t, "1", page.Logs[0].ID)
	assert.Equal(t, "5", page.Logs[4].ID)
}

func TestMemStore_SearchPagination(t *testing.T) {
	m, ctx := seed(t)
	page, err := m.Search(ctx, model.LogQuery{Limit: 2, Offset: 2, Order: "asc"})
	require.NoError(t, err)
	assert.Equal(t, uint64(5), page.Total, "total reflects all matches, not the page")
	require.Len(t, page.Logs, 2)
	assert.Equal(t, "3", page.Logs[0].ID)
	assert.Equal(t, "4", page.Logs[1].ID)
}

func TestMemStore_SearchOffsetBeyondEnd(t *testing.T) {
	m, ctx := seed(t)
	page, err := m.Search(ctx, model.LogQuery{Offset: 999})
	require.NoError(t, err)
	assert.Empty(t, page.Logs)
	assert.Equal(t, uint64(5), page.Total)
}

func TestMemStore_SearchCombinedFilters(t *testing.T) {
	m, ctx := seed(t)
	page, err := m.Search(ctx, model.LogQuery{
		Levels:   []model.Level{model.LevelError},
		Services: []string{"worker"},
	})
	require.NoError(t, err)
	require.Equal(t, uint64(1), page.Total)
	assert.Equal(t, "4", page.Logs[0].ID)
}

func TestMemStore_Stats(t *testing.T) {
	m, ctx := seed(t)
	stats, err := m.Stats(ctx, model.LogQuery{})
	require.NoError(t, err)

	assert.Equal(t, uint64(5), stats.Total)
	assert.Equal(t, uint64(2), stats.ByLevel[model.LevelInfo])
	assert.Equal(t, uint64(2), stats.ByLevel[model.LevelError])
	assert.Equal(t, uint64(1), stats.ByLevel[model.LevelWarn])
}

func TestMemStore_StatsTopServices(t *testing.T) {
	m, ctx := seed(t)
	stats, err := m.Stats(ctx, model.LogQuery{})
	require.NoError(t, err)

	require.NotEmpty(t, stats.TopServices)
	// api and worker both have 2; cron has 1. The top entry must have count 2.
	assert.Equal(t, uint64(2), stats.TopServices[0].Count)
	// Services are sorted by count descending.
	for i := 1; i < len(stats.TopServices); i++ {
		assert.GreaterOrEqual(t,
			stats.TopServices[i-1].Count, stats.TopServices[i].Count)
	}
}

func TestMemStore_StatsHistogramOrdered(t *testing.T) {
	m, ctx := seed(t)
	stats, err := m.Stats(ctx, model.LogQuery{})
	require.NoError(t, err)

	require.NotEmpty(t, stats.Histogram)
	for i := 1; i < len(stats.Histogram); i++ {
		assert.True(t,
			stats.Histogram[i-1].Bucket.Before(stats.Histogram[i].Bucket),
			"histogram buckets must be in ascending time order")
	}
}

func TestMemStore_StatsRespectsFilter(t *testing.T) {
	m, ctx := seed(t)
	stats, err := m.Stats(ctx, model.LogQuery{Services: []string{"api"}})
	require.NoError(t, err)
	assert.Equal(t, uint64(2), stats.Total)
}

func TestMemStore_PingAlwaysOK(t *testing.T) {
	m := NewMemStore()
	assert.NoError(t, m.Ping(context.Background()))
	assert.NoError(t, m.Close())
}
