package app

import (
	"context"
	"log/slog"
	"sync"
	"time"
)

type Scheduler struct {
	store    *Store
	logger   *slog.Logger
	done     chan struct{}
	wg       sync.WaitGroup
	stopOnce sync.Once
}

func NewScheduler(store *Store, logger *slog.Logger) *Scheduler {
	return &Scheduler{
		store:  store,
		logger: logger,
		done:   make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	s.logger.Info("scheduler started", "next_eligibility_run_at", nextMidnight(time.Now()).Format(time.RFC3339), "broadcast_interval", s.interval().String())

	s.wg.Add(2)
	go s.runEligibilityLoop()
	go s.runBroadcastLoop()
}

func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		close(s.done)
	})
	s.wg.Wait()
}

func (s *Scheduler) runEligibilityLoop() {
	defer s.wg.Done()

	for {
		timer := time.NewTimer(time.Until(nextMidnight(time.Now())))
		select {
		case <-timer.C:
			s.runEligibilityRefresh()
		case <-s.done:
			timer.Stop()
			s.logger.Info("scheduler stopped", "loop", "eligibility")
			return
		}
	}
}

func (s *Scheduler) runBroadcastLoop() {
	defer s.wg.Done()

	s.runBroadcastExpiration()

	ticker := time.NewTicker(s.interval())
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			s.runBroadcastExpiration()
		case <-s.done:
			s.logger.Info("scheduler stopped", "loop", "broadcasts")
			return
		}
	}
}

func (s *Scheduler) runEligibilityRefresh() {
	startedAt := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	updated, err := s.store.RefreshEligibility(ctx)
	duration := time.Since(startedAt)
	if err != nil {
		s.logger.Error("refresh donor eligibility", "error", err, "duration", duration.String())
		return
	}

	s.logger.Info("refreshed donor eligibility", "donors_updated", updated, "duration", duration.String())
}

func (s *Scheduler) runBroadcastExpiration() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	broadcastsExpired, err := s.store.ExpireBroadcasts(ctx)
	if err != nil {
		s.logger.Error("expire emergency broadcasts", "error", err)
		return
	}
	s.logger.Info("expired emergency broadcasts", "broadcasts_expired", broadcastsExpired)
}

func (s *Scheduler) interval() time.Duration {
	if s.store != nil && s.store.cfg.SchedulerInterval > 0 {
		return s.store.cfg.SchedulerInterval
	}
	return time.Hour
}

func nextMidnight(now time.Time) time.Time {
	next := now.AddDate(0, 0, 1)
	return time.Date(next.Year(), next.Month(), next.Day(), 0, 0, 0, 0, now.Location())
}
