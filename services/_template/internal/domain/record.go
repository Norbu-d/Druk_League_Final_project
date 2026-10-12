package domain

import "time"

type Record struct {
	ID        string
	Payload   []byte
	CreatedAt time.Time
}