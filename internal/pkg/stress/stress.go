package stress

import (
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"gin-example/internal/pkg/ratelimit"
	"gin-example/internal/pkg/circuitbreaker"
	"gin-example/internal/pkg/loadbalancer"
)

// StressTestConfig 压力测试配置
type StressTestConfig struct {
	Concurrency    int           // 并发数
	TotalRequests  int64         // 总请求数
	RateLimitRPS   int           // 限流RPS
	TestDuration   time.Duration // 测试持续时间
}

// DefaultStressTestConfig 默认压力测试配置
func DefaultStressTestConfig() *StressTestConfig {
	return &StressTestConfig{
		Concurrency:   100,
		TotalRequests: 10000,
		RateLimitRPS:  1000,
		TestDuration:  60 * time.Second,
	}
}

// StressTestResult 压力测试结果
type StressTestResult struct {
	TotalRequests      int64         // 总请求数
	SuccessfulRequests int64         // 成功请求数
	FailedRequests     int64         // 失败请求数
	TotalDuration      time.Duration // 总耗时
	AverageLatency     time.Duration // 平均延迟
	MaxLatency         time.Duration // 最大延迟
	MinLatency         time.Duration // 最小延迟
	Throughput         float64       // 吞吐量 (req/s)
	ErrorRate          float64       // 错误率
}

// String 格式化输出测试结果
func (r *StressTestResult) String() string {
	return fmt.Sprintf(`
压力测试结果:
  总请求数: %d
  成功请求数: %d
  失败请求数: %d
  总耗时: %v
  平均延迟: %v
  最大延迟: %v
  最小延迟: %v
  吞吐量: %.2f req/s
  错误率: %.2f%%
`,
		r.TotalRequests,
		r.SuccessfulRequests,
		r.FailedRequests,
		r.TotalDuration,
		r.AverageLatency,
		r.MaxLatency,
		r.MinLatency,
		r.Throughput,
		r.ErrorRate*100)
}

// StressTester 压力测试器
type StressTester struct {
	config *StressTestConfig
}

// NewStressTester 创建压力测试器
func NewStressTester(config *StressTestConfig) *StressTester {
	if config == nil {
		config = DefaultStressTestConfig()
	}
	
	return &StressTester{
		config: config,
	}
}

// Run 执行压力测试
func (st *StressTester) Run(targetURL string) *StressTestResult {
	fmt.Printf("开始压力测试: %s\n", targetURL)
	fmt.Printf("并发数: %d, 总请求数: %d, 持续时间: %v\n",
		st.config.Concurrency, st.config.TotalRequests, st.config.TestDuration)

	// 初始化测试组件
	rateLimiter := ratelimit.NewLeakyBucketLimiter(float64(st.config.RateLimitRPS), st.config.RateLimitRPS)
	breaker := circuitbreaker.NewCircuitBreaker(circuitbreaker.DefaultConfig())
	
	// 初始化模拟的负载均衡器
	lb, err := st.createMockLoadBalancer()
	if err != nil {
		fmt.Printf("创建负载均衡器失败: %v\n", err)
		return nil
	}

	// 测试结果统计
	var totalRequests int64
	var successfulRequests int64
	var failedRequests int64
	var totalLatency time.Duration
	var maxLatency time.Duration
	var minLatency = time.Hour * 24 // 初始化为一个大值
	var latencySum time.Duration

	// 启动时间
	startTime := time.Now()

	// 创建等待组
	var wg sync.WaitGroup
	requestsChan := make(chan struct{}, st.config.TotalRequests)

	// 启动worker
	for i := 0; i < st.config.Concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			st.worker(targetURL, requestsChan, rateLimiter, breaker, lb,
				&successfulRequests, &failedRequests, &totalLatency, &maxLatency, &minLatency, &latencySum)
		}()
	}

	// 发送请求信号
	for i := int64(0); i < st.config.TotalRequests; i++ {
		requestsChan <- struct{}{}
	}
	close(requestsChan)

	// 等待所有请求完成
	wg.Wait()

	// 计算总请求数
	totalRequests = successfulRequests + failedRequests

	// 计算测试总耗时
	totalDuration := time.Since(startTime)

	// 计算平均延迟
	var averageLatency time.Duration
	if totalRequests > 0 {
		averageLatency = time.Duration(int64(latencySum) / totalRequests)
	}

	// 计算吞吐量
	throughput := float64(totalRequests) / totalDuration.Seconds()

	// 计算错误率
	var errorRate float64
	if totalRequests > 0 {
		errorRate = float64(failedRequests) / float64(totalRequests)
	}

	result := &StressTestResult{
		TotalRequests:      totalRequests,
		SuccessfulRequests: successfulRequests,
		FailedRequests:     failedRequests,
		TotalDuration:      totalDuration,
		AverageLatency:     averageLatency,
		MaxLatency:         maxLatency,
		MinLatency:         minLatency,
		Throughput:         throughput,
		ErrorRate:          errorRate,
	}

	fmt.Println(result.String())
	return result
}

// worker 工作协程
func (st *StressTester) worker(targetURL string, requestsChan <-chan struct{},
	rateLimiter *ratelimit.LeakyBucketLimiter, breaker *circuitbreaker.CircuitBreaker,
	lb *mockLoadBalancer,
	successfulRequests, failedRequests *int64, totalLatency, maxLatency, minLatency *time.Duration, latencySum *time.Duration) {
	
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	for range requestsChan {
		// 限流检查
		if !rateLimiter.Allow() {
			atomic.AddInt64(failedRequests, 1)
			continue
		}

		// 熔断器检查
		if breaker.IsOpen() {
			atomic.AddInt64(failedRequests, 1)
			continue
		}

		// 负载均衡器获取服务地址
		addr, err := lb.GetService("test-service")
		if err != nil {
			atomic.AddInt64(failedRequests, 1)
			continue
		}

		// 构造完整URL
		url := fmt.Sprintf("http://%s%s", addr, targetURL)

		// 执行HTTP请求
		startTime := time.Now()
		err = st.executeRequest(client, url)
		latency := time.Since(startTime)

		// 更新统计信息
		if err != nil {
			// 熔断器记录失败
			breaker.Execute(func() error { return err })
			atomic.AddInt64(failedRequests, 1)
		} else {
			// 熔断器记录成功
			breaker.Execute(func() error { return nil })
			atomic.AddInt64(successfulRequests, 1)
		}

		// 更新延迟统计
		atomic.AddInt64((*int64)(latencySum), int64(latency))

		// 更新最大延迟
		for {
			currentMax := atomic.LoadInt64((*int64)(maxLatency))
			if int64(latency) <= currentMax || atomic.CompareAndSwapInt64((*int64)(maxLatency), currentMax, int64(latency)) {
				break
			}
		}

		// 更新最小延迟
		for {
			currentMin := atomic.LoadInt64((*int64)(minLatency))
			if int64(latency) >= currentMin || atomic.CompareAndSwapInt64((*int64)(minLatency), currentMin, int64(latency)) {
				break
			}
		}
	}
}

// executeRequest 执行HTTP请求
func (st *StressTester) executeRequest(client *http.Client, url string) error {
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查HTTP状态码
	if resp.StatusCode >= 400 {
		return fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	return nil
}

// mockLoadBalancer 模拟的负载均衡器
type mockLoadBalancer struct {
	ServiceMap map[string][]loadbalancer.ServiceInstance
}

// GetService 获取服务地址
func (mlb *mockLoadBalancer) GetService(serviceName string) (string, error) {
	instances, exists := mlb.ServiceMap[serviceName]
	if !exists || len(instances) == 0 {
		return "", fmt.Errorf("service not found: %s", serviceName)
	}

	// 简单随机选择
	return instances[0].Address, nil
}

// createMockLoadBalancer 创建模拟的负载均衡器
func (st *StressTester) createMockLoadBalancer() (*mockLoadBalancer, error) {
	lb := &mockLoadBalancer{
		ServiceMap: make(map[string][]loadbalancer.ServiceInstance),
	}

	// 添加模拟的服务实例
	lb.ServiceMap["test-service"] = []loadbalancer.ServiceInstance{
		{Address: "127.0.0.1:8080", Weight: 1, Status: "active"},
		{Address: "127.0.0.1:8081", Weight: 2, Status: "active"},
		{Address: "127.0.0.1:8082", Weight: 1, Status: "active"},
	}

	return lb, nil
}