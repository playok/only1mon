package collector

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/playok/only1mon/internal/model"
	"github.com/shirou/gopsutil/v4/disk"
)

type diskCollector struct{}

func NewDiskCollector() Collector { return &diskCollector{} }

func (c *diskCollector) ID() string          { return "disk" }
func (c *diskCollector) Name() string        { return "Disk" }
func (c *diskCollector) Description() string { return "Disk I/O stats and filesystem usage" }
func (c *diskCollector) Impact() model.ImpactLevel { return model.ImpactNone }
func (c *diskCollector) Warning() string     { return "" }

func (c *diskCollector) MetricNames() []string {
	return []string{
		"disk.*.read_bytes_sec", "disk.*.write_bytes_sec",
		"disk.*.read_iops", "disk.*.write_iops",
		"disk.*.total", "disk.*.used", "disk.*.free", "disk.*.used_pct",
	}
}

func (c *diskCollector) Collect(ctx context.Context) ([]model.MetricSample, error) {
	now := time.Now().Unix()
	var samples []model.MetricSample

	// IO counters per device
	counters, err := disk.IOCountersWithContext(ctx)
	if err == nil {
		for name, io := range counters {
			dev := sanitizeName(name)
			samples = append(samples,
				makeSample(now, "disk", fmt.Sprintf("disk.%s.read_bytes", dev), float64(io.ReadBytes)),
				makeSample(now, "disk", fmt.Sprintf("disk.%s.write_bytes", dev), float64(io.WriteBytes)),
				makeSample(now, "disk", fmt.Sprintf("disk.%s.read_count", dev), float64(io.ReadCount)),
				makeSample(now, "disk", fmt.Sprintf("disk.%s.write_count", dev), float64(io.WriteCount)),
				makeSample(now, "disk", fmt.Sprintf("disk.%s.io_time", dev), float64(io.IoTime)),
			)
		}
	}

	// Filesystem usage
	partitions, err := disk.PartitionsWithContext(ctx, false)
	if err == nil {
		for _, p := range partitions {
			usage, err := disk.UsageWithContext(ctx, p.Mountpoint)
			if err != nil {
				continue
			}
			mount := sanitizeName(p.Mountpoint)
			if mount == "" {
				mount = "root"
			}
			samples = append(samples,
				makeSample(now, "disk", fmt.Sprintf("disk.%s.total", mount), float64(usage.Total)),
				makeSample(now, "disk", fmt.Sprintf("disk.%s.used", mount), float64(usage.Used)),
				makeSample(now, "disk", fmt.Sprintf("disk.%s.free", mount), float64(usage.Free)),
				makeSample(now, "disk", fmt.Sprintf("disk.%s.used_pct", mount), usage.UsedPercent),
			)
		}
	}

	return samples, nil
}

func sanitizeName(s string) string {
	s = strings.ReplaceAll(s, "/", "_")
	s = strings.ReplaceAll(s, " ", "_")
	s = strings.TrimLeft(s, "_")
	if s == "" {
		return "root"
	}
	return s
}
