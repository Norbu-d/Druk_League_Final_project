package postgres

import (
	"context"
	"errors"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/packages/servicekit/testdb"
	"github.com/drukleague/platform/services/report/internal/domain"
)

func TestRepositoryPersistsAndReadsReportExport(t *testing.T) {
	repo := New(testdb.New(t, "../../../migrations"))
	item, err := repo.Create(context.Background(), domain.Report{
		ReportType: "standings", Status: "complete", RequesterID: "11111111-1111-4111-8111-111111111111",
		TournamentID: "22222222-2222-4222-8222-222222222222", Format: "csv", FileName: "standings.csv",
		ContentType: "text/csv; charset=utf-8", Content: []byte("team_name,points\nThimphu FC,3\n"),
	})
	if err != nil || item.ID == "" || item.CreatedAt.IsZero() {
		t.Fatalf("create report: item=%+v err=%v", item, err)
	}
	read, err := repo.Get(context.Background(), item.ID)
	if err != nil || read.ID != item.ID || read.Status != "complete" || read.TournamentID != item.TournamentID || string(read.Content) != string(item.Content) {
		t.Fatalf("read report: item=%+v err=%v", read, err)
	}
	if _, err := repo.Get(context.Background(), "33333333-3333-4333-8333-333333333333"); !errors.Is(err, core.ErrNotFound) {
		t.Fatalf("missing report error = %v, want not found", err)
	}
}
