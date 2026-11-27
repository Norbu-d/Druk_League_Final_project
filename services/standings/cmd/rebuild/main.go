package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	drukleaguev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	"github.com/drukleague/platform/packages/eventkit"
	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/servicekit"
	"github.com/drukleague/platform/services/standings/internal/domain"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/nats-io/nats.go"
)

type teamKey struct{ tournamentID, teamID string }
type totals struct {
	name, district                                           string
	played, won, drawn, lost, goalsFor, goalsAgainst, points int32
}

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	if err := rebuild(ctx); err != nil {
		log.Fatal(err)
	}
}

func rebuild(ctx context.Context) error {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return errors.New("DATABASE_URL is required")
	}
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return fmt.Errorf("open standings database: %w", err)
	}
	defer db.Close()
	if err := db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect standings database: %w", err)
	}

	teamConn, err := servicekit.Dial(env("TEAM_SERVICE_ADDR", "localhost:50051"), servicekit.ClientOptions{
		Auth: servicekit.ServiceIdentity, Tokens: servicekit.NewTokenSource("standings-service", servicekit.Secret(), 0), Timeout: 5 * time.Second,
	})
	if err != nil {
		return fmt.Errorf("connect Team service: %w", err)
	}
	defer teamConn.Close()
	teams := drukleaguev1.NewTeamServiceClient(teamConn)

	nc, js, err := eventkit.Connect(env("NATS_URL", "nats://localhost:4222"), "drukleague-standings-rebuild")
	if err != nil {
		return fmt.Errorf("connect NATS: %w", err)
	}
	defer nc.Close()
	info, err := js.StreamInfo(eventkit.EventsStream)
	if err != nil {
		return fmt.Errorf("read event stream: %w", err)
	}

	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	// The live consumer inserts into processed_events. Holding this lock prevents it from
	// applying events while the rebuild captures the stream tail and replaces the projection.
	if _, err = tx.ExecContext(ctx, `LOCK TABLE processed_events IN ACCESS EXCLUSIVE MODE`); err != nil {
		return fmt.Errorf("lock standings event processing: %w", err)
	}
	info, err = js.StreamInfo(eventkit.EventsStream)
	if err != nil {
		return fmt.Errorf("capture event stream tail: %w", err)
	}
	if info.State.Msgs == 0 {
		return errors.New("events stream is empty; refusing to clear standings")
	}
	if info.State.FirstSeq > 1 {
		return fmt.Errorf("event history begins at sequence %d; refusing an incomplete rebuild", info.State.FirstSeq)
	}
	first, last := info.State.FirstSeq, info.State.LastSeq
	events := make([]domain.MatchFinalized, 0)
	for sequence := first; info.State.Msgs > 0 && sequence <= last; sequence++ {
		message, getErr := js.GetMsg(eventkit.EventsStream, sequence)
		if errors.Is(getErr, nats.ErrMsgNotFound) {
			continue
		}
		if getErr != nil {
			return fmt.Errorf("read event sequence %d: %w", sequence, getErr)
		}
		if message.Subject != eventkit.SubjectMatchFinalized {
			continue
		}
		var event domain.MatchFinalized
		if err := json.Unmarshal(message.Data, &event); err != nil {
			return fmt.Errorf("decode MatchFinalized sequence %d: %w", sequence, err)
		}
		if !core.IsUUID(event.EventID) || !core.IsUUID(event.MatchID) || !core.IsUUID(event.TournamentID) || !core.IsUUID(event.HomeTeamID) || !core.IsUUID(event.AwayTeamID) || event.HomeTeamID == event.AwayTeamID || event.HomeScore < 0 || event.AwayScore < 0 {
			return fmt.Errorf("invalid MatchFinalized event at sequence %d", sequence)
		}
		events = append(events, event)
	}

	metadata := make(map[teamKey]domain.Team)
	aggregates := make(map[teamKey]*totals)
	seen := make(map[string]struct{}, len(events))
	for _, event := range events {
		if _, duplicate := seen[event.EventID]; duplicate {
			continue
		}
		seen[event.EventID] = struct{}{}
		home, err := getTeam(ctx, teams, metadata, event.TournamentID, event.HomeTeamID)
		if err != nil {
			return err
		}
		away, err := getTeam(ctx, teams, metadata, event.TournamentID, event.AwayTeamID)
		if err != nil {
			return err
		}
		apply(aggregates, event.TournamentID, home, event.HomeScore, event.AwayScore)
		apply(aggregates, event.TournamentID, away, event.AwayScore, event.HomeScore)
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM standings`); err != nil {
		return fmt.Errorf("clear standings projection: %w", err)
	}
	if _, err = tx.ExecContext(ctx, `DELETE FROM processed_events`); err != nil {
		return fmt.Errorf("clear processed event IDs: %w", err)
	}
	for eventID := range seen {
		if _, err = tx.ExecContext(ctx, `INSERT INTO processed_events(event_id) VALUES($1)`, eventID); err != nil {
			return fmt.Errorf("restore processed event ID: %w", err)
		}
	}
	for key, value := range aggregates {
		if _, err = tx.ExecContext(ctx, `INSERT INTO standings(tournament_id,team_id,team_name,dzongkhag,played,won,drawn,lost,goals_for,goals_against,points) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			key.tournamentID, key.teamID, value.name, value.district, value.played, value.won, value.drawn, value.lost, value.goalsFor, value.goalsAgainst, value.points); err != nil {
			return fmt.Errorf("write rebuilt standing for team %s: %w", key.teamID, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit rebuilt standings: %w", err)
	}
	fmt.Printf("rebuilt %d event IDs into %d team standings from stream sequences %d-%d\n", len(seen), len(aggregates), first, last)
	return nil
}

func getTeam(ctx context.Context, client drukleaguev1.TeamServiceClient, cache map[teamKey]domain.Team, tournamentID, teamID string) (domain.Team, error) {
	key := teamKey{tournamentID: tournamentID, teamID: teamID}
	if team, ok := cache[key]; ok {
		return team, nil
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	response, err := client.GetTeam(lookupCtx, &drukleaguev1.GetTeamRequest{Id: teamID})
	if err != nil {
		return domain.Team{}, fmt.Errorf("get team %s: %w", teamID, err)
	}
	p := response.GetTeam()
	if p == nil || p.GetId() != teamID || p.GetTournamentId() != tournamentID || p.GetName() == "" || p.GetDzongkhag() == "" {
		return domain.Team{}, fmt.Errorf("team %s is not valid for tournament %s", teamID, tournamentID)
	}
	team := domain.Team{ID: teamID, Name: p.GetName(), Dzongkhag: p.GetDzongkhag(), TournamentID: tournamentID}
	cache[key] = team
	return team, nil
}

func apply(rows map[teamKey]*totals, tournamentID string, team domain.Team, goalsFor, goalsAgainst int32) {
	key := teamKey{tournamentID: tournamentID, teamID: team.ID}
	value := rows[key]
	if value == nil {
		value = &totals{name: team.Name, district: team.Dzongkhag}
		rows[key] = value
	}
	won, drawn, lost, points := domain.Outcome(goalsFor, goalsAgainst)
	value.played++
	value.won += won
	value.drawn += drawn
	value.lost += lost
	value.goalsFor += goalsFor
	value.goalsAgainst += goalsAgainst
	value.points += points
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
