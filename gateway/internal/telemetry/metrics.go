package telemetry

import (
	"strings"
	"sync"
)

type Counter struct {
	Count   int64            `json:"count"`
	Reasons map[string]int64 `json:"reasons,omitempty"`
}

type TaskStatusMetric struct {
	CountByStatus map[string]int64 `json:"count_by_status"`
	LastStatus    string           `json:"last_status,omitempty"`
}

type Snapshot struct {
	TaskMetrics       map[string]TaskStatusMetric `json:"tasks"`
	WorkerFailures    map[string]Counter          `json:"worker_failures"`
	SaturationMetrics map[string]Counter          `json:"saturation"`
	RateLimitMetrics  map[string]Counter          `json:"rate_limits"`
}

var global = struct {
	mu             sync.Mutex
	taskMetrics    map[string]TaskStatusMetric
	workerFailures map[string]Counter
	saturation     map[string]Counter
	rateLimits     map[string]Counter
}{
	taskMetrics:    map[string]TaskStatusMetric{},
	workerFailures: map[string]Counter{},
	saturation:     map[string]Counter{},
	rateLimits:     map[string]Counter{},
}

func RecordTask(taskType, status string) {
	taskType = normalize(taskType, "unknown")
	status = normalize(status, "unknown")

	global.mu.Lock()
	defer global.mu.Unlock()

	metric := global.taskMetrics[taskType]
	if metric.CountByStatus == nil {
		metric.CountByStatus = map[string]int64{}
	}
	metric.CountByStatus[status]++
	metric.LastStatus = status
	global.taskMetrics[taskType] = metric
}

func RecordWorkerFailure(operation, reason string) {
	recordCounter(global.workerFailures, normalize(operation, "unknown"), normalize(reason, "unknown"))
}

func RecordSaturation(resource, reason string) {
	recordCounter(global.saturation, normalize(resource, "unknown"), normalize(reason, "unknown"))
}

func RecordRateLimit(resource, reason string) {
	recordCounter(global.rateLimits, normalize(resource, "unknown"), normalize(reason, "unknown"))
}

func SnapshotMetrics() Snapshot {
	global.mu.Lock()
	defer global.mu.Unlock()

	out := Snapshot{
		TaskMetrics:       map[string]TaskStatusMetric{},
		WorkerFailures:    map[string]Counter{},
		SaturationMetrics: map[string]Counter{},
		RateLimitMetrics:  map[string]Counter{},
	}

	for key, value := range global.taskMetrics {
		copyMetric := TaskStatusMetric{CountByStatus: map[string]int64{}, LastStatus: value.LastStatus}
		for status, count := range value.CountByStatus {
			copyMetric.CountByStatus[status] = count
		}
		out.TaskMetrics[key] = copyMetric
	}
	copyCounters(out.WorkerFailures, global.workerFailures)
	copyCounters(out.SaturationMetrics, global.saturation)
	copyCounters(out.RateLimitMetrics, global.rateLimits)
	return out
}

func copyCounters(dst, src map[string]Counter) {
	for key, value := range src {
		copyCounter := Counter{Count: value.Count, Reasons: map[string]int64{}}
		for reason, count := range value.Reasons {
			copyCounter.Reasons[reason] = count
		}
		dst[key] = copyCounter
	}
}

func recordCounter(target map[string]Counter, resource, reason string) {
	global.mu.Lock()
	defer global.mu.Unlock()
	counter := target[resource]
	counter.Count++
	if counter.Reasons == nil {
		counter.Reasons = map[string]int64{}
	}
	counter.Reasons[reason]++
	target[resource] = counter
}

func normalize(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}
