package controller

import (
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	relayhelper "github.com/QuantumNous/new-api/relay/helper"
	"github.com/QuantumNous/new-api/service"
	"github.com/QuantumNous/new-api/setting"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// kbIdentityForUser resolves the knowledge base identity (org + kb user) for a new-api user.
// Returns nil if the user has not enabled the knowledge base feature.
func kbIdentityForUser(userId int) (*service.KbIdentity, *model.UserKnowledgeBase, error) {
	kb, err := model.GetUserKnowledgeBase(userId)
	if err != nil {
		return nil, nil, err
	}
	if kb == nil {
		return nil, nil, nil
	}
	return &service.KbIdentity{
		UserId: kb.KbUserId,
		OrgId:  kb.OrgId,
		Role:   "admin", // every enabled user is admin of their own org
	}, kb, nil
}

// GetKnowledgeBaseStatus returns whether the feature is configured and whether the
// current user has enabled it.
func GetKnowledgeBaseStatus(c *gin.Context) {
	if !setting.IsKnowledgeBaseEnabled() {
		common.ApiSuccess(c, gin.H{"configured": false, "enabled": false})
		return
	}
	userId := c.GetInt("id")
	kb, err := model.GetUserKnowledgeBase(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	data := gin.H{"configured": true, "enabled": kb != nil}
	if kb != nil {
		data["org_id"] = kb.OrgId
		data["org_name"] = kb.OrgName
		data["created_at"] = kb.CreatedAt
	}
	common.ApiSuccess(c, data)
}

// EnableKnowledgeBase provisions a knowledge base organization for the current user.
func EnableKnowledgeBase(c *gin.Context) {
	if !setting.IsKnowledgeBaseEnabled() {
		common.ApiErrorMsg(c, "知识库功能未配置")
		return
	}
	userId := c.GetInt("id")
	existing, err := model.GetUserKnowledgeBase(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if existing != nil {
		common.ApiSuccess(c, gin.H{"org_id": existing.OrgId, "org_name": existing.OrgName})
		return
	}

	user, err := model.GetUserById(userId, false)
	if err != nil {
		common.ApiError(c, err)
		return
	}

	orgId := "newapi_org_" + uuid.NewString()
	kbUserId := "newapi_user_" + uuid.NewString()
	orgName := fmt.Sprintf("%s 的知识库", user.Username)
	if orgName == " 的知识库" {
		orgName = fmt.Sprintf("user-%d 的知识库", userId)
	}

	// Register org, then add the user as admin. If user add fails, roll back the org.
	if _, err := service.KbRegisterOrg(orgId, orgName); err != nil {
		common.ApiError(c, fmt.Errorf("注册知识库组织失败: %w", err))
		return
	}
	if err := service.KbAddUser(kbUserId, orgId, "admin"); err != nil {
		_ = service.KbUnregisterOrg(orgId)
		common.ApiError(c, fmt.Errorf("添加知识库用户失败: %w", err))
		return
	}

	record := &model.UserKnowledgeBase{
		UserId:   userId,
		OrgId:    orgId,
		OrgName:  orgName,
		KbUserId: kbUserId,
		Status:   "active",
	}
	if err := model.CreateUserKnowledgeBase(record); err != nil {
		_ = service.KbUnregisterOrg(orgId)
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"org_id": orgId, "org_name": orgName})
}

// DisableKnowledgeBase removes the user's knowledge base organization and all its data.
func DisableKnowledgeBase(c *gin.Context) {
	userId := c.GetInt("id")
	kb, err := model.GetUserKnowledgeBase(userId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	if kb == nil {
		common.ApiErrorMsg(c, "尚未开通知识库")
		return
	}
	if err := service.KbUnregisterOrg(kb.OrgId); err != nil {
		common.ApiError(c, fmt.Errorf("注销知识库组织失败: %w", err))
		return
	}
	if err := model.DeleteUserKnowledgeBase(userId); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

// requireKbIdentity resolves the caller's knowledge base identity or aborts with an error.
func requireKbIdentity(c *gin.Context) *service.KbIdentity {
	userId := c.GetInt("id")
	identity, _, err := kbIdentityForUser(userId)
	if err != nil {
		common.ApiError(c, err)
		c.Abort()
		return nil
	}
	if identity == nil {
		common.ApiErrorMsg(c, "请先开通知识库")
		c.Abort()
		return nil
	}
	return identity
}

func KbList(c *gin.Context) {
	identity := requireKbIdentity(c)
	if identity == nil {
		return
	}
	data, err := service.KbList(identity)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func KbCreate(c *gin.Context) {
	identity := requireKbIdentity(c)
	if identity == nil {
		return
	}
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		common.ApiErrorMsg(c, "请求参数错误: "+err.Error())
		return
	}
	if _, ok := params["kb_name"]; !ok {
		common.ApiErrorMsg(c, "缺少 kb_name")
		return
	}
	if err := service.KbCreate(identity, params); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func KbDelete(c *gin.Context) {
	identity := requireKbIdentity(c)
	if identity == nil {
		return
	}
	var req struct {
		KbName string `json:"kb_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.KbName == "" {
		common.ApiErrorMsg(c, "缺少 kb_name")
		return
	}
	if err := service.KbDelete(identity, req.KbName); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func KbClear(c *gin.Context) {
	identity := requireKbIdentity(c)
	if identity == nil {
		return
	}
	var req struct {
		KbName string `json:"kb_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.KbName == "" {
		common.ApiErrorMsg(c, "缺少 kb_name")
		return
	}
	if err := service.KbClear(identity, req.KbName); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func KbListDocs(c *gin.Context) {
	identity := requireKbIdentity(c)
	if identity == nil {
		return
	}
	var req struct {
		KbName string `json:"kb_name"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.KbName == "" {
		common.ApiErrorMsg(c, "缺少 kb_name")
		return
	}
	data, err := service.KbListDocs(identity, req.KbName)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func KbDeleteDocs(c *gin.Context) {
	identity := requireKbIdentity(c)
	if identity == nil {
		return
	}
	var req struct {
		KbName string   `json:"kb_name"`
		DocIds []string `json:"doc_ids"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.KbName == "" || len(req.DocIds) == 0 {
		common.ApiErrorMsg(c, "缺少 kb_name 或 doc_ids")
		return
	}
	if err := service.KbDeleteDocs(identity, req.KbName, req.DocIds); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func KbRunIndex(c *gin.Context) {
	identity := requireKbIdentity(c)
	if identity == nil {
		return
	}
	var req struct {
		KbName    string `json:"kb_name"`
		IndexType string `json:"index_type"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.KbName == "" || req.IndexType == "" {
		common.ApiErrorMsg(c, "缺少 kb_name 或 index_type")
		return
	}
	data, err := service.KbRunIndex(identity, req.KbName, req.IndexType)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func KbTaskStatus(c *gin.Context) {
	identity := requireKbIdentity(c)
	if identity == nil {
		return
	}
	var req struct {
		TaskId string `json:"task_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.TaskId == "" {
		common.ApiErrorMsg(c, "缺少 task_id")
		return
	}
	data, err := service.KbTaskStatus(identity, req.TaskId)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

func KbUploadDocs(c *gin.Context) {
	identity := requireKbIdentity(c)
	if identity == nil {
		return
	}
	kbName := c.PostForm("kb_name")
	if kbName == "" {
		common.ApiErrorMsg(c, "缺少 kb_name")
		return
	}
	form, err := c.MultipartForm()
	if err != nil {
		common.ApiErrorMsg(c, "解析表单失败: "+err.Error())
		return
	}
	fileHeaders := form.File["files"]
	if len(fileHeaders) == 0 {
		common.ApiErrorMsg(c, "缺少 files")
		return
	}
	files := make([]multipart.File, 0, len(fileHeaders))
	defer func() {
		for _, f := range files {
			f.Close()
		}
	}()
	for _, fh := range fileHeaders {
		f, err := fh.Open()
		if err != nil {
			common.ApiErrorMsg(c, "打开文件失败: "+err.Error())
			return
		}
		files = append(files, f)
	}
	data, err := service.KbUploadDocs(identity, kbName, files, fileHeaders)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

// ---------- Chat ----------

func KbCreateSession(c *gin.Context) {
	identity := requireKbIdentity(c)
	if identity == nil {
		return
	}
	sessionId, err := service.KbCreateSession(identity)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, gin.H{"session_id": sessionId})
}

func KbDeleteSession(c *gin.Context) {
	identity := requireKbIdentity(c)
	if identity == nil {
		return
	}
	var req struct {
		SessionId string `json:"session_id"`
	}
	if err := c.ShouldBindJSON(&req); err != nil || req.SessionId == "" {
		common.ApiErrorMsg(c, "缺少 session_id")
		return
	}
	if err := service.KbDeleteSession(identity, req.SessionId); err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, nil)
}

func KbListSessions(c *gin.Context) {
	identity := requireKbIdentity(c)
	if identity == nil {
		return
	}
	data, err := service.KbListSessions(identity)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, data)
}

// KbSessionChat proxies the SSE chat stream to the client.
func KbSessionChat(c *gin.Context) {
	identity := requireKbIdentity(c)
	if identity == nil {
		return
	}
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		common.ApiErrorMsg(c, "读取请求失败")
		return
	}
	resp, err := service.KbSessionChat(identity, payload)
	if err != nil {
		common.ApiError(c, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.JSON(resp.StatusCode, gin.H{"success": false, "message": string(body)})
		return
	}
	// Stream SSE back to client without reverse-proxy buffering.
	relayhelper.SetEventStreamHeaders(c)
	c.Writer.WriteHeader(http.StatusOK)
	flusher, ok := c.Writer.(http.Flusher)
	if !ok {
		io.Copy(c.Writer, resp.Body)
		return
	}
	buf := make([]byte, 4096)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			c.Writer.Write(buf[:n])
			flusher.Flush()
		}
		if err != nil {
			break
		}
	}
}

// ---------- Admin ----------

// AdminListKnowledgeBaseUsers lists all users who enabled the knowledge base (admin only).
func AdminListKnowledgeBaseUsers(c *gin.Context) {
	list, err := model.GetAllUserKnowledgeBases()
	if err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, list)
}
