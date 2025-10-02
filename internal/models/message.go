package models

import "time"

type Message struct {
	ID         int        `json:"id"`
	To         string     `json:"to"`
	Content    string     `json:"content"`
	SentAt     *time.Time `json:"sent_at,omitempty"`
	ExternalID *string    `json:"external_id,omitempty"`
	CreatedAt  time.Time  `json:"-"`
	UpdatedAt  time.Time  `json:"-"`
}
