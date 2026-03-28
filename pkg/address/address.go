// Package address provides an Address value object with basic validation.
package address

import (
	"fmt"
	"regexp"
	"strings"
)

var countryCodePattern = regexp.MustCompile(`^[A-Z]{2}$`)

// Address represents a physical shipping or billing address.
type Address struct {
	Line1         string
	Line2         string
	City          string
	StateOrRegion string
	PostalCode    string
	CountryCode   string
}

// NewAddress creates a validated address. Line1, City, PostalCode, and
// CountryCode are required. CountryCode must be exactly 2 uppercase letters.
func NewAddress(line1, line2, city, stateOrRegion, postalCode, countryCode string) (Address, error) {
	var missing []string

	line1 = strings.TrimSpace(line1)
	if line1 == "" {
		missing = append(missing, "line1")
	}

	city = strings.TrimSpace(city)
	if city == "" {
		missing = append(missing, "city")
	}

	postalCode = strings.TrimSpace(postalCode)
	if postalCode == "" {
		missing = append(missing, "postalCode")
	}

	countryCode = strings.TrimSpace(countryCode)
	if countryCode == "" {
		missing = append(missing, "countryCode")
	}

	if len(missing) > 0 {
		return Address{}, fmt.Errorf("address: required fields missing: %s", strings.Join(missing, ", "))
	}

	if !countryCodePattern.MatchString(countryCode) {
		return Address{}, fmt.Errorf("address: invalid country code %q, must be 2 uppercase letters", countryCode)
	}

	return Address{
		Line1:         line1,
		Line2:         strings.TrimSpace(line2),
		City:          city,
		StateOrRegion: strings.TrimSpace(stateOrRegion),
		PostalCode:    postalCode,
		CountryCode:   countryCode,
	}, nil
}
