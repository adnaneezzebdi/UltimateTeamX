Market DB Notes

This doc records schema decisions for market-svc.

Scope
- market-svc owns listings/bids state and audit only.
- economic ownership stays in club-svc (holds/settlement).

Concurrency
- DB is not used for locking; Redis handles distributed locks.
- All listing state transitions must happen under the Redis lock.

Schema
- listings: current state of each listing (never deleted).
- bids: immutable history of bids for audit/recovery.
- best_bid/best_bidder_club_id are stored on listings for fast reads and are
  updated only under lock.

Rules
- Allowed listing status values: ACTIVE, SOLD, EXPIRED.
- No foreign keys to other services' databases.
- No deletes for listings/bids; only updates and inserts.

CreateListing Flow (market-svc)
- Validate request fields (ids, prices, expiry).
- Check for an existing ACTIVE listing for the same card in market DB.
- Call club-svc LockCard; lock is the ownership/availability check.
- Insert listing with status ACTIVE in market DB.
- On DB failure, release the card lock in club-svc.

Osservabilita
- Log strutturati nel server per errori e successi del flusso CreateListing.

Configurazione
- Le variabili vengono caricate da `.env` (sovrascrivono quelle gia' presenti) usando `GO_DOTENV_PATH` se valorizzato.
