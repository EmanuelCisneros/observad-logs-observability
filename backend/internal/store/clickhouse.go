package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"

	"github.com/observa/observad/internal/model"
)

type LogStore interface {
	InsertBatch(ctx context.Context, logs []model.Log) error
	Search(ctx context.Context, q model.LogQuery) (model.LogPage, error)
	Stats(ctx context.Context, q model.LogQuery) (model.Stats, error)
	Ping(ctx context.Context) error
	Close() error
}

type ClickHouse struct {
	conn driver.Conn
	db   string
}

func NewClickHouse(ctx context.Context, addr, db, user, pass string) (*ClickHouse, error) {
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{addr},
		Auth: clickhouse.Auth{
			Database: db,
			Username: user,
			Password: pass,
		},
		Settings: clickhouse.Settings{
			"max_execution_time": 30,
		},
		DialTimeout:     5 * time.Second,
		MaxOpenConns:    16,
		MaxIdleConns:    8,
		ConnMaxLifetime: time.Hour,
		Compression: &clickhouse.Compression{
			Method: clickhouse.CompressionLZ4,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("clickhouse: open: %w", err)
	}

	c := &ClickHouse{conn: conn, db: db}
	if err := c.Ping(ctx); err != nil {
		_ = conn.Close()
		return nil, err
	}
	return c, nil
}

func (c *ClickHouse) ApplyRetention(ctx context.Context, retention time.Duration) error {
	if retention <= 0 {
		return nil
	}
	sec := int(retention.Seconds())
	rollupSec := int((retention * 3).Seconds())

	for _, stmt := range []string{
		fmt.Sprintf(
			`ALTER TABLE %s.logs MODIFY TTL toDateTime(timestamp) + INTERVAL %d SECOND`,
			c.db, sec,
		),
		fmt.Sprintf(
			`ALTER TABLE %s.logs_by_minute MODIFY TTL bucket + INTERVAL %d SECOND`,
			c.db, rollupSec,
		),
	} {
		if err := c.conn.Exec(ctx, stmt); err != nil {
			return fmt.Errorf("clickhouse: apply retention: %w", err)
		}
	}
	return nil
}

func (c *ClickHouse) Ping(ctx context.Context) error {
	if err := c.conn.Ping(ctx); err != nil {
		return fmt.Errorf("clickhouse: ping: %w", err)
	}
	return nil
}

func (c *ClickHouse) Close() error { return c.conn.Close() }

func (c *ClickHouse) InsertBatch(ctx context.Context, logs []model.Log) error {
	if len(logs) == 0 {
		return nil
	}

	batch, err := c.conn.PrepareBatch(ctx, `
		INSERT INTO logs (
			id, timestamp, level, service, message,
			trace_id, host, attr_keys, attr_values
		)`)
	if err != nil {
		return fmt.Errorf("clickhouse: prepare batch: %w", err)
	}

	for i := range logs {
		l := &logs[i]
		keys, vals := splitAttrs(l.Attributes)
		if err := batch.Append(
			l.ID,
			l.Timestamp,
			string(l.Level),
			l.Service,
			l.Message,
			l.TraceID,
			l.Host,
			keys,
			vals,
		); err != nil {
			_ = batch.Abort()
			return fmt.Errorf("clickhouse: append row %d: %w", i, err)
		}
	}

	if err := batch.Send(); err != nil {
		return fmt.Errorf("clickhouse: send batch of %d: %w", len(logs), err)
	}
	return nil
}

func (c *ClickHouse) Search(ctx context.Context, q model.LogQuery) (model.LogPage, error) {
	q.Clamp()

	where, args := buildWhere(q)
	order := "DESC"
	if q.Order == "asc" {
		order = "ASC"
	}

	sql := fmt.Sprintf(`
		SELECT
			id, timestamp, level, service, message,
			trace_id, host, attr_keys, attr_values,
			count() OVER () AS total
		FROM logs FINAL
		%s
		ORDER BY timestamp %s
		LIMIT ? OFFSET ?`, where, order)
	args = append(args, q.Limit, q.Offset)

	rows, err := c.conn.Query(ctx, sql, args...)
	if err != nil {
		return model.LogPage{}, fmt.Errorf("clickhouse: search query: %w", err)
	}
	defer rows.Close()

	page := model.LogPage{Logs: make([]model.Log, 0, q.Limit)}
	for rows.Next() {
		var (
			l     model.Log
			lvl   string
			keys  []string
			vals  []string
			total uint64
		)
		if err := rows.Scan(
			&l.ID, &l.Timestamp, &lvl, &l.Service, &l.Message,
			&l.TraceID, &l.Host, &keys, &vals, &total,
		); err != nil {
			return model.LogPage{}, fmt.Errorf("clickhouse: scan log: %w", err)
		}
		l.Level = model.Level(lvl)
		l.Attributes = joinAttrs(keys, vals)
		page.Logs = append(page.Logs, l)
		page.Total = total
	}
	if err := rows.Err(); err != nil {
		return model.LogPage{}, fmt.Errorf("clickhouse: iterate logs: %w", err)
	}
	return page, nil
}

// Stats uses logs_by_minute unless a message search filter is set.
func (c *ClickHouse) Stats(ctx context.Context, q model.LogQuery) (model.Stats, error) {
	q.Clamp()
	if q.Search != "" {
		return c.statsFromRaw(ctx, q)
	}
	return c.statsFromRollup(ctx, q)
}

func (c *ClickHouse) statsFromRollup(ctx context.Context, q model.LogQuery) (model.Stats, error) {
	where, args := buildRollupWhere(q)

	stats := model.Stats{
		ByLevel:     map[model.Level]uint64{},
		TopServices: []model.ServiceCount{},
		Histogram:   []model.HistogramBucket{},
	}

	levelSQL := fmt.Sprintf(
		`SELECT level, sum(cnt) AS n FROM logs_by_minute %s GROUP BY level`, where)
	rows, err := c.conn.Query(ctx, levelSQL, args...)
	if err != nil {
		return model.Stats{}, fmt.Errorf("clickhouse: rollup level stats: %w", err)
	}
	for rows.Next() {
		var lvl string
		var n uint64
		if err := rows.Scan(&lvl, &n); err != nil {
			rows.Close()
			return model.Stats{}, fmt.Errorf("clickhouse: scan level stat: %w", err)
		}
		stats.ByLevel[model.Level(lvl)] = n
		stats.Total += n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return model.Stats{}, err
	}

	svcSQL := fmt.Sprintf(`
		SELECT service, sum(cnt) AS n
		FROM logs_by_minute %s
		GROUP BY service
		ORDER BY n DESC
		LIMIT 10`, where)
	rows, err = c.conn.Query(ctx, svcSQL, args...)
	if err != nil {
		return model.Stats{}, fmt.Errorf("clickhouse: rollup service stats: %w", err)
	}
	for rows.Next() {
		var sc model.ServiceCount
		if err := rows.Scan(&sc.Service, &sc.Count); err != nil {
			rows.Close()
			return model.Stats{}, fmt.Errorf("clickhouse: scan service stat: %w", err)
		}
		stats.TopServices = append(stats.TopServices, sc)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return model.Stats{}, err
	}

	histSQL := fmt.Sprintf(`
		SELECT bucket, level, sum(cnt) AS n
		FROM logs_by_minute %s
		GROUP BY bucket, level
		ORDER BY bucket ASC`, where)
	rows, err = c.conn.Query(ctx, histSQL, args...)
	if err != nil {
		return model.Stats{}, fmt.Errorf("clickhouse: rollup histogram: %w", err)
	}
	bucketIdx := map[time.Time]int{}
	for rows.Next() {
		var (
			bucket time.Time
			lvl    string
			n      uint64
		)
		if err := rows.Scan(&bucket, &lvl, &n); err != nil {
			rows.Close()
			return model.Stats{}, fmt.Errorf("clickhouse: scan histogram: %w", err)
		}
		idx, ok := bucketIdx[bucket]
		if !ok {
			idx = len(stats.Histogram)
			bucketIdx[bucket] = idx
			stats.Histogram = append(stats.Histogram, model.HistogramBucket{
				Bucket: bucket,
				Counts: map[model.Level]uint64{},
			})
		}
		stats.Histogram[idx].Counts[model.Level(lvl)] = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return model.Stats{}, err
	}

	return stats, nil
}

func (c *ClickHouse) statsFromRaw(ctx context.Context, q model.LogQuery) (model.Stats, error) {
	where, args := buildWhere(q)

	stats := model.Stats{
		ByLevel:     map[model.Level]uint64{},
		TopServices: []model.ServiceCount{},
		Histogram:   []model.HistogramBucket{},
	}

	levelSQL := fmt.Sprintf(`SELECT level, count() FROM logs FINAL %s GROUP BY level`, where)
	rows, err := c.conn.Query(ctx, levelSQL, args...)
	if err != nil {
		return model.Stats{}, fmt.Errorf("clickhouse: level stats: %w", err)
	}
	for rows.Next() {
		var lvl string
		var n uint64
		if err := rows.Scan(&lvl, &n); err != nil {
			rows.Close()
			return model.Stats{}, fmt.Errorf("clickhouse: scan level stat: %w", err)
		}
		stats.ByLevel[model.Level(lvl)] = n
		stats.Total += n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return model.Stats{}, err
	}

	svcSQL := fmt.Sprintf(`
		SELECT service, count() AS n
		FROM logs FINAL %s
		GROUP BY service
		ORDER BY n DESC
		LIMIT 10`, where)
	rows, err = c.conn.Query(ctx, svcSQL, args...)
	if err != nil {
		return model.Stats{}, fmt.Errorf("clickhouse: service stats: %w", err)
	}
	for rows.Next() {
		var sc model.ServiceCount
		if err := rows.Scan(&sc.Service, &sc.Count); err != nil {
			rows.Close()
			return model.Stats{}, fmt.Errorf("clickhouse: scan service stat: %w", err)
		}
		stats.TopServices = append(stats.TopServices, sc)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return model.Stats{}, err
	}

	histSQL := fmt.Sprintf(`
		SELECT toStartOfMinute(timestamp) AS bucket, level, count() AS n
		FROM logs FINAL %s
		GROUP BY bucket, level
		ORDER BY bucket ASC`, where)
	rows, err = c.conn.Query(ctx, histSQL, args...)
	if err != nil {
		return model.Stats{}, fmt.Errorf("clickhouse: histogram: %w", err)
	}
	bucketIdx := map[time.Time]int{}
	for rows.Next() {
		var (
			bucket time.Time
			lvl    string
			n      uint64
		)
		if err := rows.Scan(&bucket, &lvl, &n); err != nil {
			rows.Close()
			return model.Stats{}, fmt.Errorf("clickhouse: scan histogram: %w", err)
		}
		idx, ok := bucketIdx[bucket]
		if !ok {
			idx = len(stats.Histogram)
			bucketIdx[bucket] = idx
			stats.Histogram = append(stats.Histogram, model.HistogramBucket{
				Bucket: bucket,
				Counts: map[model.Level]uint64{},
			})
		}
		stats.Histogram[idx].Counts[model.Level(lvl)] = n
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return model.Stats{}, err
	}

	return stats, nil
}

func buildWhere(q model.LogQuery) (string, []any) {
	var (
		conds []string
		args  []any
	)

	if !q.From.IsZero() {
		conds = append(conds, "timestamp >= ?")
		args = append(args, q.From)
	}
	if !q.To.IsZero() {
		conds = append(conds, "timestamp <= ?")
		args = append(args, q.To)
	}
	if len(q.Levels) > 0 {
		levels := make([]string, len(q.Levels))
		for i, l := range q.Levels {
			levels[i] = string(l)
		}
		conds = append(conds, "level IN (?)")
		args = append(args, levels)
	}
	if len(q.Services) > 0 {
		conds = append(conds, "service IN (?)")
		args = append(args, q.Services)
	}
	if q.Search != "" {
		conds = append(conds, "positionCaseInsensitive(message, ?) > 0")
		args = append(args, q.Search)
	}

	if len(conds) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

func buildRollupWhere(q model.LogQuery) (string, []any) {
	var (
		conds []string
		args  []any
	)

	if !q.From.IsZero() {
		conds = append(conds, "bucket >= toStartOfMinute(?)")
		args = append(args, q.From)
	}
	if !q.To.IsZero() {
		conds = append(conds, "bucket <= toStartOfMinute(?)")
		args = append(args, q.To)
	}
	if len(q.Levels) > 0 {
		levels := make([]string, len(q.Levels))
		for i, l := range q.Levels {
			levels[i] = string(l)
		}
		conds = append(conds, "level IN (?)")
		args = append(args, levels)
	}
	if len(q.Services) > 0 {
		conds = append(conds, "service IN (?)")
		args = append(args, q.Services)
	}

	if len(conds) == 0 {
		return "", args
	}
	return "WHERE " + strings.Join(conds, " AND "), args
}

func splitAttrs(m map[string]string) (keys, vals []string) {
	if len(m) == 0 {
		return []string{}, []string{}
	}
	keys = make([]string, 0, len(m))
	vals = make([]string, 0, len(m))
	for k, v := range m {
		keys = append(keys, k)
		vals = append(vals, v)
	}
	return keys, vals
}

func joinAttrs(keys, vals []string) map[string]string {
	if len(keys) == 0 {
		return nil
	}
	m := make(map[string]string, len(keys))
	for i := range keys {
		if i < len(vals) {
			m[keys[i]] = vals[i]
		}
	}
	return m
}
