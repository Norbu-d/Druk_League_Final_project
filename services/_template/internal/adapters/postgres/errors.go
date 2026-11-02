package postgres

import "errors"

var ErrInvalidPayload = errors.New("payload must be valid JSON")
