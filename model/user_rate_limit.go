package model

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const (
	RateLimitScopeGlobal     = "global"
	RateLimitScopeGroup      = "group"
	RateLimitScopeUser       = "user"
	RateLimitScopeModel      = "model"
	RateLimitScopeGroupModel = "group_model"
	RateLimitScopeUserModel  = "user_model"
	RateLimitTypeRPM         = "rpm"
	RateLimitTypeTPM         = "tpm"
)

type UserRateLimit struct {
	Id        int    `json:"id" gorm:"primaryKey"`
	Scope     string `json:"scope" gorm:"type:varchar(16);uniqueIndex:uk_user_rate_limit,priority:1;not null"`
	UserId    int    `json:"user_id" gorm:"uniqueIndex:uk_user_rate_limit,priority:2;not null"`
	Username  string `json:"username" gorm:"type:varchar(64);index"`
	GroupName string `json:"group_name" gorm:"type:varchar(64);uniqueIndex:uk_user_rate_limit,priority:3;not null"`
	ModelName string `json:"model_name" gorm:"type:varchar(128);uniqueIndex:uk_user_rate_limit,priority:4;not null"`
	RPM       int64  `json:"rpm" gorm:"default:0"`
	TPM       int64  `json:"tpm" gorm:"default:0"`
	CreatedAt int64  `json:"created_at" gorm:"bigint;autoCreateTime:milli"`
	UpdatedAt int64  `json:"updated_at" gorm:"bigint;autoUpdateTime:milli"`
}

type userRateLimitEntry struct {
	at        time.Time
	requestID string
	tokens    int64
}

type UserRateLimitReservation struct {
	key             string
	requestID       string
	estimatedTokens int64
}

var userRateLimitMemoryStore = struct {
	sync.Mutex
	entries map[string][]userRateLimitEntry
}{entries: make(map[string][]userRateLimitEntry)}

const userRateLimitLua = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local rpm = tonumber(ARGV[2])
local tpm = tonumber(ARGV[3])
local requested = tonumber(ARGV[4])
local cutoff = now - 60000

while true do
	local oldest = redis.call('LINDEX', key, 0)
	if not oldest then break end
	local separator = string.find(oldest, ':')
	if not separator or tonumber(string.sub(oldest, 1, separator - 1)) > cutoff then break end
	redis.call('LPOP', key)
end

local count = redis.call('LLEN', key)
if rpm > 0 and count >= rpm then return 2 end

if tpm > 0 then
	local total = 0
	local entries = redis.call('LRANGE', key, 0, -1)
	for _, entry in ipairs(entries) do
		local tokens = string.match(entry, ':(%d+)$')
		if tokens then total = total + tonumber(tokens) end
	end
	if total + requested > tpm then return 3 end
end

redis.call('RPUSH', key, tostring(now) .. ':' .. ARGV[5] .. ':' .. tostring(requested))
redis.call('PEXPIRE', key, 120000)
return 1
`

const settleUserRateLimitLua = `
local key = KEYS[1]
local requestID = ARGV[1]
local actualTokens = ARGV[2]
local entries = redis.call('LRANGE', key, 0, -1)

for index, entry in ipairs(entries) do
	local first = string.find(entry, ':')
	local last = first
	if first then
		while true do
			local colon = string.find(entry, ':', last + 1)
			if not colon then break end
			last = colon
		end
		if last > first and string.sub(entry, first + 1, last - 1) == requestID then
			local timestamp = string.sub(entry, 1, first - 1)
			redis.call('LSET', key, index - 1, timestamp .. ':' .. requestID .. ':' .. actualTokens)
			return 1
		end
	end
end
return 0
`

func normalizeUserRateLimit(limit *UserRateLimit) error {
	if limit == nil {
		return errors.New("rate limit is nil")
	}
	limit.Scope = strings.TrimSpace(limit.Scope)
	limit.GroupName = strings.TrimSpace(limit.GroupName)
	limit.ModelName = normalizeModelName(limit.ModelName)
	if limit.RPM < 0 || limit.TPM < 0 || limit.RPM > math.MaxInt32 || limit.TPM > math.MaxInt32 {
		return errors.New("RPM and TPM must be between 0 and 2147483647")
	}
	switch limit.Scope {
	case RateLimitScopeGlobal:
		limit.UserId = 0
		limit.Username = ""
		limit.GroupName = ""
		limit.ModelName = ""
	case RateLimitScopeGroup:
		if limit.GroupName == "" {
			return errors.New("group_name is required")
		}
		limit.UserId = 0
		limit.Username = ""
		limit.ModelName = ""
	case RateLimitScopeUser:
		if limit.UserId <= 0 {
			return errors.New("user_id is required")
		}
		limit.GroupName = ""
		limit.ModelName = ""
	case RateLimitScopeUserModel:
		if limit.UserId <= 0 || limit.ModelName == "" {
			return errors.New("user_id and model_name are required")
		}
		limit.GroupName = ""
	case RateLimitScopeModel:
		if limit.ModelName == "" {
			return errors.New("model_name is required")
		}
		limit.UserId = 0
		limit.Username = ""
		limit.GroupName = ""
	case RateLimitScopeGroupModel:
		if limit.GroupName == "" || limit.ModelName == "" {
			return errors.New("group_name and model_name are required")
		}
		limit.UserId = 0
		limit.Username = ""
	default:
		return errors.New("invalid rate limit scope")
	}
	return nil
}

func UpsertUserRateLimit(limit *UserRateLimit) error {
	if err := normalizeUserRateLimit(limit); err != nil {
		return err
	}
	if limit.UserId > 0 {
		username, err := GetUsernameById(limit.UserId, false)
		if err != nil {
			return err
		}
		limit.Username = username
	}

	var existing UserRateLimit
	err := DB.Where("scope = ? AND user_id = ? AND group_name = ? AND model_name = ?", limit.Scope, limit.UserId, limit.GroupName, limit.ModelName).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return DB.Create(limit).Error
	}
	if err != nil {
		return err
	}
	existing.Username = limit.Username
	existing.RPM = limit.RPM
	existing.TPM = limit.TPM
	return DB.Save(&existing).Error
}

func GetUserRateLimits(keyword string, pageInfo *common.PageInfo) ([]*UserRateLimit, int64, error) {
	var limits []*UserRateLimit
	var total int64
	tx := DB.Model(&UserRateLimit{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		tx = tx.Where("username LIKE ? OR group_name LIKE ? OR model_name LIKE ? OR CAST(user_id AS CHAR) LIKE ?", like, like, like, like)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&limits).Error; err != nil {
		return nil, 0, err
	}
	return limits, total, nil
}

func GetUserRateLimitById(id int) (*UserRateLimit, error) {
	var limit UserRateLimit
	if err := DB.First(&limit, id).Error; err != nil {
		return nil, err
	}
	return &limit, nil
}

func DeleteUserRateLimit(id int) error {
	return DB.Delete(&UserRateLimit{}, id).Error
}

func UpdateUserRateLimit(id int, limit *UserRateLimit) error {
	if err := normalizeUserRateLimit(limit); err != nil {
		return err
	}
	var existing UserRateLimit
	if err := DB.First(&existing, id).Error; err != nil {
		return err
	}
	if limit.UserId > 0 {
		username, err := GetUsernameById(limit.UserId, false)
		if err != nil {
			return err
		}
		limit.Username = username
	}
	var conflicting UserRateLimit
	err := DB.Where("scope = ? AND user_id = ? AND group_name = ? AND model_name = ? AND id <> ?", limit.Scope, limit.UserId, limit.GroupName, limit.ModelName, id).First(&conflicting).Error
	if err == nil {
		return errors.New("a rate limit already exists for this scope")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}
	existing.Scope = limit.Scope
	existing.UserId = limit.UserId
	existing.Username = limit.Username
	existing.GroupName = limit.GroupName
	existing.ModelName = limit.ModelName
	existing.RPM = limit.RPM
	existing.TPM = limit.TPM
	return DB.Save(&existing).Error
}

// GetEffectiveUserRateLimit returns the highest-priority matching configuration.
func GetEffectiveUserRateLimit(userId int, groupName, modelName string) (*UserRateLimit, error) {
	modelName = normalizeModelName(modelName)
	groupName = strings.TrimSpace(groupName)
	query := DB.Where("scope = ? AND user_id = 0 AND group_name = '' AND model_name = ''", RateLimitScopeGlobal)
	if groupName != "" {
		query = query.Or("scope = ? AND user_id = 0 AND group_name = ? AND model_name = ''", RateLimitScopeGroup, groupName)
	}
	if userId > 0 {
		query = query.Or("scope = ? AND user_id = ? AND group_name = '' AND model_name = ''", RateLimitScopeUser, userId)
	}
	if modelName != "" {
		query = query.Or("scope = ? AND user_id = 0 AND group_name = '' AND model_name = ?", RateLimitScopeModel, modelName)
		if groupName != "" {
			query = query.Or("scope = ? AND user_id = 0 AND group_name = ? AND model_name = ?", RateLimitScopeGroupModel, groupName, modelName)
		}
		if userId > 0 {
			query = query.Or("scope = ? AND user_id = ? AND group_name = '' AND model_name = ?", RateLimitScopeUserModel, userId, modelName)
		}
	}
	var limit UserRateLimit
	err := query.Order("CASE scope WHEN 'user_model' THEN 0 WHEN 'group_model' THEN 1 WHEN 'model' THEN 2 WHEN 'user' THEN 3 WHEN 'group' THEN 4 WHEN 'global' THEN 5 END").First(&limit).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &limit, nil
}

// AllowUserRateLimit atomically reserves one request and its estimated tokens
// in the matching one-minute window. A zero RPM or TPM means that dimension is
// not limited.
func AllowUserRateLimit(limit *UserRateLimit, userId, requestedTokens int64, requestID string) (bool, string, *UserRateLimitReservation, error) {
	if limit == nil || (limit.RPM <= 0 && limit.TPM <= 0) {
		return true, "", nil, nil
	}
	if userId <= 0 || requestedTokens < 0 || requestID == "" {
		return false, "", nil, errors.New("invalid rate limit request")
	}
	key := fmt.Sprintf("userRateLimit:%s:%d:%d", limit.Scope, limit.Id, userId)
	if common.RedisEnabled {
		result, err := common.RDB.Eval(context.Background(), userRateLimitLua, []string{key}, time.Now().UnixMilli(), limit.RPM, limit.TPM, requestedTokens, requestID).Int()
		if err != nil {
			return false, "", nil, fmt.Errorf("rate limit check failed: %w", err)
		}
		switch result {
		case 1:
			return true, "", &UserRateLimitReservation{key: key, requestID: requestID, estimatedTokens: requestedTokens}, nil
		case 2:
			return false, RateLimitTypeRPM, nil, nil
		case 3:
			return false, RateLimitTypeTPM, nil, nil
		default:
			return false, "", nil, fmt.Errorf("unexpected rate limit result: %d", result)
		}
	}

	userRateLimitMemoryStore.Lock()
	defer userRateLimitMemoryStore.Unlock()
	now := time.Now()
	entries := userRateLimitMemoryStore.entries[key]
	windowStart := now.Add(-time.Minute)
	firstActive := 0
	for firstActive < len(entries) && !entries[firstActive].at.After(windowStart) {
		firstActive++
	}
	entries = entries[firstActive:]
	if limit.RPM > 0 && int64(len(entries)) >= limit.RPM {
		userRateLimitMemoryStore.entries[key] = entries
		return false, RateLimitTypeRPM, nil, nil
	}
	var usedTokens int64
	for _, entry := range entries {
		usedTokens += entry.tokens
	}
	if limit.TPM > 0 && usedTokens+requestedTokens > limit.TPM {
		userRateLimitMemoryStore.entries[key] = entries
		return false, RateLimitTypeTPM, nil, nil
	}
	userRateLimitMemoryStore.entries[key] = append(entries, userRateLimitEntry{at: now, requestID: requestID, tokens: requestedTokens})
	return true, "", &UserRateLimitReservation{key: key, requestID: requestID, estimatedTokens: requestedTokens}, nil
}

// SettleUserRateLimitReservation replaces a request's pre-reserved estimated
// token count with the actual count persisted in the consume log.
func SettleUserRateLimitReservation(reservation *UserRateLimitReservation, actualTokens int64) error {
	if reservation == nil || reservation.requestID == "" || actualTokens < 0 || actualTokens == reservation.estimatedTokens {
		return nil
	}
	if common.RedisEnabled {
		_, err := common.RDB.Eval(context.Background(), settleUserRateLimitLua, []string{reservation.key}, reservation.requestID, actualTokens).Int()
		if err != nil {
			return fmt.Errorf("rate limit settlement failed: %w", err)
		}
		return nil
	}

	userRateLimitMemoryStore.Lock()
	defer userRateLimitMemoryStore.Unlock()
	entries := userRateLimitMemoryStore.entries[reservation.key]
	for index := range entries {
		if entries[index].requestID == reservation.requestID {
			entries[index].tokens = actualTokens
			break
		}
	}
	userRateLimitMemoryStore.entries[reservation.key] = entries
	return nil
}
