package main

import (
	"log/slog"
	"net"
	"os"

	"github.com/joho/godotenv"
	"UltimateTeamX/service/market/internal/config"
	"UltimateTeamX/service/market/internal/db"
	"UltimateTeamX/service/market/internal/market"
	clubv1 "UltimateTeamX/proto/club/v1"
	marketv1 "UltimateTeamX/proto/market/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	// Bootstrap di logging e config.
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Carica le variabili da .env se presente (solo per dev).
	envPath := os.Getenv("GO_DOTENV_PATH")
	if envPath == "" {
		envPath = ".env"
	}
	if err := godotenv.Overload(envPath); err != nil {
		// Se manca il file .env, continuiamo con le env già presenti.
		logger.Warn("impossibile caricare .env", "path", envPath, "error", err)
	} else {
		logger.Info(".env caricato", "path", envPath)
	}

	cfg := config.Load()

	// DB richiesto per la persistenza dei listing.
	database, err := db.Open(cfg.DBDSN)
	if err != nil {
		logger.Error("db connection failed", "error", err)
		os.Exit(1)
	}
	defer database.Close()

	// Club-svc richiesto per bloccare le carte dei listing.
	clubConn, err := grpc.Dial(cfg.ClubGRPCAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Error("club grpc dial failed", "error", err)
		os.Exit(1)
	}
	defer clubConn.Close()

	// Registra MarketService.
	server := grpc.NewServer()
	repo := market.NewRepo(database)
	clubClient := clubv1.NewClubServiceClient(clubConn)
	marketv1.RegisterMarketServiceServer(server, market.NewServer(logger, repo, clubClient))
	reflection.Register(server)

	// Avvia il listener gRPC.
	listener, err := net.Listen("tcp", cfg.GRPCAddr)
	if err != nil {
		logger.Error("grpc listen failed", "error", err)
		os.Exit(1)
	}

	logger.Info("market grpc listening", "addr", cfg.GRPCAddr)
	if err := server.Serve(listener); err != nil {
		logger.Error("grpc serve failed", "error", err)
		os.Exit(1)
	}
}
