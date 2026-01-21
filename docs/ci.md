# Continuous Integration (CI)

UltimateTeamX usa GitHub Actions come sistema di Continuous Integration.

La CI è **obbligatoria** per tutte le Pull Request verso `main`.
Nessun merge è permesso se la pipeline non è verde.

---

## Obiettivi

La CI serve a garantire:

- coerenza architetturale e di repository
- validazione dei file `.proto`
- build e test dei servizi Go
- prevenzione di regressioni

---

## Trigger

La pipeline CI viene eseguita su:

- `pull_request` verso `main`
- `push` su `main`

---

## Contenuto della pipeline

La CI esegue i seguenti step:

1. **Buf lint**
2. **Buf generate**
3. **Go build**
4. **Go test**

---

## Requisiti di merge

- almeno 1 approvazione PR
- CI verde
- branch protection abilitata su `main`

---

## Variabili d’ambiente

La CI usa variabili d’ambiente standard per i servizi:

- `DATABASE_URL`
- `REDIS_ADDR`
- `JWT_SECRET`

In fase iniziale queste variabili possono essere opzionali per consentire lo sviluppo incrementale.
In fasi successive (pre-prod/prod) saranno obbligatorie.

---

## Ambiente CI

La pipeline esegue un container PostgreSQL e Redis per simulare l’ambiente runtime.

---

## Best practice

- Non committare credenziali in chiaro
- La CI deve essere deterministica e ripetibile
- Tutti i servizi devono essere testati nello stesso modo

---

## Estensioni future

In fasi successive la CI potrà includere:

- test di integrazione tra servizi
- `go test -race`
- analisi statiche aggiuntive
- code coverage