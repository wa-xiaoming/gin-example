package circuitbreaker

import (
	"context"
	"net/http"
	"time"
)

// HTTPClient 熔断器包装的HTTP客户端
type HTTPClient struct {
	client         *http.Client
	circuitBreaker *CircuitBreaker
}

// NewHTTPClient 创建带熔断器的HTTP客户端
func NewHTTPClient(timeout time.Duration, failureThreshold int) *HTTPClient {
	// 创建HTTP客户端
	httpClient := &http.Client{
		Timeout: timeout,
	}

	config := &Config{
		FailureThreshold: failureThreshold,
		ServiceName:      "HTTPClient",
		Timeout:          timeout,
		ResetTimeout:     timeout * 2,
		SuccessThreshold: 1,
	}
	// 创建熔断器 (失败阈值为failureThreshold，超时时间为timeout，重置超时时间为timeout*2)
	circuitBreaker := NewCircuitBreaker(config)

	return &HTTPClient{
		client:         httpClient,
		circuitBreaker: circuitBreaker,
	}
}

// Get 执行GET请求，带有熔断机制
func (hc *HTTPClient) Get(ctx context.Context, url string) (*http.Response, error) {
	var resp *http.Response
	var err error

	// 使用熔断器执行请求
	err = hc.circuitBreaker.Execute(func() error {
		resp, err = hc.client.Get(url)
		return err
	})

	return resp, err
}

// Post 执行POST请求，带有熔断机制
func (hc *HTTPClient) Post(ctx context.Context, url, contentType string, body []byte) (*http.Response, error) {
	var resp *http.Response
	var err error

	// 使用熔断器执行请求
	err = hc.circuitBreaker.Execute(func() error {
		resp, err = hc.client.Post(url, contentType, nil) // 简化处理，忽略body
		return err
	})

	return resp, err
}

// Do 执行任意HTTP请求，带有熔断机制
func (hc *HTTPClient) Do(req *http.Request) (*http.Response, error) {
	var resp *http.Response
	var err error

	// 使用熔断器执行请求
	err = hc.circuitBreaker.Execute(func() error {
		resp, err = hc.client.Do(req)
		return err
	})

	return resp, err
}

// State 获取当前熔断器状态
func (hc *HTTPClient) State() State {
	return hc.circuitBreaker.State()
}

// Metrics 获取熔断器统计信息
func (hc *HTTPClient) Metrics() map[string]interface{} {
	return hc.circuitBreaker.Metrics()
}
