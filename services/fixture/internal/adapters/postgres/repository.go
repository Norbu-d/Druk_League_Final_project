package postgres

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/fixture/internal/domain"
)

type Repository struct{ db *sql.DB }

func New(db *sql.DB) *Repository { return &Repository{db: db} }

// SaveSchedule serialises generation per tournament with an advisory lock so two admins
// pressing "Generate" at once cannot create two schedules.
func (r *Repository) SaveSchedule(ctx context.Context, tournamentID string, fixtures []domain.Fixture, event domain.OutboxEvent) ([]domain.Fixture, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('fixtures:'||$1))`, tournamentID); err != nil {
		return nil, false, err
	}
	var existing int
	if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM fixtures WHERE tournament_id=$1`, tournamentID).Scan(&existing); err != nil {
		return nil, false, err
	}
	created := existing == 0
	if created {
		for _, f := range fixtures {
			if _, err = tx.ExecContext(ctx, `INSERT INTO fixtures(tournament_id,home_team_id,away_team_id,venue_id,venue_name,round,kickoff,format) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8)`,
				f.TournamentID, f.HomeTeamID, f.AwayTeamID, f.VenueID, f.VenueName, f.Round, f.Kickoff, f.Format); err != nil {
				return nil, false, err
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO event_outbox(event_id,payload) VALUES($1,$2::jsonb)`, event.EventID, string(event.Payload)); err != nil {
			return nil, false, err
		}
	}
	all, err := list(ctx, tx, tournamentID, 10000, 0)
	if err != nil {
		return nil, false, err
	}
	return all, created, tx.Commit()
}

func (r *Repository) SaveKnockoutSchedule(ctx context.Context, tournamentID string, plan domain.KnockoutPlan, event domain.OutboxEvent) ([]domain.Fixture, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, false, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('fixtures:'||$1))`, tournamentID); err != nil {
		return nil, false, err
	}
	var exists bool
	if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM fixture_bracket_state WHERE tournament_id=$1)`, tournamentID).Scan(&exists); err != nil {
		return nil, false, err
	}
	created := !exists
	if created {
		if _, err = tx.ExecContext(ctx, `INSERT INTO fixture_bracket_state(tournament_id,bracket_size,total_rounds) VALUES($1,$2,$3)`, tournamentID, plan.BracketSize, plan.Rounds); err != nil {
			return nil, false, err
		}
		for _, f := range plan.Fixtures {
			if _, err = tx.ExecContext(ctx, `INSERT INTO fixtures(tournament_id,home_team_id,away_team_id,venue_id,venue_name,round,kickoff,format,bracket_match,home_seed,away_seed) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,$10,$11)`,
				f.TournamentID, f.HomeTeamID, f.AwayTeamID, f.VenueID, f.VenueName, f.Round, f.Kickoff, domain.FormatKnockout, f.BracketMatch, f.HomeSeed, f.AwaySeed); err != nil {
				return nil, false, err
			}
		}
		for _, bye := range plan.Byes {
			if _, err = tx.ExecContext(ctx, `INSERT INTO fixture_bracket_entries(tournament_id,round,match_no,slot_no,team_id,seed) VALUES($1,$2,$3,$4,$5,$6)`, tournamentID, bye.Round, bye.Match, bye.Slot, bye.TeamID, bye.Seed); err != nil {
				return nil, false, err
			}
		}
		if _, err = tx.ExecContext(ctx, `INSERT INTO event_outbox(event_id,payload) VALUES($1,$2::jsonb)`, event.EventID, string(event.Payload)); err != nil {
			return nil, false, err
		}
	}
	all, err := list(ctx, tx, tournamentID, 10000, 0)
	if err != nil {
		return nil, false, err
	}
	stored := make([]domain.Fixture, 0, len(all))
	for _, fixture := range all {
		if fixture.Format == domain.FormatKnockout {
			stored = append(stored, fixture)
		}
	}
	return stored, created, tx.Commit()
}

// AdvanceKnockout records each finalized match once and creates its parent match only
// after both child matches have supplied a winner.
func (r *Repository) AdvanceKnockout(ctx context.Context, event domain.MatchFinalized) (domain.Fixture, bool, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.Fixture{}, false, err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `SELECT pg_advisory_xact_lock(hashtext('fixtures:'||$1))`, event.TournamentID); err != nil {
		return domain.Fixture{}, false, err
	}
	var current domain.Fixture
	var kickoff sql.NullTime
	err = tx.QueryRowContext(ctx, `SELECT id::text,tournament_id::text,home_team_id::text,away_team_id::text,COALESCE(venue_id::text,''),venue_name,COALESCE(match_id::text,''),round,kickoff,format,COALESCE(bracket_match,0),COALESCE(home_seed,0),COALESCE(away_seed,0) FROM fixtures WHERE match_id=$1 AND tournament_id=$2`, event.MatchID, event.TournamentID).Scan(
		&current.ID, &current.TournamentID, &current.HomeTeamID, &current.AwayTeamID, &current.VenueID, &current.VenueName, &current.MatchID,
		&current.Round, &kickoff, &current.Format, &current.BracketMatch, &current.HomeSeed, &current.AwaySeed)
	if errors.Is(err, sql.ErrNoRows) {
		return domain.Fixture{}, false, tx.Commit()
	}
	if err != nil {
		return domain.Fixture{}, false, err
	}
	current.Kickoff = kickoff.Time
	if event.HomeTeamID != current.HomeTeamID || event.AwayTeamID != current.AwayTeamID {
		return domain.Fixture{}, false, errors.New("MatchFinalized teams do not match the knockout fixture")
	}
	if current.Format != domain.FormatLeagueKnockout && current.Format != domain.FormatKnockout {
		return domain.Fixture{}, false, tx.Commit()
	}
	result, err := tx.ExecContext(ctx, `INSERT INTO fixture_match_results(event_id,match_id,tournament_id,home_team_id,away_team_id,home_score,away_score,finalized_at) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`,
		event.EventID, event.MatchID, event.TournamentID, event.HomeTeamID, event.AwayTeamID, event.HomeScore, event.AwayScore, event.FinalizedAt)
	if err != nil {
		return domain.Fixture{}, false, err
	}
	inserted, err := result.RowsAffected()
	if err != nil {
		return domain.Fixture{}, false, err
	}
	if current.Format == domain.FormatLeagueKnockout {
		var bracketExists bool
		if err = tx.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM fixture_bracket_state WHERE tournament_id=$1)`, event.TournamentID).Scan(&bracketExists); err != nil {
			return domain.Fixture{}, false, err
		}
		if bracketExists {
			return domain.Fixture{}, false, tx.Commit()
		}
		var unfinished int
		if err = tx.QueryRowContext(ctx, `SELECT count(*) FROM fixtures f LEFT JOIN fixture_match_results r ON r.match_id=f.match_id WHERE f.tournament_id=$1 AND f.format=$2 AND (f.match_id IS NULL OR r.match_id IS NULL)`, event.TournamentID, domain.FormatLeagueKnockout).Scan(&unfinished); err != nil {
			return domain.Fixture{}, false, err
		}
		if unfinished == 0 {
			return domain.Fixture{TournamentID: event.TournamentID, Format: domain.FormatLeagueKnockout}, true, tx.Commit()
		}
		return domain.Fixture{}, false, tx.Commit()
	}
	if current.Format != domain.FormatKnockout {
		return domain.Fixture{}, false, tx.Commit()
	}
	var totalRounds int32
	if err = tx.QueryRowContext(ctx, `SELECT total_rounds FROM fixture_bracket_state WHERE tournament_id=$1`, event.TournamentID).Scan(&totalRounds); err != nil {
		return domain.Fixture{}, false, err
	}
	if inserted == 0 {
		if current.Round >= totalRounds {
			return domain.Fixture{}, false, tx.Commit()
		}
		nextRound := current.Round + 1
		nextMatch := (current.BracketMatch + 1) / 2
		var pending domain.Fixture
		var pendingKickoff sql.NullTime
		err = tx.QueryRowContext(ctx, `SELECT id::text,tournament_id::text,home_team_id::text,away_team_id::text,COALESCE(venue_id::text,''),venue_name,COALESCE(match_id::text,''),round,kickoff,format,COALESCE(bracket_match,0),COALESCE(home_seed,0),COALESCE(away_seed,0) FROM fixtures WHERE tournament_id=$1 AND round=$2 AND bracket_match=$3 AND match_id IS NULL`, event.TournamentID, nextRound, nextMatch).Scan(
			&pending.ID, &pending.TournamentID, &pending.HomeTeamID, &pending.AwayTeamID, &pending.VenueID, &pending.VenueName, &pending.MatchID,
			&pending.Round, &pendingKickoff, &pending.Format, &pending.BracketMatch, &pending.HomeSeed, &pending.AwaySeed)
		if errors.Is(err, sql.ErrNoRows) {
			return domain.Fixture{}, false, tx.Commit()
		}
		if err != nil {
			return domain.Fixture{}, false, err
		}
		pending.Kickoff = pendingKickoff.Time
		return pending, true, tx.Commit()
	}
	if current.Round >= totalRounds {
		return domain.Fixture{}, false, tx.Commit()
	}
	winner, seed := current.HomeTeamID, current.HomeSeed
	if event.AwayScore > event.HomeScore || (event.AwayScore == event.HomeScore && current.AwaySeed < current.HomeSeed) {
		winner, seed = current.AwayTeamID, current.AwaySeed
	}
	nextRound := current.Round + 1
	nextMatch := (current.BracketMatch + 1) / 2
	slot := (current.BracketMatch-1)%2 + 1
	if _, err = tx.ExecContext(ctx, `INSERT INTO fixture_bracket_entries(tournament_id,round,match_no,slot_no,team_id,seed) VALUES($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`, event.TournamentID, nextRound, nextMatch, slot, winner, seed); err != nil {
		return domain.Fixture{}, false, err
	}
	rows, err := tx.QueryContext(ctx, `SELECT slot_no,team_id::text,seed FROM fixture_bracket_entries WHERE tournament_id=$1 AND round=$2 AND match_no=$3 ORDER BY slot_no`, event.TournamentID, nextRound, nextMatch)
	if err != nil {
		return domain.Fixture{}, false, err
	}
	var teams [2]string
	var seeds [2]int32
	count := 0
	for rows.Next() {
		var slotNo int32
		var teamID string
		var teamSeed int32
		if err = rows.Scan(&slotNo, &teamID, &teamSeed); err != nil {
			if closeErr := rows.Close(); closeErr != nil {
				return domain.Fixture{}, false, errors.Join(err, closeErr)
			}
			return domain.Fixture{}, false, err
		}
		teams[slotNo-1], seeds[slotNo-1] = teamID, teamSeed
		count++
	}
	if err = rows.Err(); err != nil {
		if closeErr := rows.Close(); closeErr != nil {
			return domain.Fixture{}, false, errors.Join(err, closeErr)
		}
		return domain.Fixture{}, false, err
	}
	if err = rows.Close(); err != nil {
		return domain.Fixture{}, false, err
	}
	if count < 2 {
		return domain.Fixture{}, false, tx.Commit()
	}
	fixture := domain.Fixture{TournamentID: event.TournamentID, HomeTeamID: teams[0], AwayTeamID: teams[1], VenueID: current.VenueID, VenueName: current.VenueName,
		Round: nextRound, Kickoff: current.Kickoff.AddDate(0, 0, 7), Format: domain.FormatKnockout, BracketMatch: nextMatch, HomeSeed: seeds[0], AwaySeed: seeds[1]}
	if err = tx.QueryRowContext(ctx, `INSERT INTO fixtures(tournament_id,home_team_id,away_team_id,venue_id,venue_name,round,kickoff,format,bracket_match,home_seed,away_seed) VALUES($1,$2,$3,NULLIF($4,'')::uuid,$5,$6,$7,$8,$9,$10,$11) RETURNING id::text`,
		fixture.TournamentID, fixture.HomeTeamID, fixture.AwayTeamID, fixture.VenueID, fixture.VenueName, fixture.Round, fixture.Kickoff, fixture.Format, fixture.BracketMatch, fixture.HomeSeed, fixture.AwaySeed).Scan(&fixture.ID); err != nil {
		return domain.Fixture{}, false, err
	}
	eventID, err := core.NewUUID()
	if err != nil {
		return domain.Fixture{}, false, err
	}
	payload, err := json.Marshal(domain.FixturesGenerated{EventID: eventID, TournamentID: event.TournamentID, Fixtures: 1, GeneratedAt: time.Now().UTC()})
	if err != nil {
		return domain.Fixture{}, false, err
	}
	if _, err = tx.ExecContext(ctx, `INSERT INTO event_outbox(event_id,payload) VALUES($1,$2::jsonb)`, eventID, string(payload)); err != nil {
		return domain.Fixture{}, false, err
	}
	return fixture, true, tx.Commit()
}

func (r *Repository) LinkMatch(ctx context.Context, fixtureID, matchID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE fixtures SET match_id=$2 WHERE id=$1 AND match_id IS NULL`, fixtureID, matchID)
	return err
}

func (r *Repository) List(ctx context.Context, tournamentID string, limit, offset int) ([]domain.Fixture, bool, error) {
	items, err := list(ctx, r.db, tournamentID, limit+1, offset)
	if err != nil {
		return nil, false, err
	}
	if len(items) > limit {
		return items[:limit], true, nil
	}
	return items, false, nil
}

func (r *Repository) PendingFixtures(ctx context.Context, tournamentID string) ([]domain.Fixture, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT id::text,tournament_id::text,home_team_id::text,away_team_id::text,COALESCE(venue_id::text,''),venue_name,COALESCE(match_id::text,''),round,kickoff,format,COALESCE(bracket_match,0),COALESCE(home_seed,0),COALESCE(away_seed,0) FROM fixtures WHERE tournament_id=$1 AND format=$2 AND match_id IS NULL ORDER BY round,bracket_match`, tournamentID, domain.FormatKnockout)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]domain.Fixture, 0)
	for rows.Next() {
		var fixture domain.Fixture
		var kickoff sql.NullTime
		if err := rows.Scan(&fixture.ID, &fixture.TournamentID, &fixture.HomeTeamID, &fixture.AwayTeamID, &fixture.VenueID, &fixture.VenueName, &fixture.MatchID,
			&fixture.Round, &kickoff, &fixture.Format, &fixture.BracketMatch, &fixture.HomeSeed, &fixture.AwaySeed); err != nil {
			return nil, err
		}
		fixture.Kickoff = kickoff.Time
		items = append(items, fixture)
	}
	return items, rows.Err()
}

type querier interface {
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
}

func list(ctx context.Context, q querier, tournamentID string, limit, offset int) ([]domain.Fixture, error) {
	rows, err := q.QueryContext(ctx, `SELECT id::text,tournament_id::text,home_team_id::text,away_team_id::text,COALESCE(venue_id::text,''),venue_name,COALESCE(match_id::text,''),round,kickoff,format,COALESCE(bracket_match,0),COALESCE(home_seed,0),COALESCE(away_seed,0) FROM fixtures WHERE tournament_id=$1 ORDER BY round,kickoff,id LIMIT $2 OFFSET $3`, tournamentID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []domain.Fixture{}
	for rows.Next() {
		var f domain.Fixture
		var kickoff sql.NullTime
		if err := rows.Scan(&f.ID, &f.TournamentID, &f.HomeTeamID, &f.AwayTeamID, &f.VenueID, &f.VenueName, &f.MatchID, &f.Round, &kickoff, &f.Format, &f.BracketMatch, &f.HomeSeed, &f.AwaySeed); err != nil {
			return nil, err
		}
		f.Kickoff = kickoff.Time
		items = append(items, f)
	}
	return items, rows.Err()
}

func (r *Repository) PendingEvents(ctx context.Context, limit int) ([]domain.OutboxEvent, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT event_id::text,payload::text FROM event_outbox WHERE published_at IS NULL ORDER BY created_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	events := []domain.OutboxEvent{}
	for rows.Next() {
		var event domain.OutboxEvent
		var payload string
		if err := rows.Scan(&event.EventID, &payload); err != nil {
			return nil, err
		}
		event.Payload = []byte(payload)
		events = append(events, event)
	}
	return events, rows.Err()
}

func (r *Repository) MarkPublished(ctx context.Context, eventID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE event_outbox SET published_at=now() WHERE event_id=$1 AND published_at IS NULL`, eventID)
	return err
}

func (r *Repository) PrunePublished(ctx context.Context, before time.Time) (int64, error) {
	result, err := r.db.ExecContext(ctx, `DELETE FROM event_outbox WHERE published_at IS NOT NULL AND published_at < $1`, before)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected()
}
