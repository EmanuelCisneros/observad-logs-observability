package ingest

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/observa/observad/internal/model"
)

func testBuffer(t *testing.T) (*Buffer, *miniredis.Miniredis) {
	t.Helper()
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	buf, err := NewBuffer(context.Background(), rdb, "observa:test:ingest")
	require.NoError(t, err)
	return buf, mr
}

func TestBuffer_PushAndClaimNew(t *testing.T) {
	buf, _ := testBuffer(t)
	ctx := context.Background()

	logs := []model.Log{
		{ID: "a", Timestamp: time.Now().UTC(), Level: model.LevelInfo, Service: "api", Message: "one"},
		{ID: "b", Timestamp: time.Now().UTC(), Level: model.LevelInfo, Service: "api", Message: "two"},
	}
	require.NoError(t, buf.Push(ctx, logs))

	claimed, ids, err := buf.Claim(ctx, "worker-0", 10, time.Second)
	require.NoError(t, err)
	require.Len(t, claimed, 2)
	require.Len(t, ids, 2)
	assert.Equal(t, "one", claimed[0].Message)

	require.NoError(t, buf.Ack(ctx, ids))

	pending, err := buf.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), pending)
}

func TestBuffer_ClaimPendingWithoutAck(t *testing.T) {
	buf, _ := testBuffer(t)
	ctx := context.Background()

	require.NoError(t, buf.Push(ctx, []model.Log{
		{ID: "x", Timestamp: time.Now().UTC(), Level: model.LevelInfo, Service: "api", Message: "retry-me"},
	}))

	_, ids, err := buf.Claim(ctx, "worker-0", 1, time.Second)
	require.NoError(t, err)
	require.Len(t, ids, 1)

	pending, err := buf.PendingCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), pending)

	// Same consumer redelivers its pending entries via the "0" stream ID.
	claimed, _, err := buf.Claim(ctx, "worker-0", 1, 100*time.Millisecond)
	require.NoError(t, err)
	require.Len(t, claimed, 1)
	assert.Equal(t, "retry-me", claimed[0].Message)
}

func TestBuffer_DepthAndPending(t *testing.T) {
	buf, _ := testBuffer(t)
	ctx := context.Background()

	depth, err := buf.Depth(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), depth)

	require.NoError(t, buf.Push(ctx, []model.Log{
		{ID: "1", Timestamp: time.Now().UTC(), Level: model.LevelInfo, Service: "s", Message: "m"},
	}))

	depth, err = buf.Depth(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(1), depth)
}
