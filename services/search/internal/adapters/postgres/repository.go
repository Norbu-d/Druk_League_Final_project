package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"github.com/drukleague/platform/services/search/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Search(ctx context.Context, query string, entityTypes []string, limit, offset int) ([]domain.Hit, error) {
	var statement string
	var args []any
	switch len(entityTypes) {
	case 0:
		statement = `SELECT entity_type, entity_id::text, content
			FROM search_documents
			WHERE strpos(lower(content), lower($1)) > 0
			ORDER BY updated_at DESC, entity_type, entity_id LIMIT $2 OFFSET $3`
		args = []any{query, limit, offset}
	case 1:
		statement = `SELECT entity_type, entity_id::text, content
			FROM search_documents
			WHERE strpos(lower(content), lower($1)) > 0 AND entity_type = $2
			ORDER BY updated_at DESC, entity_type, entity_id LIMIT $3 OFFSET $4`
		args = []any{query, entityTypes[0], limit, offset}
	case 2:
		statement = `SELECT entity_type, entity_id::text, content
			FROM search_documents
			WHERE strpos(lower(content), lower($1)) > 0 AND entity_type IN ($2, $3)
			ORDER BY updated_at DESC, entity_type, entity_id LIMIT $4 OFFSET $5`
		args = []any{query, entityTypes[0], entityTypes[1], limit, offset}
	case 3:
		statement = `SELECT entity_type, entity_id::text, content
			FROM search_documents
			WHERE strpos(lower(content), lower($1)) > 0 AND entity_type IN ($2, $3, $4)
			ORDER BY updated_at DESC, entity_type, entity_id LIMIT $5 OFFSET $6`
		args = []any{query, entityTypes[0], entityTypes[1], entityTypes[2], limit, offset}
	default:
		return nil, errors.New("too many search entity types")
	}

	rows, err := r.db.QueryContext(ctx, statement, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	hits := make([]domain.Hit, 0, limit)
	for rows.Next() {
		var hit domain.Hit
		var content string
		if err := rows.Scan(&hit.EntityType, &hit.EntityID, &content); err != nil {
			return nil, err
		}
		hit.Title, hit.Subtitle = displayFields(content)
		hits = append(hits, hit)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return hits, nil
}

func displayFields(content string) (title, subtitle string) {
	var fields struct {
		Title    string `json:"title"`
		Subtitle string `json:"subtitle"`
	}
	if json.Unmarshal([]byte(content), &fields) == nil && strings.TrimSpace(fields.Title) != "" {
		return strings.TrimSpace(fields.Title), strings.TrimSpace(fields.Subtitle)
	}
	return strings.TrimSpace(content), ""
}
