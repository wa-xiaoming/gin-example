package metrics

import (
	"runtime"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/spf13/cast"
)

const (
	namespace = "custom_namespace"
	subsystem = "gin_example"
)

// metricsRequestsTotal metrics for request total 计数器（Counter）
var metricsRequestsTotal = prometheus.NewCounterVec(
	prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "requests_total",
		Help:      "request(ms) total",
	},
	[]string{"method", "path"},
)

// metricsRequestsCost metrics for requests cost 累积直方图（Histogram）
var metricsRequestsCost = prometheus.NewHistogramVec(
	prometheus.HistogramOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "requests_cost",
		Help:      "request(ms) cost milliseconds",
		Buckets:   prometheus.ExponentialBuckets(1, 2, 15), // 增加更多的桶以获得更精确的分布
	},
	[]string{"method", "path", "success", "http_code", "business_code", "cost_milliseconds"},
)

// 系统级指标
var (
	// Goroutines数量
	goroutines = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "goroutines",
		Help:      "Number of goroutines that currently exist",
	})

	// 内存使用情况
	memoryAlloc = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "memory_alloc_bytes",
		Help:      "Number of bytes allocated and still in use",
	})

	memorySys = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "memory_sys_bytes",
		Help:      "Number of bytes obtained from system",
	})

	memoryGC = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "memory_gc_bytes",
		Help:      "Number of bytes released by GC",
	})

	// GC相关指标
	gcPauseTotal = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "gc_pause_total_seconds",
		Help:      "Total GC pause time",
	})

	gcPauseQuantiles = prometheus.NewSummary(prometheus.SummaryOpts{
		Namespace:  namespace,
		Subsystem:  subsystem,
		Name:       "gc_pause_quantiles_seconds",
		Help:       "GC pause time quantiles",
		Objectives: map[float64]float64{0.5: 0.05, 0.9: 0.01, 0.99: 0.001},
	})

	// CPU使用情况
	cpuUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "cpu_usage_percent",
		Help:      "Current CPU usage percentage",
	})

	// 业务相关指标
	activeUsers = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "active_users",
		Help:      "Number of active users",
	})

	apiErrors = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "api_errors_total",
			Help:      "Total number of API errors",
		},
		[]string{"endpoint", "error_type"},
	)

	cacheHits = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "cache_hits_total",
		Help:      "Total number of cache hits",
	})

	cacheMisses = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "cache_misses_total",
		Help:      "Total number of cache misses",
	})

	cacheHitRatio = prometheus.NewGauge(prometheus.GaugeOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "cache_hit_ratio",
		Help:      "Cache hit ratio",
	})

	// 限流相关指标
	rateLimitAllowed = prometheus.NewCounter(prometheus.CounterOpts{
		Namespace: namespace,
		Subsystem: subsystem,
		Name:      "rate_limit_allowed_total",
		Help:      "Total number of requests allowed by rate limiter",
	})

	rateLimitExceeded = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "rate_limit_exceeded_total",
			Help:      "Total number of requests rejected by rate limiter",
		},
		[]string{"limit_type"},
	)

	// 熔断器相关指标
	circuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "circuit_breaker_state",
			Help:      "Current state of circuit breaker (0=closed, 1=open, 2=half-open)",
		},
		[]string{"service"},
	)

	circuitBreakerTripped = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "circuit_breaker_tripped_total",
			Help:      "Total number of times circuit breaker has tripped",
		},
		[]string{"service"},
	)

	// 负载均衡相关指标
	loadBalancerRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "load_balancer_requests_total",
			Help:      "Total number of load balancer requests",
		},
		[]string{"service", "instance"},
	)

	loadBalancerFailures = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "load_balancer_failures_total",
			Help:      "Total number of load balancer failures",
		},
		[]string{"service", "instance"},
	)

	// 数据库相关指标
	dbConnections = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "db_connections",
			Help:      "Current number of database connections",
		},
		[]string{"db_type", "state"}, // db_type: read/write, state: idle/used
	)

	dbQueries = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "db_queries_total",
			Help:      "Total number of database queries",
		},
		[]string{"db_type", "operation", "success"},
	)

	dbQueryDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "db_query_duration_seconds",
			Help:      "Database query duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 15),
		},
		[]string{"db_type", "operation"},
	)

	// HTTP客户端相关指标
	httpClientRequests = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_client_requests_total",
			Help:      "Total number of HTTP client requests",
		},
		[]string{"method", "host", "status"},
	)

	httpClientDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "http_client_duration_seconds",
			Help:      "HTTP client request duration in seconds",
			Buckets:   prometheus.ExponentialBuckets(0.0001, 2, 15),
		},
		[]string{"method", "host"},
	)

	// 告警相关指标
	alertsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystem,
			Name:      "alerts_total",
			Help:      "Total number of alerts triggered",
		},
		[]string{"alert_type", "severity"},
	)
)

// 根据需要，可定制其他指标，操作如下：
// 1. 定义需要的指标
// 2. init() 中注册
// 3. RecordMetrics() 中传值

func init() {
	prometheus.MustRegister(
		metricsRequestsTotal,
		metricsRequestsCost,
		goroutines,
		memoryAlloc,
		memorySys,
		memoryGC,
		gcPauseTotal,
		gcPauseQuantiles,
		cpuUsage,
		activeUsers,
		apiErrors,
		cacheHits,
		cacheMisses,
		cacheHitRatio,
		rateLimitAllowed,
		rateLimitExceeded,
		circuitBreakerState,
		circuitBreakerTripped,
		loadBalancerRequests,
		loadBalancerFailures,
		dbConnections,
		dbQueries,
		dbQueryDuration,
		httpClientRequests,
		httpClientDuration,
		alertsTotal,
	)

	// 启动系统指标收集
	go collectSystemMetrics()
}

// RecordMetrics 记录指标
func RecordMetrics(method, path string, success bool, httpCode, businessCode int, costSeconds float64) {
	metricsRequestsTotal.With(prometheus.Labels{
		"method": method,
		"path":   path,
	}).Inc()

	metricsRequestsCost.With(prometheus.Labels{
		"method":            method,
		"path":              path,
		"success":           cast.ToString(success),
		"http_code":         cast.ToString(httpCode),
		"business_code":     cast.ToString(businessCode),
		"cost_milliseconds": cast.ToString(costSeconds * 1000),
	}).Observe(costSeconds)
}

// RecordError 记录API错误
func RecordError(endpoint, errorType string) {
	apiErrors.With(prometheus.Labels{
		"endpoint":   endpoint,
		"error_type": errorType,
	}).Inc()
}

// RecordCacheHit 记录缓存命中
func RecordCacheHit() {
	cacheHits.Inc()
	// 更新缓存命中率
	updateCacheHitRatio()
}

// RecordCacheMiss 记录缓存未命中
func RecordCacheMiss() {
	cacheMisses.Inc()
	// 更新缓存命中率
	updateCacheHitRatio()
}

// updateCacheHitRatio 更新缓存命中率
func updateCacheHitRatio() {
	hits := cacheHits.Desc().String()
	misses := cacheMisses.Desc().String()
	
	// 简化处理，实际项目中应该正确计算比率
	_ = hits
	_ = misses
	
	// cacheHitRatio.Set(hitRatio)
}

// SetActiveUsers 设置活跃用户数
func SetActiveUsers(count float64) {
	activeUsers.Set(count)
}

// RecordRateLimitAllowed 记录允许的请求数
func RecordRateLimitAllowed() {
	rateLimitAllowed.Inc()
}

// RecordRateLimitExceeded 记录超出限流的请求数
func RecordRateLimitExceeded(limitType string) {
	rateLimitExceeded.With(prometheus.Labels{
		"limit_type": limitType,
	}).Inc()
}

// SetCircuitBreakerState 设置熔断器状态
func SetCircuitBreakerState(service string, state float64) {
	circuitBreakerState.With(prometheus.Labels{
		"service": service,
	}).Set(state)
}

// RecordCircuitBreakerTripped 记录熔断器跳闸次数
func RecordCircuitBreakerTripped(service string) {
	circuitBreakerTripped.With(prometheus.Labels{
		"service": service,
	}).Inc()
}

// RecordLoadBalancerRequest 记录负载均衡请求
func RecordLoadBalancerRequest(service, instance string) {
	loadBalancerRequests.With(prometheus.Labels{
		"service":  service,
		"instance": instance,
	}).Inc()
}

// RecordLoadBalancerFailure 记录负载均衡失败
func RecordLoadBalancerFailure(service, instance string) {
	loadBalancerFailures.With(prometheus.Labels{
		"service":  service,
		"instance": instance,
	}).Inc()
}

// SetDBConnections 设置数据库连接数
func SetDBConnections(dbType, state string, count float64) {
	dbConnections.With(prometheus.Labels{
		"db_type": dbType,
		"state":   state,
	}).Set(count)
}

// RecordDBQuery 记录数据库查询
func RecordDBQuery(dbType, operation, success string) {
	dbQueries.With(prometheus.Labels{
		"db_type":   dbType,
		"operation": operation,
		"success":   success,
	}).Inc()
}

// ObserveDBQueryDuration 记录数据库查询耗时
func ObserveDBQueryDuration(dbType, operation string, duration float64) {
	dbQueryDuration.With(prometheus.Labels{
		"db_type":   dbType,
		"operation": operation,
	}).Observe(duration)
}

// RecordHTTPClientRequest 记录HTTP客户端请求
func RecordHTTPClientRequest(method, host, status string) {
	httpClientRequests.With(prometheus.Labels{
		"method": method,
		"host":   host,
		"status": status,
	}).Inc()
}

// ObserveHTTPClientDuration 记录HTTP客户端请求耗时
func ObserveHTTPClientDuration(method, host string, duration float64) {
	httpClientDuration.With(prometheus.Labels{
		"method": method,
		"host":   host,
	}).Observe(duration)
}

// RecordAlert 记录告警
func RecordAlert(alertType, severity string) {
	alertsTotal.With(prometheus.Labels{
		"alert_type": alertType,
		"severity":   severity,
	}).Inc()
}

// collectSystemMetrics 收集系统指标
func collectSystemMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// 收集goroutines数量
			goroutines.Set(float64(runtime.NumGoroutine()))

			// 收集内存信息
			var ms runtime.MemStats
			runtime.ReadMemStats(&ms)
			memoryAlloc.Set(float64(ms.Alloc))
			memorySys.Set(float64(ms.Sys))
			memoryGC.Set(float64(ms.TotalAlloc - ms.Alloc))

			// 收集GC暂停时间
			gcPauseTotal.Add(float64(ms.PauseTotalNs) / 1e9)
			
			// 收集GC暂停时间分位数
			gcPauseQuantiles.Observe(float64(ms.PauseTotalNs) / 1e9)
		}
	}
}