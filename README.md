# Only1Mon

Lightweight system monitoring dashboard built with Go and Alpine.js.
Single binary, zero external dependencies, real-time WebSocket streaming.

## Features

- **Real-time Dashboard** — Drag-and-drop widgets with live metric charts (uPlot + GridStack)
- **Multiple Widget Types** — Chart, Table, Top (process CPU/mem), IoTop (process disk I/O)
- **7 Built-in Collectors** — CPU, Memory, Disk, Network, Process, Kernel, GPU
- **Per-metric Control** — Enable/disable individual metrics without restarting
- **Alert Engine** — Configurable threshold-based alerts with EN/KO messages
- **Chart Cursor Sync** — Hover on one chart, all charts follow the same timestamp
- **Human-readable Units** — Bytes, bytes/s, %, ms, us, etc. auto-formatted
- **Daemon Mode** — `start` / `stop` / `status` with PID file management
- **Nginx Ready** — Built-in `base_path` support and `-nginx` config generator
- **Single Binary** — `CGO_ENABLED=0`, no cgo required (purego for macOS syscalls)
- **i18n** — English / Korean UI
- **Dark Theme** — Modern dark UI designed for always-on monitoring

## Quick Start

### Build

```bash
git clone https://github.com/playok/only1mon.git
cd only1mon
make build
```

Requires Go 1.22+. Produces `build/only1mon`.

### Run

```bash
# Foreground
./build/only1mon run

# Daemon
./build/only1mon start
./build/only1mon status
./build/only1mon stop
```

Open http://127.0.0.1:9923 in your browser.

### Cross-compile

```bash
make build-all    # linux/darwin x amd64/arm64
```

## Configuration

Create `config.yaml` in the working directory (or specify with `-config`):

```yaml
listen: "127.0.0.1:9923"
database: "only1mon.db"
base_path: "/"
pid_file: "only1mon.pid"
log_file: "only1mon.log"
```

Priority: `config.yaml` < environment variables < command-line flags.

| Flag | Env Var | Default | Description |
|------|---------|---------|-------------|
| `-listen` | `ONLY1MON_LISTEN` | `127.0.0.1:9923` | HTTP bind address |
| `-db` | `ONLY1MON_DB` | `only1mon.db` | SQLite database path |
| `-base-path` | `ONLY1MON_BASE_PATH` | `/` | URL prefix for reverse proxy |
| `-config` | — | `config.yaml` | Config file path |
| `-pid-file` | — | `only1mon.pid` | PID file path |
| `-log-file` | — | `only1mon.log` | Log file path |

Runtime settings (collection interval, retention, chart colors, top process count) are managed in the web UI Settings page and persisted to SQLite.

## Nginx Reverse Proxy

Generate a sample nginx config:

```bash
./build/only1mon -nginx
```

Example output:

```nginx
location /mon/ {
    proxy_pass         http://127.0.0.1:9923/;
    proxy_http_version 1.1;
    proxy_set_header   Upgrade $http_upgrade;
    proxy_set_header   Connection "upgrade";
    proxy_set_header   Host $host;
    proxy_buffering    off;
    proxy_read_timeout 86400s;
}
```

Set `base_path: "/mon"` in `config.yaml` to match.

## Commands

```
only1mon start           Start as daemon (background)
only1mon stop            Stop daemon (SIGTERM)
only1mon status          Check daemon status
only1mon run             Run in foreground
only1mon version         Print version
only1mon -nginx          Print nginx reverse proxy config
```

## Collectors

| Collector | Metrics | Description |
|-----------|---------|-------------|
| **cpu** | usage, user, system, iowait, idle, per-core, load avg | CPU utilization and load |
| **memory** | total, used, free, available, cached, buffers, swap | Memory and swap usage |
| **disk** | total, used, free, used_pct, read/write bytes/sec | Per-mount usage and I/O |
| **network** | bytes sent/recv, packets, errors, per-interface | Network throughput |
| **process** | top CPU, top memory, top I/O processes | Process resource ranking |
| **kernel** | context switches, interrupts, procs blocked/running | Kernel-level stats |
| **gpu** | utilization, temperature, memory, power | GPU monitoring (NVIDIA) |

On first run, `cpu`, `memory`, and `disk` collectors are enabled by default. Other collectors are auto-enabled when you add widgets that require their metrics.

## Dashboard Widgets

- **Chart** — Time-series line chart with uPlot. Supports multiple metrics, cursor sync across charts, unit-aware Y-axis and tooltips.
- **Table** — Real-time metric values in a grid with flash animations on change.
- **Top** — Linux `top`-like view: CPU bar, memory bar, swap bar, load average, top-N processes by CPU.
- **IoTop** — Linux `iotop`-like view: total read/write throughput, top-N processes by disk I/O.

Layouts are saved/loaded from the database and persist across sessions.

## Alert System

Built-in alert rules monitor critical thresholds:

- CPU user > 90%, system > 50%, iowait > 30%
- Memory > 80% / 90%, swap > 1GB
- Disk > 85% / 95%
- Network errors, blocked processes, GPU temperature

Rules are managed via the API (CRUD) and evaluated on every collection cycle. Virtual filesystem mounts (`/dev`, `/proc`, `/sys`, `/run`) are automatically excluded.

## Architecture

```
┌─────────┐    ┌─────────┐    ┌──────────┐    ┌───────────┐
│ Config  │───>│  Store   │───>│ Registry │───>│ Scheduler │
│ (YAML)  │    │ (SQLite) │    │(Collector)│   │  (loop)   │
└─────────┘    └─────────┘    └──────────┘    └─────┬─────┘
                                                     │
                              ┌───────────┐    ┌─────▼─────┐
                              │   Alert   │<───│    API    │───> HTTP
                              │  Engine   │    │ (net/http) │
                              └───────────┘    └─────┬─────┘
                                                     │
                                               ┌─────▼─────┐
                                               │ WebSocket │───> Browser
                                               │    Hub    │
                                               └───────────┘
```

- **Backend**: Go, `net/http` (Go 1.22 routing), SQLite (WAL mode), WebSocket
- **Frontend**: Alpine.js, uPlot, GridStack, vanilla JS (no build step)
- **Storage**: SQLite with automatic schema migrations (v1-v5)
- **macOS**: Process I/O via `purego` calling `proc_pid_rusage` from `libSystem.B.dylib`

## API

### Collectors
```
GET    /api/v1/collectors
PUT    /api/v1/collectors/{id}/enable
PUT    /api/v1/collectors/{id}/disable
PUT    /api/v1/metrics/ensure-enabled
```

### Metrics
```
GET    /api/v1/metrics/available
GET    /api/v1/metrics/query?name=cpu.total.usage&from=&to=&step=
PUT    /api/v1/metrics/state/{name}/enable
PUT    /api/v1/metrics/state/{name}/disable
```

### Alerts
```
GET    /api/v1/alerts
GET    /api/v1/alert-rules
POST   /api/v1/alert-rules
PUT    /api/v1/alert-rules/{id}
DELETE /api/v1/alert-rules/{id}
```

### Dashboard & Settings
```
GET    /api/v1/settings
PUT    /api/v1/settings
GET    /api/v1/settings/db-info
DELETE /api/v1/settings/db-purge
GET    /api/v1/dashboard/layouts
POST   /api/v1/dashboard/layouts
GET    /api/v1/dashboard/layouts/{id}
PUT    /api/v1/dashboard/layouts/{id}
DELETE /api/v1/dashboard/layouts/{id}
GET    /api/v1/ws
```

## Development

```bash
make dev             # go run (foreground, auto-recompile not included)
make run             # build + run foreground
go vet ./...         # static analysis
go test ./...        # run tests
```

### Adding a Collector

1. Implement the `Collector` interface in `internal/collector/`
2. Register in `registerAllCollectors()` in `cmd/only1mon/main.go`
3. Add metric descriptions in `internal/collector/descriptions.go`

## License

MIT
