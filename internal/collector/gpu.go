package collector

import (
	"bufio"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/playok/only1mon/internal/model"
)

type gpuCollector struct{}

func NewGPUCollector() Collector { return &gpuCollector{} }

func (c *gpuCollector) ID() string          { return "gpu" }
func (c *gpuCollector) Name() string        { return "GPU (NVIDIA)" }
func (c *gpuCollector) Description() string { return "NVIDIA GPU utilization, memory, temperature, power via nvidia-smi" }
func (c *gpuCollector) Impact() model.ImpactLevel { return model.ImpactMedium }
func (c *gpuCollector) Warning() string {
	return "Runs nvidia-smi subprocess; requires NVIDIA drivers"
}

func (c *gpuCollector) MetricNames() []string {
	return []string{
		"gpu.*.util_pct", "gpu.*.mem_util_pct", "gpu.*.temp_c",
		"gpu.*.mem_used", "gpu.*.mem_total", "gpu.*.power_watts",
	}
}

func (c *gpuCollector) Collect(ctx context.Context) ([]model.MetricSample, error) {
	now := time.Now().Unix()

	// Check if nvidia-smi exists
	path, err := exec.LookPath("nvidia-smi")
	if err != nil {
		return nil, nil // no NVIDIA GPU, skip silently
	}

	cmd := exec.CommandContext(ctx, path,
		"--query-gpu=index,utilization.gpu,utilization.memory,temperature.gpu,memory.used,memory.total,power.draw",
		"--format=csv,noheader,nounits")

	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("nvidia-smi: %w", err)
	}

	var samples []model.MetricSample
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ", ")
		if len(fields) < 7 {
			continue
		}
		idx := strings.TrimSpace(fields[0])
		utilGPU, _ := strconv.ParseFloat(strings.TrimSpace(fields[1]), 64)
		utilMem, _ := strconv.ParseFloat(strings.TrimSpace(fields[2]), 64)
		temp, _ := strconv.ParseFloat(strings.TrimSpace(fields[3]), 64)
		memUsed, _ := strconv.ParseFloat(strings.TrimSpace(fields[4]), 64)
		memTotal, _ := strconv.ParseFloat(strings.TrimSpace(fields[5]), 64)
		power, _ := strconv.ParseFloat(strings.TrimSpace(fields[6]), 64)

		samples = append(samples,
			makeSample(now, "gpu", fmt.Sprintf("gpu.%s.util_pct", idx), utilGPU),
			makeSample(now, "gpu", fmt.Sprintf("gpu.%s.mem_util_pct", idx), utilMem),
			makeSample(now, "gpu", fmt.Sprintf("gpu.%s.temp_c", idx), temp),
			makeSample(now, "gpu", fmt.Sprintf("gpu.%s.mem_used", idx), memUsed*1024*1024), // MiB to bytes
			makeSample(now, "gpu", fmt.Sprintf("gpu.%s.mem_total", idx), memTotal*1024*1024),
			makeSample(now, "gpu", fmt.Sprintf("gpu.%s.power_watts", idx), power),
		)
	}

	return samples, nil
}
