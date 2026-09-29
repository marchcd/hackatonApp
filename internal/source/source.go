package source

import (
	"context"
	"time"

	"hackatonApp/internal/domain"
)

type FetchParams struct {
	Cities      []string
	StartsAtMin time.Time
	StartsAtMax time.Time
}

type EventSource interface {
	Name() string
	FetchEvents(ctx context.Context, p FetchParams) ([]domain.Event, error)
}
