#!/usr/bin/env bash
set -euo pipefail

root=$(cd -- "$(dirname -- "$0")/.." && pwd)
mapfile -d '' modules < <(find "$root/packages" "$root/services" "$root/tools" -name go.mod -type f -print0 | sort -z)

run_govulncheck() {
  local attempt status output
  output=$(mktemp)
  for attempt in 1 2 3; do
    if govulncheck ./... >"$output" 2>&1; then
      cat "$output"
      rm -f "$output"
      return 0
    else
      status=$?
    fi

    if (( attempt < 3 )) && grep -Eiq 'fetching vulnerabilities|connection reset|unexpected EOF|i/o timeout|temporary failure in name resolution|TLS handshake timeout|no such host' "$output"; then
      echo "govulncheck network request failed (attempt $attempt/3); retrying in $((attempt * 5)) seconds." >&2
      sleep $((attempt * 5))
      continue
    fi

    cat "$output" >&2
    rm -f "$output"
    return "$status"
  done
}

for modfile in "${modules[@]}"; do
  module=${modfile%/go.mod}
  relative=${module#"$root"/}
  case "$relative" in
    services/service-builder|services/service-runtime|services/_template) continue ;;
    *) ;;
  esac

  echo "Scanning $relative"
  (
    cd "$module"
    run_govulncheck
    gosec -exclude-generated ./...
  )
done
