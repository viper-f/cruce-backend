package Entities

import "time"

type Notification struct {
	Id          int       `json:"id"`
	UserId      int       `json:"user_id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Message     string    `json:"message"`
	DateCreated time.Time `json:"date_created"`
	IsRead      bool      `json:"is_read"`
}
