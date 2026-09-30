package money

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

func TestZeroDecimalCurrency(t *testing.T) {
	amount := FromMinor(250_000, MustCurrency("XOF"))

	if got, want := amount.String(), "250,000 XOF"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
	if got, want := amount.Minor(), int64(250_000); got != want {
		t.Fatalf("Minor() = %d, want %d", got, want)
	}
}

func TestTwoDecimalCurrency(t *testing.T) {
	amount := FromMinor(123_456, MustCurrency("EUR"))

	if got, want := amount.String(), "1,234.56 EUR"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestParseRoundTrip(t *testing.T) {
	cases := []struct {
		input string
		minor int64
	}{
		{"250000 XOF", 250_000},
		{"1234.56 EUR", 123_456},
		{"-12.5 USD", -1_250},
		{"0.01 GBP", 1},
		{"1000 NGN", 100_000},
	}

	for _, tc := range cases {
		parsed, err := Parse(tc.input)
		if err != nil {
			t.Fatalf("Parse(%q): %v", tc.input, err)
		}
		if parsed.Minor() != tc.minor {
			t.Fatalf("Parse(%q).Minor() = %d, want %d", tc.input, parsed.Minor(), tc.minor)
		}
	}
}

func TestParseRejectsExcessPrecision(t *testing.T) {
	if _, err := Parse("10.005 XOF"); !errors.Is(err, ErrInvalidAmount) {
		t.Fatalf("err = %v, want ErrInvalidAmount", err)
	}
}

func TestAddRejectsMixedCurrencies(t *testing.T) {
	_, err := FromMinor(100, MustCurrency("XOF")).Add(FromMinor(100, MustCurrency("EUR")))
	if !errors.Is(err, ErrCurrencyMismatch) {
		t.Fatalf("err = %v, want ErrCurrencyMismatch", err)
	}
}

func TestAddIsExact(t *testing.T) {
	// 0.1 + 0.2 must be exactly 0.3 in minor units, which a float cannot promise.
	a := FromMinor(1, MustCurrency("USD"))
	b := FromMinor(2, MustCurrency("USD"))

	sum, err := a.Add(b)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Minor() != 3 {
		t.Fatalf("Minor() = %d, want 3", sum.Minor())
	}
	if sum.Currency().Code() != "USD" {
		t.Fatalf("currency = %s, want USD", sum.Currency().Code())
	}
	if !sum.MustAdd(Zero(MustCurrency("USD"))).Equal(sum) {
		t.Fatal("adding zero changed the amount")
	}
}

func TestOverflowIsReported(t *testing.T) {
	eur := MustCurrency("EUR")
	huge := FromMinor(1<<62, eur)

	if _, err := huge.Add(huge); !errors.Is(err, ErrOverflow) {
		t.Fatalf("err = %v, want ErrOverflow", err)
	}
	if _, err := huge.Mul(1 << 10); !errors.Is(err, ErrOverflow) {
		t.Fatalf("Mul err = %v, want ErrOverflow", err)
	}
}

func TestMulRatioRounding(t *testing.T) {
	amount := FromMinor(1, MustCurrency("USD"))

	half, err := amount.MulRatio(1, 2, RoundHalfEven)
	if err != nil {
		t.Fatal(err)
	}
	if half.Minor() != 0 {
		t.Fatalf("RoundHalfEven(0.5) = %d, want 0", half.Minor())
	}

	up, err := amount.MulRatio(1, 2, RoundHalfUp)
	if err != nil {
		t.Fatal(err)
	}
	if up.Minor() != 1 {
		t.Fatalf("RoundHalfUp(0.5) = %d, want 1", up.Minor())
	}

	down, err := amount.MulRatio(1, 3, RoundDown)
	if err != nil {
		t.Fatal(err)
	}
	if down.Minor() != 0 {
		t.Fatalf("RoundDown(0.33) = %d, want 0", down.Minor())
	}
}

func TestPercentage(t *testing.T) {
	// 2.5% of 200000 XOF is 5000 XOF exactly.
	amount := FromMinor(200_000, MustCurrency("XOF"))

	quarter, err := amount.Percentage(250, RoundHalfEven)
	if err != nil {
		t.Fatal(err)
	}
	if quarter.Minor() != 5_000 {
		t.Fatalf("2.5%% of 200000 = %d, want 5000", quarter.Minor())
	}
}

func TestSplitLosesNoMinorUnits(t *testing.T) {
	amount := FromMinor(100, MustCurrency("XOF"))

	parts, err := amount.Split(3)
	if err != nil {
		t.Fatal(err)
	}
	if len(parts) != 3 {
		t.Fatalf("len(parts) = %d, want 3", len(parts))
	}

	total, err := Sum(parts...)
	if err != nil {
		t.Fatal(err)
	}
	if total.Minor() != amount.Minor() {
		t.Fatalf("sum = %d, want %d", total.Minor(), amount.Minor())
	}
	if parts[0].Minor() != 34 || parts[1].Minor() != 33 || parts[2].Minor() != 33 {
		t.Fatalf("unexpected split: %d %d %d", parts[0].Minor(), parts[1].Minor(), parts[2].Minor())
	}
}

func TestSplitNegativeAmount(t *testing.T) {
	amount := FromMinor(-100, MustCurrency("XOF"))

	parts, err := amount.Split(3)
	if err != nil {
		t.Fatal(err)
	}

	total, err := Sum(parts...)
	if err != nil {
		t.Fatal(err)
	}
	if total.Minor() != -100 {
		t.Fatalf("sum = %d, want -100", total.Minor())
	}
}

func TestJSONRoundTrip(t *testing.T) {
	original := FromMinor(250_000, MustCurrency("XOF"))

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := string(data), `{"amountMinor":250000,"currency":"XOF"}`; got != want {
		t.Fatalf("json = %s, want %s", got, want)
	}

	var decoded Money
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded != original {
		t.Fatalf("decoded = %v, want %v", decoded, original)
	}
}

func TestCurrencyLookupIsCaseInsensitive(t *testing.T) {
	if _, err := LookupCurrency(" xof "); err != nil {
		t.Fatalf("LookupCurrency: %v", err)
	}
	if _, err := LookupCurrency("XYZ"); !errors.Is(err, ErrInvalidCurrency) {
		t.Fatalf("err = %v, want ErrInvalidCurrency", err)
	}
}

func TestStringGroupsThousands(t *testing.T) {
	amount := FromMinor(123_456_789, MustCurrency("EUR"))
	if got, want := amount.String(), "1,234,567.89 EUR"; got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}
}

func TestNegateAndAbs(t *testing.T) {
	amount := FromMinor(500, MustCurrency("USD"))

	negated, err := amount.Negate()
	if err != nil {
		t.Fatal(err)
	}
	if negated.Minor() != -500 {
		t.Fatalf("Negate() = %d, want -500", negated.Minor())
	}
	if !negated.Abs().Equal(amount) {
		t.Fatal("Abs() did not restore the original amount")
	}
}

func (m Money) Equal(other Money) bool {
	return m.minor == other.minor && m.currency == other.currency
}

func TestParseRejectsGarbage(t *testing.T) {
	for _, input := range []string{"", "abc", "100", "100 XYZ", "1.2.3 EUR"} {
		if _, err := Parse(strings.TrimSpace(input)); err == nil {
			t.Fatalf("Parse(%q) succeeded, want error", input)
		}
	}
}
