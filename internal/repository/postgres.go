package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"hackatonApp/internal/domain"
	"hackatonApp/internal/storage"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("unable to parse DSN: %w", err)
	}

	config.MaxConns = 10
	config.MinConns = 2
	config.MaxConnIdleTime = time.Minute * 3

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("unable to create connection pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		return nil, fmt.Errorf("unable to ping database: %w", err)
	}
	return pool, nil
}

type PostgresRepo struct {
	pool *pgxpool.Pool
}

func NewPostgresRepo(pool *pgxpool.Pool) *PostgresRepo {
	return &PostgresRepo{pool: pool}
}

func (r *PostgresRepo) UpsertEvents(ctx context.Context, events []domain.Event) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("postgres: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO events (
			source, external_id, title, description, url, image_url,
			city, address, lat, lng, starts_at, ends_at,
			price_min, price_max, is_free, is_pushkin_card, category, fetched_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17, now())
		ON CONFLICT (source, external_id) DO UPDATE SET
			title = EXCLUDED.title, description = EXCLUDED.description,
			url = EXCLUDED.url, image_url = EXCLUDED.image_url,
			city = EXCLUDED.city, address = EXCLUDED.address,
			lat = EXCLUDED.lat, lng = EXCLUDED.lng,
			starts_at = EXCLUDED.starts_at, ends_at = EXCLUDED.ends_at,
			price_min = EXCLUDED.price_min, price_max = EXCLUDED.price_max,
			is_free = EXCLUDED.is_free, is_pushkin_card = EXCLUDED.is_pushkin_card,
			category = EXCLUDED.category, fetched_at = now()
	`

	for _, ev := range events {
		if _, err := tx.Exec(ctx, query, ev.Source, ev.ExternalID, ev.Title, ev.Description, ev.URL, ev.ImageURL,
			ev.City, ev.Address, ev.Lat, ev.Lng, ev.StartAt, ev.EndAt, ev.PriceMin, ev.PriceMax,
			ev.IsFree, ev.IsPushkinCard, ev.Category,
		); err != nil {
			return fmt.Errorf("postgres: upsert %s/%s: %w", ev.Source, ev.ExternalID, err)
		}
	}

	return tx.Commit(ctx)
}

func (r *PostgresRepo) SearchEvents(ctx context.Context, f storage.EventFilter) ([]domain.Event, error) {
	where := []string{"starts_at >= now()"} // прошедшие события не показываем
	// where := []string{"1=1"}
	args := []interface{}{}
	argN := 1

	if f.City != "" {
		where = append(where, fmt.Sprintf("city ILIKE $%d", argN))
		args = append(args, f.City)
		argN++
	}
	if f.Category != "" {
		where = append(where, fmt.Sprintf("category ILIKE $%d", argN))
		args = append(args, f.Category)
		argN++
	}
	if f.OnlyFree {
		where = append(where, "is_free = true")
	}
	if f.OnlyPushkin {
		where = append(where, "is_pushkin_card = true")
	}

	limit := f.Limit
	if limit == 0 || limit > 50 {
		limit = 20
	}

	query := fmt.Sprintf(`
		SELECT source, external_id, title, description, url, image_url,
		       city, address, lat, lng, starts_at, ends_at,
		       price_min, price_max, is_free, is_pushkin_card, category
		FROM events WHERE %s
		ORDER BY starts_at ASC LIMIT %d OFFSET %d
	`, strings.Join(where, " AND "), limit, f.Offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("postgres: search: %w", err)
	}
	defer rows.Close()

	var events []domain.Event
	for rows.Next() {
		var ev domain.Event
		if err := rows.Scan(
			&ev.Source, &ev.ExternalID, &ev.Title, &ev.Description, &ev.URL, &ev.ImageURL,
			&ev.City, &ev.Address, &ev.Lat, &ev.Lng, &ev.StartAt, &ev.EndAt,
			&ev.PriceMin, &ev.PriceMax, &ev.IsFree, &ev.IsPushkinCard, &ev.Category,
		); err != nil {
			return nil, fmt.Errorf("postgres: scan: %w", err)
		}
		events = append(events, ev)
	}
	return events, rows.Err()
}

func (r *PostgresRepo) DistinctCities(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT city FROM events
		WHERE city IS NOT NULL AND city <> '' AND starts_at >= now()
		ORDER BY city
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: distinct cities: %w", err)
	}
	defer rows.Close()

	var cities []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("postgres: scan city: %w", err)
		}
		cities = append(cities, c)
	}

	return cities, rows.Err()
}

func (r *PostgresRepo) DistinctCategories(ctx context.Context) ([]string, error) {
	query := `
		SELECT DISTINCT category FROM events
		WHERE category IS NOT NULL AND category <> '' AND starts_at >= now()
		ORDER BY category
	`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("postgres: distinct categories: %w", err)
	}
	defer rows.Close()

	var categories []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, fmt.Errorf("postgres: scan category: %w", err)
		}
		categories = append(categories, c)
	}
	return categories, rows.Err()
}
