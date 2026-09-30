// Package ids provides globally unique, time-ordered UUIDv7 identifiers.
//
// EduPilot nodes create records independently — School A, School B, an offline
// Node C and EduPilot Cloud — and later converge. Identifiers therefore must be
// generatable without coordination, sortable by creation time, and free of any
// machine or volume information. See docs/adr/ADR-004.
package ids

import (
	"crypto/rand"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

// UUID is a canonical 128-bit identifier in its textual form.
type UUID [16]byte

// Nil is the zero UUID, used as the "not set" sentinel.
var Nil UUID

// String returns the canonical lowercase hyphenated representation.
func (u UUID) String() string {
	var buf [36]byte
	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], u[10:16])
	return string(buf[:])
}

// IsNil reports whether the UUID is the zero value.
func (u UUID) IsNil() bool {
	return u == Nil
}

// Timestamp returns the creation time encoded in the UUIDv7 timestamp field.
// It returns the zero time for a nil UUID.
func (u UUID) Timestamp() time.Time {
	if u.IsNil() {
		return time.Time{}
	}
	millis := int64(0)
	for _, b := range u[0:6] {
		millis = millis<<8 | int64(b)
	}
	return time.UnixMilli(millis).UTC()
}

// Version returns the UUID version nibble. UUIDv7 identifiers report 7.
func (u UUID) Version() int {
	return int(u[6] >> 4)
}

// Variant returns the RFC 4122 variant of the identifier.
func (u UUID) Variant() int {
	switch {
	case u[8]&0x80 == 0x00:
		return 0
	case u[8]&0xc0 == 0x80:
		return 1
	case u[8]&0xe0 == 0xc0:
		return 2
	default:
		return 3
	}
}

// Parse decodes the canonical textual form of a UUID.
func Parse(s string) (UUID, error) {
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return Nil, fmt.Errorf("ids: %q is not a canonical UUID", s)
	}

	digits := make([]byte, 0, 32)
	for i := 0; i < len(s); i++ {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			continue
		}
		digits = append(digits, s[i])
	}

	var u UUID
	if _, err := hex.Decode(u[:], digits); err != nil {
		return Nil, fmt.Errorf("ids: %q is not a canonical UUID: %w", s, err)
	}
	return u, nil
}

// MustParse decodes a canonical UUID and panics on failure. It is intended for
// constants in tests and seed data, never for user input.
func MustParse(s string) UUID {
	u, err := Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}

// Generator produces UUIDv7 identifiers. It is safe for concurrent use and
// preserves monotonicity within a single millisecond, which keeps generated
// indexes well ordered.
type Generator struct {
	mu       sync.Mutex
	lastMs   int64
	lastRand [10]byte
}

// NewGenerator returns a ready UUIDv7 generator.
func NewGenerator() *Generator {
	return &Generator{}
}

// New returns a new UUIDv7 identifier.
func (g *Generator) New() UUID {
	g.mu.Lock()
	defer g.mu.Unlock()

	millis := time.Now().UTC().UnixMilli()
	if millis == g.lastMs {
		incrementCounter(g.lastRand[:])
	} else {
		_, _ = rand.Read(g.lastRand[:])
	}
	g.lastMs = millis

	var u UUID
	var ts [8]byte
	binary.BigEndian.PutUint64(ts[:], uint64(millis))
	copy(u[0:6], ts[2:8]) // 48-bit millisecond timestamp

	u[6] = 0x70 | (g.lastRand[0] & 0x0F) // version 7
	u[7] = g.lastRand[1]
	u[8] = 0x80 | (g.lastRand[2] & 0x3F) // RFC 4122 variant
	copy(u[9:16], g.lastRand[3:10])

	return u
}

// New returns a new UUIDv7 identifier using a shared generator.
func New() UUID {
	return shared.New()
}

var shared = NewGenerator()

func incrementCounter(b []byte) {
	for i := len(b) - 1; i >= 0; i-- {
		b[i]++
		if b[i] != 0 {
			return
		}
	}
}
