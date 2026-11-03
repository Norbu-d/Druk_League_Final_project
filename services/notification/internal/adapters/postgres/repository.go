package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/notification/internal/domain"
)

type Repository struct {
	db *sql.DB
}

func New(db *sql.DB) *Repository {
	return &Repository{db: db}
}

const columns = "id::text,user_id::text,channel,subject,body,COALESCE(event_id::text,''),created_at"

func (r *Repository) Create(ctx context.Context, item domain.Notification) (domain.Notification, error) {
	query := "INSERT INTO notifications(user_id,channel,subject,body,event_id) " +
		"VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid) " +
		"ON CONFLICT (event_id,user_id) WHERE event_id IS NOT NULL DO NOTHING RETURNING " + columns
	saved, err := scan(r.db.QueryRowContext(ctx, query, item.UserID, item.Channel, item.Subject, item.Body, item.EventID))
	if err == nil {
		return saved, nil
	}
	if !errors.Is(err, sql.ErrNoRows) || item.EventID == "" {
		return domain.Notification{}, err
	}
	return scan(r.db.QueryRowContext(ctx, "SELECT "+columns+" FROM notifications WHERE event_id=$1 AND user_id=$2", item.EventID, item.UserID))
}

func (r *Repository) List(ctx context.Context, userID string, size, offset int) ([]domain.Notification, bool, error) {
	rows, err := r.db.QueryContext(ctx,
		"SELECT "+columns+" FROM notifications WHERE ($1='' OR user_id::text=$1) "+
			"ORDER BY created_at DESC,id DESC LIMIT $2 OFFSET $3",
		userID, size+1, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()

	items := make([]domain.Notification, 0, size+1)
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

	more := len(items) > size
	if more {
		items = items[:size]
	}
	return items, more, nil
}

type scanner interface {
	Scan(...any) error
}

func scan(row scanner) (domain.Notification, error) {
	var item domain.Notification
	err := row.Scan(&item.ID, &item.UserID, &item.Channel, &item.Subject, &item.Body, &item.EventID, &item.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Notification{}, core.NotFound("notification not found")
	}
	return item, err
}
