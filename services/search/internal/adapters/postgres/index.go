package postgres

import (
	"context"
	"encoding/json"

	"github.com/drukleague/platform/services/search/internal/domain"
)

func (r *Repository) ApplyIndexEvent(ctx context.Context, event domain.IndexEvent) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `INSERT INTO search_processed_events(event_id) VALUES($1) ON CONFLICT DO NOTHING`, event.EventID)
	if err != nil {
		return err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if inserted == 0 {
		return tx.Commit()
	}

	result, err = tx.ExecContext(ctx, `
		INSERT INTO search_entity_versions(entity_type,entity_id,version,deleted,last_event_id)
		VALUES($1,$2,$3,$4,$5)
		ON CONFLICT(entity_type,entity_id) DO UPDATE
		SET version=EXCLUDED.version, deleted=EXCLUDED.deleted, last_event_id=EXCLUDED.last_event_id
		WHERE search_entity_versions.version < EXCLUDED.version`,
		event.EntityType, event.EntityID, event.Version, event.Operation == domain.OperationDelete, event.EventID)
	if err != nil {
		return err
	}
	advanced, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if advanced == 0 {
		return tx.Commit()
	}

	if event.Operation == domain.OperationDelete {
		if _, err := tx.ExecContext(ctx, `DELETE FROM search_documents WHERE entity_type=$1 AND entity_id=$2`, event.EntityType, event.EntityID); err != nil {
			return err
		}
	} else {
		content, err := json.Marshal(event.Fields)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO search_documents(entity_type,entity_id,content,updated_at,version,last_event_id)
			VALUES($1,$2,$3,$4,$5,$6)
			ON CONFLICT(entity_type,entity_id) DO UPDATE
			SET content=EXCLUDED.content, updated_at=EXCLUDED.updated_at,
				version=EXCLUDED.version, last_event_id=EXCLUDED.last_event_id
			WHERE search_documents.version <= EXCLUDED.version`,
			event.EntityType, event.EntityID, string(content), event.OccurredAt, event.Version, event.EventID); err != nil {
			return err
		}
	}
	return tx.Commit()
}
