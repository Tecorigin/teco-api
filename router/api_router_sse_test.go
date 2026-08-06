package router

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestKnowledgeBaseSessionChatBypassesGzip(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	api := engine.Group("/api")
	api.Use(gzip.Gzip(
		gzip.DefaultCompression,
		gzip.WithExcludedPaths([]string{"/api/knowledge-base/chat/session_chat"}),
	))
	api.POST("/knowledge-base/chat/session_chat", func(c *gin.Context) {
		c.Header("Content-Type", "text/event-stream")
		c.String(http.StatusOK, "data: first\n\n")
	})

	request := httptest.NewRequest(http.MethodPost, "/api/knowledge-base/chat/session_chat", nil)
	request.Header.Set("Accept-Encoding", "gzip")
	response := httptest.NewRecorder()

	engine.ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Empty(t, response.Header().Get("Content-Encoding"))
	assert.Equal(t, "data: first\n\n", response.Body.String())
}
