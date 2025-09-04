package jaeger

import (
	"context"
	"fmt"
	"time"

	"gin-example/configs"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
)

// Tracer 封装了OpenTelemetry追踪器
type Tracer struct {
	provider *sdktrace.TracerProvider
	logger   *zap.Logger
}

// NewTracer 创建一个新的Jaeger追踪器
func NewTracer(logger *zap.Logger) (*Tracer, error) {
	cfg := configs.Get().Jaeger
	
	// 如果没有配置Jaeger，则返回空的追踪器
	if cfg.Endpoint == "" {
		logger.Warn("Jaeger endpoint not configured, tracing disabled")
		return &Tracer{logger: logger}, nil
	}

	// 创建Jaeger导出器
	exp, err := jaeger.New(jaeger.WithCollectorEndpoint(jaeger.WithEndpoint(cfg.Endpoint)))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// 创建追踪器提供者
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(resource.NewWithAttributes(
			semconv.SchemaURL,
			semconv.ServiceNameKey.String(configs.ProjectName),
		)),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(cfg.Sampler))),
	)

	// 设置全局追踪器
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{}, propagation.Baggage{}))

	tracer := &Tracer{
		provider: tp,
		logger:   logger,
	}

	return tracer, nil
}

// StartSpan 开始一个新的span
func (t *Tracer) StartSpan(ctx context.Context, spanName string) (context.Context, trace.Span) {
	if t.provider == nil {
		// 如果没有初始化追踪器，返回一个空的span
		return ctx, trace.SpanFromContext(ctx)
	}
	
	tracer := otel.Tracer(configs.ProjectName)
	ctx, span := tracer.Start(ctx, spanName)
	return ctx, span
}

// Close 关闭追踪器，确保所有span都被发送
func (t *Tracer) Close() error {
	if t.provider == nil {
		return nil
	}
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	return t.provider.Shutdown(ctx)
}