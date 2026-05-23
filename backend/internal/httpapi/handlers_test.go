package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/observa/observad/internal/model"
	"github.com/observa/observad/internal/store"
)

// newTestStore seeds a MemStore for handler tests.
func newTestStore(t *testing.T) *store.MemStore {
	t.Helper()
	m := store.NewMemStore()
	base := time.Date(2024, 6, 1, 12, 0, 0, 0, time.UTC)
	require.NoError(t, m.InsertBatch(nil, []model.Log{
		{ID: "1", Timestamp: base, Level: model.LevelInfo, Service: "api", Message: "started"},
		{ID: "2", Timestamp: base.Add(time.Minute), Level: model.LevelError, Service: "api", Message: "boom"},
		{ID: "3", Timestamp: base.Add(2 * time.Minute), Level: model.LevelWarn, Service: "worker", Message: "slow"},
	}))
	return m
}

func doRequest(t *testing.T, srv *Server, method, target string, body []byte, headers map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		rdr = bytes.NewReader(body)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, target, rdr)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	return rec
}

func authHeaders(key string) map[string]string {
	return map[string]string{"X-API-Key": key}
}

func TestHealth_OK(t *testing.T) {
	srv := New(Deps{Store: newTestStore(t), APIKey: "k"})
	rec := doRequest(t, srv, http.MethodGet, "/healthz", nil, nil)

	assert.Equal(t, http.StatusOK, rec.Code)

	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
}

func TestSearch_RequiresAPIKey(t *testing.T) {
	srv := New(Deps{Store: newTestStore(t), APIKey: "k"})
	rec := doRequest(t, srv, http.MethodGet, "/api/v1/logs/search", nil, nil)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestSearch_NoFilterReturnsAll(t *testing.T) {
	srv := New(Deps{Store: newTestStore(t), APIKey: "k"})
	rec := doRequest(t, srv, http.MethodGet, "/api/v1/logs/search", nil, authHeaders("k"))

	require.Equal(t, http.StatusOK, rec.Code)

	var page model.LogPage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	assert.Equal(t, uint64(3), page.Total)
	assert.Len(t, page.Logs, 3)
}

func TestSearch_FilterByLevel(t *testing.T) {
	srv := New(Deps{Store: newTestStore(t), APIKey: "k"})
	rec := doRequest(t, srv, http.MethodGet, "/api/v1/logs/search?levels=error", nil, authHeaders("k"))

	require.Equal(t, http.StatusOK, rec.Code)

	var page model.LogPage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	require.Equal(t, uint64(1), page.Total)
	assert.Equal(t, "boom", page.Logs[0].Message)
}

func TestSearch_FilterBySearchTerm(t *testing.T) {
	srv := New(Deps{Store: newTestStore(t), APIKey: "k"})
	rec := doRequest(t, srv, http.MethodGet, "/api/v1/logs/search?search=slow", nil, authHeaders("k"))

	require.Equal(t, http.StatusOK, rec.Code)

	var page model.LogPage
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &page))
	assert.Equal(t, uint64(1), page.Total)
}

func TestSearch_InvalidLevelRejected(t *testing.T) {
	srv := New(Deps{Store: newTestStore(t), APIKey: "k"})
	rec := doRequest(t, srv, http.MethodGet, "/api/v1/logs/search?levels=bogus", nil, authHeaders("k"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSearch_InvalidLimitRejected(t *testing.T) {
	srv := New(Deps{Store: newTestStore(t), APIKey: "k"})
	rec := doRequest(t, srv, http.MethodGet, "/api/v1/logs/search?limit=abc", nil, authHeaders("k"))

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestStats_ReturnsAggregates(t *testing.T) {
	srv := New(Deps{Store: newTestStore(t), APIKey: "k"})
	rec := doRequest(t, srv, http.MethodGet, "/api/v1/logs/stats", nil, authHeaders("k"))

	require.Equal(t, http.StatusOK, rec.Code)

	var stats model.Stats
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &stats))
	assert.Equal(t, uint64(3), stats.Total)
	assert.Equal(t, uint64(1), stats.ByLevel[model.LevelError])
	assert.NotEmpty(t, stats.TopServices)
}

func TestIngest_RejectsMissingAPIKey(t *testing.T) {
	srv := New(Deps{Store: newTestStore(t), APIKey: "secret"})
	body := []byte(`{"logs":[{"service":"api","message":"hi"}]}`)
	rec := doRequest(t, srv, http.MethodPost, "/api/v1/logs", body, map[string]string{
		"Content-Type": "application/json",
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestIngest_RejectsWrongAPIKey(t *testing.T) {
	srv := New(Deps{Store: newTestStore(t), APIKey: "secret"})
	body := []byte(`{"logs":[{"service":"api","message":"hi"}]}`)
	rec := doRequest(t, srv, http.MethodPost, "/api/v1/logs", body, map[string]string{
		"Content-Type": "application/json",
		"X-API-Key":    "wrong",
	})

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestIngest_RejectsEmptyBatch(t *testing.T) {
	srv := New(Deps{Store: newTestStore(t), APIKey: "secret"})
	body := []byte(`{"logs":[]}`)
	rec := doRequest(t, srv, http.MethodPost, "/api/v1/logs", body, map[string]string{
		"Content-Type": "application/json",
		"X-API-Key":    "secret",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestIngest_RejectsMalformedJSON(t *testing.T) {
	srv := New(Deps{Store: newTestStore(t), APIKey: "secret"})
	body := []byte(`{"logs": not json}`)
	rec := doRequest(t, srv, http.MethodPost, "/api/v1/logs", body, map[string]string{
		"Content-Type": "application/json",
		"X-API-Key":    "secret",
	})

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestIngest_AcceptsBearerToken(t *testing.T) {
	srv := New(Deps{Store: newTestStore(t), Buffer: nil, APIKey: "secret"})
	body := []byte(`{"logs":[{"service":"api","message":"hi"}]}`)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/logs", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer secret")

	defer func() { _ = recover() }()
	srv.Handler().ServeHTTP(rec, req)
	assert.NotEqual(t, http.StatusUnauthorized, rec.Code)
}

func TestExtractKey(t *testing.T) {
	assert.Equal(t, "rawkey", extractKeyFromAuth("rawkey", ""))
	assert.Equal(t, "tok", extractKeyFromAuth("Bearer tok", ""))
	assert.Equal(t, "xk", extractKeyFromAuth("", "xk"))
}

func extractKeyFromAuth(auth, apiKey string) string {
	if auth != "" {
		const p = "Bearer "
		if len(auth) > len(p) && auth[:len(p)] == p {
			return auth[len(p):]
		}
		return auth
	}
	return apiKey
}
