package timepad

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"hackatonApp/internal/domain"
	"hackatonApp/internal/source"
)

const baseURL = "https://api.timepad.ru/v1/events.json"

type Coordinate float64

func (c *Coordinate) UnmarshalJSON(data []byte) error {
	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		*c = Coordinate(f)
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}

	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return err
	}

	*c = Coordinate(f)
	return nil
}

type CategoryList []category

func (cl *CategoryList) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	var list []category
	if err := json.Unmarshal(data, &list); err == nil {
		*cl = CategoryList(list)
		return nil
	}

	var single category
	if err := json.Unmarshal(data, &single); err != nil {
		return fmt.Errorf("failed to unmarshal categories as array or object: %w", err)
	}

	*cl = CategoryList{single}
	return nil
}

type FlexibleString string

func (fs *FlexibleString) UnmarshalJSON(data []byte) error {
	if len(data) == 0 {
		return nil
	}

	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		*fs = FlexibleString(s)
		return nil
	}

	var n float64
	if err := json.Unmarshal(data, &n); err == nil {
		*fs = FlexibleString(strconv.FormatFloat(n, 'f', -1, 64))
		return nil
	}

	return fmt.Errorf("failed to unmarshal flexible string")
}

type Client struct {
	apiToken   string
	httpClient *http.Client
}

func NewClient(apiToken string) *Client {
	return &Client{
		apiToken:   apiToken,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Name() string {
	return "timepad"
}

// --- Сырая структура ответа API ---

type eventsResponse struct {
	Total  int             `json:"total"`
	Values []eventResponse `json:"values"`
}

type eventResponse struct {
	ID               int64             `json:"id"`
	Name             string            `json:"name"`
	DescriptionShort string            `json:"description_short"`
	StartsAt         string            `json:"starts_at"`
	EndsAt           string            `json:"ends_at"`
	URL              string            `json:"url"`
	AgeLimit         FlexibleString    `json:"age_limit"`
	PosterImage      *posterImage      `json:"poster_image"`
	Location         *location         `json:"location"`
	Categories       CategoryList      `json:"categories"`
	RegistrationData *registrationData `json:"registration_data"`
}

type posterImage struct {
	DefaultURL string `json:"default_url"`
}

type location struct {
	City        string       `json:"city"`
	Address     string       `json:"address"`
	Coordinates []Coordinate `json:"coordinates"`
}

type category struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type registrationData struct {
	PriceMin float64 `json:"price_min"`
	PriceMax float64 `json:"price_max"`
}

func (c *Client) FetchEvents(ctx context.Context, p source.FetchParams) ([]domain.Event, error) {
	const pageSize = 100
	const hardSafetyCap = 2000

	var all []domain.Event
	skip := 0
	total := -1

	for page := 0; ; page++ {
		if total >= 0 && skip >= total {
			break
		}
		if page >= hardSafetyCap {
			return nil, fmt.Errorf("timepad: hit hard safety cap (%d pages), total=%d, skip=%d - something wrong with API response", hardSafetyCap, total, skip)
		}

		if page%20 == 0 {
			slog.Info("timepad: fetched page", "page", page, "skip", skip, "total", total)
		}

		q := url.Values{}
		q.Set("fields", "location,registration_data,description_short,ends_at,age_limit")
		q.Set("sort", "+starts_at")
		q.Set("limit", strconv.Itoa(pageSize))
		q.Set("skip", strconv.Itoa(skip))

		if len(p.Cities) > 0 {
			q.Set("cities", strings.Join(p.Cities, ","))
		}

		if !p.StartsAtMin.IsZero() {
			q.Set("starts_at_min", p.StartsAtMin.Format("2006-01-02"))
		}

		if !p.StartsAtMax.IsZero() {
			q.Set("starts_at_max", p.StartsAtMax.Format("2006-01-02"))
		}

		reqURL := baseURL + "?" + q.Encode()

		req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
		if err != nil {
			return nil, fmt.Errorf("timepad: build request: %w", err)
		}

		req.Header.Set("Authorization", "Bearer "+c.apiToken)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("timepad: do request: %w", err)
		}

		if resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()
			time.Sleep(60 * time.Second)
			page--
			continue
		}

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			return nil, fmt.Errorf("timepad: unexpected status %d: %s", resp.StatusCode, body)
		}

		var parsed eventsResponse
		err = json.NewDecoder(resp.Body).Decode(&parsed)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("timepad: decode response: %w", err)
		}

		total = parsed.Total
		for _, raw := range parsed.Values {
			all = append(all, mapEvent(raw))
		}

		if len(parsed.Values) == 0 {
			break
		}

		skip += pageSize

		time.Sleep(1100 * time.Millisecond)
	}

	return all, nil
}

func mapEvent(raw eventResponse) domain.Event {
	ev := domain.Event{
		ExternalID:  strconv.FormatInt(raw.ID, 10),
		Source:      domain.SourceTimepad,
		Title:       raw.Name,
		Description: raw.DescriptionShort,
		URL:         raw.URL,
		StartAt:     parseTimepadTime(raw.StartsAt),
		EndAt:       parseTimepadTime(raw.EndsAt),
	}

	if raw.PosterImage != nil {
		ev.ImageURL = raw.PosterImage.DefaultURL
	}
	if raw.Location != nil {
		ev.City = raw.Location.City
		ev.Address = raw.Location.Address
		if len(raw.Location.Coordinates) == 2 {
			ev.Lat = float64(raw.Location.Coordinates[0])
			ev.Lng = float64(raw.Location.Coordinates[1])
		}
	}
	if len(raw.Categories) > 0 {
		ev.Category = raw.Categories[0].Name
	}
	if raw.RegistrationData != nil {
		ev.PriceMin = raw.RegistrationData.PriceMin
		ev.PriceMax = raw.RegistrationData.PriceMax
		ev.IsFree = raw.RegistrationData.PriceMin == 0 && raw.RegistrationData.PriceMax == 0
	}

	return ev
}

func parseTimepadTime(s string) time.Time {
	if s == "" {
		return time.Time{}
	}

	layouts := []string{
		time.RFC3339,
		"2006-01-02T15:04:05-0700",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05Z0700",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, l := range layouts {
		if t, err := time.Parse(l, s); err == nil {
			return t
		}
	}

	return time.Time{}
}
