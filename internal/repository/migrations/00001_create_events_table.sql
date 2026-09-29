-- +goose UP
-- +goose StatementBegin
CREATE TABLE events (
    id BIGSERIAL PRIMARY KEY,
    external_id TEXT NOT NULL,
    source TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT,
    url TEXT,
    image_url TEXT,
    city TEXT,
    address TEXT, 
    lat DOUBLE PRECISION,
    lng DOUBLE PRECISION,
    starts_at TIMESTAMPTZ,
    ends_at TIMESTAMPTZ,
    price_min NUMERIC,
    price_max NUMERIC,
    is_free BOOLEAN NOT NULL DEFAULT false,
    is_pushkin_card BOOLEAN NOT NULL DEFAULT false,
    category TEXT, 
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(source, external_id)
);

CREATE INDEX idx_events_city_starts ON events(city, starts_at);
-- +goose StatementEnd

-- +goose Down

-- +goose StatementBegin
DROP TABLE IF EXISTS events;
-- +goose StatementEnd