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
  for dir in "$root"/packages/* "$root"/services/* "$root"/tools/* "$root"/functions/*; do
    [[ -d "$dir" ]] || continue
    rm -f "$dir"/.coverage-$$.out
    rm -f "$dir"/.coverage-$$.log
  done
  return 0
}
trap cleanup_profiles EXIT
trap 'cleanup_profiles; exit 1' HUP INT TERM

for dir in "$root"/packages/* "$root"/services/* "$root"/tools/* "$root"/functions/*; do
  [[ -f "$dir/go.mod" ]] || continue
  name=${dir#"$root"/}
  case "$name" in
    services/service-builder|services/service-runtime|services/_template) continue ;;
    *) ;;
  esac

  # A module-local profile path works with native Windows Go from Git Bash.
  profile=.coverage-$$.out
  output=.coverage-$$.log
  if [[ "$name" == packages/servicekit ]]; then
    coverage_packages=./core/...,./dzongkhags/...,./servicekit/...
    test_packages=(./core/... ./dzongkhags/... ./servicekit/...)
  elif [[ -d "$dir/internal" ]]; then
    coverage_packages=./internal/...
    test_packages=(./internal/...)
  else
    coverage_packages=./...
    test_packages=(./...)
  fi
  if ! (cd "$dir" && go test -count=1 -coverpkg="$coverage_packages" -coverprofile="$profile" "${test_packages[@]}" >"$output" 2>&1); then
    cat "$dir/$output"
    printf '%-32s %12s\n' "$name" TEST-FAIL
    rm -f "$dir/$profile"
    rm -f "$dir/$output"
    status=1
    continue
  fi

  total=$(cd "$dir" && go tool cover -func="$profile" 2>/dev/null | awk '/^total:/ {sub("%", "", $3); print $3}')
  if [[ "$name" == packages/servicekit ]]; then
    has_tests=$(find "$dir"/core "$dir"/dzongkhags "$dir"/servicekit -name '*_test.go' -type f -print -quit)
  elif [[ -d "$dir/internal" ]]; then
    has_tests=$(find "$dir/internal" -name '*_test.go' -type f -print -quit)
  else
    has_tests=$(find "$dir" -name '*_test.go' -type f -print -quit)
  fi
  rm -f "$dir/$profile"
  rm -f "$dir/$output"

  if [[ -z "$has_tests" ]]; then
    printf '%-32s %12s\n' "$name" 'NO TESTS'
  else
    [[ -n "$total" ]] || total=0
    printf '%-32s %11s%%\n' "$name" "$total"
    if awk -v total="$total" -v minimum="$minimum" 'BEGIN { exit !(total + 0 < minimum + 0) }'; then
      status=1
    fi
  fi
done

exit "$status"
