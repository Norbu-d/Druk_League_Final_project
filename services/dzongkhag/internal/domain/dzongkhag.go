package domain

import "errors"

var ErrNotFound = errors.New("dzongkhag not found")

type Dzongkhag struct {
	Code string
	Name string
}
