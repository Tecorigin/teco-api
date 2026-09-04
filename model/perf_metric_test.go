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
		{ModelName: "gpt-4o", Group: "default", BucketTs: 100, LatencyMs: 100, TtftMs: 100, TpotMs: 10, HasTtft: true, OutputTokens: 50, Success: true, CreatedAt: 100},
		{ModelName: "gpt-4o", Group: "default", BucketTs: 101, LatencyMs: 200, TtftMs: 200, TpotMs: 20, HasTtft: true, OutputTokens: 60, Success: true, CreatedAt: 101},
		{ModelName: "gpt-4o", Group: "default", BucketTs: 102, LatencyMs: 300, TtftMs: 300, TpotMs: 3000, HasTtft: true, OutputTokens: 70, Success: false, CreatedAt: 102},
		{ModelName: "gpt-4o", Group: "default", BucketTs: 103, LatencyMs: 900, TtftMs: 900, TpotMs: 3000, HasTtft: false, OutputTokens: 80, Success: true, CreatedAt: 103},
		{ModelName: "claude-3-5-sonnet", Group: "default", BucketTs: 104, LatencyMs: 150, TtftMs: 150, TpotMs: 15, OutputTokens: 30, Success: true, CreatedAt: 104},
	}
	for _, sample := range samples {
		require.NoError(t, CreatePerfMetricSample(sample))
	}

	// SLO = success AND TTFT <= 250 AND TPOT <= 25
	// gpt-4o samples: 4 total, 3 success
	//   sample 100: success, TTFT=100<=250, TPOT=10<=25 -> SLO hit
	//   sample 101: success, TTFT=200<=250, TPOT=20<=25 -> SLO hit
	//   sample 102: failed -> no SLO hit
	//   sample 103: success, TTFT=900>250 -> no SLO hit
	// SLO = 2/4 = 50%
	rows, err := GetModelSLOStats(90, 200, "gpt", []string{"default"}, 250, 25)
	require.NoError(t, err)
	require.Len(t, rows, 1)

	row := rows[0]
	assert.Equal(t, "gpt-4o", row.ModelName)
	assert.Equal(t, int64(4), row.RequestCount)
	assert.Equal(t, int64(3), row.SuccessCount)
	assert.Equal(t, int64(900), row.P99LatencyMs)
	assert.Equal(t, int64(900), row.P99TtftMs)
	assert.Equal(t, int64(20), row.P99TpotMs)
	assert.Equal(t, int64(2), row.TpotSampleCount)
	assert.Equal(t, 50.00, row.Slo)
	assert.InDelta(t, 1.0, row.Rpm, 0.01)
	assert.InDelta(t, 80.0, row.Tpm, 0.01)
	assert.Greater(t, row.Rpm, 0.0)
	assert.Greater(t, row.Tpm, 0.0)

	require.NoError(t, DB.Exec("DELETE FROM perf_metric_samples").Error)
}
