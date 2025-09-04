package alert

import (
	"fmt"
	"log"
	"sync"
	"time"

	"gin-example/internal/metrics"
)

// AlertSeverity 告警严重程度
type AlertSeverity string

const (
	AlertSeverityLow    AlertSeverity = "low"
	AlertSeverityMedium AlertSeverity = "medium"
	AlertSeverityHigh   AlertSeverity = "high"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertType 告警类型
type AlertType string

const (
	AlertTypeRateLimit    AlertType = "rate_limit"
	AlertTypeCircuitBreak AlertType = "circuit_breaker"
	AlertTypeHighError    AlertType = "high_error_rate"
	AlertTypeHighLatency  AlertType = "high_latency"
	AlertTypeLowThroughput AlertType = "low_throughput"
	AlertTypeSystem       AlertType = "system"
)

// Alert 告警信息
type Alert struct {
	ID          string        `json:"id"`
	Type        AlertType     `json:"type"`
	Severity    AlertSeverity `json:"severity"`
	Message     string        `json:"message"`
	Source      string        `json:"source"`
	Timestamp   time.Time     `json:"timestamp"`
	Value       float64       `json:"value"`
	Threshold   float64       `json:"threshold"`
	Resolved    bool          `json:"resolved"`
	ResolveTime time.Time     `json:"resolve_time,omitempty"`
}

// AlertRule 告警规则
type AlertRule struct {
	Name        string        `json:"name"`
	Type        AlertType     `json:"type"`
	Severity    AlertSeverity `json:"severity"`
	Threshold   float64       `json:"threshold"`
	Duration    time.Duration `json:"duration"`
	Enabled     bool          `json:"enabled"`
	Description string        `json:"description"`
}

// AlertHandler 告警处理器接口
type AlertHandler interface {
	HandleAlert(alert *Alert)
}

// LoggerAlertHandler 日志告警处理器
type LoggerAlertHandler struct{}

// HandleAlert 处理告警
func (l *LoggerAlertHandler) HandleAlert(alert *Alert) {
	log.Printf("[ALERT] %s - %s: %s (Value: %f, Threshold: %f)", 
		alert.Severity, alert.Type, alert.Message, alert.Value, alert.Threshold)
	
	// 记录告警指标
	metrics.RecordAlert(string(alert.Type), string(alert.Severity))
}

// EmailAlertHandler 邮件告警处理器
type EmailAlertHandler struct {
	Recipients []string
}

// HandleAlert 处理告警
func (e *EmailAlertHandler) HandleAlert(alert *Alert) {
	// 实际项目中应该发送邮件
	fmt.Printf("[EMAIL ALERT] Sending alert to %v: %s - %s\n", 
		e.Recipients, alert.Type, alert.Message)
	
	// 记录告警指标
	metrics.RecordAlert(string(alert.Type), string(alert.Severity))
}

// AlertManager 告警管理器
type AlertManager struct {
	rules       map[string]*AlertRule
	alerts      map[string]*Alert
	handlers    []AlertHandler
	mu          sync.RWMutex
	checkTicker *time.Ticker
}

// NewAlertManager 创建告警管理器
func NewAlertManager() *AlertManager {
	am := &AlertManager{
		rules:    make(map[string]*AlertRule),
		alerts:   make(map[string]*Alert),
		handlers: []AlertHandler{&LoggerAlertHandler{}},
	}

	// 初始化默认告警规则
	am.initDefaultRules()

	// 启动定期检查
	am.startChecking()

	return am
}

// initDefaultRules 初始化默认告警规则
func (am *AlertManager) initDefaultRules() {
	defaultRules := []*AlertRule{
		{
			Name:        "high_error_rate",
			Type:        AlertTypeHighError,
			Severity:    AlertSeverityHigh,
			Threshold:   0.05, // 5%错误率
			Duration:    5 * time.Minute,
			Enabled:     true,
			Description: "错误率超过阈值",
		},
		{
			Name:        "high_latency",
			Type:        AlertTypeHighLatency,
			Severity:    AlertSeverityMedium,
			Threshold:   1.0, // 1秒延迟
			Duration:    5 * time.Minute,
			Enabled:     true,
			Description: "请求延迟超过阈值",
		},
		{
			Name:        "rate_limit_exceeded",
			Type:        AlertTypeRateLimit,
			Severity:    AlertSeverityLow,
			Threshold:   100, // 每分钟100次限流
			Duration:    1 * time.Minute,
			Enabled:     true,
			Description: "限流触发次数超过阈值",
		},
		{
			Name:        "circuit_breaker_tripped",
			Type:        AlertTypeCircuitBreak,
			Severity:    AlertSeverityHigh,
			Threshold:   5, // 5次熔断
			Duration:    1 * time.Minute,
			Enabled:     true,
			Description: "熔断器触发次数超过阈值",
		},
	}

	for _, rule := range defaultRules {
		am.rules[rule.Name] = rule
	}
}

// AddHandler 添加告警处理器
func (am *AlertManager) AddHandler(handler AlertHandler) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.handlers = append(am.handlers, handler)
}

// AddRule 添加告警规则
func (am *AlertManager) AddRule(rule *AlertRule) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	am.rules[rule.Name] = rule
}

// RemoveRule 移除告警规则
func (am *AlertManager) RemoveRule(ruleName string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	delete(am.rules, ruleName)
}

// TriggerAlert 触发告警
func (am *AlertManager) TriggerAlert(alertType AlertType, severity AlertSeverity, message, source string, value, threshold float64) string {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	alert := &Alert{
		ID:        fmt.Sprintf("%s-%d", source, time.Now().UnixNano()),
		Type:      alertType,
		Severity:  severity,
		Message:   message,
		Source:    source,
		Timestamp: time.Now(),
		Value:     value,
		Threshold: threshold,
		Resolved:  false,
	}
	
	am.alerts[alert.ID] = alert
	
	// 处理告警
	am.handleAlert(alert)
	
	return alert.ID
}

// ResolveAlert 解决告警
func (am *AlertManager) ResolveAlert(alertID string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	
	if alert, exists := am.alerts[alertID]; exists {
		alert.Resolved = true
		alert.ResolveTime = time.Now()
	}
}

// handleAlert 处理告警
func (am *AlertManager) handleAlert(alert *Alert) {
	for _, handler := range am.handlers {
		handler.HandleAlert(alert)
	}
}

// startChecking 启动定期检查
func (am *AlertManager) startChecking() {
	am.checkTicker = time.NewTicker(1 * time.Minute)
	go func() {
		for range am.checkTicker.C {
			am.checkRules()
		}
	}()
}

// checkRules 检查告警规则
func (am *AlertManager) checkRules() {
	// 这里应该从监控系统获取实际指标进行检查
	// 由于这是一个示例，我们只记录日志
	log.Println("Checking alert rules...")
}

// GetActiveAlerts 获取活跃告警
func (am *AlertManager) GetActiveAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	var activeAlerts []*Alert
	for _, alert := range am.alerts {
		if !alert.Resolved {
			activeAlerts = append(activeAlerts, alert)
		}
	}
	
	return activeAlerts
}

// GetAlerts 获取所有告警
func (am *AlertManager) GetAlerts() []*Alert {
	am.mu.RLock()
	defer am.mu.RUnlock()
	
	alerts := make([]*Alert, 0, len(am.alerts))
	for _, alert := range am.alerts {
		alerts = append(alerts, alert)
	}
	
	return alerts
}

// Stop 停止告警管理器
func (am *AlertManager) Stop() {
	if am.checkTicker != nil {
		am.checkTicker.Stop()
	}
}