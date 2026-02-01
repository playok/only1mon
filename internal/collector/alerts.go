package collector

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/only1mon/only1mon/internal/model"
	"github.com/only1mon/only1mon/internal/store"
)

// excludedMetricPrefixes lists metric prefixes for virtual/pseudo filesystems
// that should be excluded from alert evaluation (always 100% or irrelevant).
var excludedMetricPrefixes = []string{
	"disk.dev.",
	"disk.proc.",
	"disk.sys.",
	"disk.run.",
	"disk.snap.",
	"disk.tmpfs.",
	"disk.devfs.",
}

// AlertRule defines a condition that triggers an alert.
type AlertRule struct {
	MetricPattern string              // metric name or prefix pattern
	Condition     func(float64) bool  // returns true when alert should fire
	Threshold     float64             // threshold value for alert metadata
	Severity      model.AlertSeverity
	MessageEN     string // format string with one %v for the value
	MessageKO     string
}

// AlertEngine evaluates metric samples against rules and generates alerts.
type AlertEngine struct {
	mu     sync.RWMutex
	rules  []AlertRule
	active map[string]model.Alert // keyed by metric name to deduplicate
}

// NewAlertEngine creates an engine with default performance rules.
func NewAlertEngine() *AlertEngine {
	e := &AlertEngine{
		active: make(map[string]model.Alert),
	}
	e.rules = defaultRules()
	return e
}

// LoadRules loads enabled rules from the database and replaces the in-memory rules.
func (e *AlertEngine) LoadRules(db *store.Store) {
	models, err := db.ListAlertRules()
	if err != nil {
		log.Printf("[alerts] failed to load rules from DB: %v", err)
		return
	}
	var rules []AlertRule
	for _, m := range models {
		if !m.Enabled {
			continue
		}
		cond := buildCondition(m.Operator, m.Threshold)
		if cond == nil {
			log.Printf("[alerts] unknown operator %q for rule %d, skipping", m.Operator, m.ID)
			continue
		}
		rules = append(rules, AlertRule{
			MetricPattern: m.MetricPattern,
			Condition:     cond,
			Threshold:     m.Threshold,
			Severity:      m.Severity,
			MessageEN:     m.MessageEN,
			MessageKO:     m.MessageKO,
		})
	}
	e.mu.Lock()
	e.rules = rules
	e.mu.Unlock()
	log.Printf("[alerts] loaded %d rules from DB", len(rules))
}

// ReloadRules reloads rules from the database (call after CUD operations).
func (e *AlertEngine) ReloadRules(db *store.Store) {
	e.LoadRules(db)
}

// buildCondition creates a comparison function from operator string and threshold.
func buildCondition(op string, threshold float64) func(float64) bool {
	switch op {
	case "gt":
		return func(v float64) bool { return v > threshold }
	case "gte":
		return func(v float64) bool { return v >= threshold }
	case "lt":
		return func(v float64) bool { return v < threshold }
	case "lte":
		return func(v float64) bool { return v <= threshold }
	default:
		return nil
	}
}

// DefaultAlertRuleModels returns the default rules as model.AlertRule for DB seeding.
func DefaultAlertRuleModels() []model.AlertRule {
	return []model.AlertRule{
		// CPU
		{MetricPattern: "cpu.total.user", Operator: "gt", Threshold: 90, Severity: model.SeverityCritical, Enabled: true,
			MessageEN: "CPU user usage is very high at %.1f%%, system may experience processing delays",
			MessageKO: "CPU 사용자 사용률이 %.1f%%로 매우 높아 처리 지연이 발생할 수 있습니다"},
		{MetricPattern: "cpu.total.system", Operator: "gt", Threshold: 50, Severity: model.SeverityWarning, Enabled: true,
			MessageEN: "CPU system usage is elevated at %.1f%%, kernel overhead may be impacting performance",
			MessageKO: "CPU 시스템 사용률이 %.1f%%로 높아 커널 오버헤드가 성능에 영향을 줄 수 있습니다"},
		{MetricPattern: "cpu.total.iowait", Operator: "gt", Threshold: 30, Severity: model.SeverityWarning, Enabled: true,
			MessageEN: "CPU I/O wait is %.1f%%, disk operations are causing processing delays",
			MessageKO: "CPU I/O 대기가 %.1f%%로 디스크 작업이 처리 지연을 유발하고 있습니다"},
		{MetricPattern: "cpu.load.1", Operator: "gt", Threshold: 4, Severity: model.SeverityWarning, Enabled: true,
			MessageEN: "System load average (1m) is %.2f, processes may be queuing up",
			MessageKO: "시스템 부하 평균(1분)이 %.2f로 프로세스가 대기 중일 수 있습니다"},

		// Memory
		{MetricPattern: "mem.used_pct", Operator: "gt", Threshold: 90, Severity: model.SeverityCritical, Enabled: true,
			MessageEN: "Memory usage is critically high at %.1f%%, system may start swapping",
			MessageKO: "메모리 사용률이 %.1f%%로 매우 높아 스와핑이 시작될 수 있습니다"},
		{MetricPattern: "mem.used_pct", Operator: "gt", Threshold: 80, Severity: model.SeverityWarning, Enabled: true,
			MessageEN: "Memory usage is high at %.1f%%, consider freeing up resources",
			MessageKO: "메모리 사용률이 %.1f%%로 높습니다. 리소스 확보를 고려하세요"},
		{MetricPattern: "mem.swap.used", Operator: "gt", Threshold: 1073741824, Severity: model.SeverityWarning, Enabled: true,
			MessageEN: "Swap usage is high (%.0f bytes), performance degradation is likely",
			MessageKO: "스왑 사용량이 높습니다 (%.0f bytes). 성능 저하가 발생할 수 있습니다"},

		// Disk
		{MetricPattern: "disk.*.used_pct", Operator: "gt", Threshold: 95, Severity: model.SeverityCritical, Enabled: true,
			MessageEN: "Disk usage is critically high at %.1f%%, system may fail to write",
			MessageKO: "디스크 사용률이 %.1f%%로 매우 높아 쓰기 실패가 발생할 수 있습니다"},
		{MetricPattern: "disk.*.used_pct", Operator: "gt", Threshold: 85, Severity: model.SeverityWarning, Enabled: true,
			MessageEN: "Disk usage is high at %.1f%%, consider freeing up space",
			MessageKO: "디스크 사용률이 %.1f%%로 높습니다. 공간 확보를 고려하세요"},

		// Network
		{MetricPattern: "net.total.errin", Operator: "gt", Threshold: 100, Severity: model.SeverityWarning, Enabled: true,
			MessageEN: "Network receive errors detected (%.0f), possible network issues",
			MessageKO: "네트워크 수신 오류가 감지되었습니다 (%.0f). 네트워크 문제가 있을 수 있습니다"},
		{MetricPattern: "net.total.errout", Operator: "gt", Threshold: 100, Severity: model.SeverityWarning, Enabled: true,
			MessageEN: "Network transmit errors detected (%.0f), possible network issues",
			MessageKO: "네트워크 송신 오류가 감지되었습니다 (%.0f). 네트워크 문제가 있을 수 있습니다"},

		// Kernel
		{MetricPattern: "kernel.procs_blocked", Operator: "gt", Threshold: 5, Severity: model.SeverityWarning, Enabled: true,
			MessageEN: "%.0f processes are blocked on I/O, possible storage bottleneck",
			MessageKO: "%.0f개의 프로세스가 I/O에서 블록되었습니다. 스토리지 병목이 발생할 수 있습니다"},
		{MetricPattern: "kernel.runqueue_latency", Operator: "gt", Threshold: 1000, Severity: model.SeverityWarning, Enabled: true,
			MessageEN: "Run queue latency is %.0f us, scheduling delays may affect responsiveness",
			MessageKO: "실행 큐 지연시간이 %.0f us로 스케줄링 지연이 응답성에 영향을 줄 수 있습니다"},

		// GPU
		{MetricPattern: "gpu.*.temp_c", Operator: "gt", Threshold: 85, Severity: model.SeverityWarning, Enabled: true,
			MessageEN: "GPU temperature is %.1f°C, thermal throttling may occur",
			MessageKO: "GPU 온도가 %.1f°C로 높아 열 쓰로틀링이 발생할 수 있습니다"},
		{MetricPattern: "gpu.*.util_pct", Operator: "gt", Threshold: 95, Severity: model.SeverityInfo, Enabled: true,
			MessageEN: "GPU utilization is %.1f%%, running at near full capacity",
			MessageKO: "GPU 사용률이 %.1f%%로 거의 최대 용량으로 실행 중입니다"},
	}
}

// Evaluate checks samples against rules and returns any new/updated alerts.
func (e *AlertEngine) Evaluate(samples []model.MetricSample) []model.Alert {
	now := time.Now().Unix()
	triggered := make(map[string]model.Alert)

	e.mu.RLock()
	rules := e.rules
	e.mu.RUnlock()

	for _, s := range samples {
		if isExcludedMetric(s.MetricName) {
			continue
		}
		for _, rule := range rules {
			if !matchPattern(rule.MetricPattern, s.MetricName) {
				continue
			}
			if rule.Condition(s.Value) {
				alert := model.Alert{
					ID:        fmt.Sprintf("alert-%s", s.MetricName),
					Timestamp: now,
					Severity:  rule.Severity,
					Metric:    s.MetricName,
					Value:     s.Value,
					Threshold: rule.Threshold,
					MessageEN: fmt.Sprintf(rule.MessageEN, s.Value),
					MessageKO: fmt.Sprintf(rule.MessageKO, s.Value),
				}
				triggered[s.MetricName] = alert
			}
		}
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	// Remove alerts that are no longer triggered
	for key := range e.active {
		if _, still := triggered[key]; !still {
			delete(e.active, key)
		}
	}

	// Update/add triggered alerts
	var result []model.Alert
	for key, alert := range triggered {
		e.active[key] = alert
		result = append(result, alert)
	}

	return result
}

// ActiveAlerts returns all currently active alerts.
func (e *AlertEngine) ActiveAlerts() []model.Alert {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]model.Alert, 0, len(e.active))
	for _, a := range e.active {
		result = append(result, a)
	}
	return result
}

// matchPattern checks if a metric name matches a rule pattern.
// Supports exact match and wildcard "*" segments (e.g. "disk.*.used_pct").
func matchPattern(pattern, name string) bool {
	if pattern == name {
		return true
	}
	pp := splitDot(pattern)
	np := splitDot(name)
	if len(pp) != len(np) {
		return false
	}
	for i := range pp {
		if pp[i] == "*" {
			continue
		}
		if pp[i] != np[i] {
			return false
		}
	}
	return true
}

func splitDot(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// isExcludedMetric returns true if the metric belongs to a virtual/pseudo filesystem
// that should be excluded from alert evaluation.
func isExcludedMetric(name string) bool {
	for _, prefix := range excludedMetricPrefixes {
		if strings.HasPrefix(name, prefix) {
			return true
		}
	}
	return false
}

func defaultRules() []AlertRule {
	var rules []AlertRule
	for _, m := range DefaultAlertRuleModels() {
		cond := buildCondition(m.Operator, m.Threshold)
		if cond == nil {
			continue
		}
		rules = append(rules, AlertRule{
			MetricPattern: m.MetricPattern,
			Condition:     cond,
			Threshold:     m.Threshold,
			Severity:      m.Severity,
			MessageEN:     m.MessageEN,
			MessageKO:     m.MessageKO,
		})
	}
	return rules
}
