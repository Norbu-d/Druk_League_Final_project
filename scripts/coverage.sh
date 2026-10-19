#!/usr/bin/env sh
# Per-module unit/integration test coverage for the Go code under internal/ (services) or the
# whole module (shared packages). cmd/ wiring is excluded: it only connects adapters.
# Set TEST_DATABASE_URL to include the PostgreSQL adapter tests, e.g.
#   TEST_DATABASE_URL='postgres://postgres:test@127.0.0.1:55432/postgres?sslmode=disable'
# Usage: scripts/coverage.sh [min-percent]   (exits 1 if any listed module is below min)
set -u
root=$(cd "$(dirname "$0")/.." && pwd)
min=${1:-0}
status=0
printf '%-28s %8s\n' MODULE COVERAGE
for dir in "$root"/packages/servicekit "$root"/packages/eventkit "$root"/services/*/; do
  dir=${dir%/}
  [ -f "$dir/go.mod" ] || continue
  name=${dir#"$root"/}
  case "$name" in services/_template|services/api-gateway|services/service-*) continue ;; esac
  if [ -d "$dir/internal" ]; then pkgs=./internal/...; else pkgs=./...; fi
  profile=$(mktemp)
  if ! (cd "$dir" && go test -count=1 -coverpkg="$pkgs" -coverprofile="$profile" $pkgs >/dev/null 2>&1); then
    printf '%-28s %8s\n' "$name" FAIL
    status=1
    rm -f "$profile"
    continue
  fi
  total=$(cd "$dir" && go tool cover -func="$profile" | awk '/^total:/ {sub("%","",$3); print $3}')
  rm -f "$profile"
  [ -z "$total" ] && total=0
  printf '%-28s %7s%%\n' "$name" "$total"
  if awk -v t="$total" -v m="$min" 'BEGIN{exit !(t+0 < m+0)}'; then status=1; fi
done
exit $status
