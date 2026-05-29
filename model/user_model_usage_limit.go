package model

import (
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

type UserModelUsageLimit struct {
	Id                  int    `json:"id" gorm:"primaryKey"`
	UserId              int    `json:"user_id" gorm:"index:idx_user_model_limit,priority:1;not null"`
	Username            string `json:"username" gorm:"type:varchar(64);index"`
	ModelName           string `json:"model_name" gorm:"type:varchar(128);index:idx_user_model_limit,priority:2;not null"`
	DailyTokenLimit     int64  `json:"daily_token_limit" gorm:"default:0"`
	DailyRequestLimit   int64  `json:"daily_request_limit" gorm:"default:0"`
	WeeklyTokenLimit    int64  `json:"weekly_token_limit" gorm:"default:0"`
	WeeklyRequestLimit  int64  `json:"weekly_request_limit" gorm:"default:0"`
	MonthlyTokenLimit   int64  `json:"monthly_token_limit" gorm:"default:0"`
	MonthlyRequestLimit int64  `json:"monthly_request_limit" gorm:"default:0"`
	DailyTokenUsed      int64  `json:"daily_token_used" gorm:"-"`
	DailyRequestUsed    int64  `json:"daily_request_used" gorm:"-"`
	WeeklyTokenUsed     int64  `json:"weekly_token_used" gorm:"-"`
	WeeklyRequestUsed   int64  `json:"weekly_request_used" gorm:"-"`
	MonthlyTokenUsed    int64  `json:"monthly_token_used" gorm:"-"`
	MonthlyRequestUsed  int64  `json:"monthly_request_used" gorm:"-"`
	CreatedAt           int64  `json:"created_at" gorm:"bigint;autoCreateTime:milli"`
	UpdatedAt           int64  `json:"updated_at" gorm:"bigint;autoUpdateTime:milli"`
}

type UserModelUsageLimitHit struct {
	Id         int    `json:"id" gorm:"primaryKey"`
	UserId     int    `json:"user_id" gorm:"index;not null"`
	Username   string `json:"username" gorm:"type:varchar(64);index"`
	ModelName  string `json:"model_name" gorm:"type:varchar(128);index;not null"`
	LimitType  string `json:"limit_type" gorm:"type:varchar(16);index;not null"`
	PeriodType string `json:"period_type" gorm:"type:varchar(16);index;not null"`
	UsedValue  int64  `json:"used_value"`
	LimitValue int64  `json:"limit_value"`
	RequestId  string `json:"request_id" gorm:"type:varchar(64);index"`
	CreatedAt  int64  `json:"created_at" gorm:"bigint;index;autoCreateTime:milli"`
}

type UserModelUsageStat struct {
	TokenCount   int64 `json:"token_count"`
	RequestCount int64 `json:"request_count"`
}

type UserModelUsageExceededResult struct {
	Exceeded   bool   `json:"exceeded"`
	LimitType  string `json:"limit_type,omitempty"`
	PeriodType string `json:"period_type,omitempty"`
	UsedValue  int64  `json:"used_value,omitempty"`
	LimitValue int64  `json:"limit_value,omitempty"`
}

func normalizeModelName(modelName string) string {
	return strings.TrimSpace(modelName)
}

func validateUserModelUsageLimit(limit *UserModelUsageLimit) error {
	if limit == nil {
		return errors.New("limit is nil")
	}
	if limit.UserId <= 0 {
		return errors.New("user_id is required")
	}
	limit.ModelName = normalizeModelName(limit.ModelName)
	if limit.ModelName == "" {
		return errors.New("model_name is required")
	}
	values := []int64{
		limit.DailyTokenLimit,
		limit.DailyRequestLimit,
		limit.WeeklyTokenLimit,
		limit.WeeklyRequestLimit,
		limit.MonthlyTokenLimit,
		limit.MonthlyRequestLimit,
	}
	for _, v := range values {
		if v < 0 {
			return errors.New("limit value cannot be negative")
		}
	}
	return nil
}

func UpsertUserModelUsageLimit(limit *UserModelUsageLimit) error {
	if err := validateUserModelUsageLimit(limit); err != nil {
		return err
	}
	username, _ := GetUsernameById(limit.UserId, false)
	limit.Username = username

	var existing UserModelUsageLimit
	err := DB.Where("user_id = ? AND model_name = ?", limit.UserId, limit.ModelName).First(&existing).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return DB.Create(limit).Error
		}
		return err
	}

	existing.Username = username
	existing.DailyTokenLimit = limit.DailyTokenLimit
	existing.DailyRequestLimit = limit.DailyRequestLimit
	existing.WeeklyTokenLimit = limit.WeeklyTokenLimit
	existing.WeeklyRequestLimit = limit.WeeklyRequestLimit
	existing.MonthlyTokenLimit = limit.MonthlyTokenLimit
	existing.MonthlyRequestLimit = limit.MonthlyRequestLimit
	return DB.Save(&existing).Error
}

func DeleteUserModelUsageLimit(id int) error {
	return DB.Delete(&UserModelUsageLimit{}, id).Error
}

func GetUserModelUsageLimitById(id int) (*UserModelUsageLimit, error) {
	var limit UserModelUsageLimit
	if err := DB.First(&limit, id).Error; err != nil {
		return nil, err
	}
	return &limit, nil
}

func fillUserModelUsageLimitCurrentUsage(limits []*UserModelUsageLimit) error {
	now := time.Now()
	dailyStart := getPeriodStart(now, "daily").Unix()
	weeklyStart := getPeriodStart(now, "weekly").Unix()
	monthlyStart := getPeriodStart(now, "monthly").Unix()
	for _, limit := range limits {
		if limit == nil {
			continue
		}
		if stat, err := GetUserModelUsageCounter(limit.UserId, limit.ModelName, "daily", dailyStart); err != nil {
			return err
		} else {
			limit.DailyTokenUsed = stat.TokenCount
			limit.DailyRequestUsed = stat.RequestCount
		}
		if stat, err := GetUserModelUsageCounter(limit.UserId, limit.ModelName, "weekly", weeklyStart); err != nil {
			return err
		} else {
			limit.WeeklyTokenUsed = stat.TokenCount
			limit.WeeklyRequestUsed = stat.RequestCount
		}
		if stat, err := GetUserModelUsageCounter(limit.UserId, limit.ModelName, "monthly", monthlyStart); err != nil {
			return err
		} else {
			limit.MonthlyTokenUsed = stat.TokenCount
			limit.MonthlyRequestUsed = stat.RequestCount
		}
	}
	return nil
}

func GetUserModelUsageLimits(keyword string, pageInfo *common.PageInfo) ([]*UserModelUsageLimit, int64, error) {
	var limits []*UserModelUsageLimit
	var total int64
	tx := DB.Model(&UserModelUsageLimit{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		tx = tx.Where("username LIKE ? OR model_name LIKE ? OR CAST(user_id AS CHAR) LIKE ?", like, like, like)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&limits).Error; err != nil {
		return nil, 0, err
	}
	if err := fillUserModelUsageLimitCurrentUsage(limits); err != nil {
		return nil, 0, err
	}
	return limits, total, nil
}

func RecordUserModelUsageLimitHit(hit *UserModelUsageLimitHit) error {
	if hit == nil {
		return nil
	}
	if hit.Username == "" && hit.UserId > 0 {
		hit.Username, _ = GetUsernameById(hit.UserId, false)
	}
	return DB.Create(hit).Error
}

func GetUserModelUsageLimitHits(keyword, periodType, limitType string, pageInfo *common.PageInfo) ([]*UserModelUsageLimitHit, int64, error) {
	var hits []*UserModelUsageLimitHit
	var total int64
	tx := DB.Model(&UserModelUsageLimitHit{})
	if keyword = strings.TrimSpace(keyword); keyword != "" {
		like := "%" + keyword + "%"
		tx = tx.Where("username LIKE ? OR model_name LIKE ? OR CAST(user_id AS CHAR) LIKE ?", like, like, like)
	}
	if periodType = strings.TrimSpace(periodType); periodType != "" {
		tx = tx.Where("period_type = ?", periodType)
	}
	if limitType = strings.TrimSpace(limitType); limitType != "" {
		tx = tx.Where("limit_type = ?", limitType)
	}
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	if err := tx.Order("id desc").Limit(pageInfo.GetPageSize()).Offset(pageInfo.GetStartIdx()).Find(&hits).Error; err != nil {
		return nil, 0, err
	}
	return hits, total, nil
}

func getPeriodStart(ts time.Time, periodType string) time.Time {
	loc := ts.Location()
	y, m, d := ts.Date()
	switch periodType {
	case "daily":
		return time.Date(y, m, d, 0, 0, 0, 0, loc)
	case "weekly":
		weekday := int(ts.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		return time.Date(y, m, d-weekday+1, 0, 0, 0, 0, loc)
	case "monthly":
		return time.Date(y, m, 1, 0, 0, 0, 0, loc)
	default:
		return time.Date(y, m, d, 0, 0, 0, 0, loc)
	}
}

func GetUserModelUsageStatsSince(userId int, modelName string, startAt int64) (*UserModelUsageStat, error) {
	now := time.Unix(startAt, 0)
	periodTypes := []string{"daily", "weekly", "monthly"}
	for _, periodType := range periodTypes {
		if getPeriodStart(now, periodType).Unix() == startAt {
			return GetUserModelUsageCounter(userId, modelName, periodType, startAt)
		}
	}
	return &UserModelUsageStat{}, nil
}

func CheckUserModelUsageLimit(userId int, modelName string, estimatedTokens int64, requestIncrement int64, requestId string) (*UserModelUsageExceededResult, error) {
	modelName = normalizeModelName(modelName)
	if userId <= 0 || modelName == "" {
		return &UserModelUsageExceededResult{}, nil
	}

	var limit UserModelUsageLimit
	if err := DB.Where("user_id = ? AND model_name = ?", userId, modelName).First(&limit).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &UserModelUsageExceededResult{}, nil
		}
		return nil, err
	}

	now := time.Now()
	checks := []struct {
		PeriodType   string
		TokenLimit   int64
		RequestLimit int64
	}{
		{"daily", limit.DailyTokenLimit, limit.DailyRequestLimit},
		{"weekly", limit.WeeklyTokenLimit, limit.WeeklyRequestLimit},
		{"monthly", limit.MonthlyTokenLimit, limit.MonthlyRequestLimit},
	}

	for _, check := range checks {
		if check.TokenLimit <= 0 && check.RequestLimit <= 0 {
			continue
		}
		startAt := getPeriodStart(now, check.PeriodType).Unix()
		stat, err := GetUserModelUsageStatsSince(userId, modelName, startAt)
		if err != nil {
			return nil, err
		}
		if check.TokenLimit > 0 {
			used := stat.TokenCount
			if used >= check.TokenLimit {
				_ = RecordUserModelUsageLimitHit(&UserModelUsageLimitHit{
					UserId:     userId,
					Username:   limit.Username,
					ModelName:  modelName,
					LimitType:  "token",
					PeriodType: check.PeriodType,
					UsedValue:  used,
					LimitValue: check.TokenLimit,
					RequestId:  requestId,
				})
				return &UserModelUsageExceededResult{Exceeded: true, LimitType: "token", PeriodType: check.PeriodType, UsedValue: used, LimitValue: check.TokenLimit}, nil
			}
		}
		if check.RequestLimit > 0 {
			used := stat.RequestCount + requestIncrement
			if used > check.RequestLimit {
				_ = RecordUserModelUsageLimitHit(&UserModelUsageLimitHit{
					UserId:     userId,
					Username:   limit.Username,
					ModelName:  modelName,
					LimitType:  "request",
					PeriodType: check.PeriodType,
					UsedValue:  used,
					LimitValue: check.RequestLimit,
					RequestId:  requestId,
				})
				return &UserModelUsageExceededResult{Exceeded: true, LimitType: "request", PeriodType: check.PeriodType, UsedValue: used, LimitValue: check.RequestLimit}, nil
			}
		}
	}
	return &UserModelUsageExceededResult{}, nil
}
