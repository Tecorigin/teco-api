package model

import (
	"github.com/QuantumNous/new-api/common"
)

type ModelTokenStatsItem struct {
	ModelName            string `json:"model_name"`
	RequestCount         int64  `json:"request_count"`
	PromptTokens         int64  `json:"prompt_tokens"`
	CompletionTokens     int64  `json:"completion_tokens"`
	CacheReadTokens      int64  `json:"cache_read_tokens"`
	CacheWriteTokens     int64  `json:"cache_write_tokens"`
	CacheCreationTokens  int64  `json:"cache_creation_tokens"`
	TotalTokens          int64  `json:"total_tokens"`
}

type logTokenStatsRow struct {
	ModelName        string `json:"model_name" gorm:"column:model_name"`
	RequestCount     int64  `json:"request_count" gorm:"column:request_count"`
	PromptTokens     int64  `json:"prompt_tokens" gorm:"column:prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens" gorm:"column:completion_tokens"`
	Other            string `json:"other" gorm:"column:other"`
}

func GetModelTokenStats(startTime int64, endTime int64, username string, userId int, isAdmin bool) ([]*ModelTokenStatsItem, error) {
	query := LOG_DB.Table("logs").
		Select("model_name, prompt_tokens, completion_tokens, other").
		Where("type = ? and created_at >= ? and created_at <= ?", LogTypeConsume, startTime, endTime)

	if isAdmin {
		if username != "" {
			query = query.Where("username = ?", username)
		}
	} else {
		query = query.Where("user_id = ?", userId)
	}

	var rows []*logTokenStatsRow
	if err := query.Order("model_name asc, id asc").Find(&rows).Error; err != nil {
		return nil, err
	}

	statsMap := make(map[string]*ModelTokenStatsItem)
	for _, row := range rows {
		if row == nil || row.ModelName == "" {
			continue
		}
		item, ok := statsMap[row.ModelName]
		if !ok {
			item = &ModelTokenStatsItem{ModelName: row.ModelName}
			statsMap[row.ModelName] = item
		}
		item.RequestCount++
		item.PromptTokens += row.PromptTokens
		item.CompletionTokens += row.CompletionTokens

		if row.Other != "" {
			var other map[string]interface{}
			if err := common.UnmarshalJsonStr(row.Other, &other); err == nil {
				item.CacheReadTokens += int64(getMapIntValue(other, "cache_tokens"))
				item.CacheWriteTokens += int64(getMapIntValue(other, "cache_write_tokens"))
				item.CacheCreationTokens += int64(getMapIntValue(other, "cache_creation_tokens"))
			}
		}
	}

	result := make([]*ModelTokenStatsItem, 0, len(statsMap))
	for _, item := range statsMap {
		item.TotalTokens = item.PromptTokens + item.CompletionTokens + item.CacheReadTokens + item.CacheWriteTokens
		result = append(result, item)
	}
	return result, nil
}

func getMapIntValue(data map[string]interface{}, key string) int {
	if data == nil {
		return 0
	}
	v, ok := data[key]
	if !ok || v == nil {
		return 0
	}
	switch value := v.(type) {
	case int:
		return value
	case int8:
		return int(value)
	case int16:
		return int(value)
	case int32:
		return int(value)
	case int64:
		return int(value)
	case float32:
		return int(value)
	case float64:
		return int(value)
	default:
		return 0
	}
}