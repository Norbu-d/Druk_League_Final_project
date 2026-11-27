package postgres

import (
	"context"
	"database/sql"

	"github.com/drukleague/platform/services/audit-log/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const insertEntry = `INSERT INTO audit_entries(event_id,actor_id,action,resource_type,resource_id,reason,occurred_at)
	VALUES(NULLIF($1,'')::uuid,$2,$3,$4,$5,$6,$7)
	ON CONFLICT (event_id) WHERE event_id IS NOT NULL DO NOTHING
	RETURNING id::text,COALESCE(event_id::text,''),actor_id,action,resource_type,resource_id,reason,occurred_at`

const getByEventID = `SELECT id::text,COALESCE(event_id::text,''),actor_id,action,resource_type,resource_id,reason,occurred_at FROM audit_entries WHERE event_id=$1`

const listEntries = `SELECT id::text,COALESCE(event_id::text,''),actor_id,action,resource_type,resource_id,reason,occurred_at
	FROM audit_entries WHERE ($1='' OR (resource_type=$1 AND resource_id=$2))
	ORDER BY occurred_at DESC,id DESC LIMIT $3 OFFSET $4`

func (r *Repository) Record(ctx context.Context, item domain.Entry) (domain.Entry, error) {
	created, err := scan(r.db.QueryRowContext(ctx, insertEntry, item.EventID, item.ActorID, item.Action, item.ResourceType, item.ResourceID, item.Reason, item.OccurredAt))
	if err == nil || err != sql.ErrNoRows || item.EventID == "" {
		return created, err
	}
	return scan(r.db.QueryRowContext(ctx, getByEventID, item.EventID))
}

func (r *Repository) List(ctx context.Context, resourceType, resourceID string, limit, offset int) ([]domain.Entry, bool, error) {
	rows, err := r.db.QueryContext(ctx, listEntries, resourceType, resourceID, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := make([]domain.Entry, 0, limit+1)
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			return nil, false, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	if len(items) > limit {
		return items[:limit], true, nil
	}
	return items, false, nil
}

type rowScanner interface{ Scan(...any) error }

func scan(row rowScanner) (domain.Entry, error) {
	var item domain.Entry
	err := row.Scan(&item.ID, &item.EventID, &item.ActorID, &item.Action, &item.ResourceType, &item.ResourceID, &item.Reason, &item.OccurredAt)
	return item, err
}
