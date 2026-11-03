// Command e2e is an end-to-end smoke test of the split services. It runs inside the Compose
// network (service ports are not published) and exercises Auth, User, Tournament, Team,
// Venue, Fixture, Official, Match, Score, Standings and Notification, including NATS event flow.
//
//	docker run --rm --network deploy_default -v D:/druk-league:/src -w /src/tools/e2e \
//	  -e E2E_ADMIN_PASSWORD=... drukleague-service-builder:local go run .
//
// It creates uniquely named test data on every run (there are no delete RPCs).
package main

import (
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	authv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/auth/v1"
	fixturev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/fixture/v1"
	notificationv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/notification/v1"
	officialv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/official/v1"
	standingsv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/standings/v1"
	v1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/v1"
	venuev1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/venue/v1"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

var failures int

func check(name string, ok bool, detail ...any) {
	if ok {
		fmt.Printf("PASS  %s\n", name)
		return
	}
	failures++
	fmt.Printf("FAIL  %s %v\n", name, detail)
}

// must stops the run when a setup step fails; later checks would only cascade.
func must[T any](value T, err error) T {
	if err != nil {
		_, _, line, _ := runtime.Caller(1)
		fmt.Printf("FAIL  step at main.go:%d: %v\n", line, err)
		os.Exit(1)
	}
	return value
}

func dial(host string) *grpc.ClientConn {
	return must(grpc.NewClient(host+":50051", grpc.WithTransportCredentials(insecure.NewCredentials())))
}

// runCtx bounds the whole run so a hung dependency cannot block it forever.
var runCtx, cancelRun = context.WithTimeout(context.Background(), 5*time.Minute)

func as(token string) context.Context {
	return metadata.AppendToOutgoingContext(runCtx, "authorization", "Bearer "+token)
}

func code(err error) codes.Code { return status.Code(err) }

func main() {
	adminUser := env("E2E_ADMIN_USER", "admin")
	adminPassword := env("E2E_ADMIN_PASSWORD", "drukleague-dev")
	runStarted := time.Now().UTC()
	run := runStarted.Format("0102-150405")
	defer cancelRun()
	bg := runCtx

	auth := authv1.NewAuthServiceClient(dial("auth"))
	users := v1.NewUserServiceClient(dial("user"))
	tournaments := v1.NewTournamentServiceClient(dial("tournament"))
	teams := v1.NewTeamServiceClient(dial("team"))
	venues := venuev1.NewVenueServiceClient(dial("venue"))
	fixtures := fixturev1.NewFixtureServiceClient(dial("fixture"))
	officials := officialv1.NewOfficialServiceClient(dial("official"))
	matches := v1.NewMatchServiceClient(dial("match"))
	scores := v1.NewScoreServiceClient(dial("score"))
	standings := standingsv1.NewStandingsServiceClient(dial("standings"))
	notifications := notificationv1.NewNotificationServiceClient(dial("notification"))

	// --- Auth (Part 4)
	_, err := auth.Login(bg, &authv1.LoginRequest{Username: adminUser, Password: "definitely-wrong"})
	check("login with wrong password is Unauthenticated", code(err) == codes.Unauthenticated, err)
	login := must(auth.Login(bg, &authv1.LoginRequest{Username: adminUser, Password: adminPassword}))
	check("admin login returns access + refresh token + session", login.GetAccessToken() != "" && login.GetRefreshToken() != "" && login.GetSessionId() != "" && login.GetUser().GetRole() == "super_admin")
	refreshed := must(auth.RefreshToken(bg, &authv1.RefreshTokenRequest{RefreshToken: login.GetRefreshToken()}))
	check("refresh rotates the refresh token", refreshed.GetRefreshToken() != login.GetRefreshToken())
	_, err = auth.RefreshToken(bg, &authv1.RefreshTokenRequest{RefreshToken: login.GetRefreshToken()})
	check("old refresh token is rejected after rotation", code(err) == codes.Unauthenticated, err)
	admin := refreshed.GetAccessToken()
	_, err = teams.ListTeams(bg, &v1.ListTeamsRequest{})
	check("call without token is Unauthenticated", code(err) == codes.Unauthenticated, err)
	locked := "lockout-" + run
	for i := 0; i < 5; i++ {
		_, _ = auth.Login(bg, &authv1.LoginRequest{Username: locked, Password: "wrong-password"})
	}
	_, err = auth.Login(bg, &authv1.LoginRequest{Username: locked, Password: "wrong-password"})
	check("sixth failed login is locked out (FailedPrecondition)", code(err) == codes.FailedPrecondition, err)

	// --- User: official account
	officialName := "official-" + run
	official := must(users.CreateUser(as(admin), &v1.CreateUserRequest{Username: officialName, Password: "official-password-1", Role: "match_official"})).GetUser()
	_, err = users.CreateUser(as(admin), &v1.CreateUserRequest{Username: "boss-" + run, Password: "long-enough-password", Role: "super_admin"})
	check("admin cannot create another super admin", code(err) == codes.InvalidArgument, err)
	officialLogin := must(auth.Login(bg, &authv1.LoginRequest{Username: officialName, Password: "official-password-1"}))
	officialToken := officialLogin.GetAccessToken()
	_, err = users.ListUsers(as(officialToken), &v1.ListUsersRequest{})
	check("official cannot list users", code(err) == codes.PermissionDenied, err)

	// --- Team manager accounts used to verify MatchFinalized notifications
	managerTokens := make([]string, 4)
	managerIDs := make([]string, 4)
	for i := range managerTokens {
		managerName := fmt.Sprintf("manager-%d-%s", i, run)
		manager := must(users.CreateUser(as(admin), &v1.CreateUserRequest{Username: managerName, Password: "manager-password-1", Role: "team_manager"})).GetUser()
		managerLogin := must(auth.Login(bg, &authv1.LoginRequest{Username: managerName, Password: "manager-password-1"}))
		managerIDs[i], managerTokens[i] = manager.GetId(), managerLogin.GetAccessToken()
	}

	// --- Venue (Part 3)
	venue := must(venues.CreateVenue(as(admin), &venuev1.CreateVenueRequest{Name: "E2E Ground " + run, Dzongkhag: "thimphu", Capacity: 1500})).GetVenue()
	check("venue dzongkhag is canonicalised via Dzongkhag service", venue.GetDzongkhag() == "Thimphu", venue.GetDzongkhag())
	_, err = venues.CreateVenue(as(admin), &venuev1.CreateVenueRequest{Name: "Nowhere", Dzongkhag: "Atlantis"})
	check("venue in unknown dzongkhag is rejected", code(err) == codes.InvalidArgument, err)
	_, err = venues.CreateVenue(as(officialToken), &venuev1.CreateVenueRequest{Name: "Other", Dzongkhag: "Paro"})
	check("official cannot create venue", code(err) == codes.PermissionDenied, err)

	// --- Tournament + teams
	tournament := must(tournaments.CreateTournament(as(admin), &v1.CreateTournamentRequest{Name: "E2E Cup " + run, Sport: "football", Season: "2026", Dzongkhag: "Thimphu"})).GetTournament()
	teamIDs := []string{}
	for i, name := range []string{"Thimphu", "Paro", "Punakha", "Haa"} {
		team := must(teams.CreateTeam(as(managerTokens[i]), &v1.CreateTeamRequest{Name: name + " E2E " + run, ShortCode: fmt.Sprintf("E%d%s", i, run[len(run)-3:]), Dzongkhag: name, TournamentId: tournament.GetId()})).GetTeam()
		teamIDs = append(teamIDs, team.GetId())
		check("team is owned by its creator", team.GetManagerId() == managerIDs[i], team.GetManagerId())
	}

	// --- Fixture generation creates matches (Part 3)
	start := runStarted.AddDate(0, 3, 0).Truncate(time.Second)
	generated := must(fixtures.GenerateFixtures(as(admin), &fixturev1.GenerateFixturesRequest{TournamentId: tournament.GetId(), StartsAt: timestamppb.New(start)}))
	check("4 teams produce 6 fixtures, all linked to matches", generated.GetCreatedCount() == 6 && len(generated.GetFixtures()) == 6 && allLinked(generated.GetFixtures()), generated.GetCreatedCount())
	districtVenues := must(venues.ListVenues(as(admin), &venuev1.ListVenuesRequest{Dzongkhag: tournament.GetDzongkhag(), PageSize: 100}))
	venueIDs := make(map[string]bool, len(districtVenues.GetVenues()))
	for _, item := range districtVenues.GetVenues() {
		venueIDs[item.GetId()] = true
	}
	fixtureVenueID := ""
	if len(generated.GetFixtures()) > 0 {
		fixtureVenueID = generated.GetFixtures()[0].GetVenueId()
	}
	check("fixtures use a venue in the tournament dzongkhag", fixtureVenueID != "" && venueIDs[fixtureVenueID], fixtureVenueID)
	again := must(fixtures.GenerateFixtures(as(admin), &fixturev1.GenerateFixturesRequest{TournamentId: tournament.GetId()}))
	check("second generation creates nothing new", again.GetCreatedCount() == 0 && len(again.GetFixtures()) == 6)
	listed := must(matches.ListMatches(as(admin), &v1.ListMatchesRequest{TournamentId: tournament.GetId()}))
	check("Match service holds the 6 generated matches", len(listed.GetMatches()) == 6, len(listed.GetMatches()))
	matchID := generated.GetFixtures()[0].GetMatchId()

	// --- Official assignment (Part 3)
	assignment := must(officials.AssignOfficial(as(admin), &officialv1.AssignOfficialRequest{MatchId: matchID, OfficialUserId: official.GetId()})).GetAssignment()
	check("assignment recorded", assignment.GetOfficialUserId() == official.GetId())
	match := must(matches.GetMatch(as(admin), &v1.GetMatchRequest{Id: matchID})).GetMatch()
	check("Match service now authorises this official", match.GetOfficialId() == official.GetId())
	mine := must(officials.ListAssignments(as(officialToken), &officialv1.ListAssignmentsRequest{}))
	check("official sees exactly their assignment", len(mine.GetAssignments()) == 1 && mine.GetAssignments()[0].GetMatchId() == matchID, len(mine.GetAssignments()))
	byTournament := must(officials.ListAssignments(as(admin), &officialv1.ListAssignmentsRequest{TournamentId: tournament.GetId()}))
	check("assignments can be filtered by tournament", len(byTournament.GetAssignments()) == 1)

	// --- Scoring, finalisation and the event flow
	_, err = scores.UpdateLiveScore(as(admin), &v1.UpdateLiveScoreRequest{MatchId: matchID, HomeScore: 1})
	check("admin cannot score (assigned official only)", code(err) == codes.PermissionDenied, err)
	_, err = scores.UpdateLiveScore(as(officialToken), &v1.UpdateLiveScoreRequest{MatchId: matchID, HomeScore: 100})
	check("score above 99 is rejected", code(err) == codes.InvalidArgument, err)
	must(scores.UpdateLiveScore(as(officialToken), &v1.UpdateLiveScoreRequest{MatchId: matchID, HomeScore: 3, AwayScore: 1}))
	final := must(scores.FinalizeMatch(as(officialToken), &v1.FinalizeMatchRequest{MatchId: matchID, Reason: "full time"})).GetScore()
	check("finalized 3-1", final.GetFinalized() && final.GetHomeScore() == 3 && final.GetAwayScore() == 1)
	_, err = scores.UpdateLiveScore(as(officialToken), &v1.UpdateLiveScoreRequest{MatchId: matchID, HomeScore: 0})
	check("finalized score is immutable", code(err) == codes.AlreadyExists, err)

	var table []*standingsv1.Standing
	completed := false
	matchTeams := map[string]bool{match.GetHomeTeamId(): true, match.GetAwayTeamId(): true}
	expectedManagers := map[string]bool{}
	for i, teamID := range teamIDs {
		if matchTeams[teamID] {
			expectedManagers[managerIDs[i]] = true
		}
	}
	managerNotices := make(map[string][]*notificationv1.Notification)
	notificationsReady := false
	for i := 0; i < 30 && (len(table) < 2 || !completed || !notificationsReady); i++ {
		time.Sleep(time.Second)
		if response, err := standings.GetStandings(as(admin), &standingsv1.GetStandingsRequest{TournamentId: tournament.GetId()}); err == nil {
			table = response.GetStandings()
		}
		if m, err := matches.GetMatch(as(admin), &v1.GetMatchRequest{Id: matchID}); err == nil {
			completed = m.GetMatch().GetStatus() == "completed"
		}
		notificationsReady = len(expectedManagers) == 2
		for i, managerID := range managerIDs {
			response, err := notifications.ListNotifications(as(managerTokens[i]), &notificationv1.ListNotificationsRequest{UserId: managerID, PageSize: 50})
			if err != nil {
				notificationsReady = false
				continue
			}
			managerNotices[managerID] = response.GetNotifications()
			if expectedManagers[managerID] && len(response.GetNotifications()) == 0 {
				notificationsReady = false
			}
		}
	}
	check("Match consumed MatchFinalized (status completed)", completed)
	check("Standings consumed MatchFinalized (2 rows)", len(table) == 2, len(table))
	managersNotified, unrelatedManagersQuiet := true, true
	for _, managerID := range managerIDs {
		items := managerNotices[managerID]
		if expectedManagers[managerID] {
			found := false
			for _, item := range items {
				if item.GetSubject() == "Match result finalized" && strings.Contains(item.GetBody(), "3–1") {
					found = true
				}
			}
			if !found {
				managersNotified = false
			}
		} else if len(items) != 0 {
			unrelatedManagersQuiet = false
		}
	}
	check("participating team managers receive MatchFinalized notice", managersNotified)
	check("other team managers receive no notice for this match", unrelatedManagersQuiet)
	if len(table) == 2 {
		check("winner has 3 points, played 1, team name from Team service", table[0].GetPoints() == 3 && table[0].GetPlayed() == 1 && table[0].GetTeamName() != "" && table[1].GetLost() == 1, table[0])
	}
	_, err = officials.AssignOfficial(as(admin), &officialv1.AssignOfficialRequest{MatchId: matchID, OfficialUserId: official.GetId()})
	check("completed match cannot be reassigned", code(err) == codes.AlreadyExists, err)

	// --- Logout
	_, err = auth.RevokeSession(as(officialToken), &authv1.RevokeSessionRequest{})
	check("official can revoke own session", err == nil, err)
	_, err = auth.RefreshToken(bg, &authv1.RefreshTokenRequest{RefreshToken: officialLogin.GetRefreshToken()})
	check("revoked session cannot refresh", code(err) == codes.Unauthenticated, err)

	fmt.Printf("\n%d checks failed\n", failures)
	if failures > 0 {
		os.Exit(1)
	}
}

func allLinked(items []*fixturev1.Fixture) bool {
	for _, f := range items {
		if f.GetMatchId() == "" {
			return false
		}
	}
	return true
}

func env(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}
