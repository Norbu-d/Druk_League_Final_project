#!/usr/bin/env bash
set -euo pipefail

root=$(cd -- "$(dirname -- "$0")/.." && pwd)
mapfile -d '' modules < <(find "$root/packages" "$root/services" "$root/tools" -name go.mod -type f -print0 | sort -z)

for modfile in "${modules[@]}"; do
  module=${modfile%/go.mod}
  relative=${module#"$root"/}
  case "$relative" in
    services/service-builder|services/service-runtime|services/_template) continue ;;
    *) ;;
  esac

  echo "Vetting and building $relative"
  (
    cd "$module"
    go vet ./...
    go build ./...
  )
done
