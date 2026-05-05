package entities

import "time"

type Note struct {
	ID        string    `json:"id"`
	Content   string    `json:"content"`
	HTML      string    `json:"html"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"created_at"`
	Tags      []string  `json:"tags"`
	Reactions []Reaction `json:"reactions"`
}

type Reaction struct {
	Emoji     string `json:"emoji"`
	EmojiID   string `json:"emoji_id"`
	EmojiImage string `json:"emoji_image"`
	Count     string `json:"count"`
	IsPaid    bool   `json:"is_paid"`
}