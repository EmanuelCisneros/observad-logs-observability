package httpapi

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/observa/observad/internal/model"
	"github.com/observa/observad/internal/query"
)

type ingestRequest struct {
	Logs []model.Log `json:"logs"`
}

type ingestResponse struct {
	Accepted int          `json:"accepted"`
	Rejected int          `json:"rejected"`
	Errors   []rejectLine `json:"errors,omitempty"`
}

type rejectLine struct {
	Index  int    `json:"index"`
	Reason string `json:"reason"`
}

const maxIngestBatch = 10_000

func (s *Server) handleIngest(c *gin.Context) {
	var req ingestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abortJSON(c, http.StatusBadRequest, "invalid JSON body: "+err.Error())
		return
	}
	if len(req.Logs) == 0 {
		abortJSON(c, http.StatusBadRequest, "batch contains no logs")
		return
	}
	if len(req.Logs) > maxIngestBatch {
		abortJSON(c, http.StatusRequestEntityTooLarge,
			"batch exceeds the 10000-log limit")
		return
	}

	now := time.Now().UTC()
	good := make([]model.Log, 0, len(req.Logs))
	resp := ingestResponse{}

	for i := range req.Logs {
		l := req.Logs[i]
		if err := l.Sanitize(now); err != nil {
			resp.Rejected++
			resp.Errors = append(resp.Errors, rejectLine{Index: i, Reason: err.Error()})
			continue
		}
		if l.ID == "" {
			l.ID = uuid.NewString()
		}
		good = append(good, l)
	}

	if len(good) > 0 {
		if err := s.buffer.Push(c.Request.Context(), good); err != nil {
			s.log.Error("failed to push batch to buffer", "error", err)
			abortJSON(c, http.StatusServiceUnavailable,
				"ingest buffer unavailable, retry shortly")
			return
		}
	}

	resp.Accepted = len(good)
	metrics.ingested.Add(float64(resp.Accepted))
	metrics.rejected.Add(float64(resp.Rejected))

	status := http.StatusAccepted
	if resp.Rejected > 0 {
		status = http.StatusMultiStatus
	}
	c.JSON(status, resp)
}

func (s *Server) handleSearch(c *gin.Context) {
	var p query.Params
	if err := c.ShouldBindQuery(&p); err != nil {
		abortJSON(c, http.StatusBadRequest, "invalid query params: "+err.Error())
		return
	}

	q, err := query.Parse(p, time.Now().UTC())
	if err != nil {
		abortJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	page, err := s.store.Search(c.Request.Context(), q)
	if err != nil {
		s.log.Error("search failed", "error", err)
		abortJSON(c, http.StatusInternalServerError, "search failed")
		return
	}
	c.JSON(http.StatusOK, page)
}

func (s *Server) handleStats(c *gin.Context) {
	var p query.Params
	if err := c.ShouldBindQuery(&p); err != nil {
		abortJSON(c, http.StatusBadRequest, "invalid query params: "+err.Error())
		return
	}

	q, err := query.Parse(p, time.Now().UTC())
	if err != nil {
		abortJSON(c, http.StatusBadRequest, err.Error())
		return
	}

	stats, err := s.store.Stats(c.Request.Context(), q)
	if err != nil {
		s.log.Error("stats failed", "error", err)
		abortJSON(c, http.StatusInternalServerError, "stats failed")
		return
	}
	c.JSON(http.StatusOK, stats)
}

func (s *Server) handleHealth(c *gin.Context) {
	ctx, cancel := contextWithTimeout(c, 2*time.Second)
	defer cancel()

	if err := s.store.Ping(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "degraded",
			"store":  "unreachable",
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func abortJSON(c *gin.Context, code int, msg string) {
	c.AbortWithStatusJSON(code, gin.H{"error": msg})
}

var errClientGone = errors.New("client disconnected")
