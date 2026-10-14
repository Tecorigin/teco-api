package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSLAResultAggregation(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	previous := DB
	DB = database
	t.Cleanup(func() { DB = previous })
	require.NoError(t, DB.AutoMigrate(&SLAResult{}, &SLABucket{}))
	failed := SLAResult{EventID: "request-model-0", RequestID: "request", Mode: "model", UserID: 1, Username: "alice", ModelName: "model", StartedAt: 610, FinishedAt: 899}
	require.NoError(t, RecordSLAResult(failed))
	require.NoError(t, RecordSLAResult(failed))
	success := failed
	success.EventID = "request-model-1"
	success.StartedAt = 1200
	success.FinishedAt = 1210
	success.Succeeded = true
	require.NoError(t, RecordSLAResult(success))
	success.EventID = "request-user"
	success.Mode = "user"
	success.StartedAt = 610
	require.NoError(t, RecordSLAResult(success))
	var buckets []SLABucket
	require.NoError(t, DB.Order("mode, bucket_ts").Find(&buckets).Error)
	require.Len(t, buckets, 3)
	assert.Equal(t, int64(1), buckets[0].Total)
	assert.Equal(t, int64(1), buckets[0].Failed)
	assert.Equal(t, int64(600), buckets[0].BucketTs)
	assert.Zero(t, buckets[1].Failed)
	assert.Equal(t, "user", buckets[2].Mode)
	assert.Equal(t, int64(600), buckets[2].BucketTs)
	assert.Zero(t, buckets[2].Failed)
}
