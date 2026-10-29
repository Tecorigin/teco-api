package billingexpr

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGroupTokenPriceExpression(t *testing.T) {
	params := TokenParams{P: 1000, C: 200, Len: 1000, CR: 100, CC: 20, CC1h: 10, Img: 100, ImgO: 50, AI: 100, AO: 20, ASRS: 1000000, ImgN: 2000000, VidS: 3000000, TTSC: 100}
	for _, test := range []struct {
		name, expression    string
		input, output, want float64
		tier                string
	}{
		{"mixed image pricing", `tier("base", p * 2 + c * 8 + img * 2.5 + img_o * 1)`, 0.5, 0.8, 2580, "base"},
		{"cache prices", `tier("base", p * 2 + c * 8 + cr * 0.2 + cc * 2.5 + cc1h * 4 + img * 2.5 + img_o)`, 0.5, 0.8, 2635, "base"},
		{"raw token condition", `p > 800 ? tier("long", p * 2 + c * 8) : tier("short", p)`, 0.5, 0.8, 2280, "long"},
		{"context condition", `len > 800 ? tier("long", p * 2 + c * 8) : tier("short", p)`, 0, 0.8, 1280, "long"},
		{"shared multiplier", `tier("base", (p * 2 + c * 8 + img * 2.5 + img_o) * 2)`, 0.5, 0.8, 5160, "base"},
		{"input allowance unchanged", `tier("base", max(p - 100, 0) * 2 + c * 8)`, 0.5, 0.8, 2180, "base"},
		{"audio and media units unchanged", `tier("base", p * 2 + c * 8 + ai * 3 + ao * 5 + asr_s * 0.02 + img_n * 0.03 + vid_s * 0.02 + tts_c * 10)`, 0.5, 0.8, 143680, "base"},
		{"media stays billable", `tier("base", p * 2 + c * 8 + img * 2.5 + img_o)`, 0, 0, 300, "base"},
		{"version prefix", `v1:tier("base", p * 2 + c * 8)`, 0.5, 0.8, 2280, "base"},
		{"default compatibility", `tier("base", p * 2 + c * 8 + img * 2.5 + img_o)`, 1, 1, 3900, "base"},
	} {
		t.Run(test.name, func(t *testing.T) {
			expression, err := ApplyGroupTokenRatios(test.expression, test.input, test.output)
			require.NoError(t, err)
			cost, trace, err := RunExpr(expression, params)
			require.NoError(t, err)
			assert.InDelta(t, test.want, cost, 0.000001)
			assert.Equal(t, test.tier, trace.MatchedTier)
		})
	}
	_, err := ApplyGroupTokenRatios(`tier("base", p * c)`, 0.5, 0.8)
	assert.Error(t, err)
}

func TestComputeTieredQuotaGroupTokenRatios(t *testing.T) {
	expression := `tier("base", p * 2 + c * 8 + img * 2.5 + img_o)`
	for _, test := range []struct {
		name          string
		input, output float64
		configured    bool
		want          int
	}{
		{"legacy snapshot", 1, 1, false, 975},
		{"independent coefficients", 0.5, 0.8, true, 645},
		{"free text keeps media", 0, 0, true, 75},
		{"coefficient saturation", 1000000000000, 1, true, math.MaxInt32},
	} {
		t.Run(test.name, func(t *testing.T) {
			snapshot := &BillingSnapshot{
				ExprString: expression, ExprHash: ExprHashString(expression),
				GroupRatio: 0.5, QuotaPerUnit: 500000, EstimatedTier: "base",
			}
			if test.configured {
				snapshot.GroupInputRatio, snapshot.GroupOutputRatio = &test.input, &test.output
			}
			result, err := ComputeTieredQuota(snapshot, TokenParams{P: 1000, C: 200, Img: 100, ImgO: 50})
			require.NoError(t, err)
			assert.Equal(t, test.want, result.ActualQuotaAfterGroup)
			assert.Equal(t, "base", result.MatchedTier)
			assert.False(t, result.CrossedTier)
			if test.want == math.MaxInt32 {
				assert.NotNil(t, result.Clamp)
			}
		})
	}
	inputRatio, outputRatio := 0.1, 2.0
	snapshot := &BillingSnapshot{
		ExprString: `tier("base", p * 2 - c)`, GroupRatio: 1, QuotaPerUnit: 500000,
		GroupInputRatio: &inputRatio, GroupOutputRatio: &outputRatio,
	}
	_, err := ComputeTieredQuota(snapshot, TokenParams{P: 10, C: 10})
	assert.Error(t, err)
}
