package ingest

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/observa/observad/internal/model"
)

type Buffer struct {
	rdb    *redis.Client
	stream string
	group  string
	maxLen int64
}

const (
	consumerGroup     = "observa-writers"
	stalePendingAfter = 30 * time.Second
)

func NewBuffer(ctx context.Context, rdb *redis.Client, stream string) (*Buffer, error) {
	b := &Buffer{
		rdb:    rdb,
		stream: stream,
		group:  consumerGroup,
		maxLen: 1_000_000,
	}
	err := rdb.XGroupCreateMkStream(ctx, stream, consumerGroup, "0").Err()
	if err != nil && !isBusyGroup(err) {
		return nil, fmt.Errorf("ingest: create consumer group: %w", err)
	}
	return b, nil
}

func (b *Buffer) Push(ctx context.Context, logs []model.Log) error {
	if len(logs) == 0 {
		return nil
	}
	pipe := b.rdb.Pipeline()
	for i := range logs {
		payload, err := json.Marshal(&logs[i])
		if err != nil {
			return fmt.Errorf("ingest: marshal log: %w", err)
		}
		pipe.XAdd(ctx, &redis.XAddArgs{
			Stream: b.stream,
			MaxLen: b.maxLen,
			Approx: true,
			Values: map[string]any{"log": payload},
		})
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("ingest: push batch: %w", err)
	}
	return nil
}

func (b *Buffer) Depth(ctx context.Context) (int64, error) {
	return b.rdb.XLen(ctx, b.stream).Result()
}

func (b *Buffer) PendingCount(ctx context.Context) (int64, error) {
	info, err := b.rdb.XPending(ctx, b.stream, b.group).Result()
	if err != nil {
		return 0, fmt.Errorf("ingest: xpending: %w", err)
	}
	return info.Count, nil
}

// Claim order: stale pending (XAUTOCLAIM), this consumer's PEL ("0"), then new (">").
func (b *Buffer) Claim(ctx context.Context, consumer string, count int64, block time.Duration) ([]model.Log, []string, error) {
	if count <= 0 {
		return nil, nil, nil
	}

	var logs []model.Log
	var ids []string

	staleLogs, staleIDs, err := b.autoClaim(ctx, consumer, count)
	if err != nil {
		return nil, nil, err
	}
	logs, ids = append(logs, staleLogs...), append(ids, staleIDs...)

	if int64(len(logs)) >= count {
		return logs[:count], ids[:count], nil
	}

	remaining := count - int64(len(logs))
	pendingLogs, pendingIDs, err := b.readGroup(ctx, consumer, remaining, 0, "0")
	if err != nil {
		return nil, nil, err
	}
	logs, ids = append(logs, pendingLogs...), append(ids, pendingIDs...)

	if int64(len(logs)) >= count {
		return logs[:count], ids[:count], nil
	}

	remaining = count - int64(len(logs))
	newLogs, newIDs, err := b.readGroup(ctx, consumer, remaining, block, ">")
	if err != nil {
		return nil, nil, err
	}
	logs, ids = append(logs, newLogs...), append(ids, newIDs...)

	return logs, ids, nil
}

func (b *Buffer) autoClaim(ctx context.Context, consumer string, count int64) ([]model.Log, []string, error) {
	res, _, err := b.rdb.XAutoClaim(ctx, &redis.XAutoClaimArgs{
		Stream:   b.stream,
		Group:    b.group,
		Consumer: consumer,
		MinIdle:  stalePendingAfter,
		Start:    "0-0",
		Count:    count,
	}).Result()
	if err == redis.Nil {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("ingest: xautoclaim: %w", err)
	}
	return parseMessages(res)
}

func (b *Buffer) readGroup(ctx context.Context, consumer string, count int64, block time.Duration, streamID string) ([]model.Log, []string, error) {
	res, err := b.rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    b.group,
		Consumer: consumer,
		Streams:  []string{b.stream, streamID},
		Count:    count,
		Block:    block,
	}).Result()
	if err == redis.Nil {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("ingest: xreadgroup: %w", err)
	}
	var logs []model.Log
	var ids []string
	for _, stream := range res {
		batchLogs, batchIDs, err := parseMessages(stream.Messages)
		if err != nil {
			return nil, nil, err
		}
		logs = append(logs, batchLogs...)
		ids = append(ids, batchIDs...)
	}
	return logs, ids, nil
}

func parseMessages(msgs []redis.XMessage) ([]model.Log, []string, error) {
	var logs []model.Log
	var ids []string
	for _, msg := range msgs {
		raw, ok := msg.Values["log"].(string)
		if !ok {
			ids = append(ids, msg.ID)
			continue
		}
		var entry model.Log
		if err := json.Unmarshal([]byte(raw), &entry); err != nil {
			ids = append(ids, msg.ID)
			continue
		}
		logs = append(logs, entry)
		ids = append(ids, msg.ID)
	}
	return logs, ids, nil
}

func (b *Buffer) Ack(ctx context.Context, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := b.rdb.XAck(ctx, b.stream, b.group, ids...).Err(); err != nil {
		return fmt.Errorf("ingest: xack: %w", err)
	}
	return nil
}

func isBusyGroup(err error) bool {
	return err != nil && strings.HasPrefix(err.Error(), "BUSYGROUP")
}
