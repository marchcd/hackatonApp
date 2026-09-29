package domain

import "time"

type EventSourceType string

const (
	SourceTimepad EventSourceType = "timepad"
	SourceKultura EventSourceType = "kultura"
)

type Event struct {
	ExternalID    string
	Source        EventSourceType
	Title         string
	Description   string
	URL           string
	ImageURL      string
	City          string
	Address       string
	Lat, Lng      float64
	StartAt       time.Time
	EndAt         time.Time
	PriceMin      float64
	PriceMax      float64
	IsFree        bool
	IsPushkinCard bool
	Category      string
}
