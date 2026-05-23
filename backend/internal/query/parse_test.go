package query

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/observa/observad/internal/model"
)

var fixedNow = time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)

func TestParse_Empty(t *testing.T) {
	q, err := Parse(Params{}, fixedNow)
	require.NoError(t, err)

	assert.True(t, q.From.IsZero())
	assert.True(t, q.To.IsZero())
	assert.Empty(t, q.Levels)
	assert.Empty(t, q.Services)
	assert.Equal(t, model.DefaultLimit, q.Limit)
	assert.Equal(t, "desc", q.Order)
}

func TestParse_RelativeFrom(t *testing.T) {
	q, err := Parse(Params{From: "15m"}, fixedNow)
	require.NoError(t, err)
	assert.Equal(t, fixedNow.Add(-15*time.Minute), q.From)
}

func TestParse_RelativeFromHours(t *testing.T) {
	q, err := Parse(Params{From: "24h"}, fixedNow)
	require.NoError(t, err)
	assert.Equal(t, fixedNow.Add(-24*time.Hour), q.From)
}

func TestParse_AbsoluteRFC3339(t *testing.T) {
	q, err := Parse(Params{From: "2024-05-01T00:00:00Z"}, fixedNow)
	require.NoError(t, err)
	assert.Equal(t, time.Date(2024, 5, 1, 0, 0, 0, 0, time.UTC), q.From)
}

func TestParse_InvalidFrom(t *testing.T) {
	_, err := Parse(Params{From: "not-a-time"}, fixedNow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "from")
}

func TestParse_FromAfterToRejected(t *testing.T) {
	_, err := Parse(Params{
		From: "2024-06-01T12:00:00Z",
		To:   "2024-06-01T11:00:00Z",
	}, fixedNow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "after")
}

func TestParse_Levels(t *testing.T) {
	q, err := Parse(Params{Levels: "error,warn"}, fixedNow)
	require.NoError(t, err)
	assert.ElementsMatch(t, []model.Level{model.LevelError, model.LevelWarn}, q.Levels)
}

func TestParse_LevelsWithWhitespace(t *testing.T) {
	q, err := Parse(Params{Levels: " error , info "}, fixedNow)
	require.NoError(t, err)
	assert.ElementsMatch(t, []model.Level{model.LevelError, model.LevelInfo}, q.Levels)
}

func TestParse_InvalidLevel(t *testing.T) {
	_, err := Parse(Params{Levels: "error,verbose"}, fixedNow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "verbose")
}

func TestParse_Services(t *testing.T) {
	q, err := Parse(Params{Services: "api,worker,cron"}, fixedNow)
	require.NoError(t, err)
	assert.Equal(t, []string{"api", "worker", "cron"}, q.Services)
}

func TestParse_LimitAndOffset(t *testing.T) {
	q, err := Parse(Params{Limit: "250", Offset: "500"}, fixedNow)
	require.NoError(t, err)
	assert.Equal(t, 250, q.Limit)
	assert.Equal(t, 500, q.Offset)
}

func TestParse_LimitCapped(t *testing.T) {
	q, err := Parse(Params{Limit: "999999"}, fixedNow)
	require.NoError(t, err)
	assert.Equal(t, model.MaxLimit, q.Limit)
}

func TestParse_InvalidLimit(t *testing.T) {
	_, err := Parse(Params{Limit: "lots"}, fixedNow)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "limit")
}

func TestParse_SearchTrimmed(t *testing.T) {
	q, err := Parse(Params{Search: "  timeout  "}, fixedNow)
	require.NoError(t, err)
	assert.Equal(t, "timeout", q.Search)
}

func TestParse_OrderAsc(t *testing.T) {
	q, err := Parse(Params{Order: "ASC"}, fixedNow)
	require.NoError(t, err)
	assert.Equal(t, "asc", q.Order)
}

func TestSplitCSV(t *testing.T) {
	assert.Nil(t, splitCSV(""))
	assert.Nil(t, splitCSV("   "))
	assert.Equal(t, []string{"a"}, splitCSV("a"))
	assert.Equal(t, []string{"a", "b"}, splitCSV("a,b"))
	assert.Equal(t, []string{"a", "b"}, splitCSV(" a , , b "))
}
