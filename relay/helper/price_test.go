package helper

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/dto"
	"github.com/QuantumNous/new-api/pkg/billingexpr"
	relaycommon "github.com/QuantumNous/new-api/relay/common"
	"github.com/QuantumNous/new-api/setting/billing_setting"
	"github.com/QuantumNous/new-api/setting/config"
	"github.com/QuantumNous/new-api/setting/ratio_setting"
	"github.com/QuantumNous/new-api/types"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelPriceHelperGroupTokenRatios(t *testing.T) {
	inputBefore, outputBefore := ratio_setting.GroupInputRatio2JSONString(), ratio_setting.GroupOutputRatio2JSONString()
	modelBefore, completionBefore := ratio_setting.ModelRatio2JSONString(), ratio_setting.CompletionRatio2JSONString()
	preConsumedBefore := common.PreConsumedQuota
	t.Cleanup(func() {
		require.NoError(t, ratio_setting.UpdateGroupInputRatioByJSONString(inputBefore))
		require.NoError(t, ratio_setting.UpdateGroupOutputRatioByJSONString(outputBefore))
		require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(modelBefore))
		require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(completionBefore))
		common.PreConsumedQuota = preConsumedBefore
	})
	common.PreConsumedQuota = 0
	require.NoError(t, ratio_setting.UpdateGroupInputRatioByJSONString(`{"default":0.5,"vip":0.2}`))
	require.NoError(t, ratio_setting.UpdateGroupOutputRatioByJSONString(`{"default":0.8,"vip":0.4}`))
	require.NoError(t, ratio_setting.UpdateModelRatioByJSONString(`{"group-token-test":1}`))
	require.NoError(t, ratio_setting.UpdateCompletionRatioByJSONString(`{"group-token-test":3}`))
	for _, test := range []struct {
		name       string
		files      []*types.FileMeta
		request    dto.Request
		autoGroup  string
		maxTokens  int
		want       int
		configured bool
	}{
		{name: "text", maxTokens: 100, want: 740, configured: true},
		{name: "auto uses selected group", autoGroup: "vip", maxTokens: 100, want: 320, configured: true},
		{name: "missing output limit", want: 20160, configured: true},
		{name: "image input", files: []*types.FileMeta{{FileType: types.FileTypeImage}}, maxTokens: 100, want: 1100},
		{name: "audio output", request: &dto.GeneralOpenAIRequest{Modalities: []byte(`["text","audio"]`)}, maxTokens: 100, want: 1100},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			if test.autoGroup != "" {
				ctx.Set("auto_group", test.autoGroup)
			}
			info := &relaycommon.RelayInfo{OriginModelName: "group-token-test", UserGroup: "default", UsingGroup: "default", Request: test.request}
			price, err := ModelPriceHelper(ctx, info, 1000, &types.TokenCountMeta{MaxTokens: test.maxTokens, Files: test.files})
			require.NoError(t, err)
			assert.Equal(t, test.want, price.QuotaToPreConsume)
			if test.configured {
				require.NotNil(t, price.GroupInputRatio)
				require.NotNil(t, price.GroupOutputRatio)
			} else {
				assert.Nil(t, price.GroupInputRatio)
				assert.Nil(t, price.GroupOutputRatio)
			}
		})
	}
}

func TestModelPriceHelperTieredFreezesGroupTokenRatios(t *testing.T) {
	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	inputBefore, outputBefore := ratio_setting.GroupInputRatio2JSONString(), ratio_setting.GroupOutputRatio2JSONString()
	quotaPerUnitBefore := common.QuotaPerUnit
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
		require.NoError(t, ratio_setting.UpdateGroupInputRatioByJSONString(inputBefore))
		require.NoError(t, ratio_setting.UpdateGroupOutputRatioByJSONString(outputBefore))
		common.QuotaPerUnit = quotaPerUnitBefore
	})
	common.QuotaPerUnit = 500000
	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode":    `{"group-tiered-test":"tiered_expr"}`,
		"billing_setting.billing_expr":    `{"group-tiered-test":"tier(\"base\", p * 2 + c * 8 + img * 2.5 + img_o)"}`,
		"group_ratio_setting.group_ratio": `{"default":1}`,
	}))
	require.NoError(t, ratio_setting.UpdateGroupInputRatioByJSONString(`{"default":0.5}`))
	require.NoError(t, ratio_setting.UpdateGroupOutputRatioByJSONString(`{"default":0.8}`))
	ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
	info := &relaycommon.RelayInfo{
		OriginModelName: "group-tiered-test", UserGroup: "default", UsingGroup: "default",
		BillingRequestInput: &billingexpr.RequestInput{Body: []byte(`{}`)},
	}
	price, err := ModelPriceHelper(ctx, info, 1000, &types.TokenCountMeta{MaxTokens: 200, Files: []*types.FileMeta{{FileType: types.FileTypeImage}}})
	require.NoError(t, err)
	assert.Equal(t, 1140, price.QuotaToPreConsume)
	require.NotNil(t, info.TieredBillingSnapshot)
	data, err := common.Marshal(info.TieredBillingSnapshot)
	require.NoError(t, err)
	var frozen billingexpr.BillingSnapshot
	require.NoError(t, common.Unmarshal(data, &frozen))
	require.NoError(t, ratio_setting.UpdateGroupInputRatioByJSONString(`{"default":9}`))
	require.NoError(t, ratio_setting.UpdateGroupOutputRatioByJSONString(`{"default":9}`))
	result, err := billingexpr.ComputeTieredQuota(&frozen, billingexpr.TokenParams{P: 1000, C: 200, Img: 100, ImgO: 50})
	require.NoError(t, err)
	assert.Equal(t, 1290, result.ActualQuotaAfterGroup)
	assert.Equal(t, "base", result.MatchedTier)
}

func TestModelPriceHelperTieredUsesPreloadedRequestInput(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode": `{"tiered-test-model":"tiered_expr"}`,
		"billing_setting.billing_expr": `{"tiered-test-model":"param(\"stream\") == true ? tier(\"stream\", p * 3) : tier(\"base\", p * 2)"}`,
	}))

	recorder := httptest.NewRecorder()
	ctx, _ := gin.CreateTestContext(recorder)
	req := httptest.NewRequest(http.MethodPost, "/api/channel/test/1", nil)
	req.Body = nil
	req.ContentLength = 0
	req.Header.Set("Content-Type", "application/json")
	ctx.Request = req
	ctx.Set("group", "default")

	info := &relaycommon.RelayInfo{
		OriginModelName: "tiered-test-model",
		UserGroup:       "default",
		UsingGroup:      "default",
		RequestHeaders:  map[string]string{"Content-Type": "application/json"},
		BillingRequestInput: &billingexpr.RequestInput{
			Headers: map[string]string{"Content-Type": "application/json"},
			Body:    []byte(`{"stream":true}`),
		},
	}

	priceData, err := ModelPriceHelper(ctx, info, 1000, &types.TokenCountMeta{})
	require.NoError(t, err)
	require.Equal(t, 1500, priceData.QuotaToPreConsume)
	require.NotNil(t, info.TieredBillingSnapshot)
	require.Equal(t, "stream", info.TieredBillingSnapshot.EstimatedTier)
	require.Equal(t, billing_setting.BillingModeTieredExpr, info.TieredBillingSnapshot.BillingMode)
	require.Equal(t, common.QuotaPerUnit, info.TieredBillingSnapshot.QuotaPerUnit)
}

func TestModelPriceHelperTieredPreConsumeMaxTokensFallback(t *testing.T) {
	gin.SetMode(gin.TestMode)

	saved := map[string]string{}
	require.NoError(t, config.GlobalConfig.SaveToDB(func(key, value string) error {
		saved[key] = value
		return nil
	}))
	t.Cleanup(func() {
		require.NoError(t, config.GlobalConfig.LoadFromDB(saved))
	})

	require.NoError(t, config.GlobalConfig.LoadFromDB(map[string]string{
		"billing_setting.billing_mode":    `{"tiered-fallback-model":"tiered_expr"}`,
		"billing_setting.billing_expr":    `{"tiered-fallback-model":"tier(\"base\", p * 3 + c * 15)"}`,
		"group_ratio_setting.group_ratio": `{"default":1,"free":0}`,
	}))

	const promptTokens = 1000

	cases := []struct {
		name      string
		group     string
		maxTokens int
		expected  int
	}{
		{
			// max_tokens omitted in a paid group -> fall back to 8192 completion tokens.
			// p*3 + c*15 = 1000*3 + 8192*15 = 125880 -> /1e6 * 500000 = 62940
			name:      "non-free group falls back to 8192 completion tokens",
			group:     "default",
			maxTokens: 0,
			expected:  62940,
		},
		{
			// explicit max_tokens is used verbatim, no fallback.
			// 1000*3 + 100*15 = 4500 -> /1e6 * 500000 = 2250
			name:      "explicit max_tokens is used verbatim",
			group:     "default",
			maxTokens: 100,
			expected:  2250,
		},
		{
			// free group (ratio 0) stays zero; fallback is gated on non-zero group ratio.
			name:      "free group stays zero without fallback",
			group:     "free",
			maxTokens: 0,
			expected:  0,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			ctx, _ := gin.CreateTestContext(recorder)
			req := httptest.NewRequest(http.MethodPost, "/v1/chat/completions", nil)
			req.Header.Set("Content-Type", "application/json")
			ctx.Request = req
			ctx.Set("group", tc.group)

			info := &relaycommon.RelayInfo{
				OriginModelName: "tiered-fallback-model",
				UserGroup:       tc.group,
				UsingGroup:      tc.group,
				RequestHeaders:  map[string]string{"Content-Type": "application/json"},
				BillingRequestInput: &billingexpr.RequestInput{
					Headers: map[string]string{"Content-Type": "application/json"},
					Body:    []byte(`{}`),
				},
			}

			priceData, err := ModelPriceHelper(ctx, info, promptTokens, &types.TokenCountMeta{MaxTokens: tc.maxTokens})
			require.NoError(t, err)
			require.Equal(t, tc.expected, priceData.QuotaToPreConsume)
		})
	}
}
