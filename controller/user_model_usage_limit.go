package controller

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
)

type userModelUsageLimitRequest struct {
	UserId              int    `json:"user_id"`
	ModelName           string `json:"model_name"`
	DailyTokenLimit     int64  `json:"daily_token_limit"`
	DailyRequestLimit   int64  `json:"daily_request_limit"`
	WeeklyTokenLimit    int64  `json:"weekly_token_limit"`
	WeeklyRequestLimit  int64  `json:"weekly_request_limit"`
	MonthlyTokenLimit   int64  `json:"monthly_token_limit"`
	MonthlyRequestLimit int64  `json:"monthly_request_limit"`
}

func GetUserModelUsageLimits(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	limits, total, err := model.GetUserModelUsageLimits(keyword, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"items": limits, "total": total}})
}

func CreateUserModelUsageLimit(c *gin.Context) {
	var req userModelUsageLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	limit := &model.UserModelUsageLimit{
		UserId:              req.UserId,
		ModelName:           strings.TrimSpace(req.ModelName),
		DailyTokenLimit:     req.DailyTokenLimit,
		DailyRequestLimit:   req.DailyRequestLimit,
		WeeklyTokenLimit:    req.WeeklyTokenLimit,
		WeeklyRequestLimit:  req.WeeklyRequestLimit,
		MonthlyTokenLimit:   req.MonthlyTokenLimit,
		MonthlyRequestLimit: req.MonthlyRequestLimit,
	}
	if err := model.UpsertUserModelUsageLimit(limit); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "保存成功"})
}

func UpdateUserModelUsageLimit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	oldLimit, err := model.GetUserModelUsageLimitById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	var req userModelUsageLimitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	oldLimit.UserId = req.UserId
	oldLimit.ModelName = strings.TrimSpace(req.ModelName)
	oldLimit.DailyTokenLimit = req.DailyTokenLimit
	oldLimit.DailyRequestLimit = req.DailyRequestLimit
	oldLimit.WeeklyTokenLimit = req.WeeklyTokenLimit
	oldLimit.WeeklyRequestLimit = req.WeeklyRequestLimit
	oldLimit.MonthlyTokenLimit = req.MonthlyTokenLimit
	oldLimit.MonthlyRequestLimit = req.MonthlyRequestLimit
	if err := model.UpsertUserModelUsageLimit(oldLimit); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "保存成功"})
}

func DeleteUserModelUsageLimit(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	if err := model.DeleteUserModelUsageLimit(id); err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "删除成功"})
}

func GetUserModelUsageLimitHits(c *gin.Context) {
	pageInfo := common.GetPageQuery(c)
	keyword := c.Query("keyword")
	periodType := c.Query("period_type")
	limitType := c.Query("limit_type")
	hits, total, err := model.GetUserModelUsageLimitHits(keyword, periodType, limitType, pageInfo)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"success": false, "message": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "message": "", "data": gin.H{"items": hits, "total": total}})
}
