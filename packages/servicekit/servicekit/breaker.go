package servicekit

import (
	"sync"
	"time"
)

// BreakerState is the state of a circuit breaker.
type BreakerState int

const (
	BreakerClosed BreakerState = iota
	BreakerOpen
	BreakerHalfOpen
)

func (s BreakerState) String() string {
	switch s {
	case BreakerOpen:
		return "open"
	case BreakerHalfOpen:
		return "half-open"
	default:
		return "closed"
	}
}

// CircuitBreaker opens after a run of consecutive failures, rejects calls while open,
// and lets one probe through after the cool-down. A successful probe closes it again.
type CircuitBreaker struct {
	threshold int
	coolDown  time.Duration
	now       func() time.Time

	mu       sync.Mutex
	state    BreakerState
	failures int
	openedAt time.Time
	probing  bool
}

func NewCircuitBreaker(threshold int, coolDown time.Duration) *CircuitBreaker {
	if threshold <= 0 {
		threshold = 5
	}
	if coolDown <= 0 {
		coolDown = 10 * time.Second
	}
	return &CircuitBreaker{threshold: threshold, coolDown: coolDown, now: time.Now}
}

// Allow reports whether a call may proceed. In half-open state only one probe is allowed at a time.
func (b *CircuitBreaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case BreakerOpen:
		if b.now().Sub(b.openedAt) < b.coolDown {
			return false
		}
		b.state = BreakerHalfOpen
		b.probing = true
		return true
	case BreakerHalfOpen:
		if b.probing {
			return false
		}
		b.probing = true
		return true
	default:
		return true
	}
}

// Record reports the outcome of an allowed call; failure means the dependency itself failed.
func (b *CircuitBreaker) Record(failure bool) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if !failure {
		b.state, b.failures, b.probing = BreakerClosed, 0, false
		return
	}
	b.failures++
	if b.state == BreakerHalfOpen || b.failures >= b.threshold {
		b.state, b.openedAt, b.probing = BreakerOpen, b.now(), false
	}
}

func (b *CircuitBreaker) State() BreakerState {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.state
}
