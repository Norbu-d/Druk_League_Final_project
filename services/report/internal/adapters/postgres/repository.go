package postgres

import (
	"context"
	"database/sql"
	"errors"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/report/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

const createReport = `INSERT INTO report_jobs(report_type,status,requester_id,tournament_id,format,file_name,content_type,content)
	VALUES($1,$2,$3,$4,$5,$6,$7,$8)
	RETURNING id::text,report_type,status,requester_id::text,tournament_id::text,format,file_name,content_type,content,created_at`

const getReport = `SELECT id::text,report_type,status,requester_id::text,tournament_id::text,format,file_name,content_type,content,created_at
	FROM report_jobs WHERE id=$1`

func (r *Repository) Create(ctx context.Context, item domain.Report) (domain.Report, error) {
	created, err := scan(r.db.QueryRowContext(ctx, createReport, item.ReportType, item.Status, item.RequesterID, item.TournamentID, item.Format, item.FileName, item.ContentType, item.Content))
	return created, err
}

func (r *Repository) Get(ctx context.Context, id string) (domain.Report, error) {
	item, err := scan(r.db.QueryRowContext(ctx, getReport, id))
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Report{}, core.NotFound("report not found")
	}
	return item, err
}

type rowScanner interface{ Scan(...any) error }

func scan(row rowScanner) (domain.Report, error) {
	var item domain.Report
	err := row.Scan(&item.ID, &item.ReportType, &item.Status, &item.RequesterID, &item.TournamentID, &item.Format, &item.FileName, &item.ContentType, &item.Content, &item.CreatedAt)
	return item, err
}
