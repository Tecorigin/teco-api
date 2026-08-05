package service

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/setting"
	"github.com/google/uuid"
)

// Knowledge base third-party API errors
var (
	ErrKnowledgeBaseNotConfigured = errors.New("knowledge base service is not configured")
	ErrKnowledgeBaseUpstream      = errors.New("knowledge base upstream error")
)

const kbDebugBodyLimit = 16 * 1024

// KbApiResponse is the common response envelope of the third-party service.
type KbApiResponse struct {
	Code int            `json:"code"`
	Msg  string         `json:"msg"`
	Data map[string]any `json:"data"`
}

// The third-party returns {"code":200,"msg":"...","data":...}.

func kbBaseUrl() (string, error) {
	if !setting.IsKnowledgeBaseEnabled() {
		return "", ErrKnowledgeBaseNotConfigured
	}
	return strings.TrimRight(setting.KnowledgeBaseServiceConfig.BaseUrl, "/"), nil
}

// kbServiceRequest performs a service-to-service call (X-Api-Key) for /org/* and /user/* endpoints.
func kbServiceRequest(method, path string, query url.Values, body []byte) ([]byte, error) {
	base, err := kbBaseUrl()
	if err != nil {
		return nil, err
	}
	u := base + path
	if len(query) > 0 {
		u += "?" + query.Encode()
	}
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", setting.KnowledgeBaseServiceConfig.ApiKey)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return doKbRequest(req)
}

// kbUserRequest performs an HMAC-signed user request for /knowledge_base/* and /chat/* endpoints.
func kbUserRequest(method, path string, userKb *KbIdentity, body []byte, contentType string) ([]byte, error) {
	base, err := kbBaseUrl()
	if err != nil {
		return nil, err
	}
	u := base + path
	var reqBody io.Reader
	if body != nil {
		reqBody = bytes.NewReader(body)
	}
	req, err := http.NewRequest(method, u, reqBody)
	if err != nil {
		return nil, err
	}
	setKbAuthHeaders(req, userKb)
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	} else if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return doKbRequest(req)
}

// KbIdentity identifies a user on the knowledge base service for HMAC signing.
type KbIdentity struct {
	UserId string
	OrgId  string
	Role   string // "admin" or "user"
}

func setKbAuthHeaders(req *http.Request, id *KbIdentity) {
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	nonce := uuid.NewString()
	signString := fmt.Sprintf("%s:%s:%s:%s:%s", id.UserId, id.OrgId, id.Role, timestamp, nonce)
	mac := hmac.New(sha256.New, []byte(setting.KnowledgeBaseServiceConfig.ApiKey))
	mac.Write([]byte(signString))
	signature := hex.EncodeToString(mac.Sum(nil))

	req.Header.Set("X-User-Id", id.UserId)
	req.Header.Set("X-Org-Id", id.OrgId)
	req.Header.Set("X-User-Role", id.Role)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature", signature)
}

func doKbRequest(req *http.Request) ([]byte, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	trace := kbDebugRequest(req)
	resp, err := client.Do(req)
	if err != nil {
		kbDebugError(trace, err)
		return nil, err
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		kbDebugError(trace, err)
		return nil, err
	}
	kbDebugResponse(trace, resp.StatusCode, resp.Header, respBody, nil)
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		message := strings.TrimSpace(string(respBody))
		if len(message) > 1024 {
			message = message[:1024]
		}
		return nil, fmt.Errorf("%w: status %d: %s", ErrKnowledgeBaseUpstream, resp.StatusCode, message)
	}
	return respBody, nil
}

// ---------- Organization / User (service-to-service) ----------

// KbRegisterOrg registers an organization on the knowledge base service.
func KbRegisterOrg(orgId, orgName string) (map[string]interface{}, error) {
	payload, _ := common.Marshal(map[string]string{"org_id": orgId, "org_name": orgName})
	return kbServiceRequestMap(http.MethodPost, "/org/register", nil, payload)
}

// KbUnregisterOrg removes an organization and all of its data.
func KbUnregisterOrg(orgId string) error {
	payload, _ := common.Marshal(map[string]string{"org_id": orgId})
	_, err := kbServiceRequest(http.MethodPost, "/org/unregister", nil, payload)
	return err
}

// KbGetOrgInfo returns organization info.
func KbGetOrgInfo(orgId string) (map[string]interface{}, error) {
	q := url.Values{"org_id": []string{orgId}}
	return kbServiceRequestMap(http.MethodGet, "/org/info", q, nil)
}

// KbAddUser adds a user to an organization with a role.
func KbAddUser(userId, orgId, role string) error {
	payload, _ := common.Marshal(map[string]string{"user_id": userId, "org_id": orgId, "role": role})
	_, err := kbServiceRequest(http.MethodPost, "/user/add", nil, payload)
	return err
}

// KbRemoveUser removes a user from an organization.
func KbRemoveUser(userId, orgId string) error {
	payload, _ := common.Marshal(map[string]string{"user_id": userId, "org_id": orgId})
	_, err := kbServiceRequest(http.MethodPost, "/user/remove", nil, payload)
	return err
}

func kbServiceRequestMap(method, path string, query url.Values, body []byte) (map[string]interface{}, error) {
	respBody, err := kbServiceRequest(method, path, query, body)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}
	if err := common.Unmarshal(respBody, &envelope); err != nil {
		return nil, kbInvalidResponseError(path, respBody)
	}
	if envelope.Code != 200 {
		return nil, fmt.Errorf("%w: code %d: %s", ErrKnowledgeBaseUpstream, envelope.Code, envelope.Msg)
	}
	return envelope.Data, nil
}

// ---------- Knowledge base management (HMAC) ----------

func kbUserRequestMap(id *KbIdentity, path string, body interface{}) (map[string]interface{}, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = common.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	respBody, err := kbUserRequest(http.MethodPost, path, id, payload, "application/json")
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}
	if err := common.Unmarshal(respBody, &envelope); err != nil {
		return nil, kbInvalidResponseError(path, respBody)
	}
	if envelope.Code != 200 {
		return nil, fmt.Errorf("%w: code %d: %s", ErrKnowledgeBaseUpstream, envelope.Code, envelope.Msg)
	}
	return envelope.Data, nil
}

func kbUserRequestList(id *KbIdentity, path string, body interface{}) ([]interface{}, error) {
	var payload []byte
	var err error
	if body != nil {
		payload, err = common.Marshal(body)
		if err != nil {
			return nil, err
		}
	}
	respBody, err := kbUserRequest(http.MethodPost, path, id, payload, "application/json")
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Code int           `json:"code"`
		Msg  string        `json:"msg"`
		Data []interface{} `json:"data"`
	}
	if err := common.Unmarshal(respBody, &envelope); err != nil {
		return nil, kbInvalidResponseError(path, respBody)
	}
	if envelope.Code != 200 {
		return nil, fmt.Errorf("%w: code %d: %s", ErrKnowledgeBaseUpstream, envelope.Code, envelope.Msg)
	}
	return envelope.Data, nil
}

func kbInvalidResponseError(path string, response []byte) error {
	message := strings.TrimSpace(string(response))
	if len(message) > 1024 {
		message = message[:1024]
	}
	return fmt.Errorf("%w: %s returned an invalid JSON response: %s", ErrKnowledgeBaseUpstream, path, message)
}

// KbList lists knowledge bases.
func KbList(id *KbIdentity) ([]interface{}, error) {
	return kbUserRequestList(id, "/knowledge_base/list", nil)
}

// KbCreate creates a knowledge base with the given parameters.
func KbCreate(id *KbIdentity, params map[string]interface{}) error {
	_, err := kbUserRequestMap(id, "/knowledge_base/create", params)
	return err
}

// KbDelete deletes a knowledge base.
func KbDelete(id *KbIdentity, kbName string) error {
	_, err := kbUserRequestMap(id, "/knowledge_base/delete", map[string]string{"kb_name": kbName})
	return err
}

// KbClear clears all documents in a knowledge base.
func KbClear(id *KbIdentity, kbName string) error {
	_, err := kbUserRequestMap(id, "/knowledge_base/clear", map[string]string{"kb_name": kbName})
	return err
}

// KbListDocs lists documents in a knowledge base.
func KbListDocs(id *KbIdentity, kbName string) ([]interface{}, error) {
	return kbUserRequestList(id, "/knowledge_base/list_docs", map[string]string{"kb_name": kbName})
}

// KbDeleteDocs deletes documents from a knowledge base.
func KbDeleteDocs(id *KbIdentity, kbName string, docIds []string) error {
	_, err := kbUserRequestMap(id, "/knowledge_base/delete_docs", map[string]interface{}{"kb_name": kbName, "doc_ids": docIds})
	return err
}

// KbRunIndex triggers index building (graph / raptor).
func KbRunIndex(id *KbIdentity, kbName, indexType string) (map[string]interface{}, error) {
	return kbUserRequestMap(id, "/knowledge_base/run_index", map[string]string{"kb_name": kbName, "index_type": indexType})
}

// KbTaskStatus queries an async task status.
func KbTaskStatus(id *KbIdentity, taskId string) (map[string]interface{}, error) {
	return kbUserRequestMap(id, "/knowledge_base/task_status", map[string]string{"task_id": taskId})
}

// KbUploadDocs uploads documents (multipart) and returns the task info.
func KbUploadDocs(id *KbIdentity, kbName string, files []multipart.File, fileHeaders []*multipart.FileHeader) (map[string]interface{}, error) {
	base, err := kbBaseUrl()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	if err := writer.WriteField("kb_name", kbName); err != nil {
		return nil, err
	}
	for i, fh := range fileHeaders {
		part, err := writer.CreateFormFile("files", fh.Filename)
		if err != nil {
			return nil, err
		}
		if _, err := io.Copy(part, files[i]); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, base+"/knowledge_base/upload_docs", &buf)
	if err != nil {
		return nil, err
	}
	setKbAuthHeaders(req, id)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	kbDebugUploadRequest(req, kbName, fileHeaders)
	respBody, err := doKbRequest(req)
	if err != nil {
		return nil, err
	}
	var envelope struct {
		Code int                    `json:"code"`
		Msg  string                 `json:"msg"`
		Data map[string]interface{} `json:"data"`
	}
	if err := common.Unmarshal(respBody, &envelope); err != nil {
		return nil, err
	}
	if envelope.Code != 200 {
		return nil, fmt.Errorf("%w: code %d: %s", ErrKnowledgeBaseUpstream, envelope.Code, envelope.Msg)
	}
	return envelope.Data, nil
}

// ---------- Chat (HMAC, SSE) ----------

// KbCreateSession creates a chat session and returns its id.
func KbCreateSession(id *KbIdentity) (string, error) {
	data, err := kbUserRequestMap(id, "/chat/create_session", nil)
	if err != nil {
		return "", err
	}
	sessionId, _ := data["session_id"].(string)
	return sessionId, nil
}

// KbDeleteSession deletes a chat session.
func KbDeleteSession(id *KbIdentity, sessionId string) error {
	_, err := kbUserRequestMap(id, "/chat/delete_session", map[string]string{"session_id": sessionId})
	return err
}

// KbListSessions lists chat sessions.
func KbListSessions(id *KbIdentity) ([]interface{}, error) {
	return kbUserRequestList(id, "/chat/list_sessions", nil)
}

// KbSessionChat performs a chat request and returns the raw SSE response body
// along with the response for streaming. Caller is responsible for closing the body.
func KbSessionChat(id *KbIdentity, payload []byte) (*http.Response, error) {
	base, err := kbBaseUrl()
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, base+"/chat/session_chat", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	setKbAuthHeaders(req, id)
	req.Header.Set("Content-Type", "application/json")
	// No client timeout for streaming; rely on context cancellation upstream.
	client := &http.Client{}
	trace := kbDebugRequest(req)
	resp, err := client.Do(req)
	if err != nil {
		kbDebugError(trace, err)
		return nil, err
	}
	resp.Body = &kbDebugReadCloser{
		ReadCloser: resp.Body,
		trace:      trace,
		statusCode: resp.StatusCode,
		header:     resp.Header,
	}
	return resp, nil
}

type kbDebugTrace struct {
	id      string
	method  string
	url     string
	started time.Time
}

type kbDebugReadCloser struct {
	io.ReadCloser
	trace      kbDebugTrace
	statusCode int
	header     http.Header
	body       bytes.Buffer
	finished   bool
}

func (r *kbDebugReadCloser) Read(p []byte) (int, error) {
	n, err := r.ReadCloser.Read(p)
	if n > 0 && r.body.Len() < kbDebugBodyLimit {
		remaining := kbDebugBodyLimit - r.body.Len()
		if n > remaining {
			_, _ = r.body.Write(p[:remaining])
		} else {
			_, _ = r.body.Write(p[:n])
		}
	}
	if err != nil {
		r.finished = true
		if err == io.EOF {
			kbDebugResponse(r.trace, r.statusCode, r.header, r.body.Bytes(), nil)
		} else {
			kbDebugResponse(r.trace, r.statusCode, r.header, r.body.Bytes(), err)
		}
	}
	return n, err
}

func (r *kbDebugReadCloser) Close() error {
	err := r.ReadCloser.Close()
	if !r.finished {
		r.finished = true
		kbDebugResponse(r.trace, r.statusCode, r.header, r.body.Bytes(), err)
	}
	return err
}

func kbDebugRequest(req *http.Request) kbDebugTrace {
	trace := kbDebugTrace{
		id:      uuid.NewString(),
		method:  req.Method,
		url:     kbDebugURL(req.URL),
		started: time.Now(),
	}
	common.SysLog(fmt.Sprintf(
		"[knowledge-base][%s] request method=%s url=%s headers=%v body=%s",
		trace.id, trace.method, trace.url, kbDebugHeaders(req.Header), kbDebugRequestBody(req),
	))
	return trace
}

func kbDebugUploadRequest(req *http.Request, kbName string, fileHeaders []*multipart.FileHeader) {
	files := make([]string, 0, len(fileHeaders))
	for _, fileHeader := range fileHeaders {
		files = append(files, fmt.Sprintf("%s(%d bytes)", fileHeader.Filename, fileHeader.Size))
	}
	common.SysLog(fmt.Sprintf(
		"[knowledge-base] upload metadata url=%s kb_name=%q files=%v; multipart content is omitted",
		kbDebugURL(req.URL), kbName, files,
	))
}

func kbDebugResponse(trace kbDebugTrace, statusCode int, header http.Header, body []byte, readErr error) {
	message := fmt.Sprintf(
		"[knowledge-base][%s] response method=%s url=%s status=%d duration=%s headers=%v body=%s",
		trace.id,
		trace.method,
		trace.url,
		statusCode,
		time.Since(trace.started).Round(time.Millisecond),
		kbDebugHeaders(header),
		kbDebugBody(body),
	)
	if readErr != nil {
		message += fmt.Sprintf(" read_error=%q", readErr.Error())
	}
	common.SysLog(message)
}

func kbDebugError(trace kbDebugTrace, err error) {
	common.SysLog(fmt.Sprintf(
		"[knowledge-base][%s] request failed method=%s url=%s duration=%s error=%q",
		trace.id,
		trace.method,
		trace.url,
		time.Since(trace.started).Round(time.Millisecond),
		err.Error(),
	))
}

func kbDebugRequestBody(req *http.Request) string {
	if strings.HasPrefix(req.Header.Get("Content-Type"), "multipart/") {
		return fmt.Sprintf("<multipart %d bytes; raw file content omitted>", req.ContentLength)
	}
	if req.GetBody == nil {
		return "<empty>"
	}
	body, err := req.GetBody()
	if err != nil {
		return fmt.Sprintf("<unavailable: %v>", err)
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, kbDebugBodyLimit+1))
	if err != nil {
		return fmt.Sprintf("<unavailable: %v>", err)
	}
	return kbDebugBody(data)
}

func kbDebugHeaders(header http.Header) http.Header {
	masked := header.Clone()
	for _, name := range []string{"Authorization", "Cookie", "Set-Cookie", "X-Api-Key", "X-Nonce", "X-Signature"} {
		if masked.Get(name) != "" {
			masked.Set(name, "***")
		}
	}
	return masked
}

func kbDebugURL(requestURL *url.URL) string {
	if requestURL == nil {
		return ""
	}
	copyURL := *requestURL
	copyURL.User = nil
	return copyURL.String()
}

func kbDebugBody(body []byte) string {
	truncated := len(body) > kbDebugBodyLimit
	if truncated {
		body = body[:kbDebugBodyLimit]
	}
	value := strings.TrimSpace(string(body))
	if value == "" {
		value = "<empty>"
	}
	if truncated {
		value += "...<truncated>"
	}
	return value
}
