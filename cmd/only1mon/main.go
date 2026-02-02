package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/playok/only1mon/internal/api"
	"github.com/playok/only1mon/internal/collector"
	"github.com/playok/only1mon/internal/config"
	"github.com/playok/only1mon/internal/model"
	"github.com/playok/only1mon/internal/store"
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

// buildForwardFlags generates flags to forward the loaded config to the child.
func buildForwardFlags(cfg *config.Config) []string {
	var args []string
	args = append(args, "-config", cfg.ConfigPath)
	return args
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
	ctx, stop := signal.NotifyContext(context.Background(), shutdownSignals...)
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
