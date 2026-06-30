package model

import (
	"fmt"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type UserModelUsageCounter struct {
	Id           int    `json:"id" gorm:"primaryKey"`
	UserId       int    `json:"user_id" gorm:"uniqueIndex:uk_user_model_usage_counter,priority:1;not null"`
	ModelName    string `json:"model_name" gorm:"type:varchar(128);uniqueIndex:uk_user_model_usage_counter,priority:2;not null"`
	PeriodType   string `json:"period_type" gorm:"type:varchar(16);uniqueIndex:uk_user_model_usage_counter,priority:3;not null"`
	PeriodStart  int64  `json:"period_start" gorm:"uniqueIndex:uk_user_model_usage_counter,priority:4;not null"`
	TokenCount   int64  `json:"token_count" gorm:"default:0"`
	RequestCount int64  `json:"request_count" gorm:"default:0"`
	CreatedAt    int64  `json:"created_at" gorm:"bigint;autoCreateTime:milli"`
	UpdatedAt    int64  `json:"updated_at" gorm:"bigint;autoUpdateTime:milli"`
}

func GetUserModelUsageCounter(userId int, modelName, periodType string, periodStart int64) (*UserModelUsageStat, error) {
	stat := &UserModelUsageStat{}
	err := DB.Model(&UserModelUsageCounter{}).
		Select("COALESCE(SUM(token_count), 0) AS token_count, COALESCE(SUM(request_count), 0) AS request_count").
		Where("user_id = ? AND model_name = ? AND period_type = ? AND period_start = ?", userId, modelName, periodType, periodStart).
		Scan(stat).Error
	if err != nil {
		return nil, err
	}
	return stat, nil
}

func IncrementUserModelUsageCounter(userId int, modelName string, tokenIncrement, requestIncrement int64, now time.Time) error {
	modelName = normalizeModelName(modelName)
	if userId <= 0 || modelName == "" {
		return nil
	}
	periodTypes := []string{"daily", "weekly", "monthly"}
	for _, periodType := range periodTypes {
		periodStart := getPeriodStart(now, periodType).Unix()
		counter := &UserModelUsageCounter{
			UserId:       userId,
			ModelName:    modelName,
			PeriodType:   periodType,
			PeriodStart:  periodStart,
			TokenCount:   tokenIncrement,
			RequestCount: requestIncrement,
		}
		if err := DB.Clauses(clause.OnConflict{
			Columns: []clause.Column{
				{Name: "user_id"},
				{Name: "model_name"},
				{Name: "period_type"},
				{Name: "period_start"},
			},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"token_count":   gorm.Expr("user_model_usage_counters.token_count + ?", tokenIncrement),
				"request_count": gorm.Expr("user_model_usage_counters.request_count + ?", requestIncrement),
				"updated_at":    common.GetTimestamp(),
			}),
		}).Create(counter).Error; err != nil {
			return fmt.Errorf("increment usage counter failed for %s: %w", periodType, err)
		}
	}
	return nil
}

func BackfillUserModelUsageCountersFromLogs() error {
	periodTypes := []string{"daily", "weekly", "monthly"}
	for _, periodType := range periodTypes {
		var rows []struct {
			UserId       int
			ModelName    string
			PeriodStart  int64
			TokenCount   int64
			RequestCount int64
		}
		var periodExpr string
		switch periodType {
		case "daily":
			if common.UsingMainDatabase(common.DatabaseTypeMySQL) {
				periodExpr = "UNIX_TIMESTAMP(DATE(FROM_UNIXTIME(created_at)))"
			} else {
				periodExpr = "strftime('%s', date(created_at, 'unixepoch'))"
			}
		case "weekly":
			if common.UsingMainDatabase(common.DatabaseTypeMySQL) {
				periodExpr = "UNIX_TIMESTAMP(DATE_SUB(DATE(FROM_UNIXTIME(created_at)), INTERVAL WEEKDAY(FROM_UNIXTIME(created_at)) DAY))"
			} else {
				periodExpr = "strftime('%s', date(created_at, 'unixepoch', '-' || ((strftime('%w', created_at, 'unixepoch') + 6) % 7) || ' days'))"
			}
		default:
			if common.UsingMainDatabase(common.DatabaseTypeMySQL) {
				periodExpr = "UNIX_TIMESTAMP(DATE_FORMAT(FROM_UNIXTIME(created_at), '%Y-%m-01'))"
			} else {
				periodExpr = "strftime('%s', strftime('%Y-%m-01', created_at, 'unixepoch'))"
			}
		}
		query := fmt.Sprintf("SELECT user_id, model_name, %s AS period_start, COALESCE(SUM(prompt_tokens + completion_tokens),0) AS token_count, COUNT(1) AS request_count FROM logs WHERE type = ? AND user_id > 0 AND model_name <> '' GROUP BY user_id, model_name, period_start", periodExpr)
		if err := LOG_DB.Raw(query, LogTypeConsume).Scan(&rows).Error; err != nil {
			return err
		}
		for _, row := range rows {
			counter := &UserModelUsageCounter{
				UserId:       row.UserId,
				ModelName:    strings.TrimSpace(row.ModelName),
				PeriodType:   periodType,
				PeriodStart:  row.PeriodStart,
				TokenCount:   row.TokenCount,
				RequestCount: row.RequestCount,
			}
			if err := DB.Clauses(clause.OnConflict{
				Columns: []clause.Column{{Name: "user_id"}, {Name: "model_name"}, {Name: "period_type"}, {Name: "period_start"}},
				DoUpdates: clause.Assignments(map[string]interface{}{
					"token_count":   row.TokenCount,
					"request_count": row.RequestCount,
					"updated_at":    common.GetTimestamp(),
				}),
			}).Create(counter).Error; err != nil {
				return err
			}
		}
	}
	return nil
}
