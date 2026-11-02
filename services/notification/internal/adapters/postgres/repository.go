package postgres

import (
	"context"
	"database/sql"

	"github.com/drukleague/platform/services/notification/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const columns = `id::text,COALESCE(event_id::text,''),user_id::text,channel,subject,body,status,created_at`

func (r *Repository) Create(ctx context.Context, item domain.Notification) (domain.Notification, bool, error) {
	created, err := scan(r.db.QueryRowContext(ctx, `INSERT INTO notifications(event_id,user_id,channel,subject,body,status)
		VALUES(NULLIF($1,'')::uuid,$2,$3,$4,$5,COALESCE(NULLIF($6,''),'pending'))
		ON CONFLICT (event_id,user_id,channel) WHERE event_id IS NOT NULL DO NOTHING
		RETURNING `+columns, item.EventID, item.UserID, item.Channel, item.Subject, item.Body, item.Status))
	if err == nil {
		return created, true, nil
	}
	if err != sql.ErrNoRows || item.EventID == "" {
		return domain.Notification{}, false, err
	}
	existing, err := scan(r.db.QueryRowContext(ctx, `SELECT `+columns+` FROM notifications WHERE event_id=$1 AND user_id=$2 AND channel=$3`, item.EventID, item.UserID, item.Channel))
	return existing, false, err
}

func (r *Repository) List(ctx context.Context, userID string, limit, offset int) ([]domain.Notification, bool, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+columns+` FROM notifications WHERE user_id=$1 ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3`, userID, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	items := make([]domain.Notification, 0, limit+1)
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

func scan(row rowScanner) (domain.Notification, error) {
	var item domain.Notification
	err := row.Scan(&item.ID, &item.EventID, &item.UserID, &item.Channel, &item.Subject, &item.Body, &item.Status, &item.CreatedAt)
	return item, err
}
