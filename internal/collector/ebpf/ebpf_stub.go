//go:build !linux || !ebpf

package ebpf

import (
	"context"

	"github.com/playok/only1mon/internal/model"
)

// EBPFCollector is a no-op stub for non-Linux or non-eBPF builds.
type EBPFCollector struct{}

func New() *EBPFCollector              { return &EBPFCollector{} }
func (c *EBPFCollector) ID() string    { return "ebpf" }
func (c *EBPFCollector) Name() string  { return "eBPF (unavailable)" }
func (c *EBPFCollector) Description() string {
	return "Kernel-level latency analysis (requires Linux + eBPF support)"
}
func (c *EBPFCollector) Impact() model.ImpactLevel { return model.ImpactHigh }
func (c *EBPFCollector) Warning() string {
	return "Requires Linux with root/CAP_BPF; not available on this build"
}
func (c *EBPFCollector) MetricNames() []string {
	return []string{
		"ebpf.bio_latency_us.p50", "ebpf.bio_latency_us.p90", "ebpf.bio_latency_us.p99",
		"ebpf.tcp_connect_latency_us.p50", "ebpf.tcp_connect_latency_us.p90", "ebpf.tcp_connect_latency_us.p99",
		"ebpf.runqueue_latency_us.avg",
		"ebpf.cache_hit_rate",
	}
}
func (c *EBPFCollector) Collect(ctx context.Context) ([]model.MetricSample, error) {
	return nil, nil
}

// Available returns false for stub builds.
func Available() bool { return false }
