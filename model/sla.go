package model

import (
	"sort"
)

// SLAStatsRow holds per-user per-model SLA availability for a time window.
type SLAStatsRow struct {
	Username     string  `json:"username"`
	ModelName    string  `json:"model_name"`
	TotalWindows int64   `json:"total_windows"`
	ErrorWindows int64   `json:"error_windows"`
	TotalReqs    int64   `json:"total_reqs"`
	FailedReqs   int64   `json:"failed_reqs"`
	SLA          float64 `json:"sla"`
}

const slaBucketSeconds int64 = 300 // 5 minutes

// GetSLAStats computes SLA per (username, model_name) over [startTs, endTs].
//
// SLA = (1 - error windows / populated windows) × 100%.
// A window is an error window when it has at least 5 requests and a failure.
func GetSLAStats(startTs int64, endTs int64, username string, modelName string) ([]SLAStatsRow, error) {
	return getSLAStats(startTs, endTs, username, modelName, false)
}

func GetUserSLAStats(startTs int64, endTs int64, username string, modelName string) ([]SLAStatsRow, error) {
	return getSLAStats(startTs, endTs, username, modelName, true)
}

func getSLAStats(startTs int64, endTs int64, username string, modelName string, userSLA bool) ([]SLAStatsRow, error) {
	if startTs <= 0 || endTs <= 0 || startTs >= endTs {
		return []SLAStatsRow{}, nil
	}

	// Align bucket boundaries to 5-minute grid.
	bucketStart := startTs - (startTs % slaBucketSeconds)
	bucketEnd := endTs - (endTs % slaBucketSeconds)
	if bucketEnd < bucketStart {
		return []SLAStatsRow{}, nil
	}

	// Combine stored buckets for the same displayed username and model.
	type bucketRow struct {
		Username  string `gorm:"column:username"`
		ModelName string `gorm:"column:model_name"`
		BucketTs  int64  `gorm:"column:bucket_ts"`
		Total     int64  `gorm:"column:total"`
		Failed    int64  `gorm:"column:failed"`
	}

	var rows []bucketRow
	mode := "model"
	if userSLA {
		mode = "user"
	}
	query := DB.Model(&SLABucket{}).
		Select("username, model_name, bucket_ts, SUM(total) AS total, SUM(failed) AS failed").
		Where("bucket_ts >= ? AND bucket_ts <= ? AND mode = ?", bucketStart, bucketEnd, mode)

	if username != "" {
		query = query.Where("username LIKE ?", "%"+username+"%")
	}
	if modelName != "" {
		query = query.Where("model_name LIKE ?", "%"+modelName+"%")
	}

	if err := query.Group("username, model_name, bucket_ts").Find(&rows).Error; err != nil {
		return nil, err
	}

	if len(rows) == 0 {
		return []SLAStatsRow{}, nil
	}

	type agg struct {
		totalWindows int64
		errorWindows int64
		totalReqs    int64
		failedReqs   int64
	}
	byKey := make(map[string]*agg)
	keyOrder := make([]string, 0, len(rows))
	for _, row := range rows {
		key := row.Username + "\x00" + row.ModelName
		current := byKey[key]
		if current == nil {
			current = &agg{}
			byKey[key] = current
			keyOrder = append(keyOrder, key)
		}
		current.totalWindows++
		current.totalReqs += row.Total
		current.failedReqs += row.Failed
		// Per-5min error rate: if valid reqs < 5, error rate = 0 (not an error window).
		if row.Total >= 5 && row.Failed > 0 {
			current.errorWindows++
		}
	}

	result := make([]SLAStatsRow, 0, len(byKey))
	for _, key := range keyOrder {
		current := byKey[key]
		// Split key back into username + model_name.
		username, modelName := splitSLAKey(key)
		sla := 100.0
		if current.totalWindows > 0 {
			sla = (1.0 - float64(current.errorWindows)/float64(current.totalWindows)) * 100.0
			// Round to 2 decimals.
			sla = float64(int(sla*100)) / 100.0
		}
		result = append(result, SLAStatsRow{
			Username:     username,
			ModelName:    modelName,
			TotalWindows: current.totalWindows,
			ErrorWindows: current.errorWindows,
			TotalReqs:    current.totalReqs,
			FailedReqs:   current.failedReqs,
			SLA:          sla,
		})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Username == result[j].Username {
			return result[i].ModelName < result[j].ModelName
		}
		return result[i].Username < result[j].Username
	})
	return result, nil
}

func splitSLAKey(key string) (string, string) {
	for i := 0; i < len(key); i++ {
		if key[i] == 0 {
			return key[:i], key[i+1:]
		}
	}
	return key, ""
}
