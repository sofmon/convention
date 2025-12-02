package types

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

/* Currency */

type Currency struct {
	code   string
	factor int64
}

func newCurrency(code string, exponent int) Currency {
	return Currency{code: code, factor: int64(math.Pow10(exponent))}
}

func (c Currency) fromMinor(n int64) Money {
	return Money{amount: n, currency: c}
}

func (c Currency) fromMajor(n int64) Money {
	return Money{amount: n * c.factor, currency: c}
}

func (c Currency) Zero() Money {
	return Money{amount: 0, currency: c}
}

type eur struct{ Currency }
type usd struct{ Currency }
type jpy struct{ Currency }

var (
	EUR = eur{newCurrency("EUR", 2)}
	USD = usd{newCurrency("USD", 2)}
	JPY = jpy{newCurrency("JPY", 0)}
)

// EUR helpers

func (e eur) FromCents(cents int64) Money {
	return e.fromMinor(cents)
}

func (e eur) FromEuro(euros int64) Money {
	return e.fromMajor(euros)
}

// USD helpers

func (u usd) FromCents(cents int64) Money {
	return u.fromMinor(cents)
}

func (u usd) FromDollars(dollars int64) Money {
	return u.fromMajor(dollars)
}

/* Money */

type Money struct {
	amount   int64
	currency Currency
}

func (m Money) Currency() Currency {
	return m.currency
}

// Value returns the monetary value as a float64.
// Note: This may introduce floating-point precision issues for very large amounts
// and no currency information is retained and carries risk of mixing currencies.
// For normal operations prefer using the Money type directly.
func (m Money) Value() float64 {
	return float64(m.amount) / float64(m.currency.factor)
}

// internal helper, you can put it near the methods
func sameCurrency(m1, m2 Money) error {
	if m1.currency.code != m2.currency.code {
		return fmt.Errorf("currency mismatch: %s vs %s", m1.currency.code, m2.currency.code)
	}
	return nil
}

// JPY, no cents

func (j jpy) FromYen(yen int64) Money {
	return j.fromMinor(yen) // exponent zero
}

// arithmetic

func (a Money) TryAdd(b Money) (Money, error) {
	if err := sameCurrency(a, b); err != nil {
		return Money{}, err
	}
	return Money{
		amount:   a.amount + b.amount,
		currency: a.currency,
	}, nil
}

func (a Money) Add(b Money) Money {
	c, err := a.TryAdd(b)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Money) TrySub(b Money) (Money, error) {
	if err := sameCurrency(a, b); err != nil {
		return Money{}, err
	}
	return Money{
		amount:   a.amount - b.amount,
		currency: a.currency,
	}, nil
}

func (a Money) Sub(b Money) Money {
	c, err := a.TrySub(b)
	if err != nil {
		panic(err)
	}
	return c
}

func (a Money) Mul(f float64) Money {
	res := float64(a.amount) * f
	return Money{
		amount:   int64(math.Round(res)),
		currency: a.currency,
	}
}

func (a Money) Div(f float64) Money {
	if f == 0 {
		panic("division by zero")
	}
	res := float64(a.amount) / f
	return Money{
		amount:   int64(math.Round(res)),
		currency: a.currency,
	}
}

func (m Money) Neg() Money {
	return Money{
		amount:   -m.amount,
		currency: m.currency,
	}
}

func (m Money) IsBlank() bool {
	return m.amount == 0 && m.currency.code == ""
}

func (m Money) IsZero() bool {
	return m.amount == 0
}

func (m Money) IsNeg() bool {
	return m.amount < 0
}

func (m Money) Equal(e Money) bool {
	return m.amount == e.amount && m.currency == e.currency
}

// currencyByCode returns the Currency for a given ISO 4217 code
func currencyByCode(code string) (Currency, error) {
	switch code {
	case "EUR":
		return EUR.Currency, nil
	case "USD":
		return USD.Currency, nil
	case "JPY":
		return JPY.Currency, nil
	default:
		return Currency{}, errors.New("unknown currency: " + code)
	}
}

// String returns a string representation of the Money value (e.g., "12.34 EUR" or "567 JPY").
func (m Money) String() string {

	// Precision note: float64 has 53 bits of mantissa, which can exactly represent integers
	// up to 2^53 (~9 quadrillion). For currencies with factor=100 (EUR, USD), this safely
	// handles amounts up to ~90 trillion in major units. The division by factor produces
	// exact results for typical monetary amounts.

	if m.currency.factor == 1 {
		return strconv.FormatInt(m.amount, 10) + " " + m.currency.code
	}
	major := float64(m.amount) / float64(m.currency.factor)
	// Determine decimal places based on factor
	decimals := 0
	for f := m.currency.factor; f > 1; f /= 10 {
		decimals++
	}
	return strconv.FormatFloat(major, 'f', decimals, 64) + " " + m.currency.code
}

// Parse parses a string representation of Money (e.g., "12.34 EUR" or "567 JPY").
func Parse(s string) (Money, error) {

	// Precision note: The float64 parsing and multiplication by factor may introduce tiny
	// floating-point errors (e.g., 12.34 * 100 = 1233.9999...). We use math.Round() to
	// correct these errors when converting back to int64 minor units, ensuring exact results
	// for all typical monetary values with up to ~15 significant digits.

	parts := strings.Fields(s)
	if len(parts) != 2 {
		return Money{}, errors.New("invalid money format: expected 'amount currency'")
	}

	amount, err := strconv.ParseFloat(parts[0], 64)
	if err != nil {
		return Money{}, fmt.Errorf("invalid amount: %w", err)
	}

	currency, err := currencyByCode(parts[1])
	if err != nil {
		return Money{}, err
	}

	minorUnits := int64(math.Round(amount * float64(currency.factor)))
	return Money{amount: minorUnits, currency: currency}, nil
}

// MarshalJSON implements json.Marshaler for Money.
func (m Money) MarshalJSON() ([]byte, error) {

	// Precision note: See String() for details on float64 precision guarantees.

	major := float64(m.amount) / float64(m.currency.factor)
	return json.Marshal(struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}{
		Amount:   major,
		Currency: m.currency.code,
	})
}

// UnmarshalJSON implements json.Unmarshaler for Money.
func (m *Money) UnmarshalJSON(data []byte) error {

	// Precision note: See Parse() for details on float64 precision and rounding.

	var raw struct {
		Amount   float64 `json:"amount"`
		Currency string  `json:"currency"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	currency, err := currencyByCode(raw.Currency)
	if err != nil {
		return err
	}

	m.amount = int64(math.Round(raw.Amount * float64(currency.factor)))
	m.currency = currency
	return nil
}
