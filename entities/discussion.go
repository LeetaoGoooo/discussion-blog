package entities

import "time"

type Discussion struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Body        string    `json:"body"`
	CreatedAt   time.Time `json:"created_at"`
	Labels      []string  `json:"labels"`
	Category    string    `json:"category"`
	Author      string    `json:"author"`
	URL         string    `json:"url"`
	Comments    []Comment `json:"comments"`
}

type Comment struct {
	ID        string    `json:"id"`
	Body      string    `json:"body"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
}

type Category struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Tag struct {
	Name string `json:"name"`
}