package collector

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/only1mon/only1mon/internal/model"
	"github.com/only1mon/only1mon/internal/store"
)

// BroadcastFunc is called with collected samples for real-time streaming.
type BroadcastFunc func(samples []model.MetricSample)

// AlertBroadcastFunc is called with alerts generated from metric analysis.
type AlertBroadcastFunc func(alerts []model.Alert)

// Scheduler runs enabled collectors at a fixed interval.
type Scheduler struct {
	registry       *Registry
	store          *store.Store
	interval       time.Duration
	broadcast      BroadcastFunc
	alertBroadcast AlertBroadcastFunc
	alertEngine    *AlertEngine
	mu             sync.Mutex
	cancel         context.CancelFunc
	intervalCh     chan time.Duration // signals the loop to reset the ticker
}

// NewScheduler creates a new scheduler.
func NewScheduler(registry *Registry, s *store.Store, intervalSec int) *Scheduler {
	return &Scheduler{
		registry:    registry,
		store:       s,
		interval:    time.Duration(intervalSec) * time.Second,
		alertEngine: NewAlertEngine(),
		intervalCh:  make(chan time.Duration, 1),
	}
}

// SetBroadcast sets the function called with each batch of samples.
func (s *Scheduler) SetBroadcast(fn BroadcastFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.broadcast = fn
}

// SetAlertBroadcast sets the function called with alerts after each collection.
func (s *Scheduler) SetAlertBroadcast(fn AlertBroadcastFunc) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.alertBroadcast = fn
}

// AlertEngine returns the scheduler's alert engine for API access.
func (s *Scheduler) AlertEngine() *AlertEngine {
	return s.alertEngine
}

// Start begins the collection loop.
func (s *Scheduler) Start(ctx context.Context) {
	ctx, cancel := context.WithCancel(ctx)
	s.mu.Lock()
	s.cancel = cancel
	s.mu.Unlock()

	go s.loop(ctx)
}

// Stop halts the scheduler.
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.cancel != nil {
		s.cancel()
	}
}

// UpdateTopN updates the top-N process count on the process collector.
func (s *Scheduler) UpdateTopN(n int) {
	for _, c := range s.registry.EnabledCollectors() {
		if pc, ok := c.(*processCollector); ok {
			pc.SetTopN(n)
			log.Printf("[scheduler] top_process_count updated to %d", n)
			return
		}
	}
	// Also check all registered collectors (even disabled ones)
	s.registry.mu.RLock()
	defer s.registry.mu.RUnlock()
	for _, c := range s.registry.collectors {
		if pc, ok := c.(*processCollector); ok {
			pc.SetTopN(n)
			log.Printf("[scheduler] top_process_count updated to %d", n)
			return
		}
	}
}

// UpdateInterval changes the collection interval at runtime.
func (s *Scheduler) UpdateInterval(sec int) {
	d := time.Duration(sec) * time.Second
	if d < 1*time.Second {
		d = 1 * time.Second
	}
	s.mu.Lock()
	s.interval = d
	s.mu.Unlock()

	// Non-blocking send to notify the loop
	select {
	case s.intervalCh <- d:
	default:
	}
	log.Printf("[scheduler] interval updated to %v", d)
}

func (s *Scheduler) loop(ctx context.Context) {
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	// Run once immediately
	s.collectAll(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case newInterval := <-s.intervalCh:
			ticker.Reset(newInterval)
		case <-ticker.C:
			s.collectAll(ctx)
		}
	}
}

func (s *Scheduler) collectAll(ctx context.Context) {
	collectors := s.registry.EnabledCollectors()
	if len(collectors) == 0 {
		return
	}

	var allSamples []model.MetricSample

	for _, c := range collectors {
		samples, err := c.Collect(ctx)
		if err != nil {
			log.Printf("[scheduler] collector %s error: %v", c.ID(), err)
			continue
		}
		allSamples = append(allSamples, samples...)
	}

	if len(allSamples) == 0 {
		return
	}

	// Filter out disabled metrics
	filtered := allSamples[:0]
	for _, sample := range allSamples {
		if s.registry.IsMetricEnabled(sample.MetricName) {
			filtered = append(filtered, sample)
		}
	}
	allSamples = filtered

	if len(allSamples) == 0 {
		return
	}

	// Store in DB
	if err := s.store.InsertSamples(allSamples); err != nil {
		log.Printf("[scheduler] store error: %v", err)
	}

	// Broadcast to WebSocket clients
	s.mu.Lock()
	fn := s.broadcast
	alertFn := s.alertBroadcast
	s.mu.Unlock()
	if fn != nil {
		fn(allSamples)
	}

	// Evaluate alert rules
	alerts := s.alertEngine.Evaluate(allSamples)
	if len(alerts) > 0 && alertFn != nil {
		alertFn(alerts)
	}
}
