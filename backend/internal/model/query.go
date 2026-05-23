package model

import "time"

type LogQuery struct {
	From     time.Time
	To       time.Time
	Levels   []Level
	Services []string
	Search   string
	Limit    int
	Offset   int
	Order    string
}

const (
	DefaultLimit = 100
	MaxLimit     = 1000
)

func (q *LogQuery) Clamp() {
	if q.Limit <= 0 {
		q.Limit = DefaultLimit
	}
	if q.Limit > MaxLimit {
		q.Limit = MaxLimit
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.Order != "asc" {
		q.Order = "desc"
	}
}

type LogPage struct {
	Logs  []Log  `json:"logs"`
	Total uint64 `json:"total"`
}

type HistogramBucket struct {
	Bucket time.Time        `json:"bucket"`
	Counts map[Level]uint64 `json:"counts"`
}

type Stats struct {
	Total       uint64            `json:"total"`
	ByLevel     map[Level]uint64  `json:"by_level"`
	TopServices []ServiceCount    `json:"top_services"`
	Histogram   []HistogramBucket `json:"histogram"`
}

type ServiceCount struct {
	Service string `json:"service"`
	Count   uint64 `json:"count"`
}
