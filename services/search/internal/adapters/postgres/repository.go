package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/drukleague/platform/services/search/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) Search(ctx context.Context, query string, entityTypes []string, limit, offset int) ([]domain.Hit, error) {
	statement := `SELECT entity_type, entity_id::text, content
		FROM search_documents
		WHERE strpos(lower(content), lower($1)) > 0`
	args := []any{query}
	if len(entityTypes) > 0 {
		placeholders := make([]string, 0, len(entityTypes))
		for _, entityType := range entityTypes {
			args = append(args, entityType)
			placeholders = append(placeholders, "$"+strconv.Itoa(len(args)))
		}
		statement += " AND entity_type IN (" + strings.Join(placeholders, ",") + ")"
	}
	args = append(args, limit, offset)
	limitPosition, offsetPosition := len(args)-1, len(args)
	statement += " ORDER BY updated_at DESC, entity_type, entity_id LIMIT $" + strconv.Itoa(limitPosition) + " OFFSET $" + strconv.Itoa(offsetPosition)

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
