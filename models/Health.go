package models

import "time"

// Health ;
type Health struct {
	Status string    `json:"status"`
	Time   time.Time `json:"time" `
}
