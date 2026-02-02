# Only1Mon

Go + Alpine.js 기반 경량 시스템 모니터링 대시보드.

## Claude Code 규칙

- 매 프롬프트 응답 완료 시 변경사항을 **git commit + push** 한다.

## 빌드 & 실행

### Linux / macOS (Make)

```bash
make build           # build/ 디렉토리에 바이너리 생성
make start           # 빌드 후 데몬 시작
make stop            # 데몬 중지
make status          # 데몬 상태 확인
make run             # 빌드 후 포그라운드 실행
make dev             # go run 으로 개발 실행
make clean           # build/, DB, PID, 로그 삭제
```

### Windows (build.cmd)

```cmd
build.cmd              # 빌드 (build\only1mon.exe)
build.cmd run          # 빌드 후 포그라운드 실행
build.cmd dev          # go run 으로 개발 실행
build.cmd clean        # build\, DB, PID, 로그 삭제
build.cmd build-all    # 크로스 컴파일 (linux/darwin/windows, amd64/arm64)
```

> Windows에서는 데몬 모드(start/stop/status)를 지원하지 않으며, `run`으로 포그라운드 실행합니다.

### 주요 명령어

```bash
# 빌드
make build                           # CGO_ENABLED=0, 버전 태그 자동 삽입 (Linux/macOS)
build.cmd build                      # CGO_ENABLED=0, 버전 태그 자동 삽입 (Windows)
go vet ./...                         # 정적 분석
go test ./...                        # 테스트
make build-all                       # 크로스 컴파일 (linux/darwin, amd64/arm64)
build.cmd build-all                  # 크로스 컴파일 (linux/darwin/windows, amd64/arm64)
goreleaser build --snapshot --clean  # GoReleaser 로컬 스냅샷 빌드
goreleaser release --skip=publish    # GoReleaser 릴리스 빌드 (업로드 생략)

# 실행
only1mon start                       # 데몬 시작 (백그라운드)
only1mon stop                        # 데몬 중지
only1mon status                      # 데몬 상태 확인
only1mon run                         # 포그라운드 실행
only1mon version                     # 버전 출력
only1mon -nginx                      # nginx 리버스 프록시 설정 출력
only1mon start -config /etc/only1mon/config.yaml  # 설정 파일 지정
```

### 설정

`config.yaml` 파일로 기본 설정 (config.yaml < 환경변수 < 플래그 우선순위).

```yaml
# config.yaml
listen: "127.0.0.1:9923"    # HTTP 바인딩 주소
database: "only1mon.db"      # SQLite DB 파일 경로
base_path: "/"               # 리버스 프록시 base path
pid_file: "only1mon.pid"     # PID 파일 경로
log_file: "only1mon.log"     # 로그 파일 경로
```

| 플래그 | 환경변수 | 기본값 | 설명 |
|--------|----------|--------|------|
| `-listen` | `ONLY1MON_LISTEN` | `127.0.0.1:9923` | HTTP 바인딩 주소 |
| `-db` | `ONLY1MON_DB` | `only1mon.db` | SQLite DB 경로 |
| `-base-path` | `ONLY1MON_BASE_PATH` | `/` | 리버스 프록시 base path |
| `-config` | — | `config.yaml` | 설정 파일 경로 |
| `-pid-file` | — | `only1mon.pid` | PID 파일 경로 |
| `-log-file` | — | `only1mon.log` | 로그 파일 경로 |

수집 주기, 데이터 보관, Top 프로세스 출력 건수 등 런타임 설정은 웹 UI Settings 페이지에서 관리 (DB 저장).

## 프로젝트 구조

```
cmd/only1mon/main.go          # 진입점, 서브커맨드(start/stop/status/run), 데몬 관리
internal/
  api/
    router.go                  # HTTP 라우터, 미들웨어 (net/http, Go 1.22 패턴), base_path 지원
    collectors.go              # 수집기 + 메트릭 ON/OFF API, ensureMetricsEnabled
    metrics.go                 # 메트릭 조회/가용 목록 API
    alerts.go                  # 알림 규칙 CRUD API
    dashboard.go               # 대시보드 레이아웃 CRUD
    settings.go                # 설정 API (top_process_count 포함)
    ws.go                      # WebSocket 허브
  collector/
    collector.go               # Collector 인터페이스 정의
    registry.go                # 수집기 레지스트리 (등록, 활성화, 자동 수집기 활성화)
    scheduler.go               # 주기적 수집 스케줄러
    alerts.go                  # 알림 엔진 (룰 평가, 가상FS 제외)
    descriptions.go            # 메트릭별 설명/단위 (EN/KO), 와일드카드 패턴 매칭
    cpu.go, memory.go, disk.go, network.go, process.go, kernel.go, gpu.go
    process_io_darwin.go       # macOS 프로세스 I/O (purego + proc_pid_rusage)
    process_io_linux.go        # Linux 프로세스 I/O (/proc 파싱)
  config/config.go             # 설정 로드 (config.yaml + env + flag)
  model/
    collector.go               # CollectorState, CollectorInfo, MetricState, ImpactLevel
    metric.go                  # MetricSample, MetricMeta
    alert.go                   # Alert, AlertRule, AlertSeverity
    dashboard.go               # DashboardLayout
    setting.go                 # Setting
  store/
    migrations.go              # DB 스키마 마이그레이션 (v1~v5)
    sqlite.go                  # SQLite CRUD (WAL 모드, 단일 writer)
web/
  embed.go                     # 정적 파일 임베딩, base_path 주입
  static/
    index.html                 # SPA 메인 (Alpine.js)
    css/app.css                # 다크 테마 스타일
    js/
      app.js                   # Alpine 앱 초기화, 공유 스토어 (chartColors, topProcessCount, i18n)
      lib/
        i18n.js                # EN/KO 번역 (Alpine store)
        api-client.js          # REST API 클라이언트 (__BASE_PATH 지원)
        ws-client.js           # WebSocket 클라이언트 (__BASE_PATH 지원)
      pages/
        dashboard.js           # 대시보드 페이지 (GridStack + uPlot), metricMeta 전달
        events.js              # 이벤트/알림 페이지
        metrics.js             # 수집기/메트릭 관리 페이지
        settings.js            # 설정 페이지 (수집주기, 보관기간, 차트색상, Top 프로세스 건수)
      components/
        chart-widget.js        # 차트 위젯 (단위별 포맷팅, 차트간 커서 동기화)
        table-widget.js        # 테이블 위젯 (단위별 포맷팅)
        top-widget.js          # Top 위젯 (CPU/메모리 상위 프로세스)
        iotop-widget.js        # IoTop 위젯 (디스크 I/O 상위 프로세스)
    vendor/                    # Alpine.js, uPlot, GridStack
```

## 아키텍처

### 백엔드 레이어

```
Config(YAML) → Store(SQLite) → Registry(Collector) → Scheduler → API → WebSocket
```

- **Config**: YAML 기반 설정, 우선순위: config.yaml < 환경변수 < 커맨드라인 플래그
- **Store**: SQLite WAL 모드, `MaxOpenConns=1` 단일 writer, 마이그레이션 기반 스키마 진화
- **Registry**: `sync.RWMutex`로 스레드 안전한 수집기 + 메트릭 상태 관리, 위젯 추가 시 수집기 자동 활성화
- **Scheduler**: 고정 주기 수집 → DB 저장 → WebSocket 브로드캐스트 → 알림 평가
- **AlertEngine**: DB 기반 룰 관리, 가상 FS(dev/proc/sys/run) 메트릭 자동 제외
- **API**: Go 1.22 `net/http` 라우팅, 미들웨어(Recovery/CORS/Logging), base_path StripPrefix

### 프론트엔드

- **Alpine.js** 컴포넌트 (`x-data`) + Alpine Store (i18n, dashboard, chartColors, topProcessCount)
- **uPlot** 차트 렌더링 (차트간 커서 동기화: `cursor.sync`)
- **GridStack** 드래그&드롭 대시보드 레이아웃
- **위젯 타입**: chart, table, top, iotop
- 페이지: Dashboard, Events, Metrics, Settings (SPA, Alpine 상태 전환)
- 단위별 값 포맷팅: bytes→KB/MB/GB, bytes/s→KB/s/MB/s, %, ms, us, °C, W, MHz

### 데몬 관리

```
only1mon start → re-exec "run" with Setsid:true → PID file + log file
only1mon stop  → SIGTERM → 10초 대기
only1mon status → PID file 확인 + signal 0
```

## DB 스키마 (v1~v5)

| 테이블 | 용도 |
|--------|------|
| `schema_version` | 마이그레이션 버전 추적 |
| `metric_samples` | 타임스탬프 기반 메트릭 값 저장 (인덱스: metric_name+ts, ts) |
| `settings` | key-value 설정 (collect_interval, retention_hours, top_process_count 등) |
| `collector_state` | 수집기별 활성화 상태 + 설정 JSON |
| `dashboard_layouts` | 대시보드 그리드 레이아웃 (위젯 metricMeta 포함) |
| `metric_state` | 개별 메트릭별 활성화 상태 (v5, opt-out 모델) |
| `alert_rules` | 알림 규칙 (메트릭 패턴, 연산자, 임계값, 심각도, 메시지) |

## API 엔드포인트

### 수집기

```
GET    /api/v1/collectors                       # 수집기 목록 (metric_states 포함)
PUT    /api/v1/collectors/{id}/enable            # 수집기 활성화
PUT    /api/v1/collectors/{id}/disable           # 수집기 비활성화
PUT    /api/v1/collectors/{id}/metrics/enable    # 수집기 내 전체 메트릭 활성화
PUT    /api/v1/collectors/{id}/metrics/disable   # 수집기 내 전체 메트릭 비활성화
PUT    /api/v1/metrics/ensure-enabled            # 메트릭 목록 활성화 + 부모 수집기 자동 활성화
```

### 메트릭

```
GET    /api/v1/metrics/available                 # 가용 메트릭 트리 (그룹별, unit 포함)
GET    /api/v1/metrics/query?name=&from=&to=&step=  # 메트릭 조회 (다운샘플링)
PUT    /api/v1/metrics/state/{name}/enable       # 개별 메트릭 활성화
PUT    /api/v1/metrics/state/{name}/disable      # 개별 메트릭 비활성화
```

### 알림

```
GET    /api/v1/alerts                            # 활성 알림 목록
GET    /api/v1/alert-rules                       # 알림 규칙 목록
POST   /api/v1/alert-rules                       # 알림 규칙 생성
PUT    /api/v1/alert-rules/{id}                  # 알림 규칙 수정
DELETE /api/v1/alert-rules/{id}                  # 알림 규칙 삭제
```

### 설정 / 대시보드

```
GET    /api/v1/settings                          # 설정 조회
PUT    /api/v1/settings                          # 설정 저장
GET    /api/v1/settings/db-info                  # DB 정보 (파일크기, 레코드수)
DELETE /api/v1/settings/db-purge                 # DB 데이터 퍼지
GET    /api/v1/dashboard/layouts                  # 레이아웃 목록
POST   /api/v1/dashboard/layouts                  # 레이아웃 생성
GET    /api/v1/dashboard/layouts/{id}             # 레이아웃 조회
PUT    /api/v1/dashboard/layouts/{id}             # 레이아웃 수정
DELETE /api/v1/dashboard/layouts/{id}             # 레이아웃 삭제
GET    /api/v1/ws                                 # WebSocket 실시간 스트림
```

## 핵심 패턴 & 컨벤션

- **수집기 추가**: `Collector` 인터페이스 구현 → `main.go`의 `registerAllCollectors()`에 등록
- **메트릭 설명 추가**: `descriptions.go`의 `metricDescriptions` 맵에 추가 (`*` 와일드카드 지원)
- **에러 처리**: 개별 수집기 실패 시 로그만 남기고 계속 진행, HTTP는 JSON `{"error": "..."}` 응답
- **동시성**: Registry는 `RWMutex`, Scheduler는 `context.Context` 기반 취소
- **i18n**: `web/static/js/lib/i18n.js`에 EN/KO 키-값 추가, 템플릿에서 `$store.i18n.t('key')` 사용
- **메트릭 필터링**: opt-out 모델 (기본 활성, `metric_state`에 `enabled=0`인 것만 비활성)
- **CGO_ENABLED=0**: purego로 macOS syscall 호출 (proc_pid_rusage), cgo 불필요
- **위젯 metricMeta**: 차트/테이블 위젯에 메트릭 단위 정보 전달 → 단위별 human-readable 포맷팅

## 주요 기능 상세

### 수집기 자동 활성화

위젯 추가 시 `EnsureMetricsEnabled` API 호출 → 백엔드에서 해당 메트릭의 부모 수집기가 비활성이면 자동으로 활성화. prefix 매칭으로 동적 메트릭(`proc.top_cpu.0.pid` 등)도 처리.

### 차트 단위 포맷팅

- `metricMeta`에서 메트릭별 unit 정보 조회 (bytes, bytes/s, %, ms, us, °C, W, MHz)
- Y축, 툴팁, 시리즈 값에 단위별 human-readable 표시
- 차트 전체 대표 단위(Y축)와 메트릭별 개별 단위(툴팁) 분리 처리

### 차트 커서 동기화

모든 ChartWidget이 `cursor.sync.key = 'only1mon-sync'` 공유 → 한 차트에서 마우스 이동 시 다른 차트의 커서/툴팁도 같은 타임스탬프로 동기화.

### 알림 제외 메트릭

가상/의사 파일시스템(`/dev`, `/proc`, `/sys`, `/run`, `/snap`, `/tmpfs`, `/devfs`)의 disk 메트릭은 알림 평가에서 자동 제외 (항상 100%이므로 무의미).

### macOS 프로세스 I/O

`purego`로 `libSystem.B.dylib`의 `proc_pid_rusage` 호출. `CGO_ENABLED=0` 빌드 호환. `sync.Once`로 초기화, 실패 시 graceful fallback.

### Top 프로세스 출력 건수

Settings에서 `top_process_count` 설정 (기본 10, 범위 1~50). 런타임 변경 시 수집기 + 프론트엔드(Alpine store) 동시 반영. 위젯은 store 변경 감지 → 자동 재구독.
