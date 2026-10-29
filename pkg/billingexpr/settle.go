package billingexpr

import (
	"fmt"
	"math"

	"github.com/QuantumNous/new-api/common"
)

// quotaConversion converts raw expression output to quota based on the
// expression version. This is the central dispatch point for future versions
// that may use a different conversion formula.
func quotaConversion(exprOutput float64, snap *BillingSnapshot) float64 {
	switch snap.ExprVersion {
	default: // v1: coefficients are $/1M tokens prices
		return exprOutput / 1_000_000 * snap.QuotaPerUnit
	}
}

// ComputeTieredQuota runs the Expr from a frozen BillingSnapshot against
// actual token counts and returns the settlement result.
func ComputeTieredQuota(snap *BillingSnapshot, params TokenParams) (TieredResult, error) {
	return ComputeTieredQuotaWithRequest(snap, params, RequestInput{})
}

func ComputeTieredQuotaWithRequest(snap *BillingSnapshot, params TokenParams, request RequestInput) (TieredResult, error) {
	inputRatio, outputRatio := 1.0, 1.0
	if snap.GroupInputRatio != nil {
		inputRatio = *snap.GroupInputRatio
	}
	if snap.GroupOutputRatio != nil {
		outputRatio = *snap.GroupOutputRatio
	}
	expression, err := ApplyGroupTokenRatios(snap.ExprString, inputRatio, outputRatio)
	if err != nil {
		return TieredResult{}, err
	}
	hash := snap.ExprHash
	if expression != snap.ExprString {
		hash = ExprHashString(expression)
	}
	cost, trace, err := RunExprByHashWithRequest(expression, hash, params, request)
	if err != nil {
		return TieredResult{}, err
	}
	if (inputRatio != 1 || outputRatio != 1) && (cost < 0 || math.IsNaN(cost)) {
		return TieredResult{}, fmt.Errorf("group-adjusted billing expression returned an invalid cost")
	}

	quotaBeforeGroup := quotaConversion(cost, snap)
	afterGroup, clamp := common.QuotaRoundChecked(quotaBeforeGroup * snap.GroupRatio)
	crossed := trace.MatchedTier != snap.EstimatedTier

	return TieredResult{
		ActualQuotaBeforeGroup: quotaBeforeGroup,
		ActualQuotaAfterGroup:  afterGroup,
		MatchedTier:            trace.MatchedTier,
		CrossedTier:            crossed,
		Clamp:                  clamp,
	}, nil
}
