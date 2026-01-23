Club Service - Guida per iniziare (Italiano)

Questa guida spiega cos'e' il club-svc, cosa fa e come provarlo anche se
non conosci il progetto.

Cos'e' il club-svc
- E' il servizio che gestisce i dati economici del club (credits),
  le carte possedute e gli hold di credito.
- Altri servizi (es. market-svc) NON scrivono nel DB del club direttamente:
  chiamano le API gRPC del club-svc.

Schema DB (migrations/clubs)
- clubs: il club dell'utente (id, user_id, credits, created_at).
- user_cards: carte possedute (id, club_id, player_id, locked).
- ledger: audit delle variazioni di credito.
- credit_holds: blocchi temporanei di crediti (es. offerte in market).

Prerequisiti
- Un database Postgres accessibile.
- Le migrations applicate nell'ordine:
  - 001_create_clubs.sql
  - 002_create_user_cards.sql
  - 003_create_ledger.up.sql
  - 004_create_credit_holds.up.sql

Configurazione (.env)
Crea `service/club/.env` con:
DB_DSN=postgresql://<user>:<pass>@<host>:5432/<db>?sslmode=require
DB_HOST=<host>
DB_PORT=5432
DB_USER=<user>
DB_PASSWORD=<pass>
DB_NAME=<db>
DB_SSLMODE=require

Esecuzione migrations (senza server)
Il repo include un runner SQL in `service/club/cmd/server/main.go`.
Esempio:
go run service/club/cmd/server/main.go migrations/clubs/001_create_clubs.sql migrations/clubs/002_create_user_cards.sql

Test manuale del dominio (senza gRPC)
Il runner `club-check` stampa credits e cards per un user_id.
Esempio:
export GO_DOTENV_PATH="service/club/.env"
export USER_ID="<UUID_UTENTE>"
go run service/club/cmd/club-check/main.go

API gRPC disponibili
Le API gRPC sono definite in `proto/club/v1/club.proto`.
Nota: il repo contiene l'handler gRPC, ma non un main server completo.
Se hai un server club in esecuzione, questi sono i JSON da usare con grpcurl.

1) GetMyClub
Richiede user_id nelle metadata gRPC (non nel JSON).
grpcurl -plaintext -d '{}' \
  -H 'user_id: <UUID_UTENTE>' \
  localhost:50052 club.v1.ClubService/GetMyClub

Risposta (esempio):
{
  "club_id": "<UUID_CLUB>",
  "credits": 1200,
  "cards": [
    { "id": "<UUID_CARD>", "player_id": "<UUID_PLAYER>", "locked": false }
  ]
}

2) LockCard
JSON da inviare:
{
  "user_id": "<UUID_UTENTE>",
  "user_card_id": "<UUID_CARD>",
  "reason": "market_listing"
}
grpcurl -plaintext -d '{
  "user_id": "<UUID_UTENTE>",
  "user_card_id": "<UUID_CARD>",
  "reason": "market_listing"
}' localhost:50052 club.v1.ClubService/LockCard

3) ReleaseCardLock
JSON da inviare:
{
  "lock_id": "<UUID_LOCK>"
}
grpcurl -plaintext -d '{
  "lock_id": "<UUID_LOCK>"
}' localhost:50052 club.v1.ClubService/ReleaseCardLock

4) CreateCreditHold
JSON da inviare:
{
  "user_id": "<UUID_UTENTE>",
  "amount": 1500,
  "reason": "market_bid"
}
grpcurl -plaintext -d '{
  "user_id": "<UUID_UTENTE>",
  "amount": 1500,
  "reason": "market_bid"
}' localhost:50052 club.v1.ClubService/CreateCreditHold

5) ReleaseCreditHold
JSON da inviare:
{
  "hold_id": "<UUID_HOLD>"
}
grpcurl -plaintext -d '{
  "hold_id": "<UUID_HOLD>"
}' localhost:50052 club.v1.ClubService/ReleaseCreditHold

6) SettleTrade
JSON da inviare:
{
  "seller_user_id": "<UUID_UTENTE_SELLER>",
  "buyer_user_id": "<UUID_UTENTE_BUYER>",
  "user_card_id": "<UUID_CARD>",
  "amount": 2000
}
grpcurl -plaintext -d '{
  "seller_user_id": "<UUID_UTENTE_SELLER>",
  "buyer_user_id": "<UUID_UTENTE_BUYER>",
  "user_card_id": "<UUID_CARD>",
  "amount": 2000
}' localhost:50052 club.v1.ClubService/SettleTrade

Errori comuni
- Unauthenticated: user_id mancante nelle metadata gRPC (GetMyClub).
- NotFound: club non trovato per l'user_id.
- Internal: errori DB o problemi di connessione.
