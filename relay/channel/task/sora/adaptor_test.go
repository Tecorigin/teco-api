package sora

import (
	"testing"

	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/stretchr/testify/require"
)

func TestAdjustBillingOnSubmitUsesActualVideoSeconds(t *testing.T) {
	tests := []struct {
		name     string
		taskData string
		want     float64
	}{
		{
			name:     "uses actual duration from completed sync response",
			taskData: `{"metadata":{"actual_video_seconds":5}}`,
			want:     5,
		},
		{
			name:     "ignores missing actual duration",
			taskData: `{}`,
			want:     2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			info := &relaycommon.RelayInfo{}
			info.PriceData.AddOtherRatio("seconds", 2)
			info.PriceData.AddOtherRatio("size", 1)

			ratios := (&TaskAdaptor{}).AdjustBillingOnSubmit(info, []byte(tt.taskData))
			if tt.want == 2 {
				require.Nil(t, ratios)
				return
			}
			require.Equal(t, tt.want, ratios["seconds"])
			require.Equal(t, float64(1), ratios["size"])
		})
	}
}
