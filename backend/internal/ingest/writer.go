package ingest

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/observa/observad/internal/model"
	"github.com/observa/observad/internal/realtime"
	"github.com/observa/observad/internal/store"
)

type Writer struct {
	buf            *Buffer
	store          store.LogStore
	hub            *realtime.Hub
	log            *slog.Logger
	batchSize      int
	flushInterval  time.Duration
	workers        int
}

type WriterConfig struct {
	Buffer        *Buffer
	Store         store.LogStore
	Hub           *realtime.Hub
	Logger        *slog.Logger
	BatchSize     int
	FlushInterval time.Duration
	Workers       int
}

func NewWriter(c WriterConfig) (*Writer, error) {
	if c.Buffer == nil || c.Store == nil {
		return nil, fmt.Errorf("ingest: writer needs a buffer and a store")
	}
	if c.BatchSize < 1 {
		c.BatchSize = 1000
	}
	if c.FlushInterval <= 0 {
		c.FlushInterval = time.Second
	}
	if c.Workers < 1 {
		c.Workers = 1
	}
	return &Writer{
		buf:           c.Buffer,
		store:         c.Store,
		hub:           c.Hub,
		log:           c.Logger,
		batchSize:     c.BatchSize,
		flushInterval: c.FlushInterval,
		workers:       c.Workers,
	}, nil
}

func (w *Writer) Run(ctx context.Context) error {
	g, ctx := errgroup.WithContext(ctx)
	for i := 0; i < w.workers; i++ {
		consumer := fmt.Sprintf("worker-%d", i)
		g.Go(func() error { return w.worker(ctx, consumer) })
	}
	if err := g.Wait(); err != nil && err != context.Canceled {
		return err
	}
	return nil
}

func (w *Writer) worker(ctx context.Context, consumer string) error {
	ticker := time.NewTicker(w.flushInterval)
	defer ticker.Stop()

	batch := make([]model.Log, 0, w.batchSize)
	batchIDs := make([]string, 0, w.batchSize)

	flush := func() bool {
		if len(batch) == 0 {
			return true
		}
		fctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()

		if err := w.store.InsertBatch(fctx, batch); err != nil {
			w.logError("batch insert failed, will retry", err, len(batch))
			batch = batch[:0]
			batchIDs = batchIDs[:0]
			return false
		}
		if err := w.buf.Ack(fctx, batchIDs); err != nil {
			w.logError("ack failed after successful insert", err, len(batchIDs))
			batch = batch[:0]
			batchIDs = batchIDs[:0]
			return false
		}
		if w.hub != nil {
			now := time.Now().UTC()
			for i := range batch {
				recordIngestLag(now.Sub(batch[i].Timestamp))
			}
			w.hub.Broadcast(batch)
		}
		batch = batch[:0]
		batchIDs = batchIDs[:0]
		return true
	}

	claimWait := w.flushInterval / 4
	if claimWait < 50*time.Millisecond {
		claimWait = 50 * time.Millisecond
	}
	if claimWait > 500*time.Millisecond {
		claimWait = 500 * time.Millisecond
	}

	for {
		select {
		case <-ctx.Done():
			flush()
			return ctx.Err()
		case <-ticker.C:
			flush()
		default:
		}

		if len(batch) >= w.batchSize {
			if !flush() {
				time.Sleep(time.Second)
			}
			continue
		}

		want := int64(w.batchSize - len(batch))
		logs, ids, err := w.buf.Claim(ctx, consumer, want, claimWait)
		if err != nil {
			if ctx.Err() != nil {
				flush()
				return ctx.Err()
			}
			w.logError("claim from buffer failed", err, 0)
			time.Sleep(time.Second)
			continue
		}
		if len(logs) == 0 {
			select {
			case <-ctx.Done():
				flush()
				return ctx.Err()
			case <-ticker.C:
				flush()
			case <-time.After(claimWait):
			}
			continue
		}

		batch = append(batch, logs...)
		batchIDs = append(batchIDs, ids...)
		if len(batch) >= w.batchSize {
			flush()
		}
	}
}

func (w *Writer) logError(msg string, err error, n int) {
	if w.log == nil {
		return
	}
	w.log.Error(msg, slog.String("error", err.Error()), slog.Int("count", n))
}
