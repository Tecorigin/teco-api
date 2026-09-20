package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUserRateLimitLuaChecksRPM(t *testing.T) {
	require.Contains(t, userRateLimitLua, "if rpm > 0 and count >= rpm then return 2 end")
	require.True(t, strings.Contains(userRateLimitLua, "local count = redis.call('LLEN', key)"))
}

func resetUserRateLimitMemoryStore() {
	userRateLimitMemoryStore.Lock()
	defer userRateLimitMemoryStore.Unlock()
	userRateLimitMemoryStore.entries = make(map[string][]userRateLimitEntry)
}

func TestGetEffectiveUserRateLimitPrefersSpecificScope(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&UserRateLimit{}))
	require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&UserRateLimit{}).Error)
	t.Cleanup(func() {
		require.NoError(t, DB.Session(&gorm.Session{AllowGlobalUpdate: true}).Unscoped().Delete(&UserRateLimit{}).Error)
	})

	global := &UserRateLimit{Scope: RateLimitScopeGlobal, RPM: 10}
	group := &UserRateLimit{Scope: RateLimitScopeGroup, GroupName: "premium", RPM: 20}
	user := &UserRateLimit{Scope: RateLimitScopeUser, UserId: 1001, RPM: 30}
	userModel := &UserRateLimit{Scope: RateLimitScopeUserModel, UserId: 1001, ModelName: "gpt-test", RPM: 40}
	for _, limit := range []*UserRateLimit{global, group, user, userModel} {
		require.NoError(t, UpsertUserRateLimit(limit))
	}

	resolved, err := GetEffectiveUserRateLimit(1001, "premium", "gpt-test")
	require.NoError(t, err)
	require.Equal(t, int64(40), resolved.RPM)

	require.NoError(t, DeleteUserRateLimit(userModel.Id))
	resolved, err = GetEffectiveUserRateLimit(1001, "premium", "gpt-test")
	require.NoError(t, err)
	require.Equal(t, int64(30), resolved.RPM)

	require.NoError(t, DeleteUserRateLimit(user.Id))
	resolved, err = GetEffectiveUserRateLimit(1001, "premium", "gpt-test")
	require.NoError(t, err)
	require.Equal(t, int64(20), resolved.RPM)

	require.NoError(t, DeleteUserRateLimit(group.Id))
	resolved, err = GetEffectiveUserRateLimit(1001, "premium", "gpt-test")
	require.NoError(t, err)
	require.Equal(t, int64(10), resolved.RPM)
}

func TestUserRateLimitSettlesEstimatedTokensWithActualUsage(t *testing.T) {
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
		resetUserRateLimitMemoryStore()
	})
	resetUserRateLimitMemoryStore()
	limit := &UserRateLimit{Id: 1, Scope: RateLimitScopeUser, UserId: 1001, TPM: 100}

	allowed, limitType, reservation, err := AllowUserRateLimit(limit, 1001, 80, "request-1")
	require.NoError(t, err)
	require.True(t, allowed)
	require.Empty(t, limitType)
	require.NoError(t, SettleUserRateLimitReservation(reservation, 20))

	allowed, limitType, _, err = AllowUserRateLimit(limit, 1001, 81, "request-2")
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, RateLimitTypeTPM, limitType)

	allowed, limitType, _, err = AllowUserRateLimit(limit, 1001, 80, "request-3")
	require.NoError(t, err)
	require.True(t, allowed)
	require.Empty(t, limitType)
}

func TestUserRateLimitReturnsRPMReason(t *testing.T) {
	originalRedisEnabled := common.RedisEnabled
	common.RedisEnabled = false
	t.Cleanup(func() {
		common.RedisEnabled = originalRedisEnabled
		resetUserRateLimitMemoryStore()
	})
	resetUserRateLimitMemoryStore()
	limit := &UserRateLimit{Id: 2, Scope: RateLimitScopeUser, UserId: 1001, RPM: 1}

	allowed, _, _, err := AllowUserRateLimit(limit, 1001, 1, "request-1")
	require.NoError(t, err)
	require.True(t, allowed)

	allowed, limitType, _, err := AllowUserRateLimit(limit, 1001, 1, "request-2")
	require.NoError(t, err)
	require.False(t, allowed)
	require.Equal(t, RateLimitTypeRPM, limitType)
}

func TestNilUserRateLimitAllowsRequest(t *testing.T) {
	allowed, limitType, reservation, err := AllowUserRateLimit(nil, 1001, 10, "request-1")
	require.NoError(t, err)
	require.True(t, allowed)
	require.Empty(t, limitType)
	require.Nil(t, reservation)
}
