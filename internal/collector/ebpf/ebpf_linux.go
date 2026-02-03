//go:build linux && ebpf

package ebpf

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/playok/only1mon/internal/model"
)

// EBPFCollector provides kernel-level metrics via eBPF probes.
type EBPFCollector struct {
	available bool
}

func New() *EBPFCollector {
	c := &EBPFCollector{}
	c.available = checkCapability()
	return c
}

func (c *EBPFCollector) ID() string   { return "ebpf" }
func (c *EBPFCollector) Name() string { return "eBPF" }
func (c *EBPFCollector) Description() string {
	return "Kernel-level latency analysis: block I/O, TCP connect, scheduler runqueue"
}
func (c *EBPFCollector) Impact() model.ImpactLevel { return model.ImpactHigh }
func (c *EBPFCollector) Warning() string {
	if !c.available {
		return "eBPF not available: insufficient privileges (need root or CAP_BPF)"
	}
	return "Attaches kernel probes; requires root/CAP_BPF"
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
	if !c.available {
		return nil, nil
	}

	now := time.Now().Unix()
	var samples []model.MetricSample

	// TODO: Implement actual eBPF probe loading and reading via cilium/ebpf
	// This is a placeholder structure showing how probes would be integrated.
	// Each probe would:
	// 1. Load a compiled BPF program
	// 2. Attach to kprobe/tracepoint
	// 3. Read from BPF map (histogram/counter)
	// 4. Convert to MetricSample

	_ = now
	_ = samples

	return nil, nil
}

// Available returns true if this build has eBPF support.
func Available() bool { return true }

func checkCapability() bool {
	// Simple check: are we root or can we read BPF subsystem
	if os.Getuid() == 0 {
		return true
	}
	// Check for CAP_BPF (simplified)
	_, err := os.ReadFile("/sys/kernel/btf/vmlinux")
	return err == nil
}
