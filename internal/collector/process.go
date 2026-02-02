package collector

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/playok/only1mon/internal/model"
	"github.com/shirou/gopsutil/v4/process"
)

type processCollector struct {
	prevTime int64
	prevIO   map[int32]procIOSnapshot // keyed by PID
	topN     int
}

type procIOSnapshot struct {
	readBytes  uint64
	writeBytes uint64
}

func NewProcessCollector() Collector { return &processCollector{topN: 10} }

// TopN returns the current top-N process count.
func (c *processCollector) TopN() int { return c.topN }

// SetTopN updates the top-N process count.
func (c *processCollector) SetTopN(n int) {
	if n < 1 {
		n = 1
	}
	if n > 50 {
		n = 50
	}
	c.topN = n
}

func (c *processCollector) ID() string          { return "process" }
func (c *processCollector) Name() string        { return "Process" }
func (c *processCollector) Description() string { return "Process count and top CPU/memory consumers" }
func (c *processCollector) Impact() model.ImpactLevel { return model.ImpactMedium }
func (c *processCollector) Warning() string {
	return "Overhead increases with 5000+ processes"
}

func (c *processCollector) MetricNames() []string {
	return []string{
		"proc.total_count",
		"proc.top_cpu.*.pid", "proc.top_cpu.*.name", "proc.top_cpu.*.cpu_pct", "proc.top_cpu.*.mem_pct",
		"proc.top_mem.*.pid", "proc.top_mem.*.name", "proc.top_mem.*.cpu_pct", "proc.top_mem.*.mem_pct",
		"proc.top_io.*.pid", "proc.top_io.*.name", "proc.top_io.*.read_bps", "proc.top_io.*.write_bps",
		"proc.io.total_read_bps", "proc.io.total_write_bps",
	}
}

type procInfo struct {
	pid      int32
	name     string
	cpuPct   float64
	memPct   float32
	readBps  float64 // bytes/sec read
	writeBps float64 // bytes/sec write
}

func (c *processCollector) Collect(ctx context.Context) ([]model.MetricSample, error) {
	now := time.Now().Unix()
	var samples []model.MetricSample

	procs, err := process.ProcessesWithContext(ctx)
	if err != nil {
		return nil, err
	}

	samples = append(samples, makeSample(now, "process", "proc.total_count", float64(len(procs))))

	elapsed := float64(now - c.prevTime)
	if c.prevTime == 0 || elapsed <= 0 {
		elapsed = 0
	}
	curIO := make(map[int32]procIOSnapshot, len(procs))

	var infos []procInfo
	for _, p := range procs {
		name, _ := p.NameWithContext(ctx)
		cpuPct, _ := p.CPUPercentWithContext(ctx)
		memPct, _ := p.MemoryPercentWithContext(ctx)

		info := procInfo{
			pid:    p.Pid,
			name:   name,
			cpuPct: cpuPct,
			memPct: memPct,
		}

		// Per-process I/O counters (platform-specific)
		if rb, wb, ok := readProcIO(p.Pid); ok {
			curIO[p.Pid] = procIOSnapshot{readBytes: rb, writeBytes: wb}
			if elapsed > 0 && c.prevIO != nil {
				if prev, ok := c.prevIO[p.Pid]; ok {
					if rb >= prev.readBytes {
						info.readBps = float64(rb-prev.readBytes) / elapsed
					}
					if wb >= prev.writeBytes {
						info.writeBps = float64(wb-prev.writeBytes) / elapsed
					}
				}
			}
		}

		infos = append(infos, info)
	}

	c.prevIO = curIO
	c.prevTime = now

	topN := c.topN
	if topN < 1 {
		topN = 10
	}

	// Top N by CPU
	sort.Slice(infos, func(i, j int) bool { return infos[i].cpuPct > infos[j].cpuPct })
	for i := 0; i < topN && i < len(infos); i++ {
		p := infos[i]
		samples = append(samples,
			makeSample(now, "process", fmt.Sprintf("proc.top_cpu.%d.pid", i), float64(p.pid)),
			model.MetricSample{Timestamp: now, Collector: "process", MetricName: fmt.Sprintf("proc.top_cpu.%d.name", i), Value: 0, Labels: p.name},
			makeSample(now, "process", fmt.Sprintf("proc.top_cpu.%d.cpu_pct", i), p.cpuPct),
			makeSample(now, "process", fmt.Sprintf("proc.top_cpu.%d.mem_pct", i), float64(p.memPct)),
		)
	}

	// Top N by Memory
	sort.Slice(infos, func(i, j int) bool { return infos[i].memPct > infos[j].memPct })
	for i := 0; i < topN && i < len(infos); i++ {
		p := infos[i]
		samples = append(samples,
			makeSample(now, "process", fmt.Sprintf("proc.top_mem.%d.pid", i), float64(p.pid)),
			model.MetricSample{Timestamp: now, Collector: "process", MetricName: fmt.Sprintf("proc.top_mem.%d.name", i), Value: 0, Labels: p.name},
			makeSample(now, "process", fmt.Sprintf("proc.top_mem.%d.cpu_pct", i), p.cpuPct),
			makeSample(now, "process", fmt.Sprintf("proc.top_mem.%d.mem_pct", i), float64(p.memPct)),
		)
	}

	// Top N by I/O (read+write rate)
	sort.Slice(infos, func(i, j int) bool {
		return (infos[i].readBps + infos[i].writeBps) > (infos[j].readBps + infos[j].writeBps)
	})
	var totalReadBps, totalWriteBps float64
	for _, info := range infos {
		totalReadBps += info.readBps
		totalWriteBps += info.writeBps
	}
	samples = append(samples,
		makeSample(now, "process", "proc.io.total_read_bps", totalReadBps),
		makeSample(now, "process", "proc.io.total_write_bps", totalWriteBps),
	)
	for i := 0; i < topN && i < len(infos); i++ {
		p := infos[i]
		samples = append(samples,
			makeSample(now, "process", fmt.Sprintf("proc.top_io.%d.pid", i), float64(p.pid)),
			model.MetricSample{Timestamp: now, Collector: "process", MetricName: fmt.Sprintf("proc.top_io.%d.name", i), Value: 0, Labels: p.name},
			makeSample(now, "process", fmt.Sprintf("proc.top_io.%d.read_bps", i), p.readBps),
			makeSample(now, "process", fmt.Sprintf("proc.top_io.%d.write_bps", i), p.writeBps),
		)
	}

	return samples, nil
}
