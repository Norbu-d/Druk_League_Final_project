param(
  [string]$ComposeFile = "deploy/docker-compose.yml"
)

$ErrorActionPreference = "Stop"

docker compose -f $ComposeFile stop team player tournament search
if ($LASTEXITCODE -ne 0) { throw "Could not stop Team, Player, Tournament, and Search." }

docker compose -f $ComposeFile exec -T postgres psql -U drukleague -d search_service -v ON_ERROR_STOP=1 -c "DELETE FROM search_documents; DELETE FROM search_entity_versions; DELETE FROM search_processed_events;"
if ($LASTEXITCODE -ne 0) { throw "Could not clear the Search index." }

foreach ($database in @("team_service", "player_service", "tournament_service")) {
  docker compose -f $ComposeFile exec -T postgres psql -U drukleague -d $database -v ON_ERROR_STOP=1 -c "UPDATE search_outbox SET published_at = NULL;"
  if ($LASTEXITCODE -ne 0) { throw "Could not reset the $database outbox." }
}

docker compose -f $ComposeFile up -d search team player tournament
if ($LASTEXITCODE -ne 0) { throw "Could not restart source services." }

Write-Host "Search replay started. Check Search's outbox/consumer logs and document counts while it catches up."
