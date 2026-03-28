// Package pagination provides cursor-based pagination utilities for list
// endpoints.
package pagination

import (
	"encoding/base64"
	"fmt"

	"github.com/google/uuid"
)

// DefaultLimit is the default page size when none is specified.
const DefaultLimit = 20

// MaxLimit is the maximum allowed page size.
const MaxLimit = 100

// Page represents validated pagination parameters.
type Page struct {
	Cursor string
	Limit  int
}

// EncodeCursor encodes a UUID string to a URL-safe base64 cursor token.
func EncodeCursor(id string) string {
	return base64.URLEncoding.EncodeToString([]byte(id))
}

// DecodeCursor decodes a URL-safe base64 cursor token back to a UUID string.
// Returns an error if the cursor is not valid base64 or does not contain a
// valid UUID.
func DecodeCursor(cursor string) (string, error) {
	raw, err := base64.URLEncoding.DecodeString(cursor)
	if err != nil {
		return "", fmt.Errorf("pagination: invalid cursor encoding: %w", err)
	}

	parsed, err := uuid.Parse(string(raw))
	if err != nil {
		return "", fmt.Errorf("pagination: cursor does not contain a valid UUID: %w", err)
	}

	return parsed.String(), nil
}

// NewPage creates pagination parameters with validation. If limit is less than
// or equal to zero, DefaultLimit is used. If limit exceeds MaxLimit, it is
// clamped to MaxLimit.
func NewPage(cursor string, limit int) Page {
	if limit <= 0 {
		limit = DefaultLimit
	}
	if limit > MaxLimit {
		limit = MaxLimit
	}

	return Page{
		Cursor: cursor,
		Limit:  limit,
	}
}
