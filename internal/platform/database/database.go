// Package database configures the PostgreSQL connection pool shared by adapters.
package database

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// URLFromEnv supports an explicit URL or the DB_* variables used by Compose.
// No database settings means the local JSON adapter remains available.
func URLFromEnv() string {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}
	host := os.Getenv("DB_HOST")
	if host == "" {
		return ""
	}
	value := func(key, fallback string) string {
		if v := os.Getenv(key); v != "" {
			return v
		}
		return fallback
	}
	u := url.URL{Scheme: "postgres", Host: net.JoinHostPort(host, value("DB_PORT", "5432")), Path: "/" + value("DB_NAME", "app_db"), User: url.UserPassword(value("DB_USER", "app"), os.Getenv("DB_PASSWORD"))}
	q := u.Query()
	q.Set("sslmode", value("DB_SSLMODE", "require"))
	u.RawQuery = q.Encode()
	return u.String()
}

func Open(ctx context.Context, connectionURL string) (*pgxpool.Pool, error) {
	if connectionURL == "" {
		return nil, fmt.Errorf("set DATABASE_URL or DB_HOST/DB_PORT/DB_USER/DB_PASSWORD/DB_NAME")
	}
	config, err := pgxpool.ParseConfig(connectionURL)
	if err != nil {
		return nil, fmt.Errorf("invalid PostgreSQL configuration")
	}
	config.MaxConns = 10
	config.MinConns = 0
	config.ConnConfig.ConnectTimeout = 5 * time.Second
	config.ConnConfig.RuntimeParams["application_name"] = "i-concept"
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("initialize PostgreSQL pool: %w", err)
	}
	if err = pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("connect to PostgreSQL: %w", err)
	}
	return pool, nil
}
