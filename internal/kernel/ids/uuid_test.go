package ids

import (
	"testing"
	"time"
)

func TestGenerateParses(t *testing.T) {
	u := New()

	if u.IsNil() {
		t.Fatal("generated UUID is nil")
	}
	if u.Version() != 7 {
		t.Fatalf("version = %d, want 7", u.Version())
	}
	if u.Variant() != 1 {
		t.Fatalf("variant = %d, want 1", u.Variant())
	}

	parsed, err := Parse(u.String())
	if err != nil {
		t.Fatalf("round trip failed: %v", err)
	}
	if parsed != u {
		t.Fatalf("round trip mismatch: %s != %s", parsed, u)
	}
}

func TestTimestampIsCurrent(t *testing.T) {
	before := time.Now().UTC().Add(-time.Second)
	u := New()
	after := time.Now().UTC().Add(time.Second)

	got := u.Timestamp()
	if got.Before(before) || got.After(after) {
		t.Fatalf("timestamp %s outside [%s, %s]", got, before, after)
	}
}

func TestGeneratorIsMonotonicWithinMillisecond(t *testing.T) {
	g := NewGenerator()

	previous := g.New()
	for i := 0; i < 10_000; i++ {
		current := g.New()
		if compare(current, previous) <= 0 {
			t.Fatalf("identifier %d (%s) is not greater than %s", i, current, previous)
		}
		previous = current
	}
}

func TestParseRejectsMalformed(t *testing.T) {
	cases := []string{
		"",
		"not-a-uuid",
		"00000000-0000-0000-0000-00000000000",
		"00000000-0000-0000-0000-00000000000g",
		"0000000000000000000000000000000",
	}

	for _, input := range cases {
		if _, err := Parse(input); err == nil {
			t.Fatalf("Parse(%q) succeeded, want error", input)
		}
	}
}

func TestNilUUID(t *testing.T) {
	if !Nil.IsNil() {
		t.Fatal("Nil.IsNil() = false")
	}
	if !Nil.Timestamp().IsZero() {
		t.Fatal("Nil.Timestamp() is not zero")
	}
	if got := Nil.String(); got != "00000000-0000-0000-0000-000000000000" {
		t.Fatalf("Nil.String() = %s", got)
	}
}

func compare(a, b UUID) int {
	for i := range a {
		switch {
		case a[i] < b[i]:
			return -1
		case a[i] > b[i]:
			return 1
		}
	}
	return 0
}
