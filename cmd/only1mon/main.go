package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/only1mon/only1mon/internal/api"
	"github.com/only1mon/only1mon/internal/collector"
	"github.com/only1mon/only1mon/internal/config"
	"github.com/only1mon/only1mon/internal/model"
	"github.com/only1mon/only1mon/internal/store"
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]

	// Handle -nginx / --nginx anywhere
	if cmd == "-nginx" || cmd == "--nginx" {
		// Strip the -nginx arg, keep remaining flags for config loading
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		cmdNginx()
		return
	}

	switch cmd {
	case "start":
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		cmdStart()
	case "stop":
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		cmdStop()
	case "status":
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		cmdStatus()
	case "run":
		// Foreground mode (also used internally by daemon child)
		os.Args = append([]string{os.Args[0]}, os.Args[2:]...)
		cmdRun(false)
	case "version":
		fmt.Printf("only1mon %s\n", version)
	default:
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	exe := filepath.Base(os.Args[0])
	fmt.Fprintf(os.Stderr, `Only1Mon — Lightweight System Monitoring Dashboard (%s)

Usage:
  %s <command> [flags]

Commands:
  start          Start daemon (background)
  stop           Stop daemon
  status         Show daemon status
  run            Run in foreground
  version        Print version

Flags:
  -nginx         Print sample nginx reverse proxy configuration
  -config PATH   Config file path (default: config.yaml)
  -listen ADDR   Listen address (default: 127.0.0.1:9923)
  -db PATH       SQLite database path
  -base-path P   Base URL path for reverse proxy
  -pid-file P    PID file path
  -log-file P    Log file path

Examples:
  %s start
  %s start -config /etc/only1mon/config.yaml
  %s stop
  %s status
  %s run
  %s -nginx
`, version, exe, exe, exe, exe, exe, exe, exe)
}

// ---------------------------------------------------------------------------
// start: daemonize by re-exec with "run" subcommand
// ---------------------------------------------------------------------------

func cmdStart() {
	cfg := config.Load()

	// Check if already running
	if pid, err := readPidFile(cfg.PidFile); err == nil {
		if processExists(pid) {
			fmt.Printf("only1mon is already running (PID %d)\n", pid)
			os.Exit(1)
		}
		// Stale PID file
		os.Remove(cfg.PidFile)
	}

	// Build args: replace "start" with "run" for the child
	childArgs := []string{"run"}
	childArgs = append(childArgs, buildForwardFlags(cfg)...)

	exe, err := os.Executable()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to find executable: %v\n", err)
		os.Exit(1)
	}

	// Open log file
	logFile, err := os.OpenFile(cfg.LogFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to open log file %s: %v\n", cfg.LogFile, err)
		os.Exit(1)
	}

	child := &exec.Cmd{
		Path:   exe,
		Args:   append([]string{filepath.Base(exe)}, childArgs...),
		Stdout: logFile,
		Stderr: logFile,
		SysProcAttr: &syscall.SysProcAttr{
			Setsid: true, // detach from terminal
		},
	}

	if err := child.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "failed to start daemon: %v\n", err)
		os.Exit(1)
	}

	pid := child.Process.Pid
	if err := writePidFile(cfg.PidFile, pid); err != nil {
		fmt.Fprintf(os.Stderr, "warning: failed to write PID file: %v\n", err)
	}

	// Release the child — parent exits
	child.Process.Release()
	logFile.Close()

	fmt.Printf("only1mon started (PID %d)\n", pid)
	fmt.Printf("  Listen : http://%s\n", cfg.Listen)
	fmt.Printf("  Config : %s\n", cfg.ConfigPath)
	fmt.Printf("  PID    : %s\n", cfg.PidFile)
	fmt.Printf("  Log    : %s\n", cfg.LogFile)
}

// buildForwardFlags generates flags to forward the loaded config to the child.
func buildForwardFlags(cfg *config.Config) []string {
	var args []string
	args = append(args, "-config", cfg.ConfigPath)
	return args
}

// ---------------------------------------------------------------------------
// stop: read PID file and send SIGTERM
// ---------------------------------------------------------------------------

func cmdStop() {
	cfg := config.Load()

	pid, err := readPidFile(cfg.PidFile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "only1mon is not running (no PID file: %s)\n", cfg.PidFile)
		os.Exit(1)
	}

	if !processExists(pid) {
		fmt.Printf("only1mon is not running (stale PID %d)\n", pid)
		os.Remove(cfg.PidFile)
		os.Exit(1)
	}

	proc, err := os.FindProcess(pid)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to find process %d: %v\n", pid, err)
		os.Exit(1)
	}

	if err := proc.Signal(syscall.SIGTERM); err != nil {
		fmt.Fprintf(os.Stderr, "failed to stop PID %d: %v\n", pid, err)
		os.Exit(1)
	}

	// Wait for process to exit (up to 10 seconds)
	for i := 0; i < 100; i++ {
		time.Sleep(100 * time.Millisecond)
		if !processExists(pid) {
			os.Remove(cfg.PidFile)
			fmt.Printf("only1mon stopped (PID %d)\n", pid)
			return
		}
	}

	fmt.Printf("only1mon stop signal sent (PID %d), waiting for exit...\n", pid)
	os.Remove(cfg.PidFile)
}

// ---------------------------------------------------------------------------
// status: check PID file
// ---------------------------------------------------------------------------

func cmdStatus() {
	cfg := config.Load()

	pid, err := readPidFile(cfg.PidFile)
	if err != nil {
		fmt.Println("only1mon is stopped")
		os.Exit(1)
	}

	if processExists(pid) {
		fmt.Printf("only1mon is running (PID %d)\n", pid)
		fmt.Printf("  Listen : http://%s\n", cfg.Listen)
		fmt.Printf("  Config : %s\n", cfg.ConfigPath)
		fmt.Printf("  PID    : %s\n", cfg.PidFile)
		fmt.Printf("  Log    : %s\n", cfg.LogFile)
	} else {
		fmt.Printf("only1mon is stopped (stale PID file, was PID %d)\n", pid)
		os.Remove(cfg.PidFile)
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------
// -nginx: print sample nginx config
// ---------------------------------------------------------------------------

func cmdNginx() {
	cfg := config.Load()

	bp := cfg.BasePath
	if bp == "/" {
		bp = "/mon"
		fmt.Println("# base_path is \"/\" — using \"/mon\" as example.")
		fmt.Println("# Set base_path in config.yaml to match your desired location.")
		fmt.Println()
	}

	// Ensure trailing slash for nginx location
	loc := bp + "/"

	fmt.Printf(`# --------------------------------------------------
# nginx reverse proxy configuration for Only1Mon
# --------------------------------------------------
# Add this inside an http { server { ... } } block.

location %s {
    proxy_pass         http://%s/;
    proxy_http_version 1.1;

    # WebSocket support
    proxy_set_header   Upgrade $http_upgrade;
    proxy_set_header   Connection "upgrade";

    # Forward client info
    proxy_set_header   Host              $host;
    proxy_set_header   X-Real-IP         $remote_addr;
    proxy_set_header   X-Forwarded-For   $proxy_add_x_forwarded_for;
    proxy_set_header   X-Forwarded-Proto $scheme;

    # Disable buffering for real-time WebSocket
    proxy_buffering    off;
    proxy_read_timeout 86400s;
}
`, loc, cfg.Listen)

	fmt.Println("# config.yaml should have:")
	fmt.Printf("#   base_path: \"%s\"\n", bp)
}

// ---------------------------------------------------------------------------
// run: foreground server (also used by daemon child)
// ---------------------------------------------------------------------------

func cmdRun(isDaemon bool) {
	cfg := config.Load()

	// In daemon mode, write our own PID (child process)
	if isDaemon {
		writePidFile(cfg.PidFile, os.Getpid())
	}

	// Open store
	db, err := store.New(cfg.DBPath)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	defer db.Close()

	// Create collector registry and restore state
	registry := collector.NewRegistry(db)
	registerAllCollectors(registry)
	if err := registry.RestoreState(); err != nil {
		log.Printf("warning: failed to restore collector state: %v", err)
	}
	if err := registry.RestoreMetricStates(); err != nil {
		log.Printf("warning: failed to restore metric states: %v", err)
	}

	// On first run, enable basic collectors and create default dashboard
	if !registry.HasAnyState() {
		bootstrapDefaults(registry, db)
	}

	// Seed default alert rules if none exist
	seedDefaultAlertRules(db)

	// Apply DB-persisted settings (override config defaults)
	applyDBSettings(db, cfg)

	// Discover actual metric names by running each collector once
	log.Println("[startup] discovering available metrics...")
	registry.DiscoverMetrics(context.Background())

	// Apply top_process_count from DB to process collector
	applyTopProcessCount(db, registry)

	// Create scheduler
	sched := collector.NewScheduler(registry, db, cfg.CollectInterval)

	// Load alert rules from DB
	sched.AlertEngine().LoadRules(db)

	// Create WebSocket hub
	hub := api.NewHub()
	go hub.Run()

	// Wire scheduler broadcast to hub
	sched.SetBroadcast(func(samples []model.MetricSample) {
		hub.Broadcast(samples)
	})
	sched.SetAlertBroadcast(func(alerts []model.Alert) {
		hub.BroadcastAlerts(alerts)
	})

	// Start scheduler
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	sched.Start(ctx)

	// Start retention purge goroutine
	go runRetentionPurge(ctx, db, cfg.RetentionHours)

	// Build HTTP router
	router := api.NewRouter(registry, db, hub, sched.AlertEngine(), sched, cfg.BasePath)

	srv := &http.Server{
		Addr:    cfg.Listen,
		Handler: router,
	}

	// Start server
	go func() {
		log.Printf("Only1Mon %s listening on http://%s (base_path: %s)", version, cfg.Listen, cfg.BasePath)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for signal
	<-ctx.Done()
	log.Println("shutting down...")

	shutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sched.Stop()
	srv.Shutdown(shutCtx)

	// Clean up PID file
	os.Remove(cfg.PidFile)
	log.Println("goodbye")
}

// ---------------------------------------------------------------------------
// PID file helpers
// ---------------------------------------------------------------------------

func writePidFile(path string, pid int) error {
	return os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0644)
}

func readPidFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil || pid <= 0 {
		return 0, fmt.Errorf("invalid PID in %s", path)
	}
	return pid, nil
}

func processExists(pid int) bool {
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	// Signal 0 checks existence without actually sending a signal
	err = proc.Signal(syscall.Signal(0))
	return err == nil
}

// ---------------------------------------------------------------------------
// Bootstrap & helpers (unchanged)
// ---------------------------------------------------------------------------

func runRetentionPurge(ctx context.Context, db *store.Store, hours int) {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			n, err := db.PurgeOlderThan(hours)
			if err != nil {
				log.Printf("[purge] error: %v", err)
			} else if n > 0 {
				log.Printf("[purge] removed %d old samples", n)
			}
		}
	}
}

func registerAllCollectors(registry *collector.Registry) {
	registry.Register(collector.NewCPUCollector())
	registry.Register(collector.NewMemoryCollector())
	registry.Register(collector.NewDiskCollector())
	registry.Register(collector.NewNetworkCollector())
	registry.Register(collector.NewProcessCollector())
	registry.Register(collector.NewKernelCollector())
	registry.Register(collector.NewGPUCollector())
}

func seedDefaultAlertRules(db *store.Store) {
	rules, err := db.ListAlertRules()
	if err != nil {
		log.Printf("[bootstrap] failed to list alert rules: %v", err)
		return
	}
	if len(rules) > 0 {
		return
	}
	defaults := collector.DefaultAlertRuleModels()
	for i := range defaults {
		if _, err := db.CreateAlertRule(&defaults[i]); err != nil {
			log.Printf("[bootstrap] failed to seed alert rule: %v", err)
		}
	}
	log.Printf("[bootstrap] seeded %d default alert rules", len(defaults))
}

func bootstrapDefaults(registry *collector.Registry, db *store.Store) {
	log.Println("[bootstrap] first run detected, enabling default collectors")

	defaults := []string{"cpu", "memory", "disk"}
	for _, id := range defaults {
		if err := registry.Enable(id); err != nil {
			log.Printf("[bootstrap] failed to enable %s: %v", id, err)
		}
	}

	type widgetDef struct {
		ID      string
		Title   string
		Metrics []string
		X, Y    int
		W, H    int
	}

	widgets := []widgetDef{
		{ID: "w-cpu-usage", Title: "CPU Usage (%)", Metrics: []string{"cpu.total.user", "cpu.total.system", "cpu.total.iowait"}, X: 0, Y: 0, W: 6, H: 3},
		{ID: "w-cpu-load", Title: "Load Average", Metrics: []string{"cpu.load.1", "cpu.load.5", "cpu.load.15"}, X: 6, Y: 0, W: 6, H: 3},
		{ID: "w-mem-usage", Title: "Memory Usage", Metrics: []string{"mem.used", "mem.available", "mem.cached"}, X: 0, Y: 3, W: 6, H: 3},
		{ID: "w-mem-pct", Title: "Memory %", Metrics: []string{"mem.used_pct"}, X: 6, Y: 3, W: 6, H: 3},
		{ID: "w-swap", Title: "Swap Usage", Metrics: []string{"mem.swap.used", "mem.swap.free"}, X: 0, Y: 6, W: 6, H: 3},
		{ID: "w-disk-pct", Title: "Disk Usage %", Metrics: []string{"disk.root.used_pct"}, X: 6, Y: 6, W: 6, H: 3},
	}

	gridItems := make([]map[string]interface{}, len(widgets))
	widgetMeta := make(map[string]map[string]interface{})

	for i, w := range widgets {
		gridItems[i] = map[string]interface{}{
			"id": w.ID, "x": w.X, "y": w.Y, "w": w.W, "h": w.H,
		}
		widgetMeta[w.ID] = map[string]interface{}{
			"title":   w.Title,
			"metrics": w.Metrics,
		}
	}

	layoutData, _ := json.Marshal(map[string]interface{}{
		"grid":    gridItems,
		"widgets": widgetMeta,
	})

	dl := &model.DashboardLayout{
		Name:   "Default",
		Layout: string(layoutData),
	}
	if _, err := db.CreateDashboardLayout(dl); err != nil {
		log.Printf("[bootstrap] failed to create default layout: %v", err)
	} else {
		log.Println("[bootstrap] created default dashboard layout")
	}
}

func applyTopProcessCount(db *store.Store, registry *collector.Registry) {
	v, err := db.GetSetting("top_process_count")
	if err != nil || v == "" {
		return
	}
	n, err := strconv.Atoi(v)
	if err != nil || n < 1 {
		return
	}
	registry.SetProcessTopN(n)
	log.Printf("[settings] top_process_count from DB: %d", n)
}

func applyDBSettings(db *store.Store, cfg *config.Config) {
	if v, err := db.GetSetting("collect_interval"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.CollectInterval = n
			log.Printf("[settings] collect_interval from DB: %ds", n)
		}
	}
	if v, err := db.GetSetting("retention_hours"); err == nil && v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.RetentionHours = n
			log.Printf("[settings] retention_hours from DB: %dh", n)
		}
	}
}
