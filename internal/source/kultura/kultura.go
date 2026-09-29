package kultura

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"hackatonApp/internal/domain"
	"hackatonApp/internal/source"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c *Client) Name() string { return "kultura" }

func (c *Client) FetchEvents(ctx context.Context, p source.FetchParams) ([]domain.Event, error) {
	if c.apiKey == "" {
		return c.mockEvents(p), nil
	}
	// TODO: подключить реальный запрос, когда придёт ответ от partners@team.culture.ru
	return nil, fmt.Errorf("kultura: real API not wired yet, using apiKey=%s", c.apiKey)
}

func (c *Client) mockEvents(p source.FetchParams) []domain.Event {
	all := []domain.Event{
		{
			ExternalID:    "mock-1",
			Source:        domain.SourceKultura,
			Title:         "Экскурсия по Пушкинскому музею",
			City:          "Джалиль",
			IsFree:        false,
			IsPushkinCard: true,
			StartAt:       time.Now().Add(24 * time.Hour),
			Category:      "Музеи",
		},
		{
			ExternalID: "mock-2",
			Source:     domain.SourceKultura,
			Title:      "Бесплатный мастер-класс по керамике",
			City:       "Джалиль",
			IsFree:     true,
			StartAt:    time.Now().Add(48 * time.Hour),
			Category:   "Мастер-классы",
		},
	}

	if len(p.Cities) == 0 {
		return all
	}

	allowed := make(map[string]bool, len(p.Cities))
	for _, c := range p.Cities {
		allowed[c] = true
	}
	filtered := make([]domain.Event, 0, len(all))
	for _, ev := range all {
		if allowed[ev.City] {
			filtered = append(filtered, ev)
		}
	}
	return filtered
}
