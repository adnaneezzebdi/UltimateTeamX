#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
LOG_DIR="${ROOT_DIR}/.tmp"

# Script operativo: avvia Redis, mock club e market in locale con log su file.
mkdir -p "${LOG_DIR}"

start_redis() {
  # Avvia Redis se non e' gia' attivo.
  if command -v redis-cli >/dev/null 2>&1; then
    if redis-cli ping >/dev/null 2>&1; then
      echo "redis: gia' in esecuzione"
      return
    fi
  fi

  if command -v redis-server >/dev/null 2>&1; then
    echo "redis: avvio su 6379"
    redis-server >"${LOG_DIR}/redis.log" 2>&1 &
    echo $! >"${LOG_DIR}/redis.pid"
  else
    echo "redis: redis-server non trovato"
  fi
}

start_mock_club() {
  # Avvia il mock club-svc.
  echo "club: avvio mock-server"
  (cd "${ROOT_DIR}" && GO_DOTENV_PATH="service/club/.env" go run service/club/cmd/mock-server/main.go) \
    >"${LOG_DIR}/club.log" 2>&1 &
  echo $! >"${LOG_DIR}/club.pid"
}

start_market() {
  # Avvia market-svc.
  echo "market: avvio server"
  (cd "${ROOT_DIR}" && GO_DOTENV_PATH="service/market/.env" go run service/market/cmd/server/main.go) \
    >"${LOG_DIR}/market.log" 2>&1 &
  echo $! >"${LOG_DIR}/market.pid"
}

cleanup() {
  # Termina i processi avviati da questo script.
  for name in market club redis; do
    pid_file="${LOG_DIR}/${name}.pid"
    if [[ -f "${pid_file}" ]]; then
      pid="$(cat "${pid_file}")"
      if kill -0 "${pid}" >/dev/null 2>&1; then
        kill "${pid}" >/dev/null 2>&1 || true
      fi
      rm -f "${pid_file}"
    fi
  done
}

trap cleanup INT TERM

start_redis
start_mock_club
start_market

echo "log: ${LOG_DIR}/club.log"
echo "log: ${LOG_DIR}/market.log"
echo "premi Ctrl+C per terminare"

wait
