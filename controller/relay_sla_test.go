package controller

import (
	"net/http/httptest"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/model"
	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestRecordRelaySLARecovery(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	previous := model.DB
	model.DB = database
	t.Cleanup(func() { model.DB = previous })
	require.NoError(t, database.AutoMigrate(&model.SLAResult{}, &model.SLABucket{}))
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	ctx.Set("id", 1)
	ctx.Set("username", "alice")
	ctx.Set("original_model", "model")
	ctx.Set("channel_id", 2)
	startedAt := time.Now().Unix()
	recordRelaySLA(ctx, "internal", "internal-model-0", "client-id", "model", startedAt, 0, false)
	recordRelaySLA(ctx, "internal", "internal-model-1", "client-id", "model", startedAt, 1, true)
	recordRelaySLA(ctx, "internal", "internal-user", "client-id", "user", startedAt, 2, true)
	recordRelaySLA(ctx, "internal", "internal-user", "client-id", "user", startedAt, 2, true)
	rows, err := model.GetSLAStats(startedAt-300, startedAt+300, "", "")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(2), rows[0].TotalReqs)
	assert.Equal(t, int64(1), rows[0].FailedReqs)
	rows, err = model.GetUserSLAStats(startedAt-300, startedAt+300, "", "")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(1), rows[0].TotalReqs)
	assert.Zero(t, rows[0].FailedReqs)
	var result model.SLAResult
	require.NoError(t, database.Where("event_id = ?", "internal-user").First(&result).Error)
	assert.Equal(t, "internal", result.InternalRequestID)
	assert.Equal(t, 2, result.Attempt)
}
