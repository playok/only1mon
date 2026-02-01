package collector

import (
	"context"
	"fmt"
	"time"

	"github.com/only1mon/only1mon/internal/model"
	"github.com/shirou/gopsutil/v4/cpu"
	"github.com/shirou/gopsutil/v4/load"
)

type cpuCollector struct {
	prevTimes    *cpu.TimesStat    // previous total CPU times for delta calculation
	prevPerCore  []cpu.TimesStat   // previous per-core CPU times
}

func NewCPUCollector() Collector { return &cpuCollector{} }

func (c *cpuCollector) ID() string          { return "cpu" }
func (c *cpuCollector) Name() string        { return "CPU" }
func (c *cpuCollector) Description() string { return "CPU usage, per-core stats, load average, context switches" }
func (c *cpuCollector) Impact() model.ImpactLevel { return model.ImpactNone }
func (c *cpuCollector) Warning() string     { return "" }

func (c *cpuCollector) MetricNames() []string {
	return []string{
		"cpu.total.usage", "cpu.total.user", "cpu.total.system", "cpu.total.idle", "cpu.total.iowait",
		"cpu.load.1", "cpu.load.5", "cpu.load.15",
		"cpu.context_switches", "cpu.interrupts",
	}
}

func (c *cpuCollector) Collect(ctx context.Context) ([]model.MetricSample, error) {
	now := time.Now().Unix()
	var samples []model.MetricSample

	// Total CPU usage — delta-based calculation
	times, err := cpu.TimesWithContext(ctx, false)
	if err == nil && len(times) > 0 {
		cur := times[0]
		if c.prevTimes != nil {
			dUser := cur.User - c.prevTimes.User
			dSystem := cur.System - c.prevTimes.System
			dIdle := cur.Idle - c.prevTimes.Idle
			dIowait := cur.Iowait - c.prevTimes.Iowait
			dSteal := cur.Steal - c.prevTimes.Steal
			dNice := cur.Nice - c.prevTimes.Nice
			dIrq := cur.Irq - c.prevTimes.Irq
			dSoftirq := cur.Softirq - c.prevTimes.Softirq
			dTotal := dUser + dSystem + dIdle + dIowait + dSteal + dNice + dIrq + dSoftirq

			if dTotal > 0 {
				busyPct := (dTotal - dIdle) / dTotal * 100
				samples = append(samples,
					makeSample(now, "cpu", "cpu.total.usage", busyPct),
					makeSample(now, "cpu", "cpu.total.user", dUser/dTotal*100),
					makeSample(now, "cpu", "cpu.total.system", dSystem/dTotal*100),
					makeSample(now, "cpu", "cpu.total.idle", dIdle/dTotal*100),
					makeSample(now, "cpu", "cpu.total.iowait", dIowait/dTotal*100),
				)
			}
		}
		c.prevTimes = &cur
	}

	// Per-core — delta-based calculation
	perCoreTimes, err := cpu.TimesWithContext(ctx, true)
	if err == nil && len(perCoreTimes) > 0 {
		if c.prevPerCore != nil && len(c.prevPerCore) == len(perCoreTimes) {
			for i, cur := range perCoreTimes {
				prev := c.prevPerCore[i]
				dBusy := (cur.User - prev.User) + (cur.System - prev.System) +
					(cur.Nice - prev.Nice) + (cur.Irq - prev.Irq) +
					(cur.Softirq - prev.Softirq) + (cur.Steal - prev.Steal)
				dTotal := dBusy + (cur.Idle - prev.Idle) + (cur.Iowait - prev.Iowait)
				pct := 0.0
				if dTotal > 0 {
					pct = dBusy / dTotal * 100
				}
				samples = append(samples, model.MetricSample{
					Timestamp:  now,
					Collector:  "cpu",
					MetricName: fmt.Sprintf("cpu.core.%d.usage", i),
					Value:      pct,
				})
			}
		}
		c.prevPerCore = perCoreTimes
	}

	// Load average
	avg, err := load.AvgWithContext(ctx)
	if err == nil {
		samples = append(samples,
			makeSample(now, "cpu", "cpu.load.1", avg.Load1),
			makeSample(now, "cpu", "cpu.load.5", avg.Load5),
			makeSample(now, "cpu", "cpu.load.15", avg.Load15),
		)
	}

	return samples, nil
}

func makeSample(ts int64, collector, name string, value float64) model.MetricSample {
	return model.MetricSample{
		Timestamp:  ts,
		Collector:  collector,
		MetricName: name,
		Value:      value,
	}
}
