package perfmetrics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
)

func TestBuildRelaySampleUsesCaptureTime(t *testing.T) {
	now := time.Now()
	info := &relaycommon.RelayInfo{
		OriginModelName:   "gpt-4o",
		UsingGroup:        "default",
		IsStream:          true,
		StartTime:         now.Add(-2 * time.Second),
		FirstResponseTime: now.Add(-time.Second),
	}

	sample := BuildRelaySample(info, true, 16)

	require.True(t, sample.HasTtft)
	assert.InDelta(t, 2000, sample.LatencyMs, 50)
	assert.InDelta(t, 1000, sample.GenerationMs, 50)
}

func TestBuildRelaySamplePreservesChannelTestMarker(t *testing.T) {
	sample := BuildRelaySample(&relaycommon.RelayInfo{
		OriginModelName: "gpt-4o",
		IsChannelTest:   true,
	}, true, 16)

	require.True(t, sample.IsChannelTest)
}

func TestBuildQueryResultIncludesGroupRequestCount(t *testing.T) {
	result := buildQueryResult("gpt-4o", map[bucketKey]counters{
		{model: "gpt-4o", group: "default", bucketTs: 100}: {
			requestCount: 3,
		},
		{model: "gpt-4o", group: "default", bucketTs: 200}: {
			requestCount: 2,
		},
	})

	require.Len(t, result.Groups, 1)
	assert.Equal(t, int64(5), result.Groups[0].RequestCount)
}
