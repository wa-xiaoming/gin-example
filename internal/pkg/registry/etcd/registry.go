package etcd

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"

	"gin-example/configs"

	"github.com/pkg/errors"
	clientv3 "go.etcd.io/etcd/client/v3"
	"go.uber.org/zap"
)

type Registry struct {
	client  *clientv3.Client
	leaseID clientv3.LeaseID
	key     string
	value   string
	ctx     context.Context
	cancel  context.CancelFunc
	logger  *zap.Logger
}

// NewRegistry 创建etcd注册实例
func NewRegistry(logger *zap.Logger) (*Registry, error) {
	// 解析端口
	portStr := strings.TrimPrefix(configs.ProjectPort, ":")
	port, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, errors.Wrap(err, "invalid port")
	}

	// 获取本机IP
	ip, err := getLocalIP()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get local ip")
	}

	// 获取etcd配置
	etcdConfig := configs.Get().Etcd
	endpoints := etcdConfig.Endpoints
	// 如果配置文件中没有配置etcd地址，则使用默认地址
	if len(endpoints) == 0 {
		endpoints = []string{"localhost:2379"}
	}

	// 构建etcd客户端，增加连接重试机制
	var cli *clientv3.Client
	var connErr error
	
	// 重试连接etcd
	for i := 0; i < 3; i++ {
		cli, connErr = clientv3.New(clientv3.Config{
			Endpoints:   endpoints,
			DialTimeout: 5 * time.Second,
		})
		if connErr == nil {
			break
		}
		logger.Warn("Failed to connect etcd, retrying...", 
			zap.Error(connErr), 
			zap.Int("attempt", i+1))
		
		// 等待一段时间再重试
		time.Sleep(time.Duration(i+1) * time.Second)
	}
	
	if connErr != nil {
		return nil, errors.Wrap(connErr, "failed to connect etcd after 3 attempts")
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &Registry{
		client: cli,
		key:    fmt.Sprintf("/services/%s/%s:%d", configs.ProjectName, ip, port),
		value:  fmt.Sprintf("%s:%d", ip, port),
		ctx:    ctx,
		cancel: cancel,
		logger: logger,
	}, nil
}

// Register 注册服务
func (r *Registry) Register() error {
	// 如果已有租约，先尝试撤销
	if r.leaseID != 0 {
		_, err := r.client.Revoke(r.ctx, r.leaseID)
		if err != nil {
			r.logger.Warn("Failed to revoke previous lease", zap.Error(err))
		}
	}

	// 申请租约
	leaseResp, err := r.client.Grant(r.ctx, 30) // 30秒租约，提高稳定性
	if err != nil {
		return errors.Wrap(err, "failed to grant lease")
	}
	r.leaseID = leaseResp.ID

	// 注册服务
	_, err = r.client.Put(r.ctx, r.key, r.value, clientv3.WithLease(r.leaseID))
	if err != nil {
		return errors.Wrap(err, "failed to register service")
	}

	// 启动心跳保持
	go r.keepAlive()

	r.logger.Info("service registered", zap.String("key", r.key), zap.String("value", r.value))
	return nil
}

// Deregister 注销服务
func (r *Registry) Deregister() error {
	// 先检查client是否为nil
	if r.client == nil {
		r.logger.Warn("Etcd client is nil, skip deregister")
		return nil
	}
	
	// 先取消上下文
	if r.cancel != nil {
		r.cancel()
	}

	// 删除服务注册信息
	_, err := r.client.Delete(r.ctx, r.key)
	if err != nil {
		r.logger.Error("Failed to deregister service", zap.Error(err))
		// 即使删除失败，也要关闭客户端
		r.client.Close()
		return errors.Wrap(err, "failed to deregister service")
	}

	// 关闭etcd客户端
	r.client.Close()
	
	r.logger.Info("Service deregistered", zap.String("key", r.key))
	return nil
}

// keepAlive 保持心跳
func (r *Registry) keepAlive() {
	ticker := time.NewTicker(10 * time.Second) // 每10秒心跳一次
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			r.logger.Info("Keep alive stopped due to context cancellation")
			return
		case <-ticker.C:
			// 检查租约是否仍然有效
			_, err := r.client.Put(r.ctx, r.key, r.value, clientv3.WithLease(r.leaseID))
			if err != nil {
				r.logger.Error("Failed to keep alive", zap.Error(err))
				// 如果是租约失效错误，则重新注册
				if strings.Contains(err.Error(), "requested lease not found") {
					r.logger.Info("Lease expired, re-registering service")
					if err := r.Register(); err != nil {
						r.logger.Error("Failed to re-register service", zap.Error(err))
					}
				} else {
					// 其他错误尝试重新注册
					if err := r.Register(); err != nil {
						r.logger.Error("Failed to re-register service", zap.Error(err))
					}
				}
			}
		}
	}
}

// reRegister 重新注册服务
func (r *Registry) reRegister() {
	r.logger.Info("re-registering service")
	
	// 等待一段时间再重新注册
	time.Sleep(2 * time.Second)
	
	// 重新申请租约
	leaseResp, err := r.client.Grant(r.ctx, 30) // 使用30秒租约保持一致性
	if err != nil {
		r.logger.Error("failed to re-grant lease", zap.Error(err))
		return
	}
	r.leaseID = leaseResp.ID

	// 重新注册服务
	_, err = r.client.Put(r.ctx, r.key, r.value, clientv3.WithLease(r.leaseID))
	if err != nil {
		r.logger.Error("failed to re-register service", zap.Error(err))
		return
	}

	r.logger.Info("service re-registered", zap.String("key", r.key), zap.String("value", r.value))
	
	// 重新启动心跳保持
	go r.keepAlive()
}

// getLocalIP 获取本机IP
func getLocalIP() (string, error) {
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		return "", err
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	return localAddr.IP.String(), nil
}