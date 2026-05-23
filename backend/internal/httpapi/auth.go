package httpapi

import (
	"crypto/subtle"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func requireAPIKey(expected string) gin.HandlerFunc {
	expectedBytes := []byte(expected)
	return func(c *gin.Context) {
		if !checkAPIKey(c, expectedBytes) {
			return
		}
		c.Next()
	}
}

func checkAPIKey(c *gin.Context, expected []byte) bool {
	got := extractKey(c)
	if got == "" {
		abortJSON(c, http.StatusUnauthorized, "missing API key")
		return false
	}
	if subtle.ConstantTimeCompare([]byte(got), expected) != 1 {
		abortJSON(c, http.StatusUnauthorized, "invalid API key")
		return false
	}
	return true
}

func extractKey(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); h != "" {
		if after, ok := strings.CutPrefix(h, "Bearer "); ok {
			return strings.TrimSpace(after)
		}
		return strings.TrimSpace(h)
	}
	if k := strings.TrimSpace(c.GetHeader("X-API-Key")); k != "" {
		return k
	}
	return strings.TrimSpace(c.Query("api_key"))
}

func allowedOrigin(origin string, allowed []string) bool {
	if origin == "" {
		return true
	}
	for _, o := range allowed {
		if o == "*" || o == origin {
			return true
		}
	}
	return false
}
