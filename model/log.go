package model

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/logger"
	"github.com/QuantumNous/new-api/types"

	"github.com/gin-gonic/gin"

	"github.com/bytedance/gopkg/util/gopool"
	"gorm.io/gorm"
)

type Log struct {
	Id                int    `json:"id" gorm:"index:idx_created_at_id,priority:1;index:idx_user_id_id,priority:2"`
	UserId            int    `json:"user_id" gorm:"index;index:idx_user_id_id,priority:1"`
	CreatedAt         int64  `json:"created_at" gorm:"bigint;index:idx_created_at_id,priority:2;index:idx_created_at_type"`
	Type              int    `json:"type" gorm:"index:idx_created_at_type"`
	Content           string `json:"content"`
	Username          string `json:"username" gorm:"index;index:index_username_model_name,priority:2;default:''"`
	TokenName         string `json:"token_name" gorm:"index;default:''"`
	ModelName         string `json:"model_name" gorm:"index;index:index_username_model_name,priority:1;default:''"`
	Quota             int    `json:"quota" gorm:"default:0"`
	PromptTokens      int    `json:"prompt_tokens" gorm:"default:0"`
	CompletionTokens  int    `json:"completion_tokens" gorm:"default:0"`
	UseTime           int    `json:"use_time" gorm:"default:0"`
	IsStream          bool   `json:"is_stream"`
	ChannelId         int    `json:"channel" gorm:"index"`
	ChannelName       string `json:"channel_name" gorm:"->"`
	TokenId           int    `json:"token_id" gorm:"default:0;index"`
	Group             string `json:"group" gorm:"index"`
	Ip                string `json:"ip" gorm:"index;default:''"`
	RequestId         string `json:"request_id,omitempty" gorm:"type:varchar(64);index:idx_logs_request_id;default:''"`
	UpstreamRequestId string `json:"upstream_request_id,omitempty" gorm:"type:varchar(128);index:idx_logs_upstream_request_id;default:''"`
	Other             string `json:"other" gorm:"type:text"`
}

// don't use iota, avoid change log type value
const (
	LogTypeUnknown = 0
	LogTypeTopup   = 1
	LogTypeConsume = 2
	LogTypeManage  = 3
	LogTypeSystem  = 4
	LogTypeError   = 5
	LogTypeRefund  = 6
)

func formatUserLogs(logs []*Log, startIdx int) {
	for i := range logs {
		logs[i].ChannelName = ""
		var otherMap map[string]interface{}
		otherMap, _ = common.StrToMap(logs[i].Other)
		if otherMap != nil {
			// Remove admin-only debug fields.
			delete(otherMap, "admin_info")
			// delete(otherMap, "reject_reason")
			delete(otherMap, "stream_status")
		}
		logs[i].Other = common.MapToJsonStr(otherMap)
		logs[i].Id = startIdx + i + 1
	}
}

func GetLogByTokenId(tokenId int) (logs []*Log, err error) {
	err = LOG_DB.Model(&Log{}).Where("token_id = ?", tokenId).Order("id desc").Limit(common.MaxRecentItems).Find(&logs).Error
	formatUserLogs(logs, 0)
	return logs, err
}

func RecordLog(userId int, logType int, content string) {
	if logType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysError("failed to record log: " + err.Error())
	}
}

func RecordLogWithAdminInfo(userId int, logType int, content string, adminInfo map[string]interface{}) {
	if logType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      logType,
		Content:   content,
	}
	if len(adminInfo) > 0 {
		other := map[string]interface{}{
			"admin_info": adminInfo,
		}
		log.Other = common.MapToJsonStr(other)
	}
	if err := LOG_DB.Create(log).Error; err != nil {
		common.SysError("failed to record log: " + err.Error())
	}
}

func RecordSystemLog(content string) {
	log := &Log{
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeSystem,
		Content:   content,
	}
	err := LOG_DB.Create(log).Error
	if err != nil {
		common.SysError("failed to record system log: " + err.Error())
	}
}

func RecordBatchLog(logs []*Log) error {
	if len(logs) == 0 {
		return nil
	}
	return LOG_DB.CreateInBatches(logs, 100).Error
}

type RecordConsumeLogParams struct {
	UserId            int
	ChannelId         int
	PromptTokens      int
	CompletionTokens  int
	ModelName         string
	TokenName         string
	Quota             int
	Content           string
	UseTimeSeconds    int
	IsStream          bool
	TokenId           int
	UserQuota         int
	Group             string
	Other             map[string]interface{}
	PricingType       int
	TaskID            string
	ModelRatio        float64
	ModelPrice        float64
	BillingByRequest  bool
	RequestCount      int
	ContextCount      int
	ContextCostQuota  int
	RequestID         string
	UpstreamRequestID string
}

type RecordTaskBillingLogParams struct {
	UserId    int
	LogType   int
	Content   string
	ChannelId int
	ModelName string
	Quota     int
	TokenId   int
	Group     string
	Other     map[string]interface{}
}

const (
	LogPricingTypeUnknown = iota
	LogPricingTypeModelRatio
	LogPricingTypeModelPrice
)

func RecordConsumeLog(c *gin.Context, userId int, params RecordConsumeLogParams) {
	if !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(userId, false)
	createdAt := common.GetTimestamp()
	otherStr := common.MapToJsonStr(params.Other)
	requestId := params.RequestID
	upstreamRequestId := params.UpstreamRequestID
	if requestId == "" && c != nil {
		requestId = c.GetString(common.RequestIdKey)
	}
	if upstreamRequestId == "" && params.Other != nil {
		if value, ok := params.Other["upstream_request_id"].(string); ok {
			upstreamRequestId = value
		}
	}
	debugParams := map[string]interface{}{
		"user_id":             userId,
		"channel_id":          params.ChannelId,
		"prompt_tokens":       params.PromptTokens,
		"completion_tokens":   params.CompletionTokens,
		"model_name":          params.ModelName,
		"token_name":          params.TokenName,
		"quota":               params.Quota,
		"content":             params.Content,
		"use_time_seconds":    params.UseTimeSeconds,
		"is_stream":           params.IsStream,
		"token_id":            params.TokenId,
		"user_quota":          params.UserQuota,
		"group":               params.Group,
		"pricing_type":        params.PricingType,
		"task_id":             params.TaskID,
		"model_ratio":         params.ModelRatio,
		"model_price":         params.ModelPrice,
		"billing_by_request":  params.BillingByRequest,
		"request_count":       params.RequestCount,
		"context_count":       params.ContextCount,
		"context_cost_quota":  params.ContextCostQuota,
		"request_id":          requestId,
		"upstream_request_id": upstreamRequestId,
	}
	if params.Other != nil {
		otherForDebug := make(map[string]interface{}, len(params.Other))
		for k, v := range params.Other {
			otherForDebug[k] = v
		}
		if value, ok := otherForDebug["audit_prompt"].(string); ok && len(value) > 500 {
			otherForDebug["audit_prompt"] = value[:500] + "...(truncated)"
		}
		if value, ok := otherForDebug["audit_reply"].(string); ok && len(value) > 500 {
			otherForDebug["audit_reply"] = value[:500] + "...(truncated)"
		}
		debugParams["other"] = otherForDebug
	}
	common.SysLog(fmt.Sprintf("record consume log: userId=%d, params=%s", userId, common.MapToJsonStr(debugParams)))
	log := &Log{
		UserId:            userId,
		Username:          username,
		CreatedAt:         createdAt,
		Type:              LogTypeConsume,
		Content:           params.Content,
		PromptTokens:      params.PromptTokens,
		CompletionTokens:  params.CompletionTokens,
		ModelName:         params.ModelName,
		TokenName:         params.TokenName,
		Quota:             params.Quota,
		UseTime:           params.UseTimeSeconds,
		IsStream:          params.IsStream,
		ChannelId:         params.ChannelId,
		TokenId:           params.TokenId,
		Group:             params.Group,
		RequestId:         requestId,
		UpstreamRequestId: upstreamRequestId,
		Other:             otherStr,
	}
	if err := LOG_DB.Create(log).Error; err != nil {
		common.SysError("failed to record consume log: " + err.Error())
		return
	}
	if err := IncrementUserModelUsageCounter(userId, params.ModelName, int64(params.PromptTokens+params.CompletionTokens), 1, time.Unix(createdAt, 0)); err != nil {
		common.SysError("failed to increment user model usage counter: " + err.Error())
	}
	LogQuotaData(userId, username, params.ModelName, params.Quota, createdAt, params.PromptTokens+params.CompletionTokens)
}

func RecordConsumeLogBatch(logs []*Log) {
	if len(logs) == 0 {
		return
	}
	if err := RecordBatchLog(logs); err != nil {
		common.SysError("failed to record consume log batch: " + err.Error())
	}
}

func RecordErrorLog(c *gin.Context, userId int, channelId int, modelName string, tokenName string, content string, tokenId int, useTimeSeconds int, isStream bool, group string, other map[string]interface{}) {
	username := ""
	requestId := ""
	upstreamRequestId := ""
	if c != nil {
		username = c.GetString("username")
		requestId = c.GetString(common.RequestIdKey)
		upstreamRequestId = c.GetString(common.UpstreamRequestIdKey)
	}
	if username == "" {
		username, _ = GetUsernameById(userId, false)
	}
	if upstreamRequestId == "" && other != nil {
		if value, ok := other["upstream_request_id"].(string); ok {
			upstreamRequestId = value
		}
	}
	otherStr := common.MapToJsonStr(other)
	log := &Log{
		UserId:            userId,
		Username:          username,
		CreatedAt:         common.GetTimestamp(),
		Type:              LogTypeError,
		Content:           content,
		TokenName:         tokenName,
		ModelName:         modelName,
		Quota:             0,
		ChannelId:         channelId,
		TokenId:           tokenId,
		UseTime:           useTimeSeconds,
		IsStream:          isStream,
		Group:             group,
		RequestId:         requestId,
		UpstreamRequestId: upstreamRequestId,
		Other:             otherStr,
	}
	if err := LOG_DB.Create(log).Error; err != nil {
		common.SysError("failed to record error log: " + err.Error())
	}
}

func RecordTaskBillingLog(params RecordTaskBillingLogParams) {
	if params.LogType == LogTypeConsume && !common.LogConsumeEnabled {
		return
	}
	username, _ := GetUsernameById(params.UserId, false)
	tokenName := ""
	if params.TokenId > 0 {
		if token, err := GetTokenById(params.TokenId); err == nil {
			tokenName = token.Name
		}
	}
	log := &Log{
		UserId:    params.UserId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      params.LogType,
		Content:   params.Content,
		TokenName: tokenName,
		ModelName: params.ModelName,
		Quota:     params.Quota,
		ChannelId: params.ChannelId,
		TokenId:   params.TokenId,
		Group:     params.Group,
		Other:     common.MapToJsonStr(params.Other),
	}
	if err := LOG_DB.Create(log).Error; err != nil {
		common.SysLog("failed to record task billing log: " + err.Error())
	}
}

func RecordTopupLog(userId int, content string, callerIp string, paymentMethod string, callbackPaymentMethod string) {
	username, _ := GetUsernameById(userId, false)
	adminInfo := map[string]interface{}{
		"server_ip":               common.GetIp(),
		"node_name":               common.NodeName,
		"caller_ip":               callerIp,
		"payment_method":          paymentMethod,
		"callback_payment_method": callbackPaymentMethod,
		"version":                 common.Version,
	}
	other := map[string]interface{}{
		"admin_info": adminInfo,
	}
	log := &Log{
		UserId:    userId,
		Username:  username,
		CreatedAt: common.GetTimestamp(),
		Type:      LogTypeTopup,
		Content:   content,
		Ip:        callerIp,
		Other:     common.MapToJsonStr(other),
	}
	if err := LOG_DB.Create(log).Error; err != nil {
		common.SysError("failed to record topup log: " + err.Error())
	}
}

func GetAllLogs(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, startIdx int, num int, channel int, group string, requestId string, upstreamRequestId string, auditOnly bool) (logs []*Log, total int64, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB
	} else {
		tx = LOG_DB.Where("logs.type = ?", logType)
	}

	if modelName != "" {
		tx = tx.Where("logs.model_name like ?", modelName)
	}
	if username != "" {
		tx = tx.Where("logs.username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("(logs.request_id = ? OR logs.upstream_request_id = ?)", requestId, requestId)
	}
	if auditOnly {
		tx = tx.Where("(logs.other LIKE ? OR logs.other LIKE ?)", "%\"audit_prompt\"%", "%\"audit_reply\"%")
	}
	if upstreamRequestId != "" {
		tx = tx.Where("logs.upstream_request_id = ?", upstreamRequestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if channel != 0 {
		tx = tx.Where("logs.channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Count(&total).Error
	if err != nil {
		return nil, 0, err
	}
	err = tx.Order("logs.id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		return nil, 0, err
	}

	channelIds := types.NewSet[int]()
	for _, log := range logs {
		if log.ChannelId != 0 {
			channelIds.Add(log.ChannelId)
		}
	}

	if channelIds.Len() > 0 {
		var channels []struct {
			Id   int    `gorm:"column:id"`
			Name string `gorm:"column:name"`
		}
		if common.MemoryCacheEnabled {
			for _, channelId := range channelIds.Items() {
				if cacheChannel, err := CacheGetChannel(channelId); err == nil {
					channels = append(channels, struct {
						Id   int    `gorm:"column:id"`
						Name string `gorm:"column:name"`
					}{
						Id:   channelId,
						Name: cacheChannel.Name,
					})
				}
			}
		} else {
			if err = DB.Table("channels").Select("id, name").Where("id IN ?", channelIds.Items()).Find(&channels).Error; err != nil {
				return logs, total, err
			}
		}
		channelMap := make(map[int]string, len(channels))
		for _, channel := range channels {
			channelMap[channel.Id] = channel.Name
		}
		for i := range logs {
			logs[i].ChannelName = channelMap[logs[i].ChannelId]
		}
	}

	return logs, total, err
}

const logSearchCountLimit = 10000

func GetUserLogs(userId int, logType int, startTimestamp int64, endTimestamp int64, modelName string, tokenName string, startIdx int, num int, group string, requestId string, upstreamRequestId string, auditOnly bool) (logs []*Log, total int64, err error) {
	var tx *gorm.DB
	if logType == LogTypeUnknown {
		tx = LOG_DB.Where("logs.user_id = ?", userId)
	} else {
		tx = LOG_DB.Where("logs.user_id = ? and logs.type = ?", userId, logType)
	}

	if modelName != "" {
		modelNamePattern, err := sanitizeLikePattern(modelName)
		if err != nil {
			return nil, 0, err
		}
		tx = tx.Where("logs.model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if tokenName != "" {
		tx = tx.Where("logs.token_name = ?", tokenName)
	}
	if requestId != "" {
		tx = tx.Where("(logs.request_id = ? OR logs.upstream_request_id = ?)", requestId, requestId)
	}
	if auditOnly {
		tx = tx.Where("(logs.other LIKE ? OR logs.other LIKE ?)", "%\"audit_prompt\"%", "%\"audit_reply\"%")
	}
	if upstreamRequestId != "" {
		tx = tx.Where("logs.upstream_request_id = ?", upstreamRequestId)
	}
	if startTimestamp != 0 {
		tx = tx.Where("logs.created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("logs.created_at <= ?", endTimestamp)
	}
	if group != "" {
		tx = tx.Where("logs."+logGroupCol+" = ?", group)
	}
	err = tx.Model(&Log{}).Limit(logSearchCountLimit).Count(&total).Error
	if err != nil {
		common.SysError("failed to count user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}
	err = tx.Order("logs.id desc").Limit(num).Offset(startIdx).Find(&logs).Error
	if err != nil {
		common.SysError("failed to search user logs: " + err.Error())
		return nil, 0, errors.New("查询日志失败")
	}

	formatUserLogs(logs, startIdx)
	return logs, total, err
}

type Stat struct {
	Quota int `json:"quota"`
	Rpm   int `json:"rpm"`
	Tpm   int `json:"tpm"`
}

func SumUsedQuota(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string, channel int, group string) (stat Stat, err error) {
	tx := LOG_DB.Table("logs").Select("sum(quota) quota")

	rpmTpmQuery := LOG_DB.Table("logs").Select("count(*) rpm, sum(prompt_tokens) + sum(completion_tokens) tpm")

	if username != "" {
		tx = tx.Where("username = ?", username)
		rpmTpmQuery = rpmTpmQuery.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
		rpmTpmQuery = rpmTpmQuery.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		modelNamePattern, err := sanitizeLikePattern(modelName)
		if err != nil {
			return stat, err
		}
		tx = tx.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
		rpmTpmQuery = rpmTpmQuery.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if channel != 0 {
		tx = tx.Where("channel_id = ?", channel)
		rpmTpmQuery = rpmTpmQuery.Where("channel_id = ?", channel)
	}
	if group != "" {
		tx = tx.Where(logGroupCol+" = ?", group)
		rpmTpmQuery = rpmTpmQuery.Where(logGroupCol+" = ?", group)
	}

	tx = tx.Where("type = ?", LogTypeConsume)
	rpmTpmQuery = rpmTpmQuery.Where("type = ?", LogTypeConsume)

	rpmTpmQuery = rpmTpmQuery.Where("created_at >= ?", time.Now().Add(-60*time.Second).Unix())

	if err := tx.Scan(&stat).Error; err != nil {
		common.SysError("failed to query log stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}
	if err := rpmTpmQuery.Scan(&stat).Error; err != nil {
		common.SysError("failed to query rpm/tpm stat: " + err.Error())
		return stat, errors.New("查询统计数据失败")
	}

	return stat, nil
}

func SumUsedToken(logType int, startTimestamp int64, endTimestamp int64, modelName string, username string, tokenName string) (token int) {
	tx := LOG_DB.Table("logs").Select("ifnull(sum(prompt_tokens),0) + ifnull(sum(completion_tokens),0)")
	if username != "" {
		tx = tx.Where("username = ?", username)
	}
	if tokenName != "" {
		tx = tx.Where("token_name = ?", tokenName)
	}
	if startTimestamp != 0 {
		tx = tx.Where("created_at >= ?", startTimestamp)
	}
	if endTimestamp != 0 {
		tx = tx.Where("created_at <= ?", endTimestamp)
	}
	if modelName != "" {
		modelNamePattern, err := sanitizeLikePattern(modelName)
		if err != nil {
			common.SysError("failed to sanitize model name pattern: " + err.Error())
			return 0
		}
		tx = tx.Where("model_name LIKE ? ESCAPE '!'", modelNamePattern)
	}
	if logType != LogTypeUnknown {
		tx = tx.Where("type = ?", logType)
	}
	if err := tx.Scan(&token).Error; err != nil {
		common.SysError("failed to query used token: " + err.Error())
	}
	return token
}

func DeleteOldLog(targetTimestamp int64) (int64, error) {
	result := LOG_DB.Where("created_at < ?", targetTimestamp).Delete(&Log{})
	return result.RowsAffected, result.Error
}

func DeleteOldLogBatch(ctx context.Context, targetTimestamp int64, batchSize int) (int64, error) {
	var totalDeleted int64
	for {
		select {
		case <-ctx.Done():
			return totalDeleted, ctx.Err()
		default:
		}
		var ids []int
		err := LOG_DB.Model(&Log{}).Where("created_at < ?", targetTimestamp).Limit(batchSize).Pluck("id", &ids).Error
		if err != nil {
			return totalDeleted, err
		}
		if len(ids) == 0 {
			break
		}
		result := LOG_DB.Where("id IN ?", ids).Delete(&Log{})
		if result.Error != nil {
			return totalDeleted, result.Error
		}
		totalDeleted += result.RowsAffected
		if len(ids) < batchSize {
			break
		}
	}
	return totalDeleted, nil
}

func DeleteOldLogAsync(targetTimestamp int64, batchSize int) {
	gopool.Go(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
		defer cancel()
		rows, err := DeleteOldLogBatch(ctx, targetTimestamp, batchSize)
		if err != nil {
			logger.LogError(ctx, fmt.Sprintf("failed to delete old logs asynchronously: %v", err))
			return
		}
		logger.LogInfo(ctx, fmt.Sprintf("deleted %d old logs asynchronously", rows))
	})
}
