package model

import (
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestSLAStatsFromBuckets(t *testing.T) {
	database, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	previous := DB
	DB = database
	t.Cleanup(func() { DB = previous })
	require.NoError(t, DB.AutoMigrate(&SLABucket{}))
	require.NoError(t, DB.Create(&[]SLABucket{
		{BucketKey: "model-a", Mode: "model", UserID: 1, Username: "alice", ModelName: "model", BucketTs: 600, Total: 5, Failed: 1},
		{BucketKey: "model-b", Mode: "model", UserID: 1, Username: "alice", ModelName: "model", BucketTs: 900, Total: 4, Failed: 1},
		{BucketKey: "user-a", Mode: "user", UserID: 1, Username: "alice", ModelName: "model", BucketTs: 600, Total: 5},
		{BucketKey: "user-b", Mode: "user", UserID: 1, Username: "alice", ModelName: "model", BucketTs: 900, Total: 5, Failed: 1},
	}).Error)
	rows, err := GetSLAStats(600, 1200, "ali", "mod")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(9), rows[0].TotalReqs)
	assert.Equal(t, int64(2), rows[0].FailedReqs)
	assert.Equal(t, int64(2), rows[0].TotalWindows)
	assert.Equal(t, int64(1), rows[0].ErrorWindows)
	assert.Equal(t, 50.0, rows[0].SLA)
	rows, err = GetUserSLAStats(600, 1200, "", "")
	require.NoError(t, err)
	require.Len(t, rows, 1)
	assert.Equal(t, int64(10), rows[0].TotalReqs)
	assert.Equal(t, int64(1), rows[0].FailedReqs)
	assert.Equal(t, 50.0, rows[0].SLA)
	rows, err = GetUserSLAStats(1200, 1500, "", "")
	require.NoError(t, err)
	assert.Empty(t, rows)
}
