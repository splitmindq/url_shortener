package postgres

import (
	"URL-Shortener/internal/config"
	"URL-Shortener/internal/storage"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4/pgxpool"
	_ "github.com/jackc/pgx/v4/stdlib"
)

type Storage struct {
	pool *pgxpool.Pool
}

func NewStorage(conn string, cfg *config.Config) (*Storage, error) {
	const op = "storage.postgres.NewStorage"

	ctx := context.Background()

	poolConfig, err := pgxpool.ParseConfig(conn)
	if err != nil {
		return nil, fmt.Errorf("%s: parse poolConfig: %w", op, err)
	}

	poolConfig.MaxConns = cfg.MaxConnections
	poolConfig.MinConns = cfg.MinConnections
	poolConfig.MaxConnLifetime = cfg.MaxConnectionLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnectionIdleTime

	pool, err := pgxpool.ConnectConfig(ctx, poolConfig)
	if err != nil {
		fmt.Printf("%s Error: %s\n", op, err)
	}

	if err := pool.Ping(ctx); err != nil {
		fmt.Printf("%s Error: %s\n", op, err)
	}

	_, err = pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS urls (
			id SERIAL PRIMARY KEY,
			alias TEXT UNIQUE NOT NULL,
			url TEXT NOT NULL
		);
	`)
	if err != nil {
		fmt.Printf("%s Error: %s\n", op, err)
	}

	return &Storage{pool: pool}, nil
}

func (s *Storage) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *Storage) SaveURL(alias, urlToSave string) (int64, error) {
	const op = "storage.postgres.SaveURL"

	ctx := context.Background()
	query := `INSERT INTO url(alias, url) VALUES($1, $2) RETURNING id`
	var id int64
	err := s.pool.QueryRow(ctx, query, alias, urlToSave).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" { // unique_violation
			return 0, fmt.Errorf("%s: %w", op, storage.ErrAliasExists)
		}
		return 0, fmt.Errorf("%s: exec: %w", op, err)
	}

	return id, nil
}

func (s *Storage) GetUrl(alias string) (string, error) {
	const op = "storage.postgres.GetUrl"

	ctx := context.Background()
	var url string
	err := s.pool.QueryRow(ctx, `SELECT url FROM urls WHERE alias = $1`, alias).Scan(&url)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", storage.ErrUrlNotFound
		}
		return "", fmt.Errorf("%s: query: %w", op, err)
	}

	return url, nil
}

func (s *Storage) DeleteUrl(alias string) error {
	const op = "storage.postgres.DeleteUrl"

	ctx := context.Background()

	result, err := s.pool.Exec(ctx, `DELETE FROM urls  WHERE alias = $1`, alias)
	if err != nil {
		return fmt.Errorf("%s: exec: %w", op, err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return storage.ErrUrlNotFound
	}

	return nil
}

func (s *Storage) AliasExists(alias string) (bool, error) {
	const op = "storage.postgres.AliasExists"

	ctx := context.Background()

	var exists bool
	err := s.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM urls WHERE alias = $1)`, alias).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("%s: query: %w", op, err)
	}

	return exists, nil
}

func (s *Storage) GetPoolStats() *pgxpool.Stat {
	if s.pool != nil {
		return s.pool.Stat()
	}
	return nil
}
