package domain

import (
	"strings"
	"time"

	"github.com/drukleague/platform/packages/servicekit/core"
)

// MaxCapacity bounds the seating figure; the largest ground in Bhutan holds about 25,000.
const MaxCapacity = 100000

type Venue struct {
	ID, Name, Dzongkhag string
	Capacity            int32
	CreatedAt           time.Time
}

// Normalize trims input and checks the venue rules shared by every use case.
func (v Venue) Normalize() (Venue, error) {
	v.Name, v.Dzongkhag = strings.Join(strings.Fields(v.Name), " "), strings.TrimSpace(v.Dzongkhag)
	switch {
	case len(v.Name) < 2 || len(v.Name) > 120:
		return v, core.Invalid("venue name must be 2 to 120 characters")
	case v.Dzongkhag == "":
		return v, core.Invalid("dzongkhag is required")
	case v.Capacity < 0 || v.Capacity > MaxCapacity:
		return v, core.Invalid("capacity must be between 0 and 100000")
	}
	return v, nil
}
