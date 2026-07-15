package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetModelSLOStatsComputesPercentilesAndFilter(t *testing.T) {
	require.NoError(t, DB.AutoMigrate(&PerfMetricSample{}))
	require.NoError(t, DB.Exec("DELETE FROM perf_metric_samples").Error)

	samples := []*PerfMetricSample{
		{ModelName: "gpt-4o", Group: "default", BucketTs: 100, LatencyMs: 100, TpotMs: 10, OutputTokens: 50, Success: true, CreatedAt: 100},
		{ModelName: "gpt-4o", Group: "default", BucketTs: 101, LatencyMs: 200, TpotMs: 20, OutputTokens: 60, Success: true, CreatedAt: 101},
		{ModelName: "gpt-4o", Group: "default", BucketTs: 102, LatencyMs: 300, TpotMs: 30, OutputTokens: 70, Success: false, CreatedAt: 102},
		{ModelName: "gpt-4o", Group: "default", BucketTs: 103, LatencyMs: 900, TpotMs: 90, OutputTokens: 80, Success: true, CreatedAt: 103},
		{ModelName: "claude-3-5-sonnet", Group: "default", BucketTs: 104, LatencyMs: 150, TpotMs: 15, OutputTokens: 30, Success: true, CreatedAt: 104},
	}
	for _, sample := range samples {
		require.NoError(t, CreatePerfMetricSample(sample))
	}

	rows, err := GetModelSLOStats(90, 200, "gpt", []string{"default"}, 250)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	row := rows[0]
	assert.Equal(t, "gpt-4o", row.ModelName)
	assert.Equal(t, int64(4), row.RequestCount)
	assert.Equal(t, int64(3), row.SuccessCount)
	assert.Equal(t, int64(900), row.P99LatencyMs)
	assert.Equal(t, int64(90), row.P99TpotMs)
	assert.Equal(t, 50.00, row.Slo)
	assert.InDelta(t, 2.18, row.Rpm, 0.01)
	assert.InDelta(t, 141.82, row.Tpm, 0.01)
	assert.Greater(t, row.Rpm, 0.0)
	assert.Greater(t, row.Tpm, 0.0)

	require.NoError(t, DB.Exec("DELETE FROM perf_metric_samples").Error)
}