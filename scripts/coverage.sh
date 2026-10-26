#!/usr/bin/env bash
# Test every Go module and enforce the coverage threshold where tests exist.
# Profiles stay inside their module because native Windows Go does not handle
# Git Bash's /tmp paths reliably.
# Usage: scripts/coverage.sh [minimum-percent]
set -u

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
minimum=${1:-80}
status=0
printf '%-32s %12s\n' MODULE COVERAGE

# Remove any profile left behind if a run is interrupted.
cleanup_profiles() {
  for dir in "$root"/packages/* "$root"/services/* "$root"/tools/*; do
    [[ -d "$dir" ]] || continue
    rm -f "$dir"/.coverage-$$.out
  done
  return 0
}
trap cleanup_profiles EXIT
trap 'cleanup_profiles; exit 1' HUP INT TERM

for dir in "$root"/packages/* "$root"/services/* "$root"/tools/*; do
  [[ -f "$dir/go.mod" ]] || continue
  name=${dir#"$root"/}
  case "$name" in
    services/service-builder|services/service-runtime|services/_template) continue ;;
    *) ;;
  esac

  # A module-local profile path works with native Windows Go from Git Bash.
  profile=.coverage-$$.out
  # Let Go instrument each package that contains tests; including every
  # untested wiring/adapter package in the denominator hides useful coverage.
  if ! (cd "$dir" && go test -count=1 -coverprofile="$profile" ./...); then
    printf '%-32s %12s\n' "$name" TEST-FAIL
    rm -f "$dir/$profile"
    status=1
    continue
  fi

  total=$(cd "$dir" && go tool cover -func="$profile" | awk '/^total:/ {sub("%", "", $3); print $3}')
  has_tests=$(find "$dir" -name '*_test.go' -type f -print -quit)
  rm -f "$dir/$profile"
  [[ -n "$total" ]] || total=0

  if [[ -z "$has_tests" ]]; then
    printf '%-32s %12s\n' "$name" 'NO TESTS'
  else
    printf '%-32s %11s%%\n' "$name" "$total"
    if awk -v total="$total" -v minimum="$minimum" 'BEGIN { exit !(total + 0 < minimum + 0) }'; then
      status=1
    fi
  fi
done

exit "$status"
