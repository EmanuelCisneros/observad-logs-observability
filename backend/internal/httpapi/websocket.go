package httpapi

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/observa/observad/internal/model"
	"github.com/observa/observad/internal/realtime"
)

const (
	wsWriteWait    = 10 * time.Second
	wsPingInterval = 30 * time.Second
)

func (s *Server) handleTail(c *gin.Context) {
	if !checkAPIKey(c, []byte(s.apiKey)) {
		return
	}

	filter := realtime.Filter{
		Service: c.Query("service"),
		Level:   model.Level(c.Query("level")),
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 4096,
		CheckOrigin: func(r *http.Request) bool {
			return allowedOrigin(r.Header.Get("Origin"), s.corsOriginsList())
		},
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.log.Debug("websocket upgrade failed", "error", err)
		return
	}
	defer conn.Close()

	logCh, cancel := s.hub.Subscribe(filter)
	defer cancel()

	closed := make(chan struct{})
	go func() {
		defer close(closed)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	ping := time.NewTicker(wsPingInterval)
	defer ping.Stop()

	for {
		select {
		case <-closed:
			return
		case <-c.Request.Context().Done():
			return
		case l, ok := <-logCh:
			if !ok {
				return
			}
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteJSON(l); err != nil {
				return
			}
		case <-ping.C:
			_ = conn.SetWriteDeadline(time.Now().Add(wsWriteWait))
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func contextWithTimeout(c *gin.Context, d time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(c.Request.Context(), d)
}
