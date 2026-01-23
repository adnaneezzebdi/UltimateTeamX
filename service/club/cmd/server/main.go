package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// 1) Carica env dedicato al flow club.
	// Carica le variabili da .env per il flow club.
	envPath := os.Getenv("GO_DOTENV_PATH")
	if envPath == "" {
		envPath = "service/club/.env"
	}
	if err := godotenv.Overload(envPath); err != nil {
		logger.Warn("impossibile caricare .env", "path", envPath, "error", err)
	} else {
		logger.Info(".env caricato", "path", envPath)
	}

	// 2) Costruisce la DSN e apre il DB.
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		dsn = buildDSN()
	}

	db, err := openDB(dsn)
	if err != nil {
		logger.Error("db connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 3) Legge i file SQL da CLI ed esegue in transazione.
	files := os.Args[1:]
	if len(files) == 0 {
		logger.Error("nessun file sql passato", "usage", "go run service/club/cmd/server/main.go <file.sql> [file2.sql]")
		os.Exit(1)
	}

	for _, file := range files {
		if err := execSQLFile(db, file); err != nil {
			logger.Error("esecuzione sql fallita", "file", file, "error", err)
			os.Exit(1)
		}
		logger.Info("sql eseguito", "file", file)
	}
}

func openDB(dsn string) (*sql.DB, error) {
	if dsn == "" {
		return nil, errors.New("DB_DSN is required")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}

	return db, nil
}

func execSQLFile(db *sql.DB, path string) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, string(content)); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func buildDSN() string {
	host := os.Getenv("DB_HOST")
	port := getEnv("DB_PORT", "5432")
	user := os.Getenv("DB_USER")
	password := os.Getenv("DB_PASSWORD")
	name := os.Getenv("DB_NAME")
	sslmode := getEnv("DB_SSLMODE", "require")
	if host == "" || user == "" || name == "" {
		return ""
	}
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s", user, password, host, port, name, sslmode)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
