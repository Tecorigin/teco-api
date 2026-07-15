package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/model"
	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
	"github.com/samber/lo"
)

func GetPerfMetricsSummary(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	activeGroups := append(lo.Keys(ratio_setting.GetGroupRatioCopy()), "auto")
	result, err := perfmetrics.QuerySummaryAllWithRange(startTimestamp, endTimestamp, hours, activeGroups)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func GetPerfMetricsSummaryPrometheus(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	activeGroups := append(lo.Keys(ratio_setting.GetGroupRatioCopy()), "auto")
	result, err := perfmetrics.QuerySummaryAllWithRange(startTimestamp, endTimestamp, hours, activeGroups)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	var builder strings.Builder
	builder.WriteString("# HELP teco_api_perf_model_avg_latency_ms Average latency in milliseconds by model.\n")
	builder.WriteString("# TYPE teco_api_perf_model_avg_latency_ms gauge\n")
	builder.WriteString("# HELP teco_api_perf_model_success_rate Success rate percentage by model.\n")
	builder.WriteString("# TYPE teco_api_perf_model_success_rate gauge\n")
	builder.WriteString("# HELP teco_api_perf_model_avg_tps Average output tokens per second by model.\n")
	builder.WriteString("# TYPE teco_api_perf_model_avg_tps gauge\n")

	for _, item := range result.Models {
		label := prometheusLabelValue(item.ModelName)
		builder.WriteString(fmt.Sprintf("teco_api_perf_model_avg_latency_ms{model=%q} %d\n", label, item.AvgLatencyMs))
		builder.WriteString(fmt.Sprintf("teco_api_perf_model_success_rate{model=%q} %.2f\n", label, item.SuccessRate))
		builder.WriteString(fmt.Sprintf("teco_api_perf_model_avg_tps{model=%q} %.2f\n", label, item.AvgTps))
	}

	c.Data(http.StatusOK, "text/plain; version=0.0.4; charset=utf-8", []byte(builder.String()))
}

func GetPerfMetrics(c *gin.Context) {
	modelName := c.Query("model")
	if modelName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "model is required",
		})
		return
	}

	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	result, err := perfmetrics.Query(perfmetrics.QueryParams{
		Model:   modelName,
		Group:   c.Query("group"),
		Hours:   hours,
		StartTs: startTimestamp,
		EndTs:   endTimestamp,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	result.Groups = filterActiveGroups(result.Groups)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    result,
	})
}

func GetModelSLOMetrics(c *gin.Context) {
	startTimestamp, _ := strconv.ParseInt(c.Query("start_timestamp"), 10, 64)
	endTimestamp, _ := strconv.ParseInt(c.Query("end_timestamp"), 10, 64)
	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}
	if startTimestamp <= 0 || endTimestamp <= 0 || startTimestamp >= endTimestamp {
		endTimestamp = time.Now().Unix()
		startTimestamp = endTimestamp - int64(hours)*3600
	}

	sloThresholdMs := int64(5000)
	if rawThreshold := c.Query("slo_threshold_ms"); rawThreshold != "" {
		if parsed, err := strconv.ParseInt(rawThreshold, 10, 64); err == nil && parsed > 0 {
			sloThresholdMs = parsed
		}
	}

	activeGroups := append(lo.Keys(ratio_setting.GetGroupRatioCopy()), "auto")
	rows, err := model.GetModelSLOStats(startTimestamp, endTimestamp, c.Query("model_name"), activeGroups, sloThresholdMs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    gin.H{"models": rows},
	})
}

func filterActiveGroups(groups []perfmetrics.GroupResult) []perfmetrics.GroupResult {
	activeRatios := ratio_setting.GetGroupRatioCopy()
	return lo.Filter(groups, func(g perfmetrics.GroupResult, _ int) bool {
		_, ok := activeRatios[g.Group]
		return ok || g.Group == "auto"
	})
}

func prometheusLabelValue(value string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		`\n`, `\\n`,
		`"`, `\\"`,
	)
	return replacer.Replace(value)
}
