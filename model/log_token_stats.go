package model

import (
	"github.com/QuantumNous/new-api/common"
)

type ModelTokenStatsItem struct {
	ModelName           string `json:"model_name"`
	RequestCount        int64  `json:"request_count"`
	PromptTokens        int64  `json:"prompt_tokens"`
	CompletionTokens    int64  `json:"completion_tokens"`
	CacheReadTokens     int64  `json:"cache_read_tokens"`
	CacheWriteTokens    int64  `json:"cache_write_tokens"`
	CacheCreationTokens int64  `json:"cache_creation_tokens"`
	TotalTokens         int64  `json:"total_tokens"`
}

type UserModelTokenStatsItem struct {
	Username string `json:"username"`
	ModelTokenStatsItem
}

type logTokenStatsRow struct {
	Username         string `json:"username" gorm:"column:username"`
	ModelName        string `json:"model_name" gorm:"column:model_name"`
	RequestCount     int64  `json:"request_count" gorm:"column:request_count"`
	PromptTokens     int64  `json:"prompt_tokens" gorm:"column:prompt_tokens"`
	CompletionTokens int64  `json:"completion_tokens" gorm:"column:completion_tokens"`
	Other            string `json:"other" gorm:"column:other"`
}

type userModelCacheTokenStatsRow struct {
	Username  string `json:"username" gorm:"column:username"`
	ModelName string `json:"model_name" gorm:"column:model_name"`
	Other     string `json:"other" gorm:"column:other"`
}

func GetUserModelTokenStats(startTime int64, endTime int64, username string) ([]*UserModelTokenStatsItem, error) {
	type quotaDataTokenStatsRow struct {
		Username            string `gorm:"column:username"`
		ModelName           string `gorm:"column:model_name"`
		RequestCount        int64  `gorm:"column:request_count"`
		PromptTokens        int64  `gorm:"column:prompt_tokens"`
		CompletionTokens    int64  `gorm:"column:completion_tokens"`
		CacheReadTokens     int64  `gorm:"column:cache_read_tokens"`
		CacheWriteTokens    int64  `gorm:"column:cache_write_tokens"`
		CacheCreationTokens int64  `gorm:"column:cache_creation_tokens"`
	}
	var rows []*quotaDataTokenStatsRow
	query := DB.Table("quota_data").
		Select("username, model_name, COALESCE(SUM(count), 0) AS request_count, COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens, COALESCE(SUM(completion_tokens), 0) AS completion_tokens, COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens, COALESCE(SUM(cache_write_tokens), 0) AS cache_write_tokens, COALESCE(SUM(cache_creation_tokens), 0) AS cache_creation_tokens").
		Where("created_at >= ? and created_at <= ?", startTime, endTime).
		Where("username <> '' and model_name <> ''")
	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	err := query.Group("username, model_name").Find(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make([]*UserModelTokenStatsItem, 0, len(rows))
	for _, row := range rows {
		if row == nil || row.Username == "" || row.ModelName == "" {
			continue
		}
		item := &UserModelTokenStatsItem{
			Username: row.Username,
			ModelTokenStatsItem: ModelTokenStatsItem{
				ModelName:           row.ModelName,
				RequestCount:        row.RequestCount,
				PromptTokens:        row.PromptTokens,
				CompletionTokens:    row.CompletionTokens,
				CacheReadTokens:     row.CacheReadTokens,
				CacheWriteTokens:    row.CacheWriteTokens,
				CacheCreationTokens: row.CacheCreationTokens,
			},
		}
		item.TotalTokens = item.PromptTokens + item.CompletionTokens + item.CacheReadTokens + item.CacheWriteTokens
		result = append(result, item)
	}
	// If quota_data returned no rows (likely backfill not completed for this range),
	// fall back to scanning logs to preserve availability and correctness.
	if len(result) == 0 {
		var rows2 []*logTokenStatsRow
		query2 := LOG_DB.Table("logs").
			Select("username, model_name, COUNT(*) AS request_count, COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens, COALESCE(SUM(completion_tokens), 0) AS completion_tokens").
			Where("type = ? and created_at >= ? and created_at <= ?", LogTypeConsume, startTime, endTime).
			Where("username <> '' and model_name <> ''")
		if username != "" {
			query2 = query2.Where("username LIKE ?", "%"+username+"%")
		}
		err := query2.Group("username, model_name").Find(&rows2).Error
		if err != nil {
			return nil, err
		}
		statsMap := make(map[string]*UserModelTokenStatsItem)
		for _, row := range rows2 {
			if row == nil || row.Username == "" || row.ModelName == "" {
				continue
			}
			key := row.Username + "\x00" + row.ModelName
			item, ok := statsMap[key]
			if !ok {
				item = &UserModelTokenStatsItem{
					Username:            row.Username,
					ModelTokenStatsItem: ModelTokenStatsItem{ModelName: row.ModelName},
				}
				statsMap[key] = item
			}
			item.RequestCount += row.RequestCount
			item.PromptTokens += row.PromptTokens
			item.CompletionTokens += row.CompletionTokens
		}
		var cacheRows []*userModelCacheTokenStatsRow
		query3 := LOG_DB.Table("logs").
			Select("username, model_name, other").
			Where("type = ? and created_at >= ? and created_at <= ?", LogTypeConsume, startTime, endTime).
			Where("username <> '' and model_name <> '' and other <> ''")
		if username != "" {
			query3 = query3.Where("username LIKE ?", "%"+username+"%")
		}
		err = query3.Find(&cacheRows).Error
		if err != nil {
			return nil, err
		}
		for _, row := range cacheRows {
			if row == nil {
				continue
			}
			item := statsMap[row.Username+"\x00"+row.ModelName]
			if item == nil {
				continue
			}
			var other map[string]interface{}
			if err := common.UnmarshalJsonStr(row.Other, &other); err == nil {
				item.CacheReadTokens += int64(getMapIntValue(other, "cache_tokens"))
				item.CacheWriteTokens += int64(getMapIntValue(other, "cache_write_tokens"))
				item.CacheCreationTokens += int64(getMapIntValue(other, "cache_creation_tokens"))
			}
		}
		for _, item := range statsMap {
			item.TotalTokens = item.PromptTokens + item.CompletionTokens + item.CacheReadTokens + item.CacheWriteTokens
			result = append(result, item)
		}
	}
	return result, nil
}

func GetModelTokenStats(startTime int64, endTime int64, username string, userId int, isAdmin bool) ([]*ModelTokenStatsItem, error) {
	// First try quota_data aggregated table to avoid expensive logs scan
	type quotaModelStatsRow struct {
		ModelName        string `gorm:"column:model_name"`
		RequestCount     int64  `gorm:"column:request_count"`
		PromptTokens     int64  `gorm:"column:prompt_tokens"`
		CompletionTokens int64  `gorm:"column:completion_tokens"`
		CacheReadTokens  int64  `gorm:"column:cache_read_tokens"`
		CacheWriteTokens int64  `gorm:"column:cache_write_tokens"`
	}
	var qrows []*quotaModelStatsRow
	qquery := DB.Table("quota_data").
		Select("model_name, COALESCE(SUM(count), 0) AS request_count, COALESCE(SUM(prompt_tokens), 0) AS prompt_tokens, COALESCE(SUM(completion_tokens), 0) AS completion_tokens, COALESCE(SUM(cache_read_tokens), 0) AS cache_read_tokens, COALESCE(SUM(cache_write_tokens), 0) AS cache_write_tokens").
		Where("created_at >= ? and created_at <= ?", startTime, endTime).
		Where("model_name <> ''")
	if isAdmin {
		if username != "" {
			qquery = qquery.Where("username LIKE ?", "%"+username+"%")
		}
	} else {
		qquery = qquery.Where("user_id = ?", userId)
	}
	if err := qquery.Group("model_name").Find(&qrows).Error; err == nil && len(qrows) > 0 {
		result := make([]*ModelTokenStatsItem, 0, len(qrows))
		for _, r := range qrows {
			item := &ModelTokenStatsItem{
				ModelName:        r.ModelName,
				RequestCount:     r.RequestCount,
				PromptTokens:     r.PromptTokens,
				CompletionTokens: r.CompletionTokens,
				CacheReadTokens:  r.CacheReadTokens,
				CacheWriteTokens: r.CacheWriteTokens,
			}
			item.TotalTokens = item.PromptTokens + item.CompletionTokens + item.CacheReadTokens + item.CacheWriteTokens
			result = append(result, item)
		}
		return result, nil
	}

	// Fallback to logs scan if quota_data did not return results
	query := LOG_DB.Table("logs").
		Select("model_name, prompt_tokens, completion_tokens, other").
		Where("type = ? and created_at >= ? and created_at <= ?", LogTypeConsume, startTime, endTime)

	if isAdmin {
		if username != "" {
			// support fuzzy username filtering for admin
			query = query.Where("username LIKE ?", "%"+username+"%")
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
