#!/bin/sh
set -eu
: "${POSTGRES_PASSWORD:?POSTGRES_PASSWORD is required}"
for database in auth_service user_service team_service player_service tournament_service registration_service match_service score_service fixture_service standings_service venue_service official_service dzongkhag_service notification_service media_service report_service search_service analytics_service audit_log_service configuration_service; do
  exists=$(PGPASSWORD="$POSTGRES_PASSWORD" psql -h postgres -U drukleague -d postgres -tAc "SELECT 1 FROM pg_database WHERE datname='$database'")
  if [ "$exists" != "1" ]; then
    PGPASSWORD="$POSTGRES_PASSWORD" createdb -h postgres -U drukleague "$database"
  fi
done