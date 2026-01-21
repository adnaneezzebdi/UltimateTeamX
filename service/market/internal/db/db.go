package db

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"time"

	_ "github.com/lib/pq"
)

// Open crea la connessione Postgres e la valida con un ping.
func Open(dsn string) (*sql.DB, error) {
	if dsn == "" {
		slog.Error("DB_DSN mancante")
		return nil, errors.New("DB_DSN is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	// Fallisce subito se il database non è raggiungibile.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		slog.Error("ping database fallito", "error", err)
		return nil, err
	}

	return db, nil
}
