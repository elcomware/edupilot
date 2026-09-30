// Package money implements EduPilot's money representation: integer minor
// units plus a currency. Floating point is never used, in this package or in
// any caller of it. See docs/adr/ADR-005.
package money

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// ErrCurrencyMismatch is returned when an operation mixes currencies. EduPilot
// never converts implicitly; an exchange rate is an explicit, recorded act.
var ErrCurrencyMismatch = errors.New("money: currency mismatch")

// ErrInvalidCurrency is returned for an unknown or malformed currency code.
var ErrInvalidCurrency = errors.New("money: invalid currency")

// ErrInvalidAmount is returned for a malformed or unrepresentable amount.
var ErrInvalidAmount = errors.New("money: invalid amount")

// ErrDivisionByZero is returned by ratio operations with a zero denominator.
var ErrDivisionByZero = errors.New("money: division by zero")

// ErrOverflow is returned when a result cannot be represented in int64 minor
// units. Silent overflow in a ledger is unacceptable, so it is an error.
var ErrOverflow = errors.New("money: amount overflow")

// Currency identifies a currency and its minor unit scaling.
type Currency struct {
	code     string
	exponent uint8
}

// Registry of supported currencies. XOF and other zero-decimal currencies store
// whole units; EUR, USD and GBP store hundredths.
var currencies = map[string]Currency{
	"XOF": {code: "XOF", exponent: 0},
	"XAF": {code: "XAF", exponent: 0},
	"XPF": {code: "XPF", exponent: 0},
	"JPY": {code: "JPY", exponent: 0},
	"KRW": {code: "KRW", exponent: 0},
	"CLP": {code: "CLP", exponent: 0},
	"EUR": {code: "EUR", exponent: 2},
	"USD": {code: "USD", exponent: 2},
	"GBP": {code: "GBP", exponent: 2},
	"CAD": {code: "CAD", exponent: 2},
	"AUD": {code: "AUD", exponent: 2},
	"CHF": {code: "CHF", exponent: 2},
	"NGN": {code: "NGN", exponent: 2},
	"GHS": {code: "GHS", exponent: 2},
	"KES": {code: "KES", exponent: 2},
	"ZAR": {code: "ZAR", exponent: 2},
	"MAD": {code: "MAD", exponent: 2},
	"SEN": {code: "SEN", exponent: 2},
	"CIV": {code: "CIV", exponent: 0},
	"INR": {code: "INR", exponent: 2},
}

// LookupCurrency resolves an ISO 4217 code. Lookups are case-insensitive.
func LookupCurrency(code string) (Currency, error) {
	c, found := currencies[strings.ToUpper(strings.TrimSpace(code))]
	if !found {
		return Currency{}, fmt.Errorf("%w: %q", ErrInvalidCurrency, code)
	}
	return c, nil
}

// MustCurrency resolves a currency code and panics on failure. It is for
// constants and seed data, never for user input.
func MustCurrency(code string) Currency {
	c, err := LookupCurrency(code)
	if err != nil {
		panic(err)
	}
	return c
}

// Code returns the ISO 4217 code.
func (c Currency) Code() string { return c.code }

// Exponent returns the number of decimal places in the minor unit.
func (c Currency) Exponent() uint8 { return c.exponent }

// Scale returns 10^Exponent, the factor between major and minor units.
func (c Currency) Scale() int64 {
	scale := int64(1)
	for i := uint8(0); i < c.exponent; i++ {
		scale *= 10
	}
	return scale
}

// Money is an amount in the minor units of a single currency.
type Money struct {
	minor    int64
	currency Currency
}

// Zero returns zero in the given currency.
func Zero(c Currency) Money { return Money{currency: c} }

// FromMinor builds a Money from minor units. It never fails: an amount is any
// int64, and only arithmetic can overflow.
func FromMinor(amount int64, c Currency) Money { return Money{minor: amount, currency: c} }

// Minor returns the amount in minor units.
func (m Money) Minor() int64 { return m.minor }

// Currency returns the currency of the amount.
func (m Money) Currency() Currency { return m.currency }

// IsZero reports whether the amount is exactly zero.
func (m Money) IsZero() bool { return m.minor == 0 }

// IsNegative reports whether the amount is below zero.
func (m Money) IsNegative() bool { return m.minor < 0 }

// IsPositive reports whether the amount is above zero.
func (m Money) IsPositive() bool { return m.minor > 0 }

// Sign returns -1, 0 or 1.
func (m Money) Sign() int {
	switch {
	case m.minor < 0:
		return -1
	case m.minor > 0:
		return 1
	default:
		return 0
	}
}

// Add returns the exact sum. Amounts in different currencies are never added.
func (m Money) Add(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	sum, overflow := addInt64(m.minor, other.minor)
	if overflow {
		return Money{}, ErrOverflow
	}
	return Money{minor: sum, currency: m.currency}, nil
}

// MustAdd is Add for amounts already known to share a currency.
func (m Money) MustAdd(other Money) Money {
	sum, err := m.Add(other)
	if err != nil {
		panic(err)
	}
	return sum
}

// Sub returns the exact difference.
func (m Money) Sub(other Money) (Money, error) {
	if err := m.sameCurrency(other); err != nil {
		return Money{}, err
	}
	difference, overflow := subInt64(m.minor, other.minor)
	if overflow {
		return Money{}, ErrOverflow
	}
	return Money{minor: difference, currency: m.currency}, nil
}

// MustSub is Sub for amounts already known to share a currency.
func (m Money) MustSub(other Money) Money {
	difference, err := m.Sub(other)
	if err != nil {
		panic(err)
	}
	return difference
}

// Negate returns the amount with the opposite sign. Negating math.MinInt64
// overflows and is reported as ErrOverflow.
func (m Money) Negate() (Money, error) {
	if m.minor == math.MinInt64 {
		return Money{}, ErrOverflow
	}
	return Money{minor: -m.minor, currency: m.currency}, nil
}

// Abs returns the absolute amount.
func (m Money) Abs() Money {
	if m.minor < 0 {
		return Money{minor: -m.minor, currency: m.currency}
	}
	return m
}

// Cmp compares two amounts of the same currency, returning -1, 0 or 1.
func (m Money) Cmp(other Money) (int, error) {
	if err := m.sameCurrency(other); err != nil {
		return 0, err
	}
	switch {
	case m.minor < other.minor:
		return -1, nil
	case m.minor > other.minor:
		return 1, nil
	default:
		return 0, nil
	}
}

// Mul multiplies by a whole quantity. The result is exact; no rounding occurs
// because a fractional quantity is never implicit in EduPilot.
func (m Money) Mul(quantity int64) (Money, error) {
	if quantity == 0 {
		return Zero(m.currency), nil
	}
	product, overflow := mulInt64(m.minor, quantity)
	if overflow {
		return Money{}, ErrOverflow
	}
	return Money{minor: product, currency: m.currency}, nil
}

// MustMul is Mul for quantities known to be safe.
func (m Money) MustMul(quantity int64) Money {
	product, err := m.Mul(quantity)
	if err != nil {
		panic(err)
	}
	return product
}

// Rounding selects how a ratio result is reduced to whole minor units.
type Rounding int

const (
	// RoundHalfEven rounds halfway cases to the nearest even minor unit. It is
	// the default because it does not bias totals over many operations.
	RoundHalfEven Rounding = iota
	// RoundHalfUp rounds halfway cases away from zero.
	RoundHalfUp
	// RoundDown truncates toward zero.
	RoundDown
	// RoundUp rounds away from zero.
	RoundUp
)

// MulRatio returns m * num / den with explicit rounding.
func (m Money) MulRatio(num, den int64, rounding Rounding) (Money, error) {
	if den == 0 {
		return Money{}, ErrDivisionByZero
	}
	return Money{minor: scale(m.minor, num, den, rounding), currency: m.currency}, nil
}

// Percentage returns the given percentage of the amount, expressed in basis
// points. 250 bps is 2.5%.
func (m Money) Percentage(basisPoints int32, rounding Rounding) (Money, error) {
	return m.MulRatio(int64(basisPoints), 10_000, rounding)
}

// Split divides the amount into the given number of parts without losing or
// inventing a single minor unit. Remainder minor units are distributed one each
// to the earliest parts, which is deterministic and auditable.
func (m Money) Split(parts int) ([]Money, error) {
	if parts <= 0 {
		return nil, fmt.Errorf("money: split into %d parts", parts)
	}

	amount := m.minor
	quotient := amount / int64(parts)
	remainder := amount % int64(parts)

	out := make([]Money, parts)
	for i := range out {
		share := quotient
		if int64(i) < abs64(remainder) {
			if amount >= 0 {
				share++
			} else {
				share--
			}
		}
		out[i] = Money{minor: share, currency: m.currency}
	}
	return out, nil
}

// Sum adds every amount, requiring a single currency.
func Sum(amounts ...Money) (Money, error) {
	if len(amounts) == 0 {
		return Money{}, fmt.Errorf("money: cannot sum an empty set")
	}

	total := amounts[0]
	for _, amount := range amounts[1:] {
		next, err := total.Add(amount)
		if err != nil {
			return Money{}, err
		}
		total = next
	}
	return total, nil
}

// String renders the amount with its currency code, for example "250000 XOF"
// or "1234.56 EUR".
func (m Money) String() string {
	return format(m.minor, m.currency) + " " + m.currency.code
}

// Format renders the amount without a currency code, using thousands
// separators, for example "1,234.56".
func (m Money) Format() string {
	return format(m.minor, m.currency)
}

// MarshalJSON emits {amountMinor, currency}, matching the transport contract
// shared with the frontend.
func (m Money) MarshalJSON() ([]byte, error) {
	return []byte(`{"amountMinor":` + strconv.FormatInt(m.minor, 10) +
		`,"currency":"` + m.currency.code + `"}`), nil
}

// UnmarshalJSON reads the {amountMinor, currency} transport shape.
func (m *Money) UnmarshalJSON(data []byte) error {
	var wire struct {
		AmountMinor int64  `json:"amountMinor"`
		Currency    string `json:"currency"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return fmt.Errorf("%w: %s", ErrInvalidAmount, err)
	}

	currency, err := LookupCurrency(wire.Currency)
	if err != nil {
		return err
	}
	m.minor = wire.AmountMinor
	m.currency = currency
	return nil
}

// Parse reads a decimal amount and a currency, for example "250000 XOF",
// "1234.56 EUR" or "-12.5 USD". Parsing is exact: no float is involved.
func Parse(input string) (Money, error) {
	fields := strings.Fields(strings.TrimSpace(input))
	if len(fields) != 2 {
		return Money{}, fmt.Errorf("%w: %q", ErrInvalidAmount, input)
	}

	currency, err := LookupCurrency(fields[1])
	if err != nil {
		return Money{}, err
	}

	minor, err := parseDecimal(fields[0], currency)
	if err != nil {
		return Money{}, err
	}
	return Money{minor: minor, currency: currency}, nil
}

func (m Money) sameCurrency(other Money) error {
	if m.currency != other.currency {
		return fmt.Errorf("%w: %s and %s", ErrCurrencyMismatch, m.currency.code, other.currency.code)
	}
	return nil
}

func parseDecimal(text string, c Currency) (int64, error) {
	text = strings.ReplaceAll(strings.TrimSpace(text), ",", "")
	if text == "" {
		return 0, fmt.Errorf("%w: empty amount", ErrInvalidAmount)
	}

	sign := int64(1)
	switch text[0] {
	case '-':
		sign = -1
		text = text[1:]
	case '+':
		text = text[1:]
	}

	whole, fraction, _ := strings.Cut(text, ".")
	if whole == "" {
		whole = "0"
	}
	if len(fraction) > int(c.exponent) {
		return 0, fmt.Errorf("%w: %q has more precision than %s allows", ErrInvalidAmount, text, c.code)
	}

	wholeValue, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %q", ErrInvalidAmount, text)
	}

	fraction += strings.Repeat("0", int(c.exponent)-len(fraction))
	fractionValue := int64(0)
	if fraction != "" {
		fractionValue, err = strconv.ParseInt(fraction, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %q", ErrInvalidAmount, text)
		}
	}

	scaled, overflow := mulInt64(wholeValue, c.Scale())
	if overflow {
		return 0, ErrOverflow
	}
	total, overflow := addInt64(scaled, fractionValue)
	if overflow {
		return 0, ErrOverflow
	}
	return sign * total, nil
}

func format(minor int64, c Currency) string {
	negative := minor < 0
	absolute := abs64(minor)

	if c.exponent == 0 {
		digits := groupThousands(strconv.FormatInt(absolute, 10))
		if negative {
			return "-" + digits
		}
		return digits
	}

	scale := c.Scale()
	whole := absolute / scale
	fraction := absolute % scale

	grouped := groupThousands(strconv.FormatInt(whole, 10))
	fractionText := fmt.Sprintf("%0*d", int(c.exponent), fraction)

	if negative {
		return "-" + grouped + "." + fractionText
	}
	return grouped + "." + fractionText
}

func groupThousands(digits string) string {
	if len(digits) <= 3 {
		return digits
	}

	var out strings.Builder
	lead := len(digits) % 3
	if lead > 0 {
		out.WriteString(digits[:lead])
	}
	for i := lead; i < len(digits); i += 3 {
		if out.Len() > 0 {
			out.WriteByte(',')
		}
		out.WriteString(digits[i : i+3])
	}
	return out.String()
}

func scale(amount, num, den int64, rounding Rounding) int64 {
	if amount == 0 || num == 0 {
		return 0
	}

	negative := false
	product := amount
	if product < 0 {
		product = -product
		negative = true
	}
	multiplier := num
	if multiplier < 0 {
		multiplier = -multiplier
		negative = !negative
	}

	value, overflow := mulInt64(product, multiplier)
	if overflow {
		return math.MaxInt64
	}

	quotient := value / den
	remainder := value % den
	twiceRemainder := remainder * 2

	switch rounding {
	case RoundDown:
		// truncate
	case RoundUp:
		if remainder > 0 {
			quotient++
		}
	case RoundHalfUp:
		if twiceRemainder >= den {
			quotient++
		}
	case RoundHalfEven:
		if twiceRemainder > den || (twiceRemainder == den && quotient%2 == 1) {
			quotient++
		}
	}

	if negative {
		return -quotient
	}
	return quotient
}

func abs64(v int64) int64 {
	if v < 0 {
		return -v
	}
	return v
}

func addInt64(a, b int64) (int64, bool) {
	sum := a + b
	if (a > 0 && b > 0 && sum < 0) || (a < 0 && b < 0 && sum >= 0) {
		return 0, true
	}
	return sum, false
}

func subInt64(a, b int64) (int64, bool) {
	difference := a - b
	if (a >= 0 && b < 0 && difference < 0) || (a < 0 && b > 0 && difference >= 0) {
		return 0, true
	}
	return difference, false
}

func mulInt64(a, b int64) (int64, bool) {
	if a == 0 || b == 0 {
		return 0, false
	}
	product := a * b
	if product/b != a {
		return 0, true
	}
	return product, false
}
