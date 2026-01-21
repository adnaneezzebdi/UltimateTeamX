package config

import (
	"fmt"
	"os"
)

// Config contiene le impostazioni runtime per market-svc.
type Config struct {
	GRPCAddr      string
	DBDSN         string
	ClubGRPCAddr  string
}

// Load legge le variabili d'ambiente con default minimi.
func Load() Config {
	dbDSN := os.Getenv("DB_DSN")
	if dbDSN == "" {
		dbDSN = buildDSN()
	}

	return Config{
		GRPCAddr:     getEnv("GRPC_ADDR", ":50053"),
		DBDSN:        dbDSN,
		ClubGRPCAddr: os.Getenv("CLUB_GRPC_ADDR"),
	}
}

// getEnv ritorna il fallback quando la variabile non è presente.
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
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
