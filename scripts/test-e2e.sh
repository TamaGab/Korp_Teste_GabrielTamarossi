#!/usr/bin/env bash

set -uo pipefail

readonly PROJECT_NAME="korp-e2e"
readonly ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
readonly RESULTS_DIR="$ROOT_DIR/frontend/test-results"
readonly LOG_FILE="$RESULTS_DIR/e2e.log"
readonly -a COMPOSE=(
  docker compose
  --project-name "$PROJECT_NAME"
  --file "$ROOT_DIR/docker-compose.yml"
  --file "$ROOT_DIR/docker-compose.e2e.yml"
)

cleanup() {
  "${COMPOSE[@]}" down --volumes --remove-orphans >/dev/null 2>&1 || true
}

trap cleanup EXIT

cleanup
rm -rf "$RESULTS_DIR"
mkdir -p "$RESULTS_DIR"

"${COMPOSE[@]}" up \
  --build \
  --abort-on-container-exit \
  --exit-code-from e2e \
  e2e 2>&1 | tee "$LOG_FILE"
test_status=${PIPESTATUS[0]}

"${COMPOSE[@]}" cp e2e:/app/test-results/. "$RESULTS_DIR/" 2>&1 | tee -a "$LOG_FILE" || true

exit "$test_status"
