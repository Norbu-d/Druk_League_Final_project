package postgres

import (
	"context"
	"database/sql"
	"encoding/json"

	"github.com/drukleague/platform/services/service-template/internal/domain"
)

type RecordRepository struct{ db *sql.DB }

func NewRecordRepository(db *sql.DB) *RecordRepository { return &RecordRepository{db: db} }
func (r *RecordRepository) Create(ctx context.Context, payload []byte) (domain.Record, error) {
	if !json.Valid(payload) {
		return domain.Record{}, ErrInvalidPayload
	}
	var record domain.Record
	err := r.db.QueryRowContext(ctx, `INSERT INTO records(payload) VALUES($1) RETURNING id::text,payload,created_at`, payload).Scan(&record.ID, &record.Payload, &record.CreatedAt)
	return record, err
}
func (r *RecordRepository) Get(ctx context.Context, id string) (domain.Record, error) {
	var record domain.Record
	err := r.db.QueryRowContext(ctx, `SELECT id::text,payload,created_at FROM records WHERE id=$1`, id).Scan(&record.ID, &record.Payload, &record.CreatedAt)
	return record, err
}
