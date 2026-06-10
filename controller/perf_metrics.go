package controller

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	perfmetrics "github.com/QuantumNous/new-api/pkg/perf_metrics"
	"github.com/QuantumNous/new-api/setting/ratio_setting"

	"github.com/gin-gonic/gin"
)

func GetPerfMetricsSummary(c *gin.Context) {
	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	result, err := perfmetrics.QuerySummaryAll(hours)
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
	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	result, err := perfmetrics.QuerySummaryAll(hours)
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

	hours := 24
	if rawHours := c.Query("hours"); rawHours != "" {
		if parsed, err := strconv.Atoi(rawHours); err == nil {
			hours = parsed
		}
	}

	result, err := perfmetrics.Query(perfmetrics.QueryParams{
		Model: modelName,
		Group: c.Query("group"),
		Hours: hours,
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

func filterActiveGroups(groups []perfmetrics.GroupResult) []perfmetrics.GroupResult {
	activeGroups := ratio_setting.GetGroupRatioCopy()
	filtered := make([]perfmetrics.GroupResult, 0, len(groups))
	for _, g := range groups {
		if _, ok := activeGroups[g.Group]; ok || g.Group == "auto" {
			filtered = append(filtered, g)
		}
	}
	return filtered
}

func prometheusLabelValue(value string) string {
	replacer := strings.NewReplacer(
		`\\`, `\\\\`,
		`\n`, `\\n`,
		`"`, `\\"`,
	)
	return replacer.Replace(value)
}
