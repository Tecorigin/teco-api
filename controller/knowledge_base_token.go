package controller

import (
	"bufio"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	relayhelper "github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// KbTokenChatRequest is the OpenAI-compatible request for knowledge base Q&A via token.
type KbTokenChatRequest struct {
	// KnowledgeBase is the target knowledge base name. Required.
	KnowledgeBase string `json:"knowledge_base"`
	// Query is the user question. Either query or messages may be used.
	Query string `json:"query"`
	// Messages is an OpenAI-style message list; the last user message is used as the query.
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Stream     *bool `json:"stream"`
	ReturnDocs bool  `json:"return_docs"`
	// SessionId optionally continues an existing conversation.
	SessionId string `json:"session_id"`
}

// KbTokenChat exposes knowledge base Q&A through an API token, similar to calling a model.
// It resolves the token owner's knowledge base identity and proxies the third-party chat.
func KbTokenChat(c *gin.Context) {
	if !setting.IsKnowledgeBaseEnabled() {
		abortKbToken(c, http.StatusBadRequest, "knowledge base feature is not configured")
		return
	}
	userId := c.GetInt("id")
	if userId == 0 {
		abortKbToken(c, http.StatusUnauthorized, "invalid token")
		return
	}
	identity, _, err := kbIdentityForUser(userId)
	if err != nil {
		abortKbToken(c, http.StatusInternalServerError, err.Error())
		return
	}
	if identity == nil {
		abortKbToken(c, http.StatusForbidden, "knowledge base not enabled for this user")
		return
	}

	var req KbTokenChatRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		abortKbToken(c, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.KnowledgeBase == "" {
		abortKbToken(c, http.StatusBadRequest, "knowledge_base is required")
		return
	}
	query := req.Query
	if query == "" {
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role == "user" && req.Messages[i].Content != "" {
				query = req.Messages[i].Content
				break
			}
		}
	}
	if query == "" {
		abortKbToken(c, http.StatusBadRequest, "query is required")
		return
	}

	// Resolve session: reuse provided session or create a temporary one.
	sessionId := req.SessionId
	ownSession := false
	if sessionId == "" {
		sessionId, err = service.KbCreateSession(identity)
		if err != nil {
			abortKbToken(c, http.StatusBadGateway, "failed to create session: "+err.Error())
			return
		}
		ownSession = true
	}
	if ownSession {
		defer func() {
			_ = service.KbDeleteSession(identity, sessionId)
		}()
	}

	stream := true
	if req.Stream != nil {
		stream = *req.Stream
	}
	payload, _ := common.Marshal(map[string]interface{}{
		"query":       query,
		"session_id":  sessionId,
		"stream":      stream,
		"return_docs": req.ReturnDocs,
	})

	resp, err := service.KbSessionChat(identity, payload)
	if err != nil {
		abortKbToken(c, http.StatusBadGateway, err.Error())
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		abortKbToken(c, resp.StatusCode, string(body))
		return
	}

	if stream {
		streamKbTokenChat(c, resp.Body, req.KnowledgeBase)
	} else {
		collectKbTokenChat(c, resp.Body, req.KnowledgeBase)
	}
}

func abortKbToken(c *gin.Context, status int, msg string) {
	c.JSON(status, gin.H{
		"error": gin.H{
			"message": msg,
			"type":    "knowledge_base_error",
		},
	})
	c.Abort()
}

// streamKbTokenChat converts the upstream SSE into OpenAI-style chat.completion.chunk events.
func streamKbTokenChat(c *gin.Context, body io.Reader, kbName string) {
	relayhelper.SetEventStreamHeaders(c)
	c.Writer.WriteHeader(http.StatusOK)
	flusher, _ := c.Writer.(http.Flusher)

	id := "kbchatcmpl-" + uuid.NewString()
	created := time.Now().Unix()
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if raw == "" {
			continue
		}
		var evt struct {
			Content   string        `json:"content"`
			Reference []interface{} `json:"reference"`
		}
		if err := common.Unmarshal([]byte(raw), &evt); err != nil {
			continue
		}
		chunk := buildKbChunk(id, created, kbName, evt.Content, evt.Reference)
		data, _ := common.Marshal(chunk)
		c.Writer.Write([]byte("data: "))
		c.Writer.Write(data)
		c.Writer.Write([]byte("\n\n"))
		if flusher != nil {
			flusher.Flush()
		}
	}
	if err := scanner.Err(); err != nil {
		common.SysError("knowledge base token chat stream read failed: " + err.Error())
	}
	c.Writer.Write([]byte("data: [DONE]\n\n"))
	if flusher != nil {
		flusher.Flush()
	}
}

// collectKbTokenChat reads the full upstream SSE and returns a single OpenAI-style response.
func collectKbTokenChat(c *gin.Context, body io.Reader, kbName string) {
	var content strings.Builder
	var references []interface{}
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		raw := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if raw == "" {
			continue
		}
		var evt struct {
			Content   string        `json:"content"`
			Reference []interface{} `json:"reference"`
		}
		if err := common.Unmarshal([]byte(raw), &evt); err != nil {
			continue
		}
		content.WriteString(evt.Content)
		if len(evt.Reference) > 0 {
			references = evt.Reference
		}
	}
	if err := scanner.Err(); err != nil {
		abortKbToken(c, http.StatusBadGateway, "knowledge base stream read failed: "+err.Error())
		return
	}
	id := "kbchatcmpl-" + uuid.NewString()
	resp := gin.H{
		"id":      id,
		"object":  "chat.completion",
		"created": time.Now().Unix(),
		"model":   kbName,
		"choices": []gin.H{
			{
				"index": 0,
				"message": gin.H{
					"role":    "assistant",
					"content": content.String(),
				},
				"finish_reason": "stop",
			},
		},
	}
	if len(references) > 0 {
		resp["references"] = references
	}
	c.JSON(http.StatusOK, resp)
}

func buildKbChunk(id string, created int64, kbName, content string, references []interface{}) gin.H {
	delta := gin.H{}
	if content != "" {
		delta["content"] = content
	}
	chunk := gin.H{
		"id":      id,
		"object":  "chat.completion.chunk",
		"created": created,
		"model":   kbName,
		"choices": []gin.H{
			{
				"index":         0,
				"delta":         delta,
				"finish_reason": nil,
			},
		},
	}
	if len(references) > 0 {
		chunk["references"] = references
	}
	return chunk
}
