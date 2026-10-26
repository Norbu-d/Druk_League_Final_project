#!/usr/bin/env sh
# Test each Go module. Enforce the coverage floor only where tests exist so
# modules without tests can be brought under coverage incrementally.
# Usage: scripts/coverage.sh [minimum-percent]
set -u

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
minimum=${1:-80}
status=0
printf '%-32s %10s\n' MODULE COVERAGE

for dir in "$root"/packages/* "$root"/services/*/ "$root"/tools/*/; do
  [ -f "$dir/go.mod" ] || continue
  name=${dir#"$root"/}
  name=${name%/}
  case "$name" in services/_template|services/api-gateway|services/service-*) continue ;; esac

  if [ -d "$dir/internal" ]; then
    packages=./internal/...
  else
    packages=./...
  fi

  # A module-local profile path works with both Unix Go and native Windows Go
  # invoked from Git Bash; the PID avoids collisions between concurrent runs.
  profile=.coverage-$$.out
  if ! (cd "$dir" && go test -count=1 -coverpkg="$packages" -coverprofile="$profile" $packages); then
    printf '%-32s %10s\n' "$name" FAIL
    rm -f "$dir/$profile"
    status=1
    continue
  fi

  total=$(cd "$dir" && go tool cover -func="$profile" | awk '/^total:/ {sub("%", "", $3); print $3}')
  has_tests=$(find "$dir" -name '*_test.go' -type f -print -quit)
  rm -f "$dir/$profile"
  [ -n "$total" ] || total=0

  if [ -z "$has_tests" ]; then
    printf '%-32s %10s\n' "$name" 'NO TESTS'
  else
    printf '%-32s %9s%%\n' "$name" "$total"
    if awk -v total="$total" -v minimum="$minimum" 'BEGIN { exit !(total + 0 < minimum + 0) }'; then
      status=1
    fi
  fi
done

exit "$status"
