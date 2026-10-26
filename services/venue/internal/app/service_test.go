package app

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/drukleague/platform/packages/servicekit/core"
	"github.com/drukleague/platform/services/venue/internal/domain"
)

const venueID = "11111111-1111-4111-8111-111111111111"

type fakeRepo struct {
	created []domain.Venue
	filter  string
}

func (f *fakeRepo) Create(_ context.Context, v domain.Venue) (domain.Venue, error) {
	v.ID = venueID
	f.created = append(f.created, v)
	return v, nil
}
func (f *fakeRepo) Get(_ context.Context, id string) (domain.Venue, error) {
	if id != venueID {
		return domain.Venue{}, core.NotFound("venue not found")
	}
	return domain.Venue{ID: id}, nil
}
func (f *fakeRepo) List(_ context.Context, dzongkhag string, _, _ int) ([]domain.Venue, bool, error) {
	f.filter = dzongkhag
	return []domain.Venue{{ID: venueID}}, false, nil
}

type fakeDzongkhags struct{ err error }

func (f fakeDzongkhags) Canonical(_ context.Context, name string) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	if strings.EqualFold(strings.TrimSpace(name), "thimphu") {
		return "Thimphu", nil
	}
	return "", core.Invalid("unknown dzongkhag")
}

func TestCreateNormalizesAndCanonicalizes(t *testing.T) {
	repo := &fakeRepo{}
	v, err := New(repo, fakeDzongkhags{}).Create(context.Background(), core.RoleSuperAdmin, domain.Venue{Name: "  Changlimithang   Stadium ", Dzongkhag: "thimphu", Capacity: 15000})
	if err != nil {
		t.Fatal(err)
	}
	if v.Name != "Changlimithang Stadium" || v.Dzongkhag != "Thimphu" || v.ID != venueID {
		t.Fatalf("unexpected venue %+v", v)
	}
}

func TestCreateRejections(t *testing.T) {
	cases := []struct {
		name  string
		role  string
		venue domain.Venue
		dz    fakeDzongkhags
		want  error
	}{
		{"viewer", core.RoleViewer, domain.Venue{Name: "Ground", Dzongkhag: "Thimphu"}, fakeDzongkhags{}, core.ErrForbidden},
		{"short name", core.RoleSuperAdmin, domain.Venue{Name: "G", Dzongkhag: "Thimphu"}, fakeDzongkhags{}, core.ErrInvalid},
		{"long name", core.RoleSuperAdmin, domain.Venue{Name: strings.Repeat("x", 121), Dzongkhag: "Thimphu"}, fakeDzongkhags{}, core.ErrInvalid},
		{"no dzongkhag", core.RoleSuperAdmin, domain.Venue{Name: "Ground"}, fakeDzongkhags{}, core.ErrInvalid},
		{"negative capacity", core.RoleSuperAdmin, domain.Venue{Name: "Ground", Dzongkhag: "Thimphu", Capacity: -1}, fakeDzongkhags{}, core.ErrInvalid},
		{"huge capacity", core.RoleSuperAdmin, domain.Venue{Name: "Ground", Dzongkhag: "Thimphu", Capacity: domain.MaxCapacity + 1}, fakeDzongkhags{}, core.ErrInvalid},
		{"unknown district", core.RoleSuperAdmin, domain.Venue{Name: "Ground", Dzongkhag: "Atlantis"}, fakeDzongkhags{}, core.ErrInvalid},
		{"directory down", core.RoleSuperAdmin, domain.Venue{Name: "Ground", Dzongkhag: "Thimphu"}, fakeDzongkhags{err: core.Dependency("down")}, core.ErrDependency},
	}
	for _, c := range cases {
		if _, err := New(&fakeRepo{}, c.dz).Create(context.Background(), c.role, c.venue); !errors.Is(err, c.want) {
			t.Fatalf("%s: got %v want %v", c.name, err, c.want)
		}
	}
}

func TestGetAndList(t *testing.T) {
	repo := &fakeRepo{}
	s := New(repo, fakeDzongkhags{})
	if _, err := s.Get(context.Background(), "x"); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("bad id accepted")
	}
	if v, err := s.Get(context.Background(), venueID); err != nil || v.ID != venueID {
		t.Fatalf("get: %v", err)
	}
	if _, _, err := s.List(context.Background(), "thimphu", 10, 0); err != nil || repo.filter != "Thimphu" {
		t.Fatalf("list filter not canonicalised: %v %q", err, repo.filter)
	}
	if _, _, err := s.List(context.Background(), "", 10, 0); err != nil || repo.filter != "" {
		t.Fatalf("unfiltered list: %v", err)
	}
	if _, _, err := s.List(context.Background(), "Atlantis", 10, 0); !errors.Is(err, core.ErrInvalid) {
		t.Fatal("unknown filter accepted")
	}
}
