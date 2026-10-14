package model

import (
	"crypto/sha256"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type SLAResult struct {
	ID                int64  `gorm:"primaryKey"`
	EventID           string `gorm:"type:varchar(96);uniqueIndex"`
	RequestID         string `gorm:"type:varchar(64);index"`
	InternalRequestID string `gorm:"type:varchar(64);index"`
	ChannelID         int
	Mode              string `gorm:"type:varchar(8)"`
	UserID            int
	Username          string `gorm:"type:varchar(191)"`
	ModelName         string `gorm:"type:varchar(191)"`
	StartedAt         int64
	FinishedAt        int64
	Attempt           int
	Succeeded         bool
}

type SLABucket struct {
	ID        int64  `gorm:"primaryKey"`
	BucketKey string `gorm:"type:varchar(64);uniqueIndex"`
	Mode      string `gorm:"type:varchar(8);index:idx_sla_mode_time,priority:1"`
	BucketTs  int64  `gorm:"index:idx_sla_mode_time,priority:2"`
	UserID    int
	Username  string `gorm:"type:varchar(191)"`
	ModelName string `gorm:"type:varchar(191)"`
	Total     int64
	Failed    int64
}

func RecordSLAResult(result SLAResult) error {
	if result.EventID == "" || result.Username == "" || result.ModelName == "" || result.StartedAt <= 0 || result.FinishedAt < result.StartedAt || (result.Mode != "user" && result.Mode != "model") {
		return fmt.Errorf("invalid SLA result")
	}
	bucket := SLABucket{
		Mode: result.Mode, UserID: result.UserID, Username: result.Username, ModelName: result.ModelName,
		BucketTs: result.StartedAt - result.StartedAt%slaBucketSeconds, Total: 1,
	}
	if result.Mode == "model" {
		bucket.BucketTs = result.FinishedAt - result.FinishedAt%slaBucketSeconds
	}
	if !result.Succeeded {
		bucket.Failed = 1
	}
	bucket.BucketKey = fmt.Sprintf("%x", sha256.Sum256(fmt.Appendf(nil, "%s\x00%d\x00%s\x00%s\x00%d", bucket.Mode, bucket.UserID, bucket.Username, bucket.ModelName, bucket.BucketTs)))
	return DB.Transaction(func(tx *gorm.DB) error {
		insert := tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&result)
		if insert.Error != nil {
			return insert.Error
		}
		if insert.RowsAffected == 0 {
			return nil
		}
		return tx.Clauses(clause.OnConflict{
			Columns: []clause.Column{{Name: "bucket_key"}},
			DoUpdates: clause.Assignments(map[string]interface{}{
				"total":  gorm.Expr("total + ?", 1),
				"failed": gorm.Expr("failed + ?", bucket.Failed),
			}),
		}).Create(&bucket).Error
	})
}
