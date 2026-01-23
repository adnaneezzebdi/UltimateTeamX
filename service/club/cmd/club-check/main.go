package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"time"

	"UltimateTeamX/service/club/internal/club"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// 1) Carica env per connessione DB.
	envPath := os.Getenv("GO_DOTENV_PATH")
	if envPath == "" {
		envPath = "service/club/.env"
	}
	if err := godotenv.Overload(envPath); err != nil {
		logger.Warn("impossibile caricare .env", "path", envPath, "error", err)
	}

	// 2) Legge la DSN e apre la connessione.
	dsn := os.Getenv("DB_DSN")
	if dsn == "" {
		logger.Error("DB_DSN mancante")
		os.Exit(1)
	}

	db, err := openDB(dsn)
	if err != nil {
		logger.Error("db connection failed", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	// 3) Risolve user_id (da env o dal primo club nel DB).
	userID, err := loadUserID(db)
	if err != nil {
		logger.Error("user id non disponibile", "error", err)
		os.Exit(1)
	}

	// 4) Chiama il service di dominio e stampa il risultato.
	repo := club.NewRepo(db)
	service := club.NewService(repo)

	result, err := service.GetMyClub(context.Background(), userID)
	if err != nil {
		if errors.Is(err, club.ErrClubNotFound) {
			logger.Error("club non trovato", "user_id", userID)
			os.Exit(1)
		}
		logger.Error("errore lettura club", "error", err)
		os.Exit(1)
	}

	fmt.Printf("club_id=%s credits=%d cards=%d\n", result.ClubID, result.Credits, len(result.Cards))
	for _, card := range result.Cards {
		fmt.Printf("card id=%s player_id=%s locked=%v\n", card.ID, card.PlayerID, card.Locked)
	}
}

func openDB(dsn string) (*sql.DB, error) {
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

// loadUserID usa USER_ID se presente, altrimenti prende il primo club.
func loadUserID(db *sql.DB) (uuid.UUID, error) {
	userIDStr := os.Getenv("USER_ID")
	if userIDStr != "" {
		return uuid.Parse(userIDStr)
	}

	const query = `
SELECT user_id
FROM clubs
ORDER BY created_at ASC
LIMIT 1`

	var userID uuid.UUID
	if err := db.QueryRow(query).Scan(&userID); err != nil {
		return uuid.Nil, err
	}
	return userID, nil
}
