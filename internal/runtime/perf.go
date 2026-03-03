package runtime

import (
	"fmt"
	"net/http"
	"os"
	goruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

func (r *Runtime) perfPayload() map[string]any {
	snapshot := r.metrics.snapshot()
	var memStats goruntime.MemStats
	goruntime.ReadMemStats(&memStats)

	sqliteBytes := int64(0)
	if stat, err := os.Stat(r.App.Database); err == nil {
		sqliteBytes = stat.Size()
	}

	routes := make([]map[string]any, 0, len(snapshot.Series))
	for _, series := range snapshot.Series {
		status4xx := uint64(0)
		status5xx := uint64(0)
		for status, count := range series.StatusCount {
			if status >= 400 && status < 500 {
				status4xx += count
			}
			if status >= 500 {
				status5xx += count
			}
		}
		avgMs := 0.0
		if series.Count > 0 {
			avgMs = (series.SumSeconds / float64(series.Count)) * 1000
		}
		routes = append(routes, map[string]any{
			"method":    series.Method,
			"route":     series.Route,
			"count":     series.Count,
			"errors4xx": status4xx,
			"errors5xx": status5xx,
			"avgMs":     avgMs,
		})
	}

	return map[string]any{
		"uptimeSeconds": time.Since(snapshot.StartedAt).Seconds(),
		"goroutines":    goruntime.NumGoroutine(),
		"memoryBytes":   memStats.Alloc,
		"sqliteBytes":   sqliteBytes,
		"http": map[string]any{
			"totalRequests": snapshot.TotalRequests,
			"errors4xx":     snapshot.Total4xx,
			"errors5xx":     snapshot.Total5xx,
			"routes":        routes,
		},
	}
}

func (r *Runtime) writePrometheus(w http.ResponseWriter) {
	snapshot := r.metrics.snapshot()
	var memStats goruntime.MemStats
	goruntime.ReadMemStats(&memStats)

	sqliteBytes := int64(0)
	if stat, err := os.Stat(r.App.Database); err == nil {
		sqliteBytes = stat.Size()
	}

	w.Header().Set("content-type", "text/plain; version=0.0.4; charset=utf-8")

	var b strings.Builder
	appendMetricHeader(&b, "belm_process_uptime_seconds", "gauge", "Belm process uptime in seconds.")
	fmt.Fprintf(&b, "belm_process_uptime_seconds %.6f\n", time.Since(snapshot.StartedAt).Seconds())

	appendMetricHeader(&b, "belm_process_goroutines", "gauge", "Number of goroutines in the Belm process.")
	fmt.Fprintf(&b, "belm_process_goroutines %d\n", goruntime.NumGoroutine())

	appendMetricHeader(&b, "belm_process_memory_bytes", "gauge", "Allocated heap bytes in the Belm process.")
	fmt.Fprintf(&b, "belm_process_memory_bytes %d\n", memStats.Alloc)

	appendMetricHeader(&b, "belm_sqlite_file_bytes", "gauge", "Size of the SQLite database file in bytes.")
	fmt.Fprintf(&b, "belm_sqlite_file_bytes %d\n", sqliteBytes)

	appendMetricHeader(&b, "belm_http_requests_total", "counter", "HTTP requests handled by Belm, partitioned by method, route, and status.")
	appendMetricHeader(&b, "belm_http_request_duration_seconds", "histogram", "HTTP request duration histogram by method and route.")
	appendMetricHeader(&b, "belm_http_errors_total", "counter", "HTTP error responses handled by Belm, partitioned by class.")

	fmt.Fprintf(&b, "belm_http_errors_total{class=\"%s\"} %d\n", promLabel("4xx"), snapshot.Total4xx)
	fmt.Fprintf(&b, "belm_http_errors_total{class=\"%s\"} %d\n", promLabel("5xx"), snapshot.Total5xx)

	for _, series := range snapshot.Series {
		statuses := make([]int, 0, len(series.StatusCount))
		for status := range series.StatusCount {
			statuses = append(statuses, status)
		}
		sort.Ints(statuses)
		for _, status := range statuses {
			fmt.Fprintf(
				&b,
				"belm_http_requests_total{method=\"%s\",route=\"%s\",status=\"%s\"} %d\n",
				promLabel(series.Method),
				promLabel(series.Route),
				promLabel(strconv.Itoa(status)),
				series.StatusCount[status],
			)
		}

		cumulative := uint64(0)
		for i, limit := range httpDurationBucketsSeconds {
			cumulative += series.BucketHits[i]
			fmt.Fprintf(
				&b,
				"belm_http_request_duration_seconds_bucket{method=\"%s\",route=\"%s\",le=\"%s\"} %d\n",
				promLabel(series.Method),
				promLabel(series.Route),
				promLabel(strconv.FormatFloat(limit, 'f', -1, 64)),
				cumulative,
			)
		}
		fmt.Fprintf(
			&b,
			"belm_http_request_duration_seconds_bucket{method=\"%s\",route=\"%s\",le=\"%s\"} %d\n",
			promLabel(series.Method),
			promLabel(series.Route),
			promLabel("+Inf"),
			series.Count,
		)
		fmt.Fprintf(
			&b,
			"belm_http_request_duration_seconds_sum{method=\"%s\",route=\"%s\"} %.9f\n",
			promLabel(series.Method),
			promLabel(series.Route),
			series.SumSeconds,
		)
		fmt.Fprintf(
			&b,
			"belm_http_request_duration_seconds_count{method=\"%s\",route=\"%s\"} %d\n",
			promLabel(series.Method),
			promLabel(series.Route),
			series.Count,
		)
	}

	_, _ = w.Write([]byte(b.String()))
}

func appendMetricHeader(b *strings.Builder, name, metricType, help string) {
	fmt.Fprintf(b, "# HELP %s %s\n", name, help)
	fmt.Fprintf(b, "# TYPE %s %s\n", name, metricType)
}

func promLabel(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "\\", "\\\\"), "\"", "\\\"")
}
