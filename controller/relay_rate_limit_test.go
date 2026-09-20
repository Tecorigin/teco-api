package controller

import (
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/constant"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

func TestRateLimitBlockedLogReadsChannelIDFromContext(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	common.SetContextKey(context, constant.ContextKeyChannelId, 0)

	channelID := common.GetContextKeyInt(context, constant.ContextKeyChannelId)
	require.Zero(t, channelID)
}
