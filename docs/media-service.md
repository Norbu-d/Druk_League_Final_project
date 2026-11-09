# Match media API

The Nginx web origin proxies `/api/media` and `/api/matches/{match_id}/media` to the Media service. Requests use the DrukLeague `drukleague-session` HttpOnly cookie; a bearer token is also accepted for service clients. Every route except `GET /healthz` requires a valid signed session.

## Upload

`POST /api/media` accepts `multipart/form-data` fields:

| Field | Required | Rule |
|---|---:|---|
| `match_id` | Yes | UUID owned by the Match service; Media stores it as an opaque reference |
| `media_kind` | Yes | `photo` or `highlight` |
| `caption` | No | At most 1,000 characters |
| `file` | Yes | JPEG, PNG or WebP photo; MP4 highlight; 1 byte–10 MiB |

The owner is taken from the authenticated token, never from request data. The service sniffs file content, creates a random storage key, writes the bytes under `MEDIA_DATA_DIR`, then inserts metadata. If metadata persistence fails, it removes the newly written blob. Successful upload returns `201` and metadata with a `Location: /api/media/{id}` header.

## List and retrieve

- `GET /api/matches/{match_id}/media` returns metadata newest first.
- `GET /api/media/{id}` streams the original bytes with the stored content type, `nosniff`, and `private, no-store` cache policy.
- `DELETE /api/media/{id}` removes an asset for its uploader or a Super Admin.

The Compose `media-data` named volume persists binary files separately from PostgreSQL. Match existence is not checked synchronously because Match owns a separate database; clients must provide a real Match UUID. The service requires an authenticated user for reads. Full content authorization tied to tournament visibility remains a follow-up because the monolith has no viewer session for this service.

## Limits and operations

Uploads are limited to 10 MiB at the service and 11 MiB at Nginx to allow multipart overhead. The random storage key is generated server-side; submitted filenames are never used as paths. Back up the `media-data` volume together with the Media database, and treat missing blobs as a surfaced 404 requiring restore/reconciliation.
