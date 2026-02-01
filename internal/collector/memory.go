package collector

import (
	"context"
	"time"

	"github.com/playok/only1mon/internal/model"
	"github.com/shirou/gopsutil/v4/mem"
)

type memoryCollector struct{}

func NewMemoryCollector() Collector { return &memoryCollector{} }

func (c *memoryCollector) ID() string          { return "memory" }
func (c *memoryCollector) Name() string        { return "Memory" }
func (c *memoryCollector) Description() string { return "Memory and swap usage, page faults" }
func (c *memoryCollector) Impact() model.ImpactLevel { return model.ImpactNone }
func (c *memoryCollector) Warning() string     { return "" }

func (c *memoryCollector) MetricNames() []string {
	return []string{
		"mem.total", "mem.used", "mem.free", "mem.available", "mem.cached", "mem.buffers",
		"mem.swap.total", "mem.swap.used", "mem.swap.free",
	}
}

func (c *memoryCollector) Collect(ctx context.Context) ([]model.MetricSample, error) {
	now := time.Now().Unix()
	var samples []model.MetricSample

	vm, err := mem.VirtualMemoryWithContext(ctx)
	if err == nil {
		samples = append(samples,
			makeSample(now, "memory", "mem.total", float64(vm.Total)),
			makeSample(now, "memory", "mem.used", float64(vm.Used)),
			makeSample(now, "memory", "mem.free", float64(vm.Free)),
			makeSample(now, "memory", "mem.available", float64(vm.Available)),
			makeSample(now, "memory", "mem.cached", float64(vm.Cached)),
			makeSample(now, "memory", "mem.buffers", float64(vm.Buffers)),
			makeSample(now, "memory", "mem.used_pct", vm.UsedPercent),
		)
	}

	sw, err := mem.SwapMemoryWithContext(ctx)
	if err == nil {
		samples = append(samples,
			makeSample(now, "memory", "mem.swap.total", float64(sw.Total)),
			makeSample(now, "memory", "mem.swap.used", float64(sw.Used)),
			makeSample(now, "memory", "mem.swap.free", float64(sw.Free)),
		)
	}

	return samples, nil
}
