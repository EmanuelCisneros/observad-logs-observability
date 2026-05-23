package model

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeLevel(t *testing.T) {
	cases := []struct {
		in   string
		want Level
	}{
		{"debug", LevelDebug},
		{"DEBUG", LevelDebug},
		{"trace", LevelDebug},
		{"  info  ", LevelInfo},
		{"information", LevelInfo},
		{"warn", LevelWarn},
		{"WARNING", LevelWarn},
		{"error", LevelError},
		{"err", LevelError},
		{"fatal", LevelFatal},
		{"critical", LevelFatal},
		{"panic", LevelFatal},
		{"", LevelInfo},         // unknown -> info
		{"nonsense", LevelInfo}, // unknown -> info
	}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			assert.Equal(t, tc.want, NormalizeLevel(tc.in))
		})
	}
}

func TestLevelValid(t *testing.T) {
	assert.True(t, LevelInfo.Valid())
	assert.True(t, LevelError.Valid())
	assert.False(t, Level("verbose").Valid())
	assert.False(t, Level("").Valid())
}

func TestSanitize_FillsDefaults(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	l := Log{Message: "something happened", Service: "api"}

	require.NoError(t, l.Sanitize(now))

	assert.Equal(t, now, l.Timestamp, "zero timestamp should default to now")
	assert.Equal(t, LevelInfo, l.Level, "empty level should default to info")
}

func TestSanitize_TrimsWhitespace(t *testing.T) {
	now := time.Now()
	l := Log{Message: "  padded message  ", Service: "  api  ", Level: LevelInfo}

	require.NoError(t, l.Sanitize(now))

	assert.Equal(t, "padded message", l.Message)
	assert.Equal(t, "api", l.Service)
}

func TestSanitize_RejectsEmptyMessage(t *testing.T) {
	l := Log{Message: "   ", Service: "api"}
	err := l.Sanitize(time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "message")
}

func TestSanitize_RejectsEmptyService(t *testing.T) {
	l := Log{Message: "hello", Service: ""}
	err := l.Sanitize(time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "service")
}

func TestSanitize_RejectsOversizedMessage(t *testing.T) {
	l := Log{
		Message: strings.Repeat("x", maxMessageBytes+1),
		Service: "api",
	}
	err := l.Sanitize(time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "64KiB")
}

func TestSanitize_RejectsTooManyAttributes(t *testing.T) {
	attrs := make(map[string]string, maxAttributes+1)
	for i := 0; i <= maxAttributes; i++ {
		attrs[string(rune('a'+i%26))+time.Now().String()+string(rune(i))] = "v"
	}
	l := Log{Message: "hello", Service: "api", Attributes: attrs}
	err := l.Sanitize(time.Now())
	require.Error(t, err)
	assert.Contains(t, err.Error(), "attributes")
}

func TestSanitize_ClampsFutureTimestamp(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	farFuture := now.Add(48 * time.Hour)
	l := Log{Message: "hello", Service: "api", Timestamp: farFuture}

	require.NoError(t, l.Sanitize(now))

	assert.Equal(t, now, l.Timestamp,
		"timestamp more than 1h in the future should be clamped to now")
}

func TestSanitize_KeepsRecentTimestamp(t *testing.T) {
	now := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	recent := now.Add(-5 * time.Minute)
	l := Log{Message: "hello", Service: "api", Timestamp: recent}

	require.NoError(t, l.Sanitize(now))

	assert.Equal(t, recent, l.Timestamp, "a valid past timestamp must be preserved")
}

func TestSanitize_NormalizesUnknownLevel(t *testing.T) {
	l := Log{Message: "hello", Service: "api", Level: Level("WARNING")}
	require.NoError(t, l.Sanitize(time.Now()))
	assert.Equal(t, LevelWarn, l.Level)
}
