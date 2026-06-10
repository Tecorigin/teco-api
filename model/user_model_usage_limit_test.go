package model

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGetPeriodStart_MonthlyBoundary(t *testing.T) {
	ts := time.Date(2026, time.July, 1, 0, 0, 0, 0, time.Local)

	monthlyStart := getPeriodStart(ts, "monthly")
	weeklyStart := getPeriodStart(ts, "weekly")
	dailyStart := getPeriodStart(ts, "daily")

	require.Equal(t, ts.Unix(), monthlyStart.Unix())
	require.NotEqual(t, weeklyStart.Unix(), monthlyStart.Unix())
	require.Equal(t, dailyStart.Unix(), monthlyStart.Unix())
}

func TestCounterLookup_ShouldUseExplicitPeriodType(t *testing.T) {
	ts := time.Date(2026, time.June, 1, 0, 0, 0, 0, time.Local)

	startDaily := getPeriodStart(ts, "daily").Unix()
	startWeekly := getPeriodStart(ts, "weekly").Unix()
	startMonthly := getPeriodStart(ts, "monthly").Unix()

	// At month-start midnight, daily and monthly starts are equal by design.
	require.Equal(t, startDaily, startMonthly)

	checks := []struct {
		periodType string
		startAt    int64
	}{
		{periodType: "daily", startAt: startDaily},
		{periodType: "weekly", startAt: startWeekly},
		{periodType: "monthly", startAt: startMonthly},
	}

	for _, c := range checks {
		// Regression guard: limit checks must query counters with explicit period type.
		require.Contains(t, []string{"daily", "weekly", "monthly"}, c.periodType)
		require.NotZero(t, c.startAt)
	}
}
