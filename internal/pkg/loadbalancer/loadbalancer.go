package loadbalancer

import (
	"context"
	"math/rand"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pkg/errors"
	"go.etcd.io/etcd/client/v3"

	"gin-example/internal/metrics"
)

// LoadBalancer 负载均衡器接口
type LoadBalancer interface {
	// GetService 获取服务地址
	GetService(serviceName string) (string, error)
	// GetServiceWithStrategy 根据策略获取服务地址
	GetServiceWithStrategy(serviceName string, strategy Strategy) (string, error)
}

// Strategy 负载均衡策略接口
type Strategy interface {
	Select(instances []ServiceInstance) ServiceInstance
}

// BalanceType 负载均衡算法类型
type BalanceType int

const (
	Random BalanceType = iota
	RoundRobin
	WeightedRoundRobin
	LeastConnections
)

// String 实现BalanceType的字符串表示
func (b BalanceType) String() string {
	switch b {
	case Random:
		return "random"
	case RoundRobin:
		return "round-robin"
	case WeightedRoundRobin:
		return "weighted-round-robin"
	case LeastConnections:
		return "least-connections"
	default:
		return "unknown"
	}
}

// ServiceInstance 服务实例
type ServiceInstance struct {
	Address     string
	Weight      int
	Connections int64 // 当前连接数
	LastActive  time.Time
	Status      string // 状态: active, inactive, unhealthy
}

// EtcdLoadBalancer 基于etcd的负载均衡器
type EtcdLoadBalancer struct {
	client      *clientv3.Client
	serviceMap  map[string][]ServiceInstance // 服务名 -> 实例列表
	indexMap    map[string]*int32            // 服务名 -> 轮询索引
	mu          sync.RWMutex
	balanceType BalanceType
	lastUpdate  time.Time
}

// NewEtcdLoadBalancer 创建etcd负载均衡器
func NewEtcdLoadBalancer(endpoints []string, balanceType BalanceType) (*EtcdLoadBalancer, error) {
	cli, err := clientv3.New(clientv3.Config{
		Endpoints:   endpoints,
		DialTimeout: 5 * time.Second,
	})
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect etcd")
	}

	lb := &EtcdLoadBalancer{
		client:      cli,
		serviceMap:  make(map[string][]ServiceInstance),
		indexMap:    make(map[string]*int32),
		balanceType: balanceType,
		lastUpdate:  time.Now(),
	}

	// 启动服务发现
	go lb.watchServices()

	return lb, nil
}

// GetService 获取服务地址
func (lb *EtcdLoadBalancer) GetService(serviceName string) (string, error) {
	return lb.GetServiceWithStrategy(serviceName, lb.getDefaultStrategy())
}

// GetServiceWithStrategy 根据策略获取服务地址
func (lb *EtcdLoadBalancer) GetServiceWithStrategy(serviceName string, strategy Strategy) (string, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	instances, exists := lb.serviceMap[serviceName]
	if !exists || len(instances) == 0 {
		return "", errors.New("service not found: " + serviceName)
	}

	// 过滤掉不健康的实例
	healthyInstances := make([]ServiceInstance, 0)
	for _, instance := range instances {
		if instance.Status == "active" || instance.Status == "" {
			healthyInstances = append(healthyInstances, instance)
		}
	}

	if len(healthyInstances) == 0 {
		return "", errors.New("no healthy instances for service: " + serviceName)
	}

	// 使用策略选择实例
	selected := strategy.Select(healthyInstances)

	// 增加连接数
	atomic.AddInt64(&selected.Connections, 1)
	selected.LastActive = time.Now()

	// 记录负载均衡请求
	metrics.RecordLoadBalancerRequest(serviceName, selected.Address)

	return selected.Address, nil
}

// getDefaultStrategy 获取默认策略
func (lb *EtcdLoadBalancer) getDefaultStrategy() Strategy {
	switch lb.balanceType {
	case Random:
		return &RandomStrategy{}
	case RoundRobin:
		return &RoundRobinStrategy{indexMap: lb.indexMap}
	case WeightedRoundRobin:
		return &WeightedRoundRobinStrategy{indexMap: lb.indexMap}
	case LeastConnections:
		return &LeastConnectionsStrategy{}
	default:
		return &RandomStrategy{}
	}
}

// RandomStrategy 随机策略
type RandomStrategy struct{}

func (rs *RandomStrategy) Select(instances []ServiceInstance) ServiceInstance {
	rand.Seed(time.Now().UnixNano())
	return instances[rand.Intn(len(instances))]
}

// RoundRobinStrategy 轮询策略
type RoundRobinStrategy struct {
	indexMap map[string]*int32
	mu       sync.RWMutex
}

func (rrs *RoundRobinStrategy) Select(instances []ServiceInstance) ServiceInstance {
	// 使用服务实例列表的字符串表示作为key
	key := ""
	for _, instance := range instances {
		key += instance.Address + ","
	}

	rrs.mu.RLock()
	index, ok := rrs.indexMap[key]
	rrs.mu.RUnlock()

	if !ok {
		index = new(int32)
		rrs.mu.Lock()
		rrs.indexMap[key] = index
		rrs.mu.Unlock()
	}

	currentIndex := int(atomic.AddInt32(index, 1))
	selected := instances[(currentIndex-1)%len(instances)]
	return selected
}

// WeightedRoundRobinStrategy 加权轮询策略
type WeightedRoundRobinStrategy struct {
	indexMap map[string]*int32
	mu       sync.RWMutex
}

func (wrrs *WeightedRoundRobinStrategy) Select(instances []ServiceInstance) ServiceInstance {
	// 构建加权实例列表
	var weightedInstances []ServiceInstance
	for _, instance := range instances {
		for i := 0; i < instance.Weight; i++ {
			weightedInstances = append(weightedInstances, instance)
		}
	}

	if len(weightedInstances) == 0 {
		return ServiceInstance{}
	}

	// 使用服务实例列表的字符串表示作为key
	key := ""
	for _, instance := range weightedInstances {
		key += instance.Address + ","
	}

	wrrs.mu.RLock()
	index, ok := wrrs.indexMap[key]
	wrrs.mu.RUnlock()

	if !ok {
		index = new(int32)
		wrrs.mu.Lock()
		wrrs.indexMap[key] = index
		wrrs.mu.Unlock()
	}

	currentIndex := int(atomic.AddInt32(index, 1))
	selected := weightedInstances[(currentIndex-1)%len(weightedInstances)]
	return selected
}

// LeastConnectionsStrategy 最少连接策略
type LeastConnectionsStrategy struct{}

func (lcs *LeastConnectionsStrategy) Select(instances []ServiceInstance) ServiceInstance {
	if len(instances) == 0 {
		return ServiceInstance{}
	}

	minConnections := atomic.LoadInt64(&instances[0].Connections)
	selected := instances[0]

	for _, instance := range instances[1:] {
		connections := atomic.LoadInt64(&instance.Connections)
		if connections < minConnections {
			minConnections = connections
			selected = instance
		}
	}

	return selected
}

// watchServices 监听服务变化
func (lb *EtcdLoadBalancer) watchServices() {
	// 定期刷新服务列表
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	// 立即加载一次服务
	lb.refreshServices()

	for {
		select {
		case <-ticker.C:
			lb.refreshServices()
		}
	}
}

// refreshServices 刷新服务列表
func (lb *EtcdLoadBalancer) refreshServices() {
	// 获取所有服务
	resp, err := lb.client.Get(context.Background(), "/services/", clientv3.WithPrefix())
	if err != nil {
		// 日志记录错误，但不中断程序
		return
	}

	lb.mu.Lock()
	defer lb.mu.Unlock()

	// 清空旧的服务列表
	oldServiceMap := lb.serviceMap
	lb.serviceMap = make(map[string][]ServiceInstance)

	// 解析服务信息
	for _, kv := range resp.Kvs {
		key := string(kv.Key)
		value := string(kv.Value)

		// 解析服务名，格式: /services/{serviceName}/{address}
		// 这里简化处理，实际项目中可能需要更复杂的解析逻辑
		serviceName := "default" // 默认服务名
		if len(key) > 9 {        // "/services/".length = 9
			// 提取服务名
			parts := strings.Split(key[9:], "/")
			if len(parts) > 0 {
				serviceName = parts[0]
			}
		}

		// 解析权重（如果有）
		weight := 1
		addrParts := strings.Split(value, ":")
		if len(addrParts) == 3 {
			// 格式: address:port:weight
			if w, err := strconv.Atoi(addrParts[2]); err == nil {
				weight = w
			}
		}

		// 检查实例是否已存在，如果存在则保留连接数等状态
		var connections int64 = 0
		lastActive := time.Now()
		status := "active"

		if oldInstances, ok := oldServiceMap[serviceName]; ok {
			for _, oldInstance := range oldInstances {
				if oldInstance.Address == value {
					connections = oldInstance.Connections
					lastActive = oldInstance.LastActive
					status = oldInstance.Status
					break
				}
			}
		}

		instance := ServiceInstance{
			Address:     value,
			Weight:      weight,
			Connections: connections,
			LastActive:  lastActive,
			Status:      status,
		}

		lb.serviceMap[serviceName] = append(lb.serviceMap[serviceName], instance)
	}

	// 初始化索引
	for serviceName := range lb.serviceMap {
		if _, exists := lb.indexMap[serviceName]; !exists {
			lb.indexMap[serviceName] = new(int32)
		}
	}

	lb.lastUpdate = time.Now()
}

// HealthCheck 健康检查
func (lb *EtcdLoadBalancer) HealthCheck() {
	lb.mu.Lock()
	defer lb.mu.Unlock()

	for serviceName, instances := range lb.serviceMap {
		for i, instance := range instances {
			// 执行健康检查
			if lb.isInstanceHealthy(instance.Address) {
				// 实例健康
				if instance.Status != "active" {
					lb.serviceMap[serviceName][i].Status = "active"
				}
			} else {
				// 实例不健康
				lb.serviceMap[serviceName][i].Status = "unhealthy"

				// 记录负载均衡失败
				metrics.RecordLoadBalancerFailure(serviceName, instance.Address)
			}
		}
	}
}

// isInstanceHealthy 检查实例是否健康
func (lb *EtcdLoadBalancer) isInstanceHealthy(address string) bool {
	// 简单的TCP连接检查
	conn, err := net.DialTimeout("tcp", address, 3*time.Second)
	if err != nil {
		return false
	}
	defer conn.Close()
	return true
}

// GetServiceMap 获取服务映射表（用于监控）
func (lb *EtcdLoadBalancer) GetServiceMap() map[string][]ServiceInstance {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// 返回副本以避免并发问题
	result := make(map[string][]ServiceInstance)
	for k, v := range lb.serviceMap {
		instances := make([]ServiceInstance, len(v))
		copy(instances, v)
		result[k] = instances
	}
	return result
}

// GetLastUpdate 获取最后更新时间
func (lb *EtcdLoadBalancer) GetLastUpdate() time.Time {
	lb.mu.RLock()
	defer lb.mu.RUnlock()
	return lb.lastUpdate
}
