package collector

import (
	"context"
	"runtime"
	"time"

	"github.com/only1mon/only1mon/internal/model"
	"github.com/shirou/gopsutil/v4/load"
)

type kernelCollector struct{}

func NewKernelCollector() Collector { return &kernelCollector{} }

func (c *kernelCollector) ID() string          { return "kernel" }
func (c *kernelCollector) Name() string        { return "Kernel" }
func (c *kernelCollector) Description() string { return "Kernel stats: procs running/blocked, load misc" }
func (c *kernelCollector) Impact() model.ImpactLevel { return model.ImpactNone }
func (c *kernelCollector) Warning() string     { return "" }

func (c *kernelCollector) MetricNames() []string {
	return []string{
		"kernel.procs_running", "kernel.procs_blocked",
	}
}

func (c *kernelCollector) Collect(ctx context.Context) ([]model.MetricSample, error) {
	now := time.Now().Unix()
	var samples []model.MetricSample

	if runtime.GOOS == "linux" {
		misc, err := load.MiscWithContext(ctx)
		if err == nil {
			samples = append(samples,
				makeSample(now, "kernel", "kernel.procs_running", float64(misc.ProcsRunning)),
				makeSample(now, "kernel", "kernel.procs_blocked", float64(misc.ProcsBlocked)),
			)
		}
	}

	// On non-linux, we can still report goroutine count as a proxy
	samples = append(samples,
		makeSample(now, "kernel", "kernel.goroutines", float64(runtime.NumGoroutine())),
	)

	return samples, nil
}
