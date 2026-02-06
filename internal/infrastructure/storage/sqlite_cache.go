package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"misskeyRSSbot/internal/domain/repository"

	_ "modernc.org/sqlite"
)

type sqliteCache struct {
	db *sql.DB
}

func NewSQLiteCacheRepository(dbPath string) (repository.CacheRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(time.Hour)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to connect to sqlite database: %w", err)
	}

	cache := &sqliteCache{db: db}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := cache.initSchema(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return cache, nil
}

func (c *sqliteCache) initSchema(ctx context.Context) error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS latest_published (
			rss_url TEXT PRIMARY KEY,
			published_at INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS processed_guids (
			guid TEXT PRIMARY KEY,
			processed_at INTEGER NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_processed_guids_processed_at ON processed_guids(processed_at)`,
	}

	for _, query := range queries {
		if _, err := c.db.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute schema query: %w", err)
		}
	}

	return nil
}

func (c *sqliteCache) GetLatestPublishedTime(ctx context.Context, rssURL string) (time.Time, error) {
	var unixTime int64
	err := c.db.QueryRowContext(
		ctx,
		"SELECT published_at FROM latest_published WHERE rss_url = ?",
		rssURL,
	).Scan(&unixTime)

	if errors.Is(err, sql.ErrNoRows) {
		return time.Time{}, nil
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("failed to get latest published time: %w", err)
	}

	return time.Unix(unixTime, 0), nil
}

func (c *sqliteCache) SaveLatestPublishedTime(ctx context.Context, rssURL string, published time.Time) error {
	_, err := c.db.ExecContext(
		ctx,
		`INSERT INTO latest_published (rss_url, published_at) VALUES (?, ?)
		ON CONFLICT(rss_url) DO UPDATE SET published_at = excluded.published_at`,
		rssURL,
		published.Unix(),
	)
	if err != nil {
		return fmt.Errorf("failed to save latest published time: %w", err)
	}

	return nil
}

func (c *sqliteCache) IsProcessed(ctx context.Context, guid string) (bool, error) {
	var exists int
	err := c.db.QueryRowContext(
		ctx,
		"SELECT 1 FROM processed_guids WHERE guid = ?",
		guid,
	).Scan(&exists)

	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("failed to check if processed: %w", err)
	}

	return true, nil
}

func (c *sqliteCache) MarkAsProcessed(ctx context.Context, guid string) error {
	_, err := c.db.ExecContext(
		ctx,
		`INSERT INTO processed_guids (guid, processed_at) VALUES (?, ?)
		ON CONFLICT(guid) DO NOTHING`,
		guid,
		time.Now().Unix(),
	)
	if err != nil {
		return fmt.Errorf("failed to mark as processed: %w", err)
	}

	return nil
}

func (c *sqliteCache) Close() error {
	return c.db.Close()
}

func (c *sqliteCache) CleanupOldGUIDs(ctx context.Context, olderThan time.Duration) (int64, error) {
	cutoff := time.Now().Add(-olderThan).Unix()
	result, err := c.db.ExecContext(
		ctx,
		"DELETE FROM processed_guids WHERE processed_at < ?",
		cutoff,
	)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup old GUIDs: %w", err)
	}

	deleted, err := result.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return deleted, nil
}
