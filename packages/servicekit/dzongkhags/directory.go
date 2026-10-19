// Package dzongkhags is a cached client for the Dzongkhag reference-data service. The 20
// districts rarely change, so lookups are served from memory and a stale copy is used
// while the Dzongkhag service is unavailable (graceful degradation).
package dzongkhags

import (
	"context"
	"strings"
	"sync"
	"time"

	dzongkhagv1 "github.com/drukleague/platform/packages/contracts/gen/drukleague/dzongkhag/v1"
	"github.com/drukleague/platform/packages/servicekit/core"
	"google.golang.org/grpc"
)

type Directory struct {
	client dzongkhagv1.DzongkhagServiceClient
	ttl    time.Duration
	now    func() time.Time

	mu      sync.Mutex
	byKey   map[string]string // lower-case name or code -> canonical name
	fetched time.Time
}

func New(conn grpc.ClientConnInterface, ttl time.Duration) *Directory {
	if ttl <= 0 {
		ttl = 10 * time.Minute
	}
	return &Directory{client: dzongkhagv1.NewDzongkhagServiceClient(conn), ttl: ttl, now: time.Now}
}

// Canonical returns the official spelling for a dzongkhag name or code.
func (d *Directory) Canonical(ctx context.Context, nameOrCode string) (string, error) {
	key := strings.ToLower(strings.TrimSpace(nameOrCode))
	if key == "" {
		return "", core.Invalid("dzongkhag is required")
	}
	names, err := d.load(ctx)
	if err != nil {
		return "", err
	}
	canonical, ok := names[key]
	if !ok {
		return "", core.Invalid("dzongkhag must be one of the 20 Bhutanese districts")
	}
	return canonical, nil
}

func (d *Directory) load(ctx context.Context) (map[string]string, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if d.byKey != nil && d.now().Sub(d.fetched) < d.ttl {
		return d.byKey, nil
	}
	response, err := d.client.ListDzongkhags(ctx, &dzongkhagv1.ListDzongkhagsRequest{})
	if err != nil || len(response.GetDzongkhags()) == 0 {
		if d.byKey != nil {
			return d.byKey, nil // stale but still correct for fixed reference data
		}
		return nil, core.Dependency("dzongkhag directory unavailable")
	}
	byKey := make(map[string]string, 2*len(response.GetDzongkhags()))
	for _, item := range response.GetDzongkhags() {
		byKey[strings.ToLower(item.GetName())] = item.GetName()
		byKey[strings.ToLower(item.GetCode())] = item.GetName()
	}
	d.byKey, d.fetched = byKey, d.now()
	return byKey, nil
}
