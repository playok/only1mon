package collector

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/only1mon/only1mon/internal/model"
	"github.com/shirou/gopsutil/v4/net"
)

type networkCollector struct {
	prevTime     int64
	prevCounters map[string]net.IOCountersStat // keyed by interface name
}

func NewNetworkCollector() Collector { return &networkCollector{} }

func (c *networkCollector) ID() string          { return "network" }
func (c *networkCollector) Name() string        { return "Network" }
func (c *networkCollector) Description() string { return "Network interface stats, TCP connection states" }
func (c *networkCollector) Impact() model.ImpactLevel { return model.ImpactLow }
func (c *networkCollector) Warning() string     { return "May have slight overhead with many connections" }

func (c *networkCollector) MetricNames() []string {
	return []string{
		"net.total.bytes_sent", "net.total.bytes_recv",
		"net.total.packets_sent", "net.total.packets_recv",
		"net.total.errin", "net.total.errout", "net.total.dropin", "net.total.dropout",
		"net.total.bytes_sent_sec", "net.total.bytes_recv_sec",
		"net.total.packets_sent_sec", "net.total.packets_recv_sec",
		"net.*.bytes_sent", "net.*.bytes_recv",
		"net.*.packets_sent", "net.*.packets_recv",
		"net.*.errin", "net.*.errout", "net.*.dropin", "net.*.dropout",
		"net.*.bytes_sent_sec", "net.*.bytes_recv_sec",
		"net.*.packets_sent_sec", "net.*.packets_recv_sec",
		"net.tcp.established", "net.tcp.time_wait", "net.tcp.close_wait",
		"net.tcp.retransmits",
		"net.tcp.tx_queue_total", "net.tcp.rx_queue_total",
		"net.tcp.tx_queue_max", "net.tcp.rx_queue_max",
	}
}

func (c *networkCollector) Collect(ctx context.Context) ([]model.MetricSample, error) {
	now := time.Now().Unix()
	var samples []model.MetricSample

	// Per-interface counters + total aggregation
	counters, err := net.IOCountersWithContext(ctx, true)
	if err == nil {
		curMap := make(map[string]net.IOCountersStat, len(counters))

		var totalBytesSent, totalBytesRecv uint64
		var totalPktsSent, totalPktsRecv uint64
		var totalErrin, totalErrout uint64
		var totalDropin, totalDropout uint64

		for _, io := range counters {
			iface := io.Name
			curMap[iface] = io

			// Cumulative counters
			samples = append(samples,
				makeSample(now, "network", fmt.Sprintf("net.%s.bytes_sent", iface), float64(io.BytesSent)),
				makeSample(now, "network", fmt.Sprintf("net.%s.bytes_recv", iface), float64(io.BytesRecv)),
				makeSample(now, "network", fmt.Sprintf("net.%s.packets_sent", iface), float64(io.PacketsSent)),
				makeSample(now, "network", fmt.Sprintf("net.%s.packets_recv", iface), float64(io.PacketsRecv)),
				makeSample(now, "network", fmt.Sprintf("net.%s.errin", iface), float64(io.Errin)),
				makeSample(now, "network", fmt.Sprintf("net.%s.errout", iface), float64(io.Errout)),
				makeSample(now, "network", fmt.Sprintf("net.%s.dropin", iface), float64(io.Dropin)),
				makeSample(now, "network", fmt.Sprintf("net.%s.dropout", iface), float64(io.Dropout)),
			)

			// Per-interface rate (delta / elapsed seconds)
			if c.prevCounters != nil && c.prevTime > 0 {
				elapsed := float64(now - c.prevTime)
				if elapsed > 0 {
					if prev, ok := c.prevCounters[iface]; ok {
						samples = append(samples,
							makeSample(now, "network", fmt.Sprintf("net.%s.bytes_sent_sec", iface), float64(io.BytesSent-prev.BytesSent)/elapsed),
							makeSample(now, "network", fmt.Sprintf("net.%s.bytes_recv_sec", iface), float64(io.BytesRecv-prev.BytesRecv)/elapsed),
							makeSample(now, "network", fmt.Sprintf("net.%s.packets_sent_sec", iface), float64(io.PacketsSent-prev.PacketsSent)/elapsed),
							makeSample(now, "network", fmt.Sprintf("net.%s.packets_recv_sec", iface), float64(io.PacketsRecv-prev.PacketsRecv)/elapsed),
						)
					}
				}
			}

			totalBytesSent += io.BytesSent
			totalBytesRecv += io.BytesRecv
			totalPktsSent += io.PacketsSent
			totalPktsRecv += io.PacketsRecv
			totalErrin += io.Errin
			totalErrout += io.Errout
			totalDropin += io.Dropin
			totalDropout += io.Dropout
		}

		// System-wide totals (cumulative)
		samples = append(samples,
			makeSample(now, "network", "net.total.bytes_sent", float64(totalBytesSent)),
			makeSample(now, "network", "net.total.bytes_recv", float64(totalBytesRecv)),
			makeSample(now, "network", "net.total.packets_sent", float64(totalPktsSent)),
			makeSample(now, "network", "net.total.packets_recv", float64(totalPktsRecv)),
			makeSample(now, "network", "net.total.errin", float64(totalErrin)),
			makeSample(now, "network", "net.total.errout", float64(totalErrout)),
			makeSample(now, "network", "net.total.dropin", float64(totalDropin)),
			makeSample(now, "network", "net.total.dropout", float64(totalDropout)),
		)

		// System-wide rate totals
		if c.prevCounters != nil && c.prevTime > 0 {
			elapsed := float64(now - c.prevTime)
			if elapsed > 0 {
				// Sum previous totals
				var prevBytesSent, prevBytesRecv uint64
				var prevPktsSent, prevPktsRecv uint64
				for _, prev := range c.prevCounters {
					prevBytesSent += prev.BytesSent
					prevBytesRecv += prev.BytesRecv
					prevPktsSent += prev.PacketsSent
					prevPktsRecv += prev.PacketsRecv
				}
				samples = append(samples,
					makeSample(now, "network", "net.total.bytes_sent_sec", float64(totalBytesSent-prevBytesSent)/elapsed),
					makeSample(now, "network", "net.total.bytes_recv_sec", float64(totalBytesRecv-prevBytesRecv)/elapsed),
					makeSample(now, "network", "net.total.packets_sent_sec", float64(totalPktsSent-prevPktsSent)/elapsed),
					makeSample(now, "network", "net.total.packets_recv_sec", float64(totalPktsRecv-prevPktsRecv)/elapsed),
				)
			}
		}

		c.prevCounters = curMap
		c.prevTime = now
	}

	// TCP connection states
	conns, err := net.ConnectionsWithContext(ctx, "tcp")
	if err == nil {
		states := map[string]int{
			"ESTABLISHED": 0,
			"TIME_WAIT":   0,
			"CLOSE_WAIT":  0,
		}
		for _, conn := range conns {
			if _, ok := states[conn.Status]; ok {
				states[conn.Status]++
			}
		}
		samples = append(samples,
			makeSample(now, "network", "net.tcp.established", float64(states["ESTABLISHED"])),
			makeSample(now, "network", "net.tcp.time_wait", float64(states["TIME_WAIT"])),
			makeSample(now, "network", "net.tcp.close_wait", float64(states["CLOSE_WAIT"])),
		)
	}

	// TCP retransmits (from /proc/net/snmp on Linux)
	netProto, err := net.ProtoCountersWithContext(ctx, []string{"tcp"})
	if err == nil {
		for _, proto := range netProto {
			if proto.Protocol == "tcp" {
				if v, ok := proto.Stats["RetransSegs"]; ok {
					samples = append(samples,
						makeSample(now, "network", "net.tcp.retransmits", float64(v)),
					)
				}
			}
		}
	}

	// TCP socket queue depths (Linux only, from /proc/net/tcp + /proc/net/tcp6)
	if runtime.GOOS == "linux" {
		txTotal, rxTotal, txMax, rxMax := parseTCPQueueDepths()
		samples = append(samples,
			makeSample(now, "network", "net.tcp.tx_queue_total", float64(txTotal)),
			makeSample(now, "network", "net.tcp.rx_queue_total", float64(rxTotal)),
			makeSample(now, "network", "net.tcp.tx_queue_max", float64(txMax)),
			makeSample(now, "network", "net.tcp.rx_queue_max", float64(rxMax)),
		)
	}

	return samples, nil
}

// parseTCPQueueDepths reads /proc/net/tcp and /proc/net/tcp6 to aggregate
// socket send/receive queue sizes (tx_queue, rx_queue columns).
// Format of each line: sl local_address remote_address st tx_queue:rx_queue ...
// tx_queue and rx_queue are hex-encoded byte counts.
func parseTCPQueueDepths() (txTotal, rxTotal, txMax, rxMax uint64) {
	for _, path := range []string{"/proc/net/tcp", "/proc/net/tcp6"} {
		f, err := os.Open(path)
		if err != nil {
			continue
		}
		scanner := bufio.NewScanner(f)
		scanner.Scan() // skip header line
		for scanner.Scan() {
			fields := strings.Fields(scanner.Text())
			if len(fields) < 5 {
				continue
			}
			// fields[4] is "tx_queue:rx_queue" in hex
			parts := strings.SplitN(fields[4], ":", 2)
			if len(parts) != 2 {
				continue
			}
			tx, err1 := strconv.ParseUint(parts[0], 16, 64)
			rx, err2 := strconv.ParseUint(parts[1], 16, 64)
			if err1 != nil || err2 != nil {
				continue
			}
			txTotal += tx
			rxTotal += rx
			if tx > txMax {
				txMax = tx
			}
			if rx > rxMax {
				rxMax = rx
			}
		}
		f.Close()
	}
	return
}
