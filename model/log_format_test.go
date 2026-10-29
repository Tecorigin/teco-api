package model

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/gin-gonic/gin"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStreamErrorLogFilterIncludesConsumeLogs(t *testing.T) {
	modelName := "stream-filter-regression"
	fixtures := []Log{
		{UserId: 91001, Type: LogTypeConsume, ModelName: modelName, IsStream: true, Other: `{"stream_status":{"status":"error","end_reason":"client_gone"}}`},
		{UserId: 91001, Type: LogTypeConsume, ModelName: modelName, IsStream: true, Other: `{"stream_status": {"status": "error", "end_reason": "upstream_error"}}`},
		{UserId: 91002, Type: LogTypeConsume, ModelName: modelName, IsStream: true, Other: `{"stream_status":{"status":"error"}}`},
		{UserId: 91001, Type: LogTypeConsume, ModelName: modelName, IsStream: true, Other: `{"stream_status":{"status":"ok"}}`},
		{UserId: 91001, Type: LogTypeConsume, ModelName: modelName, IsStream: true, Other: `{"audit_reply":"stream_status error client_gone"}`},
		{UserId: 91001, Type: LogTypeConsume, ModelName: modelName, IsStream: false, Other: `{"stream_status":{"status":"error"}}`},
		{UserId: 91001, Type: LogTypeConsume, ModelName: modelName, IsStream: true, Other: ""},
		{UserId: 91001, Type: LogTypeError, ModelName: modelName, Other: `{"status_code":400}`},
	}
	for index := range fixtures {
		fixtures[index].CreatedAt = common.GetTimestamp()
		fixtures[index].Quota = 10
		fixtures[index].PromptTokens = 10
		require.NoError(t, createLog(&fixtures[index]))
	}
	t.Cleanup(func() {
		require.NoError(t, LOG_DB.Where("model_name = ?", modelName).Delete(&Log{}).Error)
	})
	require.NoError(t, LOG_DB.Model(&Log{}).Where("request_id IN ?", []string{
		fixtures[0].RequestId, fixtures[1].RequestId, fixtures[2].RequestId,
	}).Update("other", "{}").Error)
	require.NoError(t, LOG_DB.Model(&Log{}).Where("request_id = ?", fixtures[3].RequestId).
		Update("other", `{"stream_status":{"status":"error"}}`).Error)

	logs, total, err := GetAllLogs(LogTypeConsume, 0, 0, modelName, "", "", 0, 1, 0, "", "", "", false, true)
	require.NoError(t, err)
	assert.EqualValues(t, 3, total)
	require.Len(t, logs, 1)
	assert.Equal(t, LogTypeConsume, logs[0].Type)

	userLogs, total, err := GetUserLogs(91001, LogTypeConsume, 0, 0, modelName, "", 0, 10, "", "", "", false, true)
	require.NoError(t, err)
	assert.EqualValues(t, 2, total)
	require.Len(t, userLogs, 2)
	assert.ElementsMatch(t, []string{fixtures[0].RequestId, fixtures[1].RequestId}, []string{userLogs[0].RequestId, userLogs[1].RequestId})
	for _, log := range userLogs {
		assert.Equal(t, 91001, log.UserId)
		assert.Equal(t, LogTypeConsume, log.Type)
	}

	_, total, err = GetUserLogs(91001, LogTypeConsume, 0, 0, modelName, "", 0, 10, "", "", "", false, false)
	require.NoError(t, err)
	assert.EqualValues(t, 6, total)

	stat, err := SumUsedQuota(LogTypeConsume, 0, 0, modelName, "", "", 0, "", true)
	require.NoError(t, err)
	assert.Equal(t, 30, stat.Quota)
	assert.Equal(t, 3, stat.Rpm)
	assert.Equal(t, 30, stat.Tpm)
}

func TestRecordBatchLogPersistsStreamStatus(t *testing.T) {
	logs := []*Log{
		{RequestId: "batch-stream-error", IsStream: true, Other: `{"stream_status":{"status":"error","end_reason":"client_gone"}}`},
		{RequestId: "batch-stream-ok", IsStream: true, Other: `{"stream_status":{"status":"ok"}}`},
		{RequestId: "batch-not-stream", Other: `{"stream_status":{"status":"error"}}`},
		{RequestId: "batch-stream-invalid", IsStream: true, Other: "invalid JSON"},
	}
	t.Cleanup(func() {
		require.NoError(t, LOG_DB.Where("request_id IN ?", []string{
			"batch-stream-error", "batch-stream-ok", "batch-not-stream", "batch-stream-invalid",
		}).Delete(&Log{}).Error)
	})
	require.NoError(t, RecordBatchLog(logs))
	for index, expectedStatus := range []string{"error", "ok", "", ""} {
		var stored Log
		require.NoError(t, LOG_DB.Where("request_id = ?", logs[index].RequestId).First(&stored).Error)
		assert.Equal(t, expectedStatus, stored.StreamStatus)
	}
}

func TestRecordErrorLogPreservesUpstreamRequestID(t *testing.T) {
	user := &User{Username: "error-log-user", AffCode: "error-log-user"}
	require.NoError(t, DB.Create(user).Error)
	t.Cleanup(func() {
		require.NoError(t, LOG_DB.Where("user_id = ?", user.Id).Delete(&Log{}).Error)
		require.NoError(t, DB.Delete(user).Error)
	})

	for _, testCase := range []struct {
		name       string
		contextID  string
		otherID    string
		expectedID string
	}{
		{"standard header context", "header-id", "", "header-id"},
		{"provider id in other", "", "provider-id", "provider-id"},
		{"current response takes precedence", "old-id", "current-id", "current-id"},
		{"no upstream id", "", "", ""},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest("POST", "/v1/chat/completions", nil)
			ctx.Set(common.RequestIdKey, "gateway-"+testCase.name)
			ctx.Set(common.UpstreamRequestIdKey, testCase.contextID)
			other := map[string]interface{}{"status_code": 400}
			if testCase.otherID != "" {
				other["upstream_request_id"] = testCase.otherID
			}

			RecordErrorLog(ctx, user.Id, 1, "test-model", "test-token", "invalid request", 0, 1, false, "default", other)

			var log Log
			require.NoError(t, LOG_DB.Where("request_id = ?", ctx.GetString(common.RequestIdKey)).First(&log).Error)
			assert.Equal(t, LogTypeError, log.Type)
			assert.Equal(t, testCase.expectedID, log.UpstreamRequestId)
			logs := []*Log{&log}
			formatUserLogs(logs, 0)
			assert.Equal(t, testCase.expectedID, logs[0].UpstreamRequestId)
		})
	}
}

// TestFormatUserLogsStripsQuotaSaturation verifies the admin-only quota
// saturation marker (nested under other.admin_info) is removed for non-admin
// log views, since formatUserLogs strips the whole admin_info object.
func TestFormatUserLogsStripsQuotaSaturation(t *testing.T) {
	other := common.MapToJsonStr(map[string]interface{}{
		"model_price": 0.004,
		"admin_info": map[string]interface{}{
			"quota_saturation": map[string]interface{}{
				"op":      "QuotaFromDecimal",
				"kind":    "overflow",
				"clamped": common.MaxQuota,
			},
		},
	})
	logs := []*Log{{Other: other}}

	formatUserLogs(logs, 0)

	parsed, err := common.StrToMap(logs[0].Other)
	require.NoError(t, err)
	_, hasAdminInfo := parsed["admin_info"]
	require.False(t, hasAdminInfo, "admin_info (and nested quota_saturation) must be stripped for non-admin views")
	// Non-admin billing fields remain visible.
	require.Contains(t, parsed, "model_price")
}
