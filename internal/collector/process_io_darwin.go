package collector

import (
	"encoding/binary"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
)

const (
	rusageInfoV4    = 4   // RUSAGE_INFO_V4
	rusageV4Size    = 296 // sizeof(rusage_info_v4)
	diskReadOffset  = 144 // offset of ri_diskio_bytesread
	diskWriteOffset = 152 // offset of ri_diskio_byteswritten
)

var (
	procPidRusageFn   func(pid int32, flavor int32, buffer uintptr) int32
	procPidRusageOnce sync.Once
	procPidRusageOK   bool
)

func initProcPidRusage() {
	handle, err := purego.Dlopen("/usr/lib/libSystem.B.dylib", purego.RTLD_LAZY|purego.RTLD_GLOBAL)
	if err != nil {
		return
	}
	purego.RegisterLibFunc(&procPidRusageFn, handle, "proc_pid_rusage")
	procPidRusageOK = true
}

// readProcIO reads per-process disk I/O via proc_pid_rusage on macOS.
// Returns (readBytes, writeBytes, ok).
// Only works for processes owned by the same user (or with root privileges).
func readProcIO(pid int32) (uint64, uint64, bool) {
	procPidRusageOnce.Do(initProcPidRusage)
	if !procPidRusageOK {
		return 0, 0, false
	}

	buf := make([]byte, rusageV4Size)
	ret := procPidRusageFn(pid, rusageInfoV4, uintptr(unsafe.Pointer(&buf[0])))
	if ret != 0 {
		return 0, 0, false
	}

	readBytes := binary.LittleEndian.Uint64(buf[diskReadOffset : diskReadOffset+8])
	writeBytes := binary.LittleEndian.Uint64(buf[diskWriteOffset : diskWriteOffset+8])
	return readBytes, writeBytes, true
}
