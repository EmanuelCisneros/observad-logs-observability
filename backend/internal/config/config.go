package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	HTTPAddr        string
	LogLevel        string
	IngestAPIKey    string
	CORSOrigins     []string
	IngestRateLimit int

	ClickHouse ClickHouseConfig
	Redis      RedisConfig

	BatchSize     int
	FlushInterval time.Duration
	Workers       int
	Retention     time.Duration
}

type ClickHouseConfig struct {
	Addr     string
	Database string
	Username string
	Password string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
	Stream   string
}

func Load() (*Config, error) {
	c := &Config{
		HTTPAddr:     env("OBSERVA_HTTP_ADDR", ":8080"),
		LogLevel:     env("OBSERVA_LOG_LEVEL", "info"),
		IngestAPIKey: env("OBSERVA_INGEST_API_KEY", "dev-local-key"),
		CORSOrigins:  envCSV("OBSERVA_CORS_ORIGINS", "*"),

		ClickHouse: ClickHouseConfig{
			Addr:     env("OBSERVA_CH_ADDR", "127.0.0.1:9000"),
			Database: env("OBSERVA_CH_DB", "observa"),
			Username: env("OBSERVA_CH_USER", "default"),
			Password: env("OBSERVA_CH_PASSWORD", ""),
		},
		Redis: RedisConfig{
			Addr:     env("OBSERVA_REDIS_ADDR", "127.0.0.1:6379"),
			Password: env("OBSERVA_REDIS_PASSWORD", ""),
			Stream:   env("OBSERVA_REDIS_STREAM", "observa:ingest"),
		},
	}

	var err error
	if c.BatchSize, err = envInt("OBSERVA_BATCH_SIZE", 10_000); err != nil {
		return nil, err
	}
	if c.Workers, err = envInt("OBSERVA_WORKERS", 4); err != nil {
		return nil, err
	}
	if c.Redis.DB, err = envInt("OBSERVA_REDIS_DB", 0); err != nil {
		return nil, err
	}
	if c.FlushInterval, err = envDuration("OBSERVA_FLUSH_INTERVAL", time.Second); err != nil {
		return nil, err
	}
	if c.Retention, err = envDuration("OBSERVA_RETENTION", 30*24*time.Hour); err != nil {
		return nil, err
	}
	if c.IngestRateLimit, err = envInt("OBSERVA_INGEST_RATE_LIMIT", 1000); err != nil {
		return nil, err
	}

	if c.BatchSize < 1 {
		return nil, fmt.Errorf("config: OBSERVA_BATCH_SIZE must be >= 1, got %d", c.BatchSize)
	}
	if c.Workers < 1 {
		return nil, fmt.Errorf("config: OBSERVA_WORKERS must be >= 1, got %d", c.Workers)
	}
	return c, nil
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) (int, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be an integer: %w", key, err)
	}
	return n, nil
}

func envDuration(key string, def time.Duration) (time.Duration, error) {
	v := os.Getenv(key)
	if v == "" {
		return def, nil
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return 0, fmt.Errorf("config: %s must be a duration: %w", key, err)
	}
	return d, nil
}

func envCSV(key, def string) []string {
	raw := env(key, def)
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
