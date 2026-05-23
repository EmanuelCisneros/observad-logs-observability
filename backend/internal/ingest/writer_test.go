package ingest

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/observa/observad/internal/model"
	"github.com/observa/observad/internal/realtime"
	"github.com/observa/observad/internal/store"
)

type failStore struct {
	attempts int
	failFor  int
	inner    store.LogStore
}

func (f *failStore) InsertBatch(ctx context.Context, logs []model.Log) error {
	f.attempts++
	if f.attempts <= f.failFor {
		return errors.New("simulated insert failure")
	}
	return f.inner.InsertBatch(ctx, logs)
}

func (f *failStore) Search(ctx context.Context, q model.LogQuery) (model.LogPage, error) {
	return f.inner.Search(ctx, q)
}

func (f *failStore) Stats(ctx context.Context, q model.LogQuery) (model.Stats, error) {
	return f.inner.Stats(ctx, q)
}

func (f *failStore) Ping(ctx context.Context) error { return f.inner.Ping(ctx) }
func (f *failStore) Close() error                  { return f.inner.Close() }

func TestWriter_RetriesAfterInsertFailure(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer rdb.Close()

	buf, err := NewBuffer(context.Background(), rdb, "observa:test:writer")
	require.NoError(t, err)

	mem := store.NewMemStore()
	wrapped := &failStore{failFor: 1, inner: mem}

	writer, err := NewWriter(WriterConfig{
		Buffer:        buf,
		Store:         wrapped,
		Hub:           realtime.NewHub(),
		BatchSize:     10,
		FlushInterval: 50 * time.Millisecond,
		Workers:       1,
	})
	require.NoError(t, err)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	require.NoError(t, buf.Push(ctx, []model.Log{
		{ID: "r1", Timestamp: time.Now().UTC(), Level: model.LevelInfo, Service: "api", Message: "hello"},
	}))

	errCh := make(chan error, 1)
	go func() { errCh <- writer.Run(ctx) }()

	require.Eventually(t, func() bool {
		return mem.Len() == 1
	}, 2*time.Second, 50*time.Millisecond)

	cancel()
	<-errCh

	assert.GreaterOrEqual(t, wrapped.attempts, 2)
}
