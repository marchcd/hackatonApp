package storage

import (
	"context"

	"hackatonApp/internal/domain"
)

type EventFilter struct {
	City        string
	Category    string
	OnlyFree    bool
	OnlyPushkin bool
	Limit       int
	Offset      int
}

type EventRepository interface {
	UpsertEvents(ctx context.Context, events []domain.Event) error
	SearchEvents(ctx context.Context, f EventFilter) ([]domain.Event, error)
	DistinctCities(ctx context.Context) ([]string, error)
	DistinctCategories(ctx context.Context) ([]string, error)
}
