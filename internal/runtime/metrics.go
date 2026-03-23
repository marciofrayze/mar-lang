package runtime

import (
	"sort"
	"sync"
	"time"
)

var httpDurationBucketsSeconds = []float64{0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10}

type metricsCollector struct {
	mu            sync.RWMutex
	startedAt     time.Time
	totalRequests uint64
	total4xx      uint64
	total5xx      uint64
	series        map[httpSeriesKey]*httpSeries
}

type httpSeriesKey struct {
	Method string
	Route  string
}

type httpSeries struct {
	Count       uint64
	SumSeconds  float64
	BucketHits  []uint64
	StatusCount map[int]uint64
}

type metricsSnapshot struct {
	StartedAt     time.Time
	TotalRequests uint64
	Total4xx      uint64
	Total5xx      uint64
	Series        []httpSeriesSnapshot
}

type httpSeriesSnapshot struct {
	Method      string
	Route       string
	Count       uint64
	SumSeconds  float64
	BucketHits  []uint64
	StatusCount map[int]uint64
}

func newMetricsCollector() *metricsCollector {
	return &metricsCollector{
		startedAt: time.Now(),
		series:    map[httpSeriesKey]*httpSeries{},
	}
}

func (m *metricsCollector) recordRequest(method, route string, status int, duration time.Duration) {
	if method == "" {
		method = "UNKNOWN"
	}
	if route == "" {
		route = "/"
	}
	seconds := duration.Seconds()

	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++
	if status >= 400 && status < 500 {
		m.total4xx++
	}
	if status >= 500 {
		m.total5xx++
	}

	key := httpSeriesKey{Method: method, Route: route}
	current := m.series[key]
	if current == nil {
		current = &httpSeries{
			BucketHits:  make([]uint64, len(httpDurationBucketsSeconds)),
			StatusCount: map[int]uint64{},
		}
		m.series[key] = current
	}
	current.Count++
	current.SumSeconds += seconds
	current.StatusCount[status]++

	index := bucketIndex(seconds)
	if index >= 0 && index < len(current.BucketHits) {
		current.BucketHits[index]++
	}
}

func (m *metricsCollector) snapshot() metricsSnapshot {
	m.mu.RLock()
	defer m.mu.RUnlock()

	out := metricsSnapshot{
		StartedAt:     m.startedAt,
		TotalRequests: m.totalRequests,
		Total4xx:      m.total4xx,
		Total5xx:      m.total5xx,
		Series:        make([]httpSeriesSnapshot, 0, len(m.series)),
	}
	for key, series := range m.series {
		statusCopy := make(map[int]uint64, len(series.StatusCount))
		for status, count := range series.StatusCount {
			statusCopy[status] = count
		}
		bucketsCopy := make([]uint64, len(series.BucketHits))
		copy(bucketsCopy, series.BucketHits)
		out.Series = append(out.Series, httpSeriesSnapshot{
			Method:      key.Method,
			Route:       key.Route,
			Count:       series.Count,
			SumSeconds:  series.SumSeconds,
			BucketHits:  bucketsCopy,
			StatusCount: statusCopy,
		})
	}

	sort.Slice(out.Series, func(i, j int) bool {
		if out.Series[i].Route == out.Series[j].Route {
			return out.Series[i].Method < out.Series[j].Method
		}
		return out.Series[i].Route < out.Series[j].Route
	})
	return out
}

func bucketIndex(seconds float64) int {
	for i, limit := range httpDurationBucketsSeconds {
		if seconds <= limit {
			return i
		}
	}
	return -1
}
