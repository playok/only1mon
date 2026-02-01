package collector

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// readProcIO reads per-process I/O counters from /proc/[pid]/io (Linux).
// Returns (readBytes, writeBytes, ok).
func readProcIO(pid int32) (uint64, uint64, bool) {
	f, err := os.Open(fmt.Sprintf("/proc/%d/io", pid))
	if err != nil {
		return 0, 0, false
	}
	defer f.Close()

	var readBytes, writeBytes uint64
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "read_bytes:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "read_bytes:"))
			readBytes, _ = strconv.ParseUint(v, 10, 64)
		} else if strings.HasPrefix(line, "write_bytes:") {
			v := strings.TrimSpace(strings.TrimPrefix(line, "write_bytes:"))
			writeBytes, _ = strconv.ParseUint(v, 10, 64)
		}
	}
	return readBytes, writeBytes, true
}
