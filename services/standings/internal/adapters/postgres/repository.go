package postgres

import (
	"context"
	"database/sql"

	"github.com/drukleague/platform/services/standings/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

func (r *Repository) TeamKnown(ctx context.Context, tournamentID, teamID string) (bool, error) {
	var exists bool
	err := r.db.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM standings WHERE tournament_id=$1 AND team_id=$2 AND team_name<>'')`, tournamentID, teamID).Scan(&exists)
	return exists, err
}

func (r *Repository) ApplyFinalized(ctx context.Context, event domain.MatchFinalized, home, away domain.Team) (bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return false, err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(ctx, `INSERT INTO processed_events(event_id) VALUES($1) ON CONFLICT(event_id) DO NOTHING`, event.EventID)
	if err != nil {
		return false, err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return false, err
	}
	if inserted == 0 {
		return false, tx.Commit()
	}
	applySide := func(team domain.Team, gf, ga int32) error {
		won, drawn, lost, points := domain.Outcome(gf, ga)
		_, err := tx.ExecContext(ctx, `INSERT INTO standings(tournament_id,team_id,team_name,dzongkhag,played,won,drawn,lost,goals_for,goals_against,points) VALUES($1,$2,$3,$4,1,$5,$6,$7,$8,$9,$10) ON CONFLICT(tournament_id,team_id) DO UPDATE SET team_name=CASE WHEN EXCLUDED.team_name<>'' THEN EXCLUDED.team_name ELSE standings.team_name END,dzongkhag=CASE WHEN EXCLUDED.dzongkhag<>'' THEN EXCLUDED.dzongkhag ELSE standings.dzongkhag END,played=standings.played+1,won=standings.won+EXCLUDED.won,drawn=standings.drawn+EXCLUDED.drawn,lost=standings.lost+EXCLUDED.lost,goals_for=standings.goals_for+EXCLUDED.goals_for,goals_against=standings.goals_against+EXCLUDED.goals_against,points=standings.points+EXCLUDED.points`, event.TournamentID, team.ID, team.Name, team.Dzongkhag, won, drawn, lost, gf, ga, points)
		return err
	}
	if err = applySide(home, event.HomeScore, event.AwayScore); err != nil {
		return false, err
	}
	if err = applySide(away, event.AwayScore, event.HomeScore); err != nil {
		return false, err
	}
	return true, tx.Commit()
}

func (r *Repository) Get(ctx context.Context, tournamentID string) ([]domain.Standing, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT team_id::text,team_name,dzongkhag,played,won,drawn,lost,goals_for,goals_against,goals_for-goals_against,points FROM standings WHERE tournament_id=$1 ORDER BY points DESC,(goals_for-goals_against) DESC,goals_for DESC,team_name`, tournamentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Standing, 0)
	for rows.Next() {
		var item domain.Standing
		if err = rows.Scan(&item.TeamID, &item.TeamName, &item.Dzongkhag, &item.Played, &item.Won, &item.Drawn, &item.Lost, &item.GoalsFor, &item.GoalsAgainst, &item.GoalDifference, &item.Points); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
