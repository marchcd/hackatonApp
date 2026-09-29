package service

import (
	"context"
	"fmt"

	"hackatonApp/internal/domain"
	"hackatonApp/internal/storage"
)

type EventService struct {
	repo storage.EventRepository
}

func NewEventService(r storage.EventRepository) *EventService {
	return &EventService{repo: r}
}

type SearchParams struct {
	City        string
	Category    string
	OnlyFree    bool
	OnlyPushkin bool
	Page        int
	PageSize    int
}

const (
	defaultPageSize = 20
	maxPageSize     = 50
)

func (s *EventService) Search(ctx context.Context, p SearchParams) ([]domain.Event, error) {
	pageSize := p.PageSize
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}

	page := p.Page
	if page <= 0 {
		page = 1
	}

	filter := storage.EventFilter{
		City:        p.City,
		Category:    p.Category,
		OnlyFree:    p.OnlyFree,
		OnlyPushkin: p.OnlyPushkin,
		Limit:       pageSize,
		Offset:      (page - 1) * pageSize,
	}

	events, err := s.repo.SearchEvents(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("service: search events: %w", err)
	}

	return events, nil
}

func (s *EventService) Cities(ctx context.Context) ([]string, error) {
	return s.repo.DistinctCities(ctx)
}

func (s *EventService) Categories(ctx context.Context) ([]string, error) {
	return s.repo.DistinctCategories(ctx)
}
