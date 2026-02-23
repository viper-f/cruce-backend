package Entities

import "time"

type Notification struct {
	Id          int                  `json:"id"`
	UserId      int                  `json:"user_id"`
	Type        string               `json:"type"`
	Title       string               `json:"title"`
	Message     string               `json:"message"`
	DateCreated time.Time            `json:"date_created"`
	IsRead      bool                 `json:"is_read"`
	Mention     *NotificationMention `json:"mention"`
	Game        *NotificationGame    `json:"game"`
}

type NotificationMention struct {
	UserId        int     `json:"user_id"`
	UserName      string  `json:"user_name"`
	CharacterId   *int    `json:"character_id"`
	CharacterName *string `json:"character_name"`
	PostId        int     `json:"post_id"`
	TopicId       int     `json:"topic_id"`
}

type NotificationGame struct {
	TopicId           int    `json:"topic_id"`
	TopicName         string `json:"topic_name"`
	Type              string `json:"type"`
	UserCharacterId   int    `json:"user_character_id"`
	UserCharacterName string `json:"user_character_name"`
	CharacterId       int    `json:"character_id"`
	CharacterName     string `json:"character_name"`
}
