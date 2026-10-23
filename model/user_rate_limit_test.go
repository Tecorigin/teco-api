package model

import (
	"strings"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
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

func setupUserRateLimitTestDB(t *testing.T) {
	t.Helper()
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	sqlDB, err := database.DB()
	require.NoError(t, err)
	sqlDB.SetMaxOpenConns(1)
	previousDB, previousRedisEnabled := DB, common.RedisEnabled
	DB, common.RedisEnabled = database, false
	resetUserRateLimitMemoryStore()
	t.Cleanup(func() {
		DB, common.RedisEnabled = previousDB, previousRedisEnabled
		resetUserRateLimitMemoryStore()
		_ = sqlDB.Close()
	})
	require.NoError(t, DB.AutoMigrate(&User{}, &UserRateLimit{}))
	for _, user := range []User{
		{Id: 1001, Username: "alice", AffCode: "rate-alice", Group: "premium"},
		{Id: 1002, Username: "bob", AffCode: "rate-bob", Group: "premium"},
	} {
		require.NoError(t, DB.Create(&user).Error)
	}
}

func TestGetEffectiveUserRateLimitPrefersSpecificScope(t *testing.T) {
	setupUserRateLimitTestDB(t)
	global := &UserRateLimit{Scope: RateLimitScopeGlobal, RPM: 10, TPM: 100}
	group := &UserRateLimit{Scope: RateLimitScopeGroup, GroupName: "premium", RPM: 20, TPM: 200}
	user := &UserRateLimit{Scope: RateLimitScopeUser, UserId: 1001, RPM: 30, TPM: 300}
	modelLimit := &UserRateLimit{Scope: RateLimitScopeModel, ModelName: "gpt-test", RPM: 40, TPM: 0}
	groupModel := &UserRateLimit{Scope: RateLimitScopeGroupModel, GroupName: "premium", ModelName: "gpt-test", RPM: 50, TPM: 500}
	userModel := &UserRateLimit{Scope: RateLimitScopeUserModel, UserId: 1001, ModelName: "gpt-test", RPM: 60, TPM: 0}
	for _, limit := range []*UserRateLimit{global, group, user, modelLimit, groupModel, userModel} {
		require.NoError(t, UpsertUserRateLimit(limit))
	}
	for _, test := range []struct {
		name      string
		userID    int
		groupName string
		modelName string
		expected  *UserRateLimit
	}{
		{"user model and zero override", 1001, "premium", "gpt-test", userModel},
		{"group model for another user", 1002, "premium", "gpt-test", groupModel},
		{"model for another group", 1002, "default", "gpt-test", modelLimit},
		{"user on another model", 1001, "premium", "other-model", user},
		{"group on another model", 1002, "premium", "other-model", group},
		{"global fallback", 1002, "default", "other-model", global},
		{"trimmed group and model", 1002, " premium ", " gpt-test ", groupModel},
		{"no model", 1001, "premium", "", user},
	} {
		t.Run(test.name, func(t *testing.T) {
			resolved, err := GetEffectiveUserRateLimit(test.userID, test.groupName, test.modelName)
			require.NoError(t, err)
			require.NotNil(t, resolved)
			assert.Equal(t, test.expected.Scope, resolved.Scope)
			assert.Equal(t, test.expected.RPM, resolved.RPM)
			assert.Equal(t, test.expected.TPM, resolved.TPM)
		})
	}
	for _, test := range []struct {
		remove   *UserRateLimit
		expected *UserRateLimit
	}{
		{userModel, groupModel}, {groupModel, modelLimit}, {modelLimit, user},
		{user, group}, {group, global}, {global, nil},
	} {
		require.NoError(t, DeleteUserRateLimit(test.remove.Id))
		resolved, err := GetEffectiveUserRateLimit(1001, "premium", "gpt-test")
		require.NoError(t, err)
		if test.expected == nil {
			assert.Nil(t, resolved)
			continue
		}
		require.NotNil(t, resolved)
		assert.Equal(t, test.expected.Scope, resolved.Scope)
		assert.Equal(t, test.expected.RPM, resolved.RPM)
		assert.Equal(t, test.expected.TPM, resolved.TPM)
	}
}

func TestNewUserRateLimitScopesNormalizeAndUpsert(t *testing.T) {
	setupUserRateLimitTestDB(t)
	for _, scope := range []string{RateLimitScopeModel, RateLimitScopeGroupModel} {
		t.Run(scope, func(t *testing.T) {
			limit := &UserRateLimit{Scope: scope, UserId: 1001, Username: "unused", GroupName: " premium ", ModelName: " gpt-test ", RPM: 10, TPM: 100}
			require.NoError(t, UpsertUserRateLimit(limit))
			assert.Zero(t, limit.UserId)
			assert.Empty(t, limit.Username)
			assert.Equal(t, "gpt-test", limit.ModelName)
			if scope == RateLimitScopeModel {
				assert.Empty(t, limit.GroupName)
			} else {
				assert.Equal(t, "premium", limit.GroupName)
			}
			originalID := limit.Id
			limit.RPM, limit.TPM = 0, 200
			require.NoError(t, UpsertUserRateLimit(limit))
			stored, err := GetUserRateLimitById(originalID)
			require.NoError(t, err)
			assert.Zero(t, stored.RPM)
			assert.Equal(t, int64(200), stored.TPM)
			limit.ModelName = "other-model"
			require.NoError(t, UpdateUserRateLimit(originalID, limit))
			stored, err = GetUserRateLimitById(originalID)
			require.NoError(t, err)
			assert.Equal(t, "other-model", stored.ModelName)
		})
	}
	for _, limit := range []*UserRateLimit{
		{Scope: RateLimitScopeModel},
		{Scope: RateLimitScopeGroupModel, ModelName: "gpt-test"},
		{Scope: RateLimitScopeGroupModel, GroupName: "premium"},
	} {
		assert.Error(t, UpsertUserRateLimit(limit))
	}
}

func TestNewUserRateLimitScopesKeepUserCountersSeparate(t *testing.T) {
	for _, scope := range []string{RateLimitScopeModel, RateLimitScopeGroupModel} {
		t.Run(scope, func(t *testing.T) {
			setupUserRateLimitTestDB(t)
			limit := &UserRateLimit{Scope: scope, GroupName: "premium", ModelName: "gpt-test", RPM: 1, TPM: 10}
			require.NoError(t, UpsertUserRateLimit(limit))
			allowed, _, reservation, err := AllowUserRateLimit(limit, 1001, 4, "alice-first")
			require.NoError(t, err)
			require.True(t, allowed)
			require.NoError(t, SettleUserRateLimitReservation(reservation, 6))
			allowed, reason, _, err := AllowUserRateLimit(limit, 1001, 1, "alice-second")
			require.NoError(t, err)
			assert.False(t, allowed)
			assert.Equal(t, RateLimitTypeRPM, reason)
			allowed, _, _, err = AllowUserRateLimit(limit, 1002, 4, "bob-first")
			require.NoError(t, err)
			assert.True(t, allowed)
		})
	}
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
