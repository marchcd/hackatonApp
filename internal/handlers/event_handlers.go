package handlers

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"

	"hackatonApp/internal/api"
	"hackatonApp/internal/domain"
	"hackatonApp/internal/service"
)

type eventDTO struct {
	Title         string  `json:"title"`
	Description   string  `json:"description,omitempty"`
	URL           string  `json:"url,omitempty"`
	ImageURL      string  `json:"image_url,omitempty"`
	City          string  `json:"city,omitempty"`
	Address       string  `json:"address,omitempty"`
	Lat           float64 `json:"lat,omitempty"`
	Lng           float64 `json:"lng,omitempty"`
	StartsAt      string  `json:"starts_at,omitempty"`
	EndsAt        string  `json:"ends_at,omitempty"`
	IsFree        bool    `json:"is_free"`
	IsPushkinCard bool    `json:"is_pushkin_card"`
	Category      string  `json:"category,omitempty"`
}

type searchResponse struct {
	Events []eventDTO `json:"events"`
	Page   int        `json:"page"`
}

type EventHandler struct {
	events    *service.EventService
	botToken  string
	skipCheck bool
}

func NewEventService(events *service.EventService, botToken string, skipCheck bool) *EventHandler {
	return &EventHandler{events: events, botToken: botToken, skipCheck: skipCheck}
}

func (h *EventHandler) RegisterRoutes(mux *http.ServeMux) {
	var eventsRoute http.Handler = http.HandlerFunc(h.handleSearch)
	if !h.skipCheck {
		eventsRoute = api.ValidateMaxInitData(h.botToken)(eventsRoute)
	} else {
		slog.Warn("SKIP_INIT_DATA_CHECK=true: /api/events отдается БЕЗ проверки initData - только для локальной отладки!")
	}
	mux.Handle("GET /api/events", eventsRoute)
}

func (h *EventHandler) handleSearch(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	params := service.SearchParams{
		City:        q.Get("city"),
		Category:    q.Get("category"),
		OnlyFree:    q.Get("free") == "true",
		OnlyPushkin: q.Get("pushkin") == "true",
		Page:        parseIntOr(q.Get("page"), 1),
		PageSize:    parseIntOr(q.Get("page_size"), 0),
	}

	events, err := h.events.Search(r.Context(), params)
	if err != nil {
		http.Error(w, `{"error":"internal error"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	resp := searchResponse{Events: toDTOs(events), Page: params.Page}
	_ = json.NewEncoder(w).Encode(resp)
}

func toDTOs(events []domain.Event) []eventDTO {
	dtos := make([]eventDTO, 0, len(events))
	for _, ev := range events {
		dtos = append(dtos, eventDTO{
			Title:         ev.Title,
			Description:   ev.Description,
			URL:           ev.URL,
			ImageURL:      ev.ImageURL,
			City:          ev.City,
			Address:       ev.Address,
			Lat:           ev.Lat,
			Lng:           ev.Lng,
			StartsAt:      ev.StartAt.Format("2006-01-02T15:04:05Z07:00"),
			EndsAt:        ev.EndAt.Format("2006-01-02T15:04:05Z07:00"),
			IsFree:        ev.IsFree,
			IsPushkinCard: ev.IsPushkinCard,
			Category:      ev.Category,
		})
	}

	return dtos
}

func parseIntOr(raw string, def int) int {
	if raw == "" {
		return def
	}
	v, err := strconv.Atoi(raw)
	if err != nil {
		return def
	}
	return v
}
