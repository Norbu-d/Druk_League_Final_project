package domain

import (
	"errors"
	"testing"
	"time"
)

func TestClashNamesWhatIsBusy(t *testing.T) {
	bhutan := time.FixedZone("BTT", 6*60*60)
	err := Clash("venue Changlimithang", time.Date(2026, 11, 1, 10, 0, 0, 0, bhutan))
	if !errors.Is(err, ErrConflict) || err.Error() != "match scheduling conflict: venue Changlimithang already has a match at 2026-11-01 04:00 UTC" {
		t.Fatalf("got %v", err)
	}
}
