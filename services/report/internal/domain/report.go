package domain

import "time"

type Report struct {
	ID, ReportType, Status, RequesterID, TournamentID, Format string
	FileName, ContentType, DownloadURL                        string
	Content                                                   []byte
	CreatedAt                                                 time.Time
}

type Standing struct {
	TeamID, TeamName, Dzongkhag                    string
	Played, Won, Drawn, Lost                       int32
	GoalsFor, GoalsAgainst, GoalDifference, Points int32
}

type Fixture struct {
	ID, TournamentID, HomeTeamID, AwayTeamID, VenueID, MatchID string
	Kickoff                                                    time.Time
	Round                                                      int32
}

type CreateRequest struct {
	ReportType, TournamentID, Format string
}
