// Package identity provides UUID v7 generation and parsing for aggregate
// and entity identifiers.
package identity

import (
	"fmt"

	"github.com/google/uuid"
)

// NewID generates a new UUID v7 (time-sortable) string.
func NewID() string {
	return uuid.Must(uuid.NewV7()).String()
}

// Parse validates the given string as a UUID and returns it in canonical form.
// Returns an error if the string is not a valid UUID.
func Parse(id string) (string, error) {
	parsed, err := uuid.Parse(id)
	if err != nil {
		return "", fmt.Errorf("identity: invalid UUID %q: %w", id, err)
	}
	return parsed.String(), nil
}
