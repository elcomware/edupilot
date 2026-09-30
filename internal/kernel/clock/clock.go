// Package clock abstracts the system clock so that time-dependent rules —
// effective-dated fees, fiscal periods, audit timestamps, idempotency windows —
// are testable and deterministic.
package clock

import "time"

// Clock reports the current time. It is a port: the domain depends on it, and
// infrastructure supplies the implementation.
type Clock interface {
	Now() time.Time
}

// System is the production clock, returning UTC.
type System struct{}

// Now returns the current UTC time.
func (System) Now() time.Time { return time.Now().UTC() }

// Fixed is a clock frozen at a given instant, for tests and for reproducible
// seed data.
type Fixed struct {
	instant time.Time
}

// NewFixed returns a clock frozen at the given instant.
func NewFixed(instant time.Time) Fixed {
	return Fixed{instant: instant.UTC()}
}

// Now returns the frozen instant.
func (f Fixed) Now() time.Time { return f.instant }

// Advance moves the frozen clock forward.
func (f Fixed) Advance(d time.Duration) Fixed {
	return Fixed{instant: f.instant.Add(d)}
}

// Set moves the frozen clock to an absolute instant.
func (f Fixed) Set(instant time.Time) Fixed {
	return Fixed{instant: instant.UTC()}
}
