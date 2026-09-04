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
