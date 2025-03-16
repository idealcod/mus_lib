package models

import "time"

type Song struct {
	ID          int       `json:"id" db:"id"`
	Group       string    `json:"group" db:"group_name"`
	Song        string    `json:"song" db:"song_name"`
	ReleaseDate string    `json:"release_date" db:"release_date"`
	Text        string    `json:"text" db:"text"`
	Link        string    `json:"link" db:"link"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
	UpdatedAt   time.Time `json:"updated_at" db:"updated_at"`
}

type Verse struct {
	Number int    `json:"number"`
	Text   string `json:"text"`
}
