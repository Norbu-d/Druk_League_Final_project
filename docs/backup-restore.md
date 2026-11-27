# Local database and media backup

Run the PowerShell backup while the Compose stack is healthy:

```powershell
.
\scripts\backup.ps1
```

The script writes a `pg_dumpall` SQL dump and a gzip archive of the Compose `media-data` volume to `%LOCALAPPDATA%\DrukLeague\backups` by default. An alternate output directory can be supplied as the first argument. Keep these files outside Git and store copies somewhere separate from the development machine before a demo.

The SQL dump contains account hashes and database contents. Treat both archives as private project data. This is a development backup procedure, not an automated retention or off-site backup service.

## Restore

Restore only into a stopped/isolated development stack after confirming the target database and media volume. Restoring overwrites data in the target databases and media volume.

1. Stop the application services while leaving PostgreSQL available. Keep a separate copy of the target's existing database and `media-data` volume before proceeding.
2. Restore the SQL dump into the PostgreSQL instance using its configured superuser:

   ```powershell
   Get-Content -Raw .\drukleague-databases-<timestamp>.sql | docker compose -f deploy/docker-compose.yml exec -T postgres psql -U drukleague -d postgres
   ```

3. With the media service stopped, extract the media archive into the Compose `media-data` volume using a temporary Alpine container. Replace the archive path with the selected backup:

   ```powershell
   docker run --rm --mount type=volume,src=deploy_media-data,dst=/data --mount "type=bind,src=$((Get-Location).Path -replace '\\','/'),dst=/backup,readonly" alpine:3.22 sh -c 'tar -xzf /backup/drukleague-media-<timestamp>.tar.gz -C /data'
   ```

4. Start the application services and verify health, login, media retrieval, and the required project smoke flows before using the restored data.

The Compose project name affects named volume names. Check the resolved volume with `docker compose -f deploy/docker-compose.yml config --format json` before restoring; do not assume `deploy_media-data` if `COMPOSE_PROJECT_NAME` has been changed.
