#!/usr/bin/env bash
set -euo pipefail

compose=(docker compose --file deploy/docker-compose.yml)
api_url=${API_BASE_URL:-http://127.0.0.1:8080}

cleanup() {
  "${compose[@]}" down --volumes --remove-orphans
}
trap cleanup EXIT

"${compose[@]}" up --detach postgres nats db-init api audit-log

for attempt in {1..60}; do
  if curl --fail --silent "$api_url/healthz" >/dev/null; then
    break
  fi
  if [[ "$attempt" == 60 ]]; then
    "${compose[@]}" logs api audit-log postgres nats
    exit 1
  fi
  sleep 2
done

login=$(curl --fail --silent "$api_url/api/auth/login" \
  -H 'Content-Type: application/json' \
  --data "{\"username\":\"${BOOTSTRAP_ADMIN_USER}\",\"password\":\"${BOOTSTRAP_ADMIN_PASSWORD}\"}")
token=$(jq -er '.token' <<<"$login")
actor_id=$(jq -er '.user.id' <<<"$login")

tournament=$(curl --fail --silent "$api_url/api/tournaments" \
  -H "Authorization: Bearer $token" \
  -H 'Content-Type: application/json' \
  --data '{"name":"Audit E2E Verification","sport":"Football","season":"CI","dzongkhag":"Thimphu"}')
tournament_id=$(jq -er '.id' <<<"$tournament")

for attempt in {1..60}; do
  count=$("${compose[@]}" exec -T postgres psql -U drukleague -d audit_log_service -Atqc \
    "SELECT count(*) FROM audit_entries WHERE action='tournament.created' AND resource_type='tournament' AND resource_id='${tournament_id}' AND actor_id='${actor_id}'")
  if [[ "$count" == "1" ]]; then
    echo "Audit event persisted for tournament $tournament_id by actor $actor_id."
    exit 0
  fi
  sleep 2
done

"${compose[@]}" logs api audit-log postgres nats
echo "Expected tournament.created audit entry for $tournament_id; found $count." >&2
exit 1
