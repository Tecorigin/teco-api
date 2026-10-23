package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type userRateLimitRequest struct {
	Scope     string `json:"scope"`
	UserId    int    `json:"user_id"`
	GroupName string `json:"group_name"`
	ModelName string `json:"model_name"`
	RPM       int64  `json:"rpm"`
	TPM       int64  `json:"tpm"`
}

func userRateLimitFromRequest(req userRateLimitRequest) *model.UserRateLimit {
	return &model.UserRateLimit{
		Scope:     strings.TrimSpace(req.Scope),
		UserId:    req.UserId,
		GroupName: strings.TrimSpace(req.GroupName),
		ModelName: strings.TrimSpace(req.ModelName),
		RPM:       req.RPM,
		TPM:       req.TPM,
	}
}

func GetUserRateLimits(c *gin.Context) {
	limits, total, err := model.GetUserRateLimits(c.Query("keyword"), common.GetPageQuery(c))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"items": limits, "total": total}})
}

func GetUserRateLimitModels(c *gin.Context) {
	models := make([]string, 0)
	if err := model.DB.Model(&model.Ability{}).Where("enabled = ?", true).Distinct("model").Order("model").Pluck("model", &models).Error; err != nil {
		common.ApiError(c, err)
		return
	}
	common.ApiSuccess(c, models)
}

func CreateUserRateLimit(c *gin.Context) {
	var req userRateLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := model.UpsertUserRateLimit(userRateLimitFromRequest(req)); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "saved"})
}

func UpdateUserRateLimit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	var req userRateLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	if err := model.UpdateUserRateLimit(id, userRateLimitFromRequest(req)); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "saved"})
}

func DeleteUserRateLimit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := model.DeleteUserRateLimit(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "deleted"})
}
