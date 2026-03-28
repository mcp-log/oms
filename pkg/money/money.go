// Package money provides a Money value object for safe monetary calculations.
// Amounts are represented as decimal strings, never floating point.
package money

import (
	"fmt"
	"regexp"

	"github.com/shopspring/decimal"
)

var currencyCodePattern = regexp.MustCompile(`^[A-Z]{3}$`)

// Money represents a monetary value. Amount is a decimal string, never float.
type Money struct {
	CurrencyCode string
	Amount       string // decimal string e.g. "29.99"
}

// NewMoney creates a Money value. Validates currency code is 3 uppercase letters.
// Validates amount is a valid decimal string using shopspring/decimal.
func NewMoney(currencyCode, amount string) (Money, error) {
	if !currencyCodePattern.MatchString(currencyCode) {
		return Money{}, fmt.Errorf("money: invalid currency code %q, must be 3 uppercase letters", currencyCode)
	}

	d, err := decimal.NewFromString(amount)
	if err != nil {
		return Money{}, fmt.Errorf("money: invalid amount %q: %w", amount, err)
	}

	return Money{
		CurrencyCode: currencyCode,
		Amount:       d.String(),
	}, nil
}

// Add returns a new Money with the amounts summed. Returns an error if
// currencies differ.
func (m Money) Add(other Money) (Money, error) {
	if m.CurrencyCode != other.CurrencyCode {
		return Money{}, fmt.Errorf("money: cannot add %s to %s", other.CurrencyCode, m.CurrencyCode)
	}

	a, _ := decimal.NewFromString(m.Amount)
	b, _ := decimal.NewFromString(other.Amount)

	return Money{
		CurrencyCode: m.CurrencyCode,
		Amount:       a.Add(b).String(),
	}, nil
}

// Multiply returns a new Money with amount * quantity.
func (m Money) Multiply(quantity int) Money {
	a, _ := decimal.NewFromString(m.Amount)
	result := a.Mul(decimal.NewFromInt(int64(quantity)))

	return Money{
		CurrencyCode: m.CurrencyCode,
		Amount:       result.String(),
	}
}

// IsZero returns true if the amount equals zero.
func (m Money) IsZero() bool {
	a, err := decimal.NewFromString(m.Amount)
	if err != nil {
		return false
	}
	return a.IsZero()
}

// String returns the money value in "USD 29.99" format.
func (m Money) String() string {
	return fmt.Sprintf("%s %s", m.CurrencyCode, m.Amount)
}
