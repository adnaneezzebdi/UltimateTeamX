Note sul DB di Market

Questo documento descrive lo schema e i flussi principali del market-svc.

Scopo
- market-svc gestisce solo lo stato e l'audit di listings/bids.
- La proprieta' economica rimane in club-svc (hold e settlement).

Concorrenza
- Il DB non e' usato per i lock; Redis gestisce i lock distribuiti.
- Tutte le transizioni di stato dei listing avvengono sotto lock Redis.

Schema
- listings: stato corrente di ogni annuncio (mai cancellato).
- bids: storico immutabile delle offerte per audit/recupero.
- best_bid/best_bidder_club_id sono salvati su listings per letture rapide e
  aggiornati solo sotto lock.
- bids.hold_id serve per rilasciare l'hold precedente quando arriva un rilancio.

Regole
- Stati ammessi per listings: ACTIVE, SOLD, EXPIRED.
- Nessuna foreign key verso i DB di altri servizi.
- Nessuna delete per listings/bids; solo update e insert.

Integrazione con club-svc
- market-svc risolve seller_club_id e bidder_club_id chiamando GetMyClub
  e passando l'user_id nelle metadata gRPC.
- L'ownership/disponibilita' della carta e' verificata con LockCard.
- I crediti sono gestiti con CreateCreditHold/ReleaseCreditHold.

Flusso CreateListing (market-svc)
- Valida i campi della richiesta (id, prezzi, scadenza).
- Controlla che non esista gia' un listing ACTIVE per la stessa carta.
- Risolve seller_club_id via club-svc (GetMyClub).
- Chiama club-svc LockCard; il lock vale come verifica di ownership/disponibilita'.
- Inserisce il listing con stato ACTIVE nel DB market.
- In caso di errore DB, rilascia il lock carta in club-svc.

Flusso PlaceBid (market-svc)
- Acquisisce un lock Redis su `lock:listing:{listing_id}`.
- Verifica il listing (ACTIVE, non scaduto, importo valido).
- Risolve bidder_club_id via club-svc (GetMyClub).
- Crea un hold crediti nel club-svc per il bidder.
- Inserisce il bid e aggiorna best_bid in transazione DB.
- Rilascia l'hold precedente (se presente).
- Rilascia il lock Redis.

Osservabilita'
- Log strutturati nel server per errori e successi del flusso CreateListing.

Configurazione
- Le variabili vengono caricate da `.env` (sovrascrivono quelle gia' presenti)
  usando `GO_DOTENV_PATH` se valorizzato.

Esempi pratici (grpcurl)

Creare un listing (mettere una carta in vendita)
grpcurl -plaintext -d '{
  "seller_user_id": "11111111-1111-1111-1111-111111111111",
  "user_card_id": "22222222-2222-2222-2222-222222222222",
  "start_price": 1000,
  "buy_now_price": 2000,
  "expires_at_unix": 1893456000
}' localhost:50053 market.v1.MarketService/CreateListing

Fare un'offerta (rilanciare su un annuncio)
grpcurl -plaintext -d '{
  "listing_id": "<LISTING_ID>",
  "bidder_user_id": "33333333-3333-3333-3333-333333333333",
  "bid_amount": 1500
}' localhost:50053 market.v1.MarketService/PlaceBid
