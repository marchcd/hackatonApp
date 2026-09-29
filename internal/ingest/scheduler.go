package ingest

import (
	"context"
	"log/slog"
	"sync"
	"time"

	"hackatonApp/internal/source"
	"hackatonApp/internal/storage"
)

type Scheduler struct {
	sources  []source.EventSource
	repo     storage.EventRepository
	cities   []string
	interval time.Duration
}

func NewScheduler(sources []source.EventSource, repo storage.EventRepository, cities []string, interval time.Duration) *Scheduler {
	return &Scheduler{sources: sources, repo: repo, cities: cities, interval: interval}
}

func (s *Scheduler) Run(ctx context.Context) {
	s.fetchAll(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.fetchAll(ctx)
		}
	}
}

func (s *Scheduler) fetchAll(ctx context.Context) {
	var wg sync.WaitGroup
	for _, src := range s.sources {
		wg.Add(1)
		go func(src source.EventSource) {
			defer wg.Done()

			events, err := src.FetchEvents(ctx, source.FetchParams{
				Cities:      s.cities,
				StartsAtMin: time.Now(),
				StartsAtMax: time.Now().AddDate(0, 0, 90),
			})
			if err != nil {
				slog.Error("ingest: %s fetch error: %v", src.Name(), err)
				return
			}
			if err := s.repo.UpsertEvents(ctx, events); err != nil {
				slog.Error("ingest: %s upsert error: %v", src.Name(), err)
			}
		}(src)
	}
	wg.Wait()
}
