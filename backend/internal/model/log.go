package model

import (
	"errors"
	"strings"
	"time"
)

type Level string

const (
	LevelDebug Level = "debug"
	LevelInfo  Level = "info"
	LevelWarn  Level = "warn"
	LevelError Level = "error"
	LevelFatal Level = "fatal"
)

var knownLevels = map[Level]struct{}{
	LevelDebug: {}, LevelInfo: {}, LevelWarn: {}, LevelError: {}, LevelFatal: {},
}

func NormalizeLevel(raw string) Level {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "debug", "trace", "dbg":
		return LevelDebug
	case "info", "information", "notice":
		return LevelInfo
	case "warn", "warning":
		return LevelWarn
	case "error", "err":
		return LevelError
	case "fatal", "crit", "critical", "panic", "emerg":
		return LevelFatal
	default:
		return LevelInfo
	}
}

func (l Level) Valid() bool {
	_, ok := knownLevels[l]
	return ok
}

type Log struct {
	ID         string            `json:"id"`
	Timestamp  time.Time         `json:"timestamp"`
	Level      Level             `json:"level"`
	Service    string            `json:"service"`
	Message    string            `json:"message"`
	TraceID    string            `json:"trace_id,omitempty"`
	Host       string            `json:"host,omitempty"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

const (
	maxMessageBytes = 64 * 1024
	maxServiceLen   = 256
	maxAttributes   = 64
)

var (
	errEmptyMessage  = errors.New("log: message must not be empty")
	errEmptyService  = errors.New("log: service must not be empty")
	errMessageTooBig = errors.New("log: message exceeds 64KiB limit")
	errServiceTooBig = errors.New("log: service name exceeds 256 chars")
	errTooManyAttrs  = errors.New("log: too many attributes (max 64)")
)

func (l *Log) Sanitize(now time.Time) error {
	l.Message = strings.TrimSpace(l.Message)
	l.Service = strings.TrimSpace(l.Service)

	if l.Message == "" {
		return errEmptyMessage
	}
	if l.Service == "" {
		return errEmptyService
	}
	if len(l.Message) > maxMessageBytes {
		return errMessageTooBig
	}
	if len(l.Service) > maxServiceLen {
		return errServiceTooBig
	}
	if len(l.Attributes) > maxAttributes {
		return errTooManyAttrs
	}

	if l.Timestamp.IsZero() {
		l.Timestamp = now
	}
	if l.Timestamp.After(now.Add(1 * time.Hour)) {
		l.Timestamp = now
	}

	if !l.Level.Valid() {
		l.Level = NormalizeLevel(string(l.Level))
	}
	return nil
}
