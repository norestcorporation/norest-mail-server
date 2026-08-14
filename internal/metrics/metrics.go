package metrics

import (
	"sync"
	"time"
)

// Metrics provides lightweight metrics collection without external dependencies.
type Metrics struct {
	mu sync.RWMutex

	// HTTP metrics
	httpRequestsTotal      int64
	httpErrorsTotal        int64
	httpRequestDurationMs int64

	// Database metrics
	dbQueryDurationMs int64
	dbErrors          int64
	dbPoolUtilization float64

	// Stalwart metrics
	stalwartRequestDurationMs int64
	stalwartRequestErrors      int64
	stalwartTimeoutCount       int64

	// Provisioning metrics
	jobsStarted      int64
	jobsSucceeded    int64
	jobsFailed       int64
	jobsRetried      int64
	jobsReclaimed    int64
	jobDurationMs    int64
	pendingJobs      int64
	processingJobs   int64
	retryJobs       int64
	failedJobs       int64
	oldestPendingJob time.Time
}

var globalMetrics = &Metrics{}

// HTTPRequest records an HTTP request.
func HTTPRequest(duration time.Duration, isError bool) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	globalMetrics.httpRequestsTotal++
	globalMetrics.httpRequestDurationMs += duration.Milliseconds()
	if isError {
		globalMetrics.httpErrorsTotal++
	}
}

// DBQuery records a database query.
func DBQuery(duration time.Duration, isError bool) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	globalMetrics.dbQueryDurationMs += duration.Milliseconds()
	if isError {
		globalMetrics.dbErrors++
	}
}

// DBPoolUtilization records the current database pool utilization.
func DBPoolUtilization(utilization float64) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	globalMetrics.dbPoolUtilization = utilization
}

// StalwartRequest records a Stalwart request.
func StalwartRequest(duration time.Duration, isError bool, isTimeout bool) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	globalMetrics.stalwartRequestDurationMs += duration.Milliseconds()
	if isError {
		globalMetrics.stalwartRequestErrors++
	}
	if isTimeout {
		globalMetrics.stalwartTimeoutCount++
	}
}

// JobStarted records a job being started.
func JobStarted() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	globalMetrics.jobsStarted++
}

// JobSucceeded records a job succeeding.
func JobSucceeded(duration time.Duration) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	globalMetrics.jobsSucceeded++
	globalMetrics.jobDurationMs += duration.Milliseconds()
}

// JobFailed records a job failing.
func JobFailed() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	globalMetrics.jobsFailed++
}

// JobRetried records a job being retried.
func JobRetried() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	globalMetrics.jobsRetried++
}

// JobReclaimed records a job being reclaimed.
func JobReclaimed() {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	globalMetrics.jobsReclaimed++
}

// SetProvisioningBacklog sets the current provisioning backlog statistics.
func SetProvisioningBacklog(pending, processing, retry, failed int64, oldest time.Time) {
	globalMetrics.mu.Lock()
	defer globalMetrics.mu.Unlock()

	globalMetrics.pendingJobs = pending
	globalMetrics.processingJobs = processing
	globalMetrics.retryJobs = retry
	globalMetrics.failedJobs = failed
	globalMetrics.oldestPendingJob = oldest
}

// GetSnapshot returns a snapshot of current metrics.
func GetSnapshot() map[string]interface{} {
	globalMetrics.mu.RLock()
	defer globalMetrics.mu.RUnlock()

	return map[string]interface{}{
		"http_requests_total":      globalMetrics.httpRequestsTotal,
		"http_errors_total":        globalMetrics.httpErrorsTotal,
		"http_request_duration_ms": globalMetrics.httpRequestDurationMs,
		"db_query_duration_ms":    globalMetrics.dbQueryDurationMs,
		"db_errors":               globalMetrics.dbErrors,
		"db_pool_utilization":     globalMetrics.dbPoolUtilization,
		"stalwart_request_duration_ms": globalMetrics.stalwartRequestDurationMs,
		"stalwart_request_errors": globalMetrics.stalwartRequestErrors,
		"stalwart_timeout_count":  globalMetrics.stalwartTimeoutCount,
		"jobs_started":            globalMetrics.jobsStarted,
		"jobs_succeeded":          globalMetrics.jobsSucceeded,
		"jobs_failed":             globalMetrics.jobsFailed,
		"jobs_retried":            globalMetrics.jobsRetried,
		"jobs_reclaimed":          globalMetrics.jobsReclaimed,
		"job_duration_ms":         globalMetrics.jobDurationMs,
		"pending_jobs":            globalMetrics.pendingJobs,
		"processing_jobs":         globalMetrics.processingJobs,
		"retry_jobs":              globalMetrics.retryJobs,
		"failed_jobs":             globalMetrics.failedJobs,
		"oldest_pending_job":      globalMetrics.oldestPendingJob,
	}
}