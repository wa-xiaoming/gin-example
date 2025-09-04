// Package grpc 提供gRPC服务功能
package grpc

import (
	"context"
	"fmt"
	"net"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

// Server 封装了gRPC服务器
type Server struct {
	server   *grpc.Server
	logger   *zap.Logger
	port     string
	services []interface{} // 注册的服务列表
}

// NewServer 创建一个新的gRPC服务器
func NewServer(logger *zap.Logger) *Server {
	// 使用固定端口，实际项目中可以从配置文件读取
	port := ":50051"
	
	return &Server{
		server: grpc.NewServer(),
		logger: logger,
		port:   port,
	}
}

// RegisterService 注册gRPC服务
func (s *Server) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	s.server.RegisterService(desc, impl)
	s.services = append(s.services, impl)
	s.logger.Info("gRPC service registered", zap.String("service", desc.ServiceName))
}

// Start 启动gRPC服务器
func (s *Server) Start() error {
	lis, err := net.Listen("tcp", s.port)
	if err != nil {
		return fmt.Errorf("failed to listen on port %s: %w", s.port, err)
	}

	s.logger.Info("Starting gRPC server", zap.String("port", s.port))
	
	// 在goroutine中启动服务器
	go func() {
		if err := s.server.Serve(lis); err != nil {
			s.logger.Error("gRPC server error", zap.Error(err))
		}
	}()
	
	return nil
}

// Stop 停止gRPC服务器
func (s *Server) Stop() {
	s.logger.Info("Stopping gRPC server")
	s.server.GracefulStop()
}

// Client 封装了gRPC客户端
type Client struct {
	conn   *grpc.ClientConn
	logger *zap.Logger
}

// NewClient 创建一个新的gRPC客户端
func NewClient(logger *zap.Logger, target string) (*Client, error) {
	// 建立连接
	conn, err := grpc.Dial(target, grpc.WithInsecure())
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC server %s: %w", target, err)
	}
	
	return &Client{
		conn:   conn,
		logger: logger,
	}, nil
}

// GetConn 获取gRPC连接
func (c *Client) GetConn() *grpc.ClientConn {
	return c.conn
}

// Close 关闭gRPC客户端连接
func (c *Client) Close() error {
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

// HealthCheckService 健康检查服务接口
type HealthCheckService interface {
	Check(ctx context.Context) error
}