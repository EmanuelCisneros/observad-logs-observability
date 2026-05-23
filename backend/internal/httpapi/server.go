package httpapi

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/redis/go-redis/v9"

	"github.com/observa/observad/internal/ingest"
	"github.com/observa/observad/internal/realtime"
	"github.com/observa/observad/internal/store"
)

type Server struct {
	store       store.LogStore
	buffer      *ingest.Buffer
	hub         *realtime.Hub
	log         *slog.Logger
	apiKey      string
	corsOrigins []string
	engine      *gin.Engine
}

type Deps struct {
	Store           store.LogStore
	Buffer          *ingest.Buffer
	Hub             *realtime.Hub
	Logger          *slog.Logger
	APIKey          string
	CORSOrigins     []string
	IngestRateLimit int
	Redis           *redis.Client
}

func New(d Deps) *Server {
	gin.SetMode(gin.ReleaseMode)

	origins := d.CORSOrigins
	if len(origins) == 0 {
		origins = []string{"*"}
	}

	s := &Server{
		store:       d.Store,
		buffer:      d.Buffer,
		hub:         d.Hub,
		log:         d.Logger,
		apiKey:      d.APIKey,
		corsOrigins: origins,
	}

	registerIngestMetrics(d.Buffer)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(observability())
	r.Use(cors.New(cors.Config{
		AllowOrigins:     origins,
		AllowMethods:     []string{"GET", "POST", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization", "X-API-Key"},
		AllowCredentials: false,
		MaxAge:           12 * time.Hour,
	}))

	r.GET("/healthz", s.handleHealth)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	limiter := NewRateLimiter(d.Redis, d.IngestRateLimit, time.Second)
	auth := requireAPIKey(d.APIKey)

	v1 := r.Group("/api/v1")
	{
		v1.POST("/logs", auth, limiter.Middleware(), s.handleIngest)
		v1.GET("/logs/search", auth, s.handleSearch)
		v1.GET("/logs/stats", auth, s.handleStats)
		v1.GET("/logs/tail", auth, s.handleTail)
	}

	s.engine = r
	return s
}

func (s *Server) Handler() http.Handler { return s.engine }

func (s *Server) corsOriginsList() []string { return s.corsOrigins }
