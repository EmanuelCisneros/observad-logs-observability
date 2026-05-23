package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLogQueryClamp_DefaultsLimit(t *testing.T) {
	q := LogQuery{}
	q.Clamp()
	assert.Equal(t, DefaultLimit, q.Limit)
	assert.Equal(t, "desc", q.Order)
}

func TestLogQueryClamp_CapsLimit(t *testing.T) {
	q := LogQuery{Limit: 100_000}
	q.Clamp()
	assert.Equal(t, MaxLimit, q.Limit, "limit must be capped at MaxLimit")
}

func TestLogQueryClamp_NegativeOffset(t *testing.T) {
	q := LogQuery{Offset: -50}
	q.Clamp()
	assert.Equal(t, 0, q.Offset, "negative offset must clamp to 0")
}

func TestLogQueryClamp_PreservesAscOrder(t *testing.T) {
	q := LogQuery{Order: "asc"}
	q.Clamp()
	assert.Equal(t, "asc", q.Order)
}

func TestLogQueryClamp_NormalizesUnknownOrder(t *testing.T) {
	q := LogQuery{Order: "sideways"}
	q.Clamp()
	assert.Equal(t, "desc", q.Order, "unknown order must default to desc")
}
