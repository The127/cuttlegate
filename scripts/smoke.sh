#!/usr/bin/env bash
set -euo pipefail

PROJECT_NAME="cuttlegate-smoke"
TIMEOUT=60
COMPOSE="docker compose --project-name $PROJECT_NAME"

# Ports used by docker-compose.yml — fail early if any are occupied
REQUIRED_PORTS=(5432 8080 5002 5003)
busy=()
for port in "${REQUIRED_PORTS[@]}"; do
  if ss -tlnH "sport = :$port" | grep -q .; then
    busy+=("$port")
  fi
done
if [ ${#busy[@]} -gt 0 ]; then
  echo "FAIL: port(s) ${busy[*]} already in use. Stop conflicting services (e.g. just down) before running smoke."
  exit 1
fi

cleanup() {
  echo "Tearing down..."
  $COMPOSE down -v --remove-orphans 2>/dev/null || true
}
trap cleanup EXIT

echo "Starting compose stack (project: $PROJECT_NAME)..."
$COMPOSE up -d --build

# Services with healthchecks in docker-compose.yml
HEALTHCHECK_SERVICES=("db" "keyline-db" "valkey" "keyline")

echo "Waiting for services to become healthy (timeout: ${TIMEOUT}s)..."

elapsed=0
while [ "$elapsed" -lt "$TIMEOUT" ]; do
  all_healthy=true
  ps_json=$($COMPOSE ps --format json 2>/dev/null)

  # Check for failed one-shot services (e.g. migrate)
  while IFS= read -r line; do
    name=$(echo "$line" | jq -r '.Service')
    state=$(echo "$line" | jq -r '.State')
    exit_code=$(echo "$line" | jq -r '.ExitCode')

    if [ "$state" = "exited" ] && [ "$exit_code" != "0" ]; then
      echo "FAIL: service '$name' exited with code $exit_code"
      exit 1
    fi
  done < <(echo "$ps_json" | jq -c '.[] | {Service: .Labels["com.docker.compose.service"], State: .State, ExitCode: .ExitCode}')

  # Check services that have healthchecks — look for "(healthy)" in Status
  for svc in "${HEALTHCHECK_SERVICES[@]}"; do
    status=$(echo "$ps_json" | jq -r --arg svc "$svc" '.[] | select(.Labels["com.docker.compose.service"] == $svc) | .Status')
    if [[ "$status" != *"(healthy)"* ]]; then
      all_healthy=false
      break
    fi
  done

  if $all_healthy; then
    echo "All services healthy after ${elapsed}s."
    break
  fi

  sleep 2
  elapsed=$((elapsed + 2))
done

if [ "$elapsed" -ge "$TIMEOUT" ]; then
  echo "FAIL: timeout after ${TIMEOUT}s. Unhealthy services:"
  ps_json=$($COMPOSE ps --format json 2>/dev/null)
  for svc in "${HEALTHCHECK_SERVICES[@]}"; do
    status=$(echo "$ps_json" | jq -r --arg svc "$svc" '.[] | select(.Labels["com.docker.compose.service"] == $svc) | .Status')
    if [[ "$status" != *"(healthy)"* ]]; then
      echo "  - $svc ($status)"
    fi
  done
  exit 1
fi

# Verify endpoints
BASE_URL="http://localhost:8080"
KEYLINE_URL="http://localhost:5002"
KEYLINE_UI_URL="http://localhost:5003"

fail=0

check_endpoint() {
  local url="$1"
  local description="$2"
  local status
  status=$(curl -sf -o /dev/null -w '%{http_code}' "$url" 2>/dev/null) || status="000"
  if [ "$status" = "200" ]; then
    echo "OK:   $description ($url) -> $status"
  else
    echo "FAIL: $description ($url) -> $status"
    fail=1
  fi
}

check_json_fields() {
  local url="$1"
  local description="$2"
  shift 2
  local fields=("$@")
  local body
  body=$(curl -sf "$url" 2>/dev/null) || { echo "FAIL: $description ($url) -> request failed"; fail=1; return; }

  for field in "${fields[@]}"; do
    if ! echo "$body" | jq -e ".$field" > /dev/null 2>&1; then
      echo "FAIL: $description ($url) -> missing field '$field'"
      fail=1
      return
    fi
  done
  echo "OK:   $description ($url) -> 200, fields present"
}

echo ""
echo "Verifying endpoints..."

check_endpoint "$BASE_URL/healthz" "Server healthz"
check_endpoint "$BASE_URL/readyz" "Server readyz"
check_json_fields "$BASE_URL/api/v1/config" "Server config" authority client_id redirect_uri
check_endpoint "$KEYLINE_URL/oidc/keyline/.well-known/openid-configuration" "Keyline OIDC discovery"
check_endpoint "$KEYLINE_UI_URL" "Keyline UI"

echo ""
if [ "$fail" -ne 0 ]; then
  echo "FAIL: one or more endpoint checks failed."
  exit 1
fi

echo "All smoke checks passed."
exit 0
