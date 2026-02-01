package collector

import "strings"

// MetricDesc holds a human-readable description and unit for a metric.
type MetricDesc struct {
	Description   string `json:"description"`
	DescriptionKO string `json:"description_ko"`
	Unit          string `json:"unit"`
}

// metricDescriptions maps metric name patterns to descriptions.
// Use "*" as a wildcard segment (e.g. "cpu.core.*.usage").
var metricDescriptions = map[string]MetricDesc{
	// ========================== CPU ==========================
	"cpu.total.usage": {
		"Overall CPU busy percentage (100% minus idle). This is the single most important CPU metric — it shows how much of the total CPU capacity is being used across all cores. Values above 80% sustained may indicate CPU saturation and potential application slowdowns.",
		"전체 CPU 사용률 (100% - 유휴). 모든 코어의 총 CPU 용량 중 얼마나 사용되고 있는지 보여주는 가장 중요한 CPU 지표입니다. 80% 이상이 지속되면 CPU 포화 상태로 애플리케이션 성능 저하가 발생할 수 있습니다.",
		"%",
	},
	"cpu.total.user": {
		"CPU time spent executing user-space application code (web servers, databases, your programs). High user% with low system% is normal for compute-intensive workloads. If this is consistently high, the application may need optimization or horizontal scaling.",
		"사용자 공간 애플리케이션 코드(웹 서버, DB, 사용자 프로그램)를 실행하는 데 소비된 CPU 시간. user%가 높고 system%가 낮은 것은 연산 집약적 워크로드에서 정상입니다. 지속적으로 높으면 애플리케이션 최적화나 수평 확장이 필요할 수 있습니다.",
		"%",
	},
	"cpu.total.system": {
		"CPU time spent in kernel/system calls (file I/O, networking, memory management, process scheduling). High system% often indicates heavy I/O operations, frequent system calls, or kernel overhead. Values above 30% warrant investigation — common causes include excessive context switching, disk I/O, or network processing.",
		"커널/시스템 콜(파일 I/O, 네트워킹, 메모리 관리, 프로세스 스케줄링)에 소비된 CPU 시간. system%가 높으면 대량의 I/O 작업, 빈번한 시스템 콜, 커널 오버헤드를 의미합니다. 30% 이상이면 조사가 필요하며, 주요 원인은 과도한 컨텍스트 스위칭, 디스크 I/O, 네트워크 처리입니다.",
		"%",
	},
	"cpu.total.idle": {
		"CPU time spent doing nothing — waiting for work. This is the inverse of usage (idle + usage ≈ 100%). High idle means the CPU has spare capacity. Near 0% idle under load indicates the CPU is fully saturated and tasks are queuing up waiting for CPU time.",
		"아무 작업도 하지 않고 대기하는 CPU 시간. 사용률의 반대 (idle + usage ≈ 100%). idle이 높으면 여유 용량이 있다는 뜻입니다. 부하 상태에서 idle이 0%에 가까우면 CPU가 완전 포화 상태로 태스크가 CPU 시간을 기다리며 대기 중입니다.",
		"%",
	},
	"cpu.total.iowait": {
		"CPU time spent idle while waiting for disk or network I/O to complete. High iowait indicates the CPU is being held up by slow storage or network devices. This is a key indicator of I/O bottlenecks — the CPU wants to work but is stuck waiting for data. Common causes: slow disks, heavy disk reads/writes, NFS timeouts, network storage latency.",
		"디스크나 네트워크 I/O 완료를 기다리며 유휴 상태인 CPU 시간. iowait가 높으면 느린 스토리지나 네트워크 장치가 CPU를 지연시키고 있다는 의미입니다. I/O 병목의 핵심 지표로, CPU는 작업하고 싶지만 데이터를 기다리며 멈춰 있습니다. 주요 원인: 느린 디스크, 대량 읽기/쓰기, NFS 타임아웃, 네트워크 스토리지 지연.",
		"%",
	},
	"cpu.total.steal": {
		"CPU time 'stolen' by the hypervisor to serve other virtual machines on the same physical host. Only relevant in virtualized environments (AWS EC2, GCP, Azure VMs, etc.). High steal% means your VM is competing for CPU with noisy neighbors. Consider upgrading to a dedicated instance or a larger VM type.",
		"같은 물리 호스트의 다른 가상 머신에 할당하기 위해 하이퍼바이저가 '빼앗은' CPU 시간. 가상화 환경(AWS EC2, GCP, Azure VM 등)에서만 의미 있습니다. steal%가 높으면 다른 VM과 CPU를 경쟁 중입니다. 전용 인스턴스나 더 큰 VM 타입으로 업그레이드를 고려하세요.",
		"%",
	},
	"cpu.core.*.usage": {
		"CPU usage of an individual core. Useful for detecting unbalanced workloads where one core is maxed out while others are idle. Single-threaded applications often pin one core at 100% while others remain low. Helps identify if multi-threading or CPU affinity tuning is needed.",
		"개별 코어의 CPU 사용률. 하나의 코어만 최대치이고 나머지는 유휴인 불균형 워크로드를 감지하는 데 유용합니다. 싱글스레드 애플리케이션은 한 코어만 100%로 고정하고 나머지는 낮게 유지합니다. 멀티스레딩이나 CPU 친화도 튜닝이 필요한지 판단하는 데 도움됩니다.",
		"%",
	},
	"cpu.core.*.user":   {"User-space CPU time on this specific core. See cpu.total.user for interpretation.", "이 코어의 사용자 공간 CPU 시간. 해석은 cpu.total.user를 참고하세요.", "%"},
	"cpu.core.*.system": {"Kernel/system CPU time on this specific core. See cpu.total.system for interpretation.", "이 코어의 커널/시스템 CPU 시간. 해석은 cpu.total.system을 참고하세요.", "%"},
	"cpu.core.*.idle":   {"Idle time on this specific core. See cpu.total.idle for interpretation.", "이 코어의 유휴 시간. 해석은 cpu.total.idle을 참고하세요.", "%"},
	"cpu.core.*.iowait": {"I/O wait time on this specific core. See cpu.total.iowait for interpretation.", "이 코어의 I/O 대기 시간. 해석은 cpu.total.iowait를 참고하세요.", "%"},
	"cpu.load.1": {
		"System load average over the last 1 minute. Represents the average number of processes waiting for CPU or I/O. On a 4-core system, load of 4.0 means all cores are fully utilized. Load above core count means processes are queuing. A sudden spike indicates a burst of activity. Compare with load.5 and load.15 to see if load is increasing or decreasing.",
		"최근 1분간 시스템 부하 평균. CPU 또는 I/O를 기다리는 평균 프로세스 수를 나타냅니다. 4코어 시스템에서 load 4.0은 모든 코어가 완전히 활용되고 있다는 뜻입니다. 코어 수를 초과하면 프로세스가 대기열에 쌓이고 있습니다. 급증하면 활동이 폭증한 것이며, load.5/load.15와 비교하여 부하가 증가/감소 추세인지 파악하세요.",
		"",
	},
	"cpu.load.5": {
		"System load average over the last 5 minutes. A more stable indicator than load.1. If load.1 is high but load.5 is low, the spike is recent and may be temporary. If both are high, sustained load is occurring.",
		"최근 5분간 시스템 부하 평균. load.1보다 안정적인 지표입니다. load.1은 높지만 load.5가 낮으면 최근 발생한 일시적 급증입니다. 둘 다 높으면 지속적인 부하가 발생 중입니다.",
		"",
	},
	"cpu.load.15": {
		"System load average over the last 15 minutes. Shows the long-term trend. If load.1 > load.15, load is increasing (potential problem developing). If load.1 < load.15, load is decreasing (situation improving).",
		"최근 15분간 시스템 부하 평균. 장기 추세를 보여줍니다. load.1 > load.15이면 부하 증가 추세(잠재적 문제 발생 중). load.1 < load.15이면 부하 감소 추세(상황 개선 중).",
		"",
	},
	"cpu.context_switches": {
		"Cumulative count of context switches since boot. A context switch occurs when the CPU switches from one process/thread to another. High rates (tens of thousands per second) are normal on busy servers. Extremely high rates may indicate too many threads, lock contention, or CPU thrashing between processes. Monitor the rate of change rather than absolute value.",
		"부팅 이후 누적 컨텍스트 스위치 횟수. 컨텍스트 스위치는 CPU가 한 프로세스/스레드에서 다른 것으로 전환할 때 발생합니다. 바쁜 서버에서 초당 수만 건은 정상입니다. 극도로 높은 비율은 과도한 스레드, 락 경합, CPU 스래싱을 나타낼 수 있습니다. 절대값보다 변화율을 모니터링하세요.",
		"count",
	},
	"cpu.interrupts": {
		"Cumulative count of hardware interrupts since boot. Hardware interrupts are signals from devices (network cards, disks, timers) requesting CPU attention. High rates are normal with high network traffic or disk I/O. Sudden spikes may indicate hardware issues or driver problems.",
		"부팅 이후 누적 하드웨어 인터럽트 횟수. 하드웨어 인터럽트는 장치(네트워크 카드, 디스크, 타이머)가 CPU의 처리를 요청하는 신호입니다. 네트워크 트래픽이나 디스크 I/O가 많으면 높은 비율은 정상입니다. 급증하면 하드웨어 문제나 드라이버 문제를 나타낼 수 있습니다.",
		"count",
	},

	// ========================== Memory ==========================
	"mem.total": {
		"Total physical RAM installed in the system. This is a fixed hardware value that doesn't change during runtime. Used as the denominator for memory usage percentage calculations.",
		"시스템에 설치된 총 물리 RAM 용량. 런타임 중 변하지 않는 고정 하드웨어 값입니다. 메모리 사용률 계산의 분모로 사용됩니다.",
		"bytes",
	},
	"mem.used": {
		"Memory actively in use by processes and the OS kernel (excludes cache and buffers). This is what is truly 'consumed' and cannot be reclaimed without terminating processes. Continuously increasing used memory may indicate a memory leak.",
		"프로세스와 OS 커널이 적극적으로 사용 중인 메모리 (캐시와 버퍼 제외). 실제로 '소비'된 양으로, 프로세스를 종료하지 않으면 회수할 수 없습니다. 지속적으로 증가하면 메모리 누수를 의심하세요.",
		"bytes",
	},
	"mem.free": {
		"Memory that is completely unused — not in use by processes, cache, or buffers. On a healthy Linux system, free memory is often low because the OS uses spare memory for disk cache (which is good). Low free memory alone is NOT a problem — check mem.available instead.",
		"프로세스, 캐시, 버퍼 어디에도 사용되지 않는 완전히 빈 메모리. 정상적인 Linux 시스템에서는 OS가 여유 메모리를 디스크 캐시로 활용하므로 free가 낮은 것이 정상입니다(이는 좋은 것). free가 낮은 것 자체는 문제가 아닙니다 — mem.available을 확인하세요.",
		"bytes",
	},
	"mem.available": {
		"Memory available for new processes without swapping, including reclaimable cache and buffers. This is the most accurate measure of 'how much memory can I still use'. If available is low (below 10-15% of total), the system may start swapping soon, causing severe performance degradation.",
		"스와핑 없이 새 프로세스가 사용할 수 있는 메모리. 회수 가능한 캐시와 버퍼를 포함합니다. '메모리를 얼마나 더 쓸 수 있는가'의 가장 정확한 지표입니다. available이 total의 10-15% 이하로 떨어지면 곧 스와핑이 시작되어 심각한 성능 저하가 발생할 수 있습니다.",
		"bytes",
	},
	"mem.cached": {
		"Memory used by the OS to cache recently read files from disk (page cache). This speeds up repeated file access significantly. Cached memory is automatically released when applications need more memory, so high cache is beneficial, not a problem. If cache is very low despite available disk I/O, the system may be memory-constrained.",
		"OS가 최근 읽은 파일을 디스크에서 캐시하는 데 사용하는 메모리 (페이지 캐시). 반복적인 파일 접근 속도를 크게 향상시킵니다. 캐시 메모리는 애플리케이션이 더 많은 메모리를 필요로 하면 자동으로 해제되므로, 캐시가 높은 것은 문제가 아니라 유익합니다. 디스크 I/O가 있는데도 캐시가 매우 낮으면 메모리 부족 상태일 수 있습니다.",
		"bytes",
	},
	"mem.buffers": {
		"Memory used for kernel I/O buffers (metadata, directory entries, pending writes). Typically a small fraction of total memory. The kernel manages this automatically. Very low buffers under heavy disk I/O may indicate memory pressure.",
		"커널 I/O 버퍼(메타데이터, 디렉토리 엔트리, 대기 중 쓰기)에 사용되는 메모리. 보통 전체 메모리의 작은 비율입니다. 커널이 자동으로 관리합니다. 디스크 I/O가 많은데 버퍼가 매우 낮으면 메모리 압박 상태일 수 있습니다.",
		"bytes",
	},
	"mem.used_pct": {
		"Percentage of total physical memory currently in use. A quick overview metric. Below 70% is comfortable, 70-85% is moderate, above 85% is high and approaching capacity. Sustained high usage with decreasing available memory is a warning sign.",
		"총 물리 메모리 중 현재 사용 중인 비율. 빠른 개요를 위한 지표입니다. 70% 이하는 여유, 70-85%는 보통, 85% 이상은 높으며 용량 한계에 근접합니다. 높은 사용률이 지속되고 available이 줄어들면 경고 신호입니다.",
		"%",
	},
	"mem.swap.total": {
		"Total swap space configured on the system. Swap is disk space used as 'overflow' memory when physical RAM is full. Having swap is a safety net, but relying on it degrades performance significantly since disk is orders of magnitude slower than RAM.",
		"시스템에 설정된 총 스왑 공간. 스왑은 물리 RAM이 가득 찼을 때 '오버플로' 메모리로 사용되는 디스크 공간입니다. 스왑은 안전장치이지만, 디스크가 RAM보다 수 배 느리므로 스왑에 의존하면 성능이 크게 저하됩니다.",
		"bytes",
	},
	"mem.swap.used": {
		"Swap space currently in use. Any swap usage means physical RAM was insufficient at some point. Small amounts may be fine (inactive pages swapped out), but actively growing swap usage with high I/O wait indicates the system is memory-starved and performance is degraded. Consider adding more RAM or reducing workload.",
		"현재 사용 중인 스왑 공간. 스왑 사용은 어느 시점에 물리 RAM이 부족했음을 의미합니다. 소량은 괜찮을 수 있지만(비활성 페이지 스왑 아웃), 스왑 사용이 증가하면서 I/O wait가 높으면 메모리 부족으로 성능이 저하된 상태입니다. RAM 추가나 워크로드 감소를 고려하세요.",
		"bytes",
	},
	"mem.swap.free": {
		"Available swap space remaining. If this reaches zero while the system still needs more memory, the OOM (Out Of Memory) killer will start terminating processes to free memory.",
		"남은 사용 가능한 스왑 공간. 시스템이 더 많은 메모리를 필요로 하는데 이 값이 0에 도달하면, OOM(Out Of Memory) 킬러가 메모리 확보를 위해 프로세스를 강제 종료하기 시작합니다.",
		"bytes",
	},
	"mem.page_faults.major": {
		"Major page faults require reading data from disk (swap or memory-mapped files). Each major fault causes significant latency (milliseconds). A sudden increase indicates the system is swapping heavily or doing excessive memory-mapped I/O. Sustained high major faults are a critical performance issue.",
		"디스크(스왑 또는 메모리 매핑 파일)에서 데이터를 읽어야 하는 메이저 페이지 폴트. 각 메이저 폴트는 밀리초 단위의 큰 지연을 유발합니다. 급증하면 시스템이 심하게 스와핑하거나 과도한 메모리 매핑 I/O를 수행 중입니다. 지속적인 메이저 폴트 증가는 심각한 성능 문제입니다.",
		"count",
	},
	"mem.page_faults.minor": {
		"Minor page faults are resolved entirely in memory without disk I/O (e.g., copy-on-write, shared library mapping). These are fast (microseconds) and normal. High rates are expected when starting new processes or allocating large amounts of memory. Not a concern by themselves.",
		"디스크 I/O 없이 메모리 내에서 해결되는 마이너 페이지 폴트 (예: copy-on-write, 공유 라이브러리 매핑). 마이크로초 단위로 빠르며 정상적입니다. 새 프로세스를 시작하거나 대량 메모리를 할당할 때 높은 비율이 예상됩니다. 그 자체로는 문제가 아닙니다.",
		"count",
	},
	"mem.slab": {
		"Memory used by the kernel's slab allocator for internal data structures (inodes, dentries, network buffers, etc.). Normally a small percentage of total memory. Unusually high slab usage can indicate a kernel memory leak, excessive file system metadata caching, or a large number of open files/connections.",
		"커널 슬랩 할당자가 내부 자료구조(inode, dentry, 네트워크 버퍼 등)에 사용하는 메모리. 보통 전체 메모리의 작은 비율입니다. 비정상적으로 높은 슬랩 사용은 커널 메모리 누수, 과도한 파일시스템 메타데이터 캐싱, 대량의 열린 파일/연결을 나타낼 수 있습니다.",
		"bytes",
	},
	"mem.hugepages.total": {
		"Number of huge pages (typically 2MB each) pre-allocated by the system. Huge pages reduce TLB (Translation Lookaside Buffer) misses for applications with large memory footprints like databases. These pages are reserved and cannot be used for regular allocations even if unused.",
		"시스템이 사전 할당한 대형 페이지(보통 각 2MB) 수. 데이터베이스처럼 대량 메모리를 사용하는 애플리케이션의 TLB(변환 참조 버퍼) 미스를 줄입니다. 이 페이지는 예약되어 있어 사용하지 않아도 일반 할당에 쓸 수 없습니다.",
		"count",
	},
	"mem.hugepages.free": {
		"Number of huge pages currently not in use. If this is always equal to hugepages.total, huge pages are allocated but no application is using them — wasted memory. If always zero, applications may need more huge pages.",
		"현재 사용되지 않는 대형 페이지 수. 항상 hugepages.total과 같으면 대형 페이지가 할당되었지만 어떤 애플리케이션도 사용하지 않는 메모리 낭비입니다. 항상 0이면 애플리케이션에 더 많은 대형 페이지가 필요할 수 있습니다.",
		"count",
	},

	// ========================== Disk ==========================
	"disk.*.total": {
		"Total capacity of the disk partition. This is the full size of the filesystem as configured. Useful as a reference to calculate usage percentage and plan capacity.",
		"디스크 파티션의 총 용량. 설정된 파일시스템의 전체 크기입니다. 사용률 계산과 용량 계획의 기준으로 유용합니다.",
		"bytes",
	},
	"disk.*.used": {
		"Disk space consumed by files on this partition. Steadily increasing usage without corresponding file cleanup may indicate log files growing unchecked, core dumps accumulating, or data not being rotated.",
		"이 파티션에서 파일이 사용 중인 디스크 공간. 파일 정리 없이 지속적으로 증가하면 로그 파일 무한 증가, 코어 덤프 누적, 데이터 미정리를 나타낼 수 있습니다.",
		"bytes",
	},
	"disk.*.free": {
		"Available disk space on the partition. When this approaches zero, applications may fail to write files, databases may crash, and logs may stop recording. Aim to keep at least 10-15% free for safe operation.",
		"파티션의 여유 디스크 공간. 0에 가까워지면 애플리케이션 파일 쓰기 실패, DB 크래시, 로그 기록 중단이 발생할 수 있습니다. 안전한 운영을 위해 최소 10-15% 여유를 유지하세요.",
		"bytes",
	},
	"disk.*.used_pct": {
		"Disk space usage as a percentage. The primary disk capacity alert metric. Below 70% is comfortable. 70-85% needs attention. Above 90% is critical — plan immediate cleanup or expansion. At 100% the system may become unstable.",
		"디스크 공간 사용률. 디스크 용량 알림의 핵심 지표입니다. 70% 이하는 여유, 70-85%는 주의 필요, 90% 이상은 위험 — 즉시 정리하거나 확장을 계획하세요. 100%에서 시스템이 불안정해질 수 있습니다.",
		"%",
	},
	"disk.*.read_bytes": {
		"Cumulative total bytes read from this disk device since boot. This is a monotonically increasing counter. Track the rate of change to understand read throughput patterns over time.",
		"부팅 이후 이 디스크 장치에서 읽은 누적 총 바이트. 단조 증가 카운터입니다. 시간에 따른 읽기 처리량 패턴을 이해하려면 변화율을 추적하세요.",
		"bytes",
	},
	"disk.*.write_bytes": {
		"Cumulative total bytes written to this disk device since boot. This is a monotonically increasing counter. Track the rate of change to understand write throughput patterns. Unexpected write bursts may indicate runaway logging or backup operations.",
		"부팅 이후 이 디스크 장치에 쓴 누적 총 바이트. 단조 증가 카운터입니다. 변화율을 추적하여 쓰기 처리량 패턴을 파악하세요. 예상치 못한 쓰기 폭증은 비정상 로깅이나 백업 작업을 나타낼 수 있습니다.",
		"bytes",
	},
	"disk.*.read_count":  {"Cumulative number of read operations since boot. High read IOPS with low throughput indicates many small random reads (typical of database workloads).", "부팅 이후 읽기 작업 누적 횟수. 낮은 처리량에 높은 읽기 IOPS는 많은 작은 랜덤 읽기를 나타냅니다(DB 워크로드에 전형적).", "count"},
	"disk.*.write_count": {"Cumulative number of write operations since boot. High write IOPS with low throughput indicates many small random writes (typical of logging or database commits).", "부팅 이후 쓰기 작업 누적 횟수. 낮은 처리량에 높은 쓰기 IOPS는 많은 작은 랜덤 쓰기를 나타냅니다(로깅이나 DB 커밋에 전형적).", "count"},
	"disk.*.io_time":     {"Cumulative time the disk has been busy processing I/O requests (in milliseconds). Used to calculate I/O utilization percentage. When io_time grows at the same rate as wall clock time, the disk is 100% busy.", "디스크가 I/O 요청을 처리하느라 바빴던 누적 시간(밀리초). I/O 활용률 계산에 사용됩니다. io_time이 실시간과 같은 속도로 증가하면 디스크가 100% 사용 중입니다.", "ms"},
	"disk.*.read_bytes_sec": {
		"Current disk read throughput (bytes per second), calculated from the delta between two collection intervals. Shows how fast data is being read from disk right now. Compare with the disk's rated sequential read speed to assess saturation.",
		"현재 디스크 읽기 처리량(초당 바이트), 두 수집 간격의 차이로 계산. 지금 디스크에서 데이터를 얼마나 빨리 읽고 있는지 보여줍니다. 디스크의 공칭 순차 읽기 속도와 비교하여 포화도를 평가하세요.",
		"bytes/s",
	},
	"disk.*.write_bytes_sec": {
		"Current disk write throughput (bytes per second), calculated from the delta between two collection intervals. Shows how fast data is being written to disk right now. Sustained high write throughput may indicate heavy logging, backup jobs, or database checkpoint operations.",
		"현재 디스크 쓰기 처리량(초당 바이트), 두 수집 간격의 차이로 계산. 지금 디스크에 데이터를 얼마나 빨리 쓰고 있는지 보여줍니다. 지속적인 높은 쓰기 처리량은 대량 로깅, 백업 작업, DB 체크포인트 작업을 나타낼 수 있습니다.",
		"bytes/s",
	},
	"disk.*.read_iops": {
		"Current read operations per second (IOPS). SSDs typically handle 10,000-100,000+ IOPS, while HDDs handle 100-200 IOPS. If IOPS approaches the disk's limit, I/O latency will increase sharply.",
		"현재 초당 읽기 작업 수(IOPS). SSD는 보통 10,000-100,000+ IOPS, HDD는 100-200 IOPS를 처리합니다. IOPS가 디스크 한계에 근접하면 I/O 지연이 급격히 증가합니다.",
		"ops/s",
	},
	"disk.*.write_iops": {
		"Current write operations per second (IOPS). Write IOPS is often the bottleneck for databases and logging-heavy applications. If write IOPS is maxed out, consider using faster storage, reducing write frequency, or batching writes.",
		"현재 초당 쓰기 작업 수(IOPS). 쓰기 IOPS는 DB와 로깅이 많은 애플리케이션에서 종종 병목입니다. 쓰기 IOPS가 최대치이면 더 빠른 스토리지 사용, 쓰기 빈도 감소, 배치 쓰기를 고려하세요.",
		"ops/s",
	},
	"disk.*.io_time_pct": {
		"Percentage of time the disk was busy with I/O during the collection interval (0-100%). This is the disk utilization metric. At 100%, every moment of time is spent doing I/O and new requests must wait in queue. Sustained 80%+ indicates the disk is becoming a bottleneck. Note: for parallel devices (SSDs, RAID arrays) this can be misleading since they can serve multiple requests simultaneously.",
		"수집 간격 동안 디스크가 I/O로 사용 중이었던 시간 비율(0-100%). 디스크 활용률 지표입니다. 100%에서는 모든 시간이 I/O에 사용되고 새 요청은 큐에서 대기해야 합니다. 80% 이상 지속되면 디스크가 병목이 되고 있습니다. 참고: SSD, RAID 등 병렬 장치에서는 동시에 여러 요청을 처리할 수 있어 이 값이 정확하지 않을 수 있습니다.",
		"%",
	},
	"disk.*.queue_depth": {
		"Average number of I/O requests waiting in the disk queue. A queue depth of 0-1 means the disk is keeping up. Depth of 2-4 is normal for SSDs handling concurrent requests. Depth consistently above 8-16 indicates the disk cannot keep up with demand and requests are accumulating, leading to increased latency.",
		"디스크 큐에 대기 중인 평균 I/O 요청 수. 큐 깊이 0-1은 디스크가 요청을 따라가고 있다는 의미. 2-4는 동시 요청을 처리하는 SSD에서 정상. 8-16 이상이 지속되면 디스크가 수요를 감당하지 못해 요청이 쌓이고 지연이 증가합니다.",
		"count",
	},

	// ========================== Network ==========================
	"net.total.bytes_sent": {
		"Cumulative total bytes transmitted (TX) across all network interfaces since boot. This is a monotonically increasing counter. Track the rate of change (see net.total.bytes_sent_sec) for real-time throughput monitoring.",
		"부팅 이후 모든 네트워크 인터페이스에서 전송(TX)한 누적 총 바이트. 단조 증가 카운터입니다. 실시간 처리량 모니터링은 변화율(net.total.bytes_sent_sec)을 확인하세요.",
		"bytes",
	},
	"net.total.bytes_recv": {
		"Cumulative total bytes received (RX) across all network interfaces since boot. This is a monotonically increasing counter. Track the rate of change (see net.total.bytes_recv_sec) for real-time throughput monitoring.",
		"부팅 이후 모든 네트워크 인터페이스에서 수신(RX)한 누적 총 바이트. 단조 증가 카운터입니다. 실시간 처리량 모니터링은 변화율(net.total.bytes_recv_sec)을 확인하세요.",
		"bytes",
	},
	"net.total.packets_sent": {"Cumulative total packets sent across all interfaces since boot. Track rate of change for per-second packet rate.", "부팅 이후 모든 인터페이스에서 송신한 누적 총 패킷 수. 초당 패킷 비율은 변화율을 추적하세요.", "count"},
	"net.total.packets_recv": {"Cumulative total packets received across all interfaces since boot. Track rate of change for per-second packet rate.", "부팅 이후 모든 인터페이스에서 수신한 누적 총 패킷 수. 초당 패킷 비율은 변화율을 추적하세요.", "count"},
	"net.total.errin": {
		"Total receive errors across all interfaces. Includes CRC errors, frame errors, and other low-level reception failures. Non-zero values may indicate cable problems, NIC hardware issues, or driver bugs. Even a small error rate on a high-traffic link should be investigated.",
		"모든 인터페이스의 총 수신 오류 수. CRC 오류, 프레임 오류, 기타 저수준 수신 실패를 포함합니다. 0이 아니면 케이블 문제, NIC 하드웨어 문제, 드라이버 버그를 나타낼 수 있습니다. 높은 트래픽 링크에서 작은 오류율도 조사가 필요합니다.",
		"count",
	},
	"net.total.errout": {
		"Total transmit errors across all interfaces. Indicates failures when sending packets — could be hardware faults, driver issues, or network congestion causing buffer overflows. Investigate cable connections and NIC health.",
		"모든 인터페이스의 총 송신 오류 수. 패킷 전송 실패를 나타내며, 하드웨어 결함, 드라이버 문제, 네트워크 혼잡으로 인한 버퍼 오버플로가 원인일 수 있습니다. 케이블 연결과 NIC 상태를 점검하세요.",
		"count",
	},
	"net.total.dropin": {
		"Total incoming packets dropped across all interfaces. Dropped packets are discarded before reaching the application — usually because the kernel's receive buffer is full (the application is not reading fast enough) or iptables/netfilter rules are dropping packets. High drop rates cause retransmissions and increased latency.",
		"모든 인터페이스에서 수신 중 드롭된 총 패킷 수. 드롭된 패킷은 애플리케이션에 도달하기 전에 폐기됩니다. 보통 커널 수신 버퍼가 가득 찼거나(애플리케이션이 충분히 빠르게 읽지 못함) iptables/netfilter 규칙이 패킷을 버리기 때문입니다. 높은 드롭율은 재전송과 지연 증가를 유발합니다.",
		"count",
	},
	"net.total.dropout": {
		"Total outgoing packets dropped across all interfaces. Outbound drops usually indicate the transmit queue is full — the network link is saturated or the NIC cannot send fast enough. May also be caused by traffic shaping or QoS policies.",
		"모든 인터페이스에서 송신 중 드롭된 총 패킷 수. 송신 드롭은 보통 전송 큐가 가득 찼음을 나타냅니다 — 네트워크 링크가 포화되었거나 NIC가 충분히 빠르게 전송하지 못합니다. 트래픽 셰이핑이나 QoS 정책으로 인해 발생할 수도 있습니다.",
		"count",
	},
	"net.total.bytes_sent_sec": {
		"Real-time total network send throughput (bytes/second) across all interfaces, calculated from delta between two collection intervals. This shows the actual current outgoing bandwidth usage. Compare with your link speed (e.g., 1Gbps = ~125MB/s) to assess network saturation.",
		"모든 인터페이스의 실시간 총 네트워크 송신 처리량(초당 바이트), 두 수집 간격의 차이로 계산. 현재 실제 아웃바운드 대역폭 사용량을 보여줍니다. 링크 속도(예: 1Gbps = ~125MB/s)와 비교하여 네트워크 포화도를 평가하세요.",
		"bytes/s",
	},
	"net.total.bytes_recv_sec": {
		"Real-time total network receive throughput (bytes/second) across all interfaces, calculated from delta between two collection intervals. This shows the actual current incoming bandwidth usage. Sudden spikes may indicate DDoS attacks, large file transfers, or backup operations.",
		"모든 인터페이스의 실시간 총 네트워크 수신 처리량(초당 바이트), 두 수집 간격의 차이로 계산. 현재 실제 인바운드 대역폭 사용량을 보여줍니다. 갑작스러운 급증은 DDoS 공격, 대용량 파일 전송, 백업 작업을 나타낼 수 있습니다.",
		"bytes/s",
	},
	"net.total.packets_sent_sec": {"Total packets sent per second across all interfaces. High packet rates with low byte rates indicate many small packets (typical of API microservices or DNS). Very high pps can stress the CPU with interrupt processing.", "모든 인터페이스의 초당 총 송신 패킷 수. 높은 패킷 비율에 낮은 바이트 비율은 작은 패킷이 많음을 나타냅니다(API 마이크로서비스나 DNS에 전형적). 매우 높은 pps는 인터럽트 처리로 CPU에 부담을 줄 수 있습니다.", "pkt/s"},
	"net.total.packets_recv_sec": {"Total packets received per second across all interfaces. See net.total.packets_sent_sec for interpretation.", "모든 인터페이스의 초당 총 수신 패킷 수. 해석은 net.total.packets_sent_sec를 참고하세요.", "pkt/s"},
	"net.*.bytes_sent":       {"Cumulative bytes sent on this specific network interface since boot. Each interface (eth0, ens5, etc.) tracks its own counters independently.", "부팅 이후 이 특정 네트워크 인터페이스에서 송신한 누적 바이트. 각 인터페이스(eth0, ens5 등)는 독립적으로 카운터를 추적합니다.", "bytes"},
	"net.*.bytes_recv":       {"Cumulative bytes received on this specific network interface since boot.", "부팅 이후 이 특정 네트워크 인터페이스에서 수신한 누적 바이트.", "bytes"},
	"net.*.packets_sent":     {"Cumulative packets sent on this network interface since boot.", "부팅 이후 이 네트워크 인터페이스에서 송신한 누적 패킷 수.", "count"},
	"net.*.packets_recv":     {"Cumulative packets received on this network interface since boot.", "부팅 이후 이 네트워크 인터페이스에서 수신한 누적 패킷 수.", "count"},
	"net.*.errin":            {"Receive errors on this interface. See net.total.errin for interpretation.", "이 인터페이스의 수신 오류. 해석은 net.total.errin을 참고하세요.", "count"},
	"net.*.errout":           {"Transmit errors on this interface. See net.total.errout for interpretation.", "이 인터페이스의 송신 오류. 해석은 net.total.errout을 참고하세요.", "count"},
	"net.*.dropin":           {"Incoming packets dropped on this interface. See net.total.dropin for interpretation.", "이 인터페이스에서 수신 중 드롭된 패킷. 해석은 net.total.dropin을 참고하세요.", "count"},
	"net.*.dropout":          {"Outgoing packets dropped on this interface. See net.total.dropout for interpretation.", "이 인터페이스에서 송신 중 드롭된 패킷. 해석은 net.total.dropout을 참고하세요.", "count"},
	"net.*.bytes_sent_sec":   {"Send throughput (bytes/sec) on this specific interface, computed as delta between two intervals.", "이 인터페이스의 송신 처리량(초당 바이트), 두 수집 간격의 차이로 계산.", "bytes/s"},
	"net.*.bytes_recv_sec":   {"Receive throughput (bytes/sec) on this specific interface, computed as delta between two intervals.", "이 인터페이스의 수신 처리량(초당 바이트), 두 수집 간격의 차이로 계산.", "bytes/s"},
	"net.*.packets_sent_sec": {"Packets sent per second on this specific interface.", "이 인터페이스의 초당 송신 패킷 수.", "pkt/s"},
	"net.*.packets_recv_sec": {"Packets received per second on this specific interface.", "이 인터페이스의 초당 수신 패킷 수.", "pkt/s"},
	"net.tcp.established": {
		"Number of TCP connections currently in ESTABLISHED state — actively transferring data between two endpoints. This is the most common healthy connection state. A very high number may indicate connection pooling issues, too many concurrent clients, or connections not being closed properly.",
		"현재 ESTABLISHED 상태의 TCP 연결 수 — 두 끝점 간 데이터를 적극적으로 전송 중. 가장 일반적인 정상 연결 상태입니다. 매우 높은 수치는 연결 풀 문제, 너무 많은 동시 클라이언트, 연결이 제대로 닫히지 않는 문제를 나타낼 수 있습니다.",
		"count",
	},
	"net.tcp.time_wait": {
		"Number of TCP connections in TIME_WAIT state. After a connection is closed, it enters TIME_WAIT for 2×MSL (typically 60 seconds) to handle delayed packets. High TIME_WAIT count is common on busy web servers and load balancers handling many short-lived connections. Not usually a problem unless it consumes all available local ports — consider enabling tcp_tw_reuse if thousands accumulate.",
		"TIME_WAIT 상태의 TCP 연결 수. 연결이 닫힌 후 지연 패킷 처리를 위해 2×MSL(보통 60초) 동안 TIME_WAIT에 머뭅니다. 짧은 연결을 많이 처리하는 웹 서버/로드밸런서에서 높은 수치는 흔합니다. 사용 가능한 로컬 포트를 모두 소진하지 않는 한 문제가 아니지만, 수천 개가 쌓이면 tcp_tw_reuse 활성화를 고려하세요.",
		"count",
	},
	"net.tcp.close_wait": {
		"Number of TCP connections in CLOSE_WAIT state. This means the remote side has closed the connection, but the local application has NOT closed its end yet. Accumulating CLOSE_WAIT connections is almost always an application bug — the application is not calling close() on sockets after the peer disconnects. This can lead to file descriptor exhaustion. Investigate the application's connection handling code.",
		"CLOSE_WAIT 상태의 TCP 연결 수. 원격 측은 연결을 닫았지만 로컬 애플리케이션이 아직 자기 쪽을 닫지 않았다는 의미입니다. CLOSE_WAIT 연결이 쌓이는 것은 거의 항상 애플리케이션 버그입니다 — 상대방이 연결을 끊은 후 소켓에 close()를 호출하지 않고 있습니다. 파일 디스크립터 고갈로 이어질 수 있습니다. 애플리케이션의 연결 처리 코드를 조사하세요.",
		"count",
	},
	"net.tcp.retransmits": {
		"Cumulative number of TCP segment retransmissions since boot (Linux only, from /proc/net/snmp). Retransmissions occur when an acknowledgment is not received in time — indicating packet loss, network congestion, or an unreliable link. A consistently increasing retransmit rate degrades application throughput and increases latency. Common causes: network congestion, faulty cables, overloaded switches, or Wi-Fi interference.",
		"부팅 이후 TCP 세그먼트 재전송 누적 횟수(Linux 전용, /proc/net/snmp에서 수집). 확인 응답이 시간 내에 수신되지 않으면 재전송이 발생하며, 패킷 손실, 네트워크 혼잡, 불안정한 링크를 나타냅니다. 재전송율이 지속적으로 증가하면 애플리케이션 처리량이 저하되고 지연이 증가합니다. 주요 원인: 네트워크 혼잡, 불량 케이블, 과부하된 스위치, Wi-Fi 간섭.",
		"count",
	},
	"net.tcp.tx_queue_total": {
		"Total bytes queued in send buffers across ALL TCP sockets (Linux only, from /proc/net/tcp + /proc/net/tcp6). This is data the application has written but the kernel hasn't sent yet, or sent but not yet acknowledged by the remote side. High total tx_queue indicates network congestion — data is piling up because the network can't deliver it fast enough. Common causes: slow remote endpoint, network bandwidth saturation, high packet loss causing slow congestion window growth.",
		"모든 TCP 소켓의 송신 버퍼에 대기 중인 총 바이트 (Linux 전용, /proc/net/tcp + /proc/net/tcp6에서 수집). 애플리케이션이 쓰기는 했지만 커널이 아직 보내지 않았거나, 보냈지만 원격 측의 확인 응답을 받지 못한 데이터입니다. tx_queue 합계가 높으면 네트워크 혼잡을 나타냅니다 — 네트워크가 충분히 빠르게 전달하지 못해 데이터가 쌓이고 있습니다. 주요 원인: 느린 원격 엔드포인트, 네트워크 대역폭 포화, 높은 패킷 손실로 인한 혼잡 윈도우 성장 지연.",
		"bytes",
	},
	"net.tcp.rx_queue_total": {
		"Total bytes queued in receive buffers across ALL TCP sockets (Linux only, from /proc/net/tcp + /proc/net/tcp6). This is data the kernel has received from the network but the application hasn't read yet (via recv/read syscalls). High total rx_queue means the application is not consuming incoming data fast enough. Common causes: application is CPU-bound and can't process data, blocking I/O in the application, slow event loop, or the application is stuck/deadlocked.",
		"모든 TCP 소켓의 수신 버퍼에 대기 중인 총 바이트 (Linux 전용, /proc/net/tcp + /proc/net/tcp6에서 수집). 커널이 네트워크에서 수신했지만 애플리케이션이 아직 읽지 않은(recv/read 시스템콜) 데이터입니다. rx_queue 합계가 높으면 애플리케이션이 들어오는 데이터를 충분히 빠르게 소비하지 못합니다. 주요 원인: 애플리케이션이 CPU 바운드로 데이터 처리 불가, 블로킹 I/O, 느린 이벤트 루프, 애플리케이션 멈춤/데드락.",
		"bytes",
	},
	"net.tcp.tx_queue_max": {
		"Largest send buffer queue size among all individual TCP sockets (Linux only). While tx_queue_total shows the aggregate, this shows the single worst-case socket. A very high value on one socket while others are low indicates a specific connection is congested — possibly a slow client, a network path with packet loss, or a connection to a distant endpoint with high RTT.",
		"모든 TCP 소켓 중 가장 큰 단일 송신 버퍼 큐 크기 (Linux 전용). tx_queue_total이 전체 합계를 보여주는 반면, 이 값은 가장 심한 단일 소켓을 보여줍니다. 하나의 소켓만 매우 높고 나머지는 낮으면 특정 연결이 혼잡한 것입니다 — 느린 클라이언트, 패킷 손실이 있는 네트워크 경로, 높은 RTT의 원거리 엔드포인트가 원인일 수 있습니다.",
		"bytes",
	},
	"net.tcp.rx_queue_max": {
		"Largest receive buffer queue size among all individual TCP sockets (Linux only). Shows the single socket with the most unread data. A large rx_queue on one socket suggests that particular connection's data is not being processed — the application may have a slow handler for that connection, or that specific connection's processing is blocked by a lock or slow operation.",
		"모든 TCP 소켓 중 가장 큰 단일 수신 버퍼 큐 크기 (Linux 전용). 읽지 않은 데이터가 가장 많은 소켓을 보여줍니다. 한 소켓의 rx_queue가 크면 해당 연결의 데이터가 처리되지 않고 있습니다 — 해당 연결의 핸들러가 느리거나, 락이나 느린 작업에 의해 처리가 차단되었을 수 있습니다.",
		"bytes",
	},

	// ========================== Process ==========================
	"proc.total_count": {
		"Total number of processes currently running on the system. A steadily increasing process count without corresponding decreases may indicate a process fork bomb, runaway script spawning children, or zombie processes not being reaped. Typical servers run 100-500 processes.",
		"시스템에서 현재 실행 중인 총 프로세스 수. 프로세스 수가 감소 없이 계속 증가하면 프로세스 포크 폭탄, 자식 프로세스를 무한 생성하는 스크립트, 회수되지 않는 좀비 프로세스를 나타낼 수 있습니다. 일반적인 서버는 100-500개 프로세스를 실행합니다.",
		"count",
	},
	"proc.top_cpu.*.pid":     {"Process ID (PID) of one of the top CPU-consuming processes. Use this to identify which process to investigate with tools like strace, perf, or application profilers.", "CPU 사용량 상위 프로세스의 프로세스 ID(PID). strace, perf, 애플리케이션 프로파일러 같은 도구로 조사할 프로세스를 식별하는 데 사용하세요.", ""},
	"proc.top_cpu.*.name":    {"Process name of a top CPU consumer. Quickly identifies which application or service is using the most CPU without needing to SSH into the server.", "CPU 사용량 상위 프로세스의 이름. 서버에 SSH 접속 없이도 어떤 애플리케이션이나 서비스가 CPU를 가장 많이 사용하는지 빠르게 파악합니다.", ""},
	"proc.top_cpu.*.cpu_pct": {"CPU usage percentage of this top CPU-consuming process, normalized to total system capacity (0-100%). For example, a process using 2 full cores on an 8-core system shows 25%.", "CPU 사용량 상위 프로세스의 CPU 사용률. 전체 시스템 용량 대비 정규화된 값(0-100%)입니다. 예: 8코어 시스템에서 2코어를 사용하면 25%로 표시됩니다.", "%"},
	"proc.top_cpu.*.mem_pct": {"Memory usage percentage of a top CPU-consuming process. Helps correlate high CPU with memory behavior — a process using both high CPU and high memory may be processing large datasets.", "CPU 사용량 상위 프로세스의 메모리 사용률. 높은 CPU와 메모리 동작을 연관시키는 데 도움 — CPU와 메모리 모두 높은 프로세스는 대량 데이터셋을 처리 중일 수 있습니다.", "%"},
	"proc.top_mem.*.pid":     {"Process ID of one of the top memory-consuming processes.", "메모리 사용량 상위 프로세스의 프로세스 ID.", ""},
	"proc.top_mem.*.name":    {"Process name of a top memory consumer. Identifies which application is using the most memory — useful for finding memory leaks or unexpected memory growth.", "메모리 사용량 상위 프로세스의 이름. 어떤 애플리케이션이 가장 많은 메모리를 사용하는지 식별 — 메모리 누수나 예기치 않은 메모리 증가를 찾는 데 유용합니다.", ""},
	"proc.top_mem.*.cpu_pct": {"CPU usage of a top memory-consuming process, normalized to total system capacity (0-100%). A process with high memory but low CPU may be holding cached data; high memory with high CPU indicates active processing.", "메모리 사용량 상위 프로세스의 CPU 사용률. 전체 시스템 용량 대비 정규화된 값(0-100%)입니다. 메모리는 높지만 CPU가 낮으면 캐시 데이터를 유지 중; 메모리와 CPU 모두 높으면 활발한 처리 중입니다.", "%"},
	"proc.top_mem.*.mem_pct": {"Memory usage percentage of this top memory-consuming process. Track over time to detect memory leaks — a slowly but steadily increasing value is suspicious.", "메모리 사용량 상위 프로세스의 메모리 사용률. 시간에 따라 추적하여 메모리 누수 감지 — 느리지만 꾸준히 증가하는 값은 의심스럽습니다.", "%"},

	// Top I/O processes
	"proc.top_io.*.pid":       {"Process ID of one of the top I/O-consuming processes. Use to identify which process is generating the most disk I/O.", "I/O 사용량 상위 프로세스의 프로세스 ID. 가장 많은 디스크 I/O를 발생시키는 프로세스를 식별합니다.", ""},
	"proc.top_io.*.name":      {"Process name of a top I/O consumer. Identifies which application or service is reading/writing the most data to disk.", "I/O 사용량 상위 프로세스의 이름. 어떤 애플리케이션이나 서비스가 디스크에 가장 많은 데이터를 읽고/쓰는지 식별합니다.", ""},
	"proc.top_io.*.read_bps":  {"Disk read rate (bytes/sec) of this top I/O process. High values indicate heavy read workloads — database queries, log scanning, file processing.", "I/O 상위 프로세스의 디스크 읽기 속도(bytes/sec). 높은 값은 DB 쿼리, 로그 스캔, 파일 처리 등 대량 읽기 워크로드를 나타냅니다.", "bytes/s"},
	"proc.top_io.*.write_bps": {"Disk write rate (bytes/sec) of this top I/O process. High values indicate heavy write workloads — logging, database writes, file downloads.", "I/O 상위 프로세스의 디스크 쓰기 속도(bytes/sec). 높은 값은 로깅, DB 기록, 파일 다운로드 등 대량 쓰기 워크로드를 나타냅니다.", "bytes/s"},
	"proc.io.total_read_bps":  {"Total disk read rate (bytes/sec) across all processes. Represents the aggregate I/O read bandwidth being consumed by all processes.", "모든 프로세스의 총 디스크 읽기 속도(bytes/sec). 전체 프로세스가 사용하는 I/O 읽기 대역폭 합계입니다.", "bytes/s"},
	"proc.io.total_write_bps": {"Total disk write rate (bytes/sec) across all processes. Represents the aggregate I/O write bandwidth being consumed by all processes.", "모든 프로세스의 총 디스크 쓰기 속도(bytes/sec). 전체 프로세스가 사용하는 I/O 쓰기 대역폭 합계입니다.", "bytes/s"},

	// ========================== Kernel ==========================
	"kernel.procs_running": {
		"Number of processes currently executing on a CPU core (Linux only). This shows how many processes are actively using CPU right now, NOT waiting. On a 4-core system, procs_running above 4 means some processes are in the run queue waiting for a core. Consistently above core count indicates CPU saturation.",
		"현재 CPU 코어에서 실행 중인 프로세스 수 (Linux 전용). 대기 중이 아니라 실제로 CPU를 사용하고 있는 프로세스 수를 보여줍니다. 4코어 시스템에서 procs_running이 4를 넘으면 일부 프로세스가 코어를 기다리며 실행 큐에 있습니다. 코어 수를 지속적으로 초과하면 CPU 포화입니다.",
		"count",
	},
	"kernel.procs_blocked": {
		"Number of processes blocked waiting for I/O to complete (Linux only). These processes want to run but are stuck waiting for disk, network, or other I/O. High procs_blocked indicates I/O bottleneck — the storage or network subsystem cannot keep up. Combine with iowait% and disk metrics for diagnosis.",
		"I/O 완료를 기다리며 블록된 프로세스 수 (Linux 전용). 실행하고 싶지만 디스크, 네트워크, 기타 I/O를 기다리며 멈춰 있는 프로세스입니다. procs_blocked가 높으면 I/O 병목을 나타냅니다 — 스토리지나 네트워크 서브시스템이 따라가지 못합니다. 진단을 위해 iowait%와 디스크 메트릭을 함께 확인하세요.",
		"count",
	},
	"kernel.runqueue_latency": {
		"Average time a process waits in the CPU run queue before actually executing (microseconds). This measures scheduling delay — how long a ready-to-run process waits for a CPU core. Low latency (<100μs) means the CPU has spare capacity. High latency (>1000μs) means processes are queuing up for CPU time. Requires eBPF support for accurate measurement.",
		"프로세스가 CPU 실행 큐에서 실제 실행까지 대기하는 평균 시간(마이크로초). 스케줄링 지연을 측정합니다 — 실행 준비된 프로세스가 CPU 코어를 기다리는 시간. 낮은 지연(<100μs)은 CPU 여유 용량. 높은 지연(>1000μs)은 CPU 시간을 기다리며 프로세스가 대기 중. 정확한 측정에는 eBPF 지원이 필요합니다.",
		"us",
	},
	"kernel.vmstat.pgpgin": {
		"Pages read into memory from disk per second. Includes both regular file I/O (page cache fills) and swap-in operations. High pgpgin is normal during file-heavy workloads. If combined with high swap usage, it indicates memory pressure causing swap-in activity.",
		"초당 디스크에서 메모리로 읽어들인 페이지 수. 일반 파일 I/O(페이지 캐시 채움)와 스왑-인 작업을 모두 포함합니다. 파일 집약적 워크로드에서 높은 pgpgin은 정상입니다. 높은 스왑 사용량과 함께이면 메모리 압박으로 인한 스왑-인 활동을 나타냅니다.",
		"pages/s",
	},
	"kernel.vmstat.pgpgout": {
		"Pages written from memory to disk per second. Includes file writes (dirty page flushes) and swap-out operations. High pgpgout during normal operation usually means file write activity. Combined with increasing swap usage, it indicates the OS is evicting memory pages to disk under memory pressure.",
		"초당 메모리에서 디스크로 쓴 페이지 수. 파일 쓰기(더티 페이지 플러시)와 스왑-아웃 작업을 포함합니다. 정상 운영 중 높은 pgpgout는 보통 파일 쓰기 활동입니다. 스왑 사용량 증가와 함께이면 OS가 메모리 압박으로 메모리 페이지를 디스크로 내보내고 있습니다.",
		"pages/s",
	},
	"kernel.vmstat.pswpin": {
		"Swap pages read from disk per second. This is SPECIFICALLY swap activity (not regular file I/O). Any non-zero value means the system is actively swapping in — processes are accessing memory that was previously evicted to disk. This causes significant performance degradation. If sustained, the system needs more RAM.",
		"초당 디스크에서 읽은 스왑 페이지 수. 이것은 일반 파일 I/O가 아닌 스왑 활동만을 나타냅니다. 0이 아닌 값은 시스템이 적극적으로 스왑-인 중 — 이전에 디스크로 내보낸 메모리를 프로세스가 접근하고 있습니다. 이는 심각한 성능 저하를 유발합니다. 지속되면 더 많은 RAM이 필요합니다.",
		"pages/s",
	},
	"kernel.vmstat.pswpout": {
		"Swap pages written to disk per second. Non-zero means the kernel is actively evicting memory pages to swap because physical RAM is exhausted. The most critical memory pressure indicator — even small sustained values indicate the system is running out of memory and performance will suffer.",
		"초당 디스크에 쓴 스왑 페이지 수. 0이 아니면 물리 RAM이 소진되어 커널이 메모리 페이지를 스왑으로 내보내고 있습니다. 가장 중요한 메모리 압박 지표 — 작은 값이라도 지속되면 시스템 메모리가 부족하고 성능이 저하될 것입니다.",
		"pages/s",
	},

	// ========================== GPU ==========================
	"gpu.*.util_pct": {
		"GPU core utilization percentage. Shows how busy the GPU's compute units are. 0% means the GPU is idle, 100% means fully saturated. For ML training or inference workloads, sustained high utilization is desired (getting full value from the GPU). For desktop/rendering, sustained 100% may indicate insufficient GPU power.",
		"GPU 코어 활용률. GPU 연산 유닛이 얼마나 바쁜지 보여줍니다. 0%는 유휴, 100%는 완전 포화. ML 학습/추론 워크로드에서는 지속적인 높은 활용률이 바람직합니다(GPU를 최대한 활용). 데스크톱/렌더링에서 100%가 지속되면 GPU 성능이 부족할 수 있습니다.",
		"%",
	},
	"gpu.*.mem_util_pct": {
		"GPU memory controller utilization — how busy the memory bus is reading/writing GPU memory. High mem_util with low core util may indicate memory-bandwidth-bound workloads. Bottleneck here means the GPU cores are waiting for data from memory.",
		"GPU 메모리 컨트롤러 활용률 — GPU 메모리를 읽고 쓰는 메모리 버스가 얼마나 바쁜지. 메모리 활용률은 높지만 코어 활용률이 낮으면 메모리 대역폭 병목 워크로드입니다. 여기서 병목이 발생하면 GPU 코어가 메모리에서 데이터를 기다리고 있습니다.",
		"%",
	},
	"gpu.*.temp_c": {
		"GPU temperature in degrees Celsius. Most GPUs throttle performance at 80-90°C to prevent damage. If temperature stays near the throttle threshold, the GPU is losing performance. Ensure adequate cooling. Sustained temperatures above 85°C reduce GPU lifespan.",
		"GPU 온도(섭씨). 대부분의 GPU는 손상 방지를 위해 80-90°C에서 성능을 제한합니다. 온도가 쓰로틀링 임계치 근처에 머물면 GPU가 성능을 잃고 있습니다. 적절한 냉각을 확보하세요. 85°C 이상 지속되면 GPU 수명이 단축됩니다.",
		"°C",
	},
	"gpu.*.mem_used": {
		"GPU memory currently in use. Running out of GPU memory causes out-of-memory errors in CUDA/OpenCL applications, or the driver may fall back to system RAM (very slow). Track this to right-size your model batch sizes or detect GPU memory leaks.",
		"현재 사용 중인 GPU 메모리. GPU 메모리가 부족하면 CUDA/OpenCL 애플리케이션에서 메모리 부족 오류가 발생하거나, 드라이버가 시스템 RAM으로 폴백합니다(매우 느림). 모델 배치 크기를 적절히 조정하거나 GPU 메모리 누수를 감지하는 데 추적하세요.",
		"bytes",
	},
	"gpu.*.mem_total":   {"Total GPU memory installed. The maximum VRAM available for textures, model weights, and compute buffers.", "설치된 총 GPU 메모리. 텍스처, 모델 가중치, 연산 버퍼에 사용 가능한 최대 VRAM입니다.", "bytes"},
	"gpu.*.power_watts": {
		"Current GPU power consumption in watts. Useful for monitoring energy costs and thermal output. Approaching the GPU's TDP (Thermal Design Power) limit means the GPU is running at full capacity and may begin power-throttling.",
		"현재 GPU 전력 소비(와트). 에너지 비용과 발열 모니터링에 유용합니다. GPU의 TDP(열 설계 전력) 한계에 근접하면 GPU가 전체 용량으로 동작하며 전력 쓰로틀링을 시작할 수 있습니다.",
		"W",
	},

	// ========================== eBPF ==========================
	"ebpf.bio_latency_us.p50": {
		"Block I/O latency median (50th percentile) in microseconds. This is the 'typical' latency for disk I/O operations measured at the kernel level using eBPF. For SSDs, p50 should be under 200μs. For HDDs, 2000-5000μs is typical. Significantly higher values indicate disk performance issues.",
		"블록 I/O 지연시간 중앙값(50번째 백분위, 마이크로초). eBPF를 사용하여 커널 수준에서 측정한 디스크 I/O 작업의 '일반적인' 지연시간입니다. SSD는 200μs 이하, HDD는 2000-5000μs가 일반적입니다. 현저히 높은 값은 디스크 성능 문제를 나타냅니다.",
		"us",
	},
	"ebpf.bio_latency_us.p90": {
		"Block I/O latency 90th percentile — 90% of I/O operations complete within this time. The gap between p50 and p90 reveals latency consistency. A large gap means unpredictable I/O performance (some operations are much slower). Often caused by disk queue contention or background operations like garbage collection on SSDs.",
		"블록 I/O 지연시간 90번째 백분위 — I/O 작업의 90%가 이 시간 내에 완료. p50과 p90의 차이는 지연 일관성을 보여줍니다. 차이가 크면 I/O 성능이 불규칙(일부 작업이 훨씬 느림). 디스크 큐 경합이나 SSD 가비지 컬렉션 같은 백그라운드 작업이 원인일 수 있습니다.",
		"us",
	},
	"ebpf.bio_latency_us.p99": {
		"Block I/O latency 99th percentile — the 'worst case' (excluding extreme outliers). Critical for latency-sensitive applications. If p99 is 10x higher than p50, some I/O operations experience severe delays. This tail latency directly impacts user-facing response times in databases and web servers.",
		"블록 I/O 지연시간 99번째 백분위 — 극단적 이상치를 제외한 '최악의 경우'. 지연에 민감한 애플리케이션에 중요합니다. p99가 p50의 10배 이상이면 일부 I/O 작업이 심각한 지연을 겪습니다. 이 꼬리 지연은 DB와 웹 서버의 사용자 응답 시간에 직접 영향을 줍니다.",
		"us",
	},
	"ebpf.tcp_connect_latency_us.p50": {
		"TCP connection establishment latency median (50th percentile). The time from SYN sent to ESTABLISHED state. Measures network round-trip time plus server processing. For local network connections, should be under 1000μs. For cross-region, 10,000-50,000μs is typical.",
		"TCP 연결 수립 지연시간 중앙값(50번째 백분위). SYN 전송부터 ESTABLISHED 상태까지의 시간. 네트워크 왕복 시간과 서버 처리 시간을 측정합니다. 로컬 네트워크 연결은 1000μs 이하, 리전 간은 10,000-50,000μs가 일반적입니다.",
		"us",
	},
	"ebpf.tcp_connect_latency_us.p90": {"TCP connection latency 90th percentile. See p50 for baseline context. A large gap between p50 and p90 may indicate intermittent network issues or DNS resolution delays.", "TCP 연결 지연시간 90번째 백분위. 기준 맥락은 p50을 참고하세요. p50과 p90의 차이가 크면 간헐적 네트워크 문제나 DNS 해석 지연을 나타낼 수 있습니다.", "us"},
	"ebpf.tcp_connect_latency_us.p99": {"TCP connection latency 99th percentile — worst-case connection time. High p99 with normal p50/p90 may indicate occasional DNS timeouts, network path changes, or SYN retransmissions due to packet loss.", "TCP 연결 지연시간 99번째 백분위 — 최악의 연결 시간. p50/p90은 정상인데 p99가 높으면 간헐적 DNS 타임아웃, 네트워크 경로 변경, 패킷 손실로 인한 SYN 재전송을 나타낼 수 있습니다.", "us"},
	"ebpf.runqueue_latency_us.avg": {
		"Average CPU run-queue latency measured by eBPF. How long processes wait in the scheduler queue before getting CPU time. This is a direct measure of CPU contention. Under 100μs is excellent. 100-1000μs indicates moderate load. Above 1000μs means processes experience noticeable scheduling delays — CPU is overloaded.",
		"eBPF로 측정한 평균 CPU 실행 큐 지연. 프로세스가 CPU 시간을 받기 전 스케줄러 큐에서 대기하는 시간. CPU 경합의 직접적인 측정입니다. 100μs 이하는 우수. 100-1000μs는 보통 부하. 1000μs 이상은 프로세스가 눈에 띄는 스케줄링 지연을 겪으며 CPU가 과부하 상태입니다.",
		"us",
	},
	"ebpf.cache_hit_rate": {
		"Page cache hit rate — percentage of file read requests served from memory cache without hitting disk. 95%+ is excellent (most reads come from cache). Below 80% means many reads go to disk, increasing I/O load and latency. Low hit rate may indicate working set is larger than available memory, or applications are accessing many different files without locality.",
		"페이지 캐시 적중률 — 디스크 접근 없이 메모리 캐시에서 처리된 파일 읽기 요청 비율. 95% 이상이면 우수(대부분의 읽기가 캐시에서 처리). 80% 이하면 많은 읽기가 디스크로 가며 I/O 부하와 지연이 증가합니다. 낮은 적중률은 작업 세트가 가용 메모리보다 크거나, 지역성 없이 많은 파일에 접근 중임을 나타냅니다.",
		"%",
	},
}

// LookupMetricDesc finds the best matching description for a concrete metric name.
// It tries exact match first, then pattern matching with "*" wildcard.
func LookupMetricDesc(name string) MetricDesc {
	// Exact match
	if d, ok := metricDescriptions[name]; ok {
		return d
	}

	// Pattern match: replace variable segments with "*"
	parts := strings.Split(name, ".")
	for i := len(parts) - 1; i >= 0; i-- {
		trial := make([]string, len(parts))
		copy(trial, parts)
		trial[i] = "*"
		key := strings.Join(trial, ".")
		if d, ok := metricDescriptions[key]; ok {
			return d
		}
	}

	for i := 0; i < len(parts); i++ {
		for j := i; j < len(parts); j++ {
			trial := make([]string, len(parts))
			copy(trial, parts)
			for k := i; k <= j; k++ {
				trial[k] = "*"
			}
			key := strings.Join(trial, ".")
			if d, ok := metricDescriptions[key]; ok {
				return d
			}
		}
	}

	return MetricDesc{}
}
