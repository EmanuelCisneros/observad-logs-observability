-- Observa — ClickHouse schema
-- Applied automatically by the clickhouse container on first boot
-- (see deploy/docker-compose.yml volume mount).

CREATE DATABASE IF NOT EXISTS observa;

-- Main log table.
--
-- ReplacingMergeTree deduplicates rows with the same ORDER BY key (id) on
-- background merge, making re-inserts after a failed Redis ACK idempotent.
CREATE TABLE IF NOT EXISTS observa.logs
(
    id          String,
    timestamp   DateTime64(3, 'UTC') CODEC(DoubleDelta, ZSTD(1)),
    level       LowCardinality(String),
    service     LowCardinality(String),
    message     String CODEC(ZSTD(3)),
    trace_id    String CODEC(ZSTD(1)),
    host        LowCardinality(String),
    attr_keys   Array(String) CODEC(ZSTD(1)),
    attr_values Array(String) CODEC(ZSTD(1)),

    INDEX idx_message message TYPE tokenbf_v1(32768, 3, 0) GRANULARITY 4
)
ENGINE = ReplacingMergeTree()
PARTITION BY toDate(timestamp)
ORDER BY (id)
TTL toDateTime(timestamp) + INTERVAL 30 DAY
SETTINGS index_granularity = 8192;

-- Per-minute rollup used by the stats API (histogram + top services).
CREATE TABLE IF NOT EXISTS observa.logs_by_minute
(
    bucket   DateTime('UTC'),
    service  LowCardinality(String),
    level    LowCardinality(String),
    cnt      UInt64
)
ENGINE = SummingMergeTree
PARTITION BY toDate(bucket)
ORDER BY (service, level, bucket)
TTL bucket + INTERVAL 90 DAY;

CREATE MATERIALIZED VIEW IF NOT EXISTS observa.logs_by_minute_mv
TO observa.logs_by_minute
AS
SELECT
    toStartOfMinute(timestamp) AS bucket,
    service,
    level,
    count() AS cnt
FROM observa.logs
GROUP BY bucket, service, level;
