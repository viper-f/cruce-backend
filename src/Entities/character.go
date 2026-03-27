package Entities

import "time"

type Character struct {
	Id              int               `json:"id"`
	UserId          int               `json:"user_id"`
	Name            string            `json:"name"`
	Avatar          *string           `json:"avatar"`
	CustomFields    CustomFieldEntity `json:"custom_fields" db:"-"`
	CharacterStatus CharacterStatus   `json:"character_status"`
	TopicId         int               `json:"topic_id"`
	TotalEpisodes   int               `json:"total_episodes"`
	Factions        []Faction         `json:"factions" db:"-"`
	Episodes        []EpisodeListItem `json:"episodes" db:"-"`
	CanEdit         *bool             `json:"can_edit,omitempty" db:"-"`
}

type EpisodeListItem struct {
	Id                     int               `json:"id"`
	Name                   string            `json:"name"`
	TopicId                int               `json:"topic_id"`
	Characters             []*ShortCharacter `json:"characters"`
	DateLastPost           *time.Time        `json:"date_last_post"`
	LastPostAuthorUsername *string           `json:"last_post_author_username"`
}

func (c *Character) GetBaseFields() []string {
	return []string{"user_id", "name", "avatar", "character_status", "topic_id", "total_episodes"}
}

type ShortCharacter struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

type CharacterStatus int

const (
	ActiveCharacter   CharacterStatus = 0
	InactiveCharacter CharacterStatus = 1
	PendingCharacter  CharacterStatus = 2
)

type CharacterClaim struct {
	Id          int     `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	IsClaimed   bool    `json:"is_claimed"`
	UserId      int     `json:"user_id"`
	GuestHash   string  `json:"guest_hash"`
}

type WantedCharacter struct {
	Id               int    `json:"id"`
	Name             string `json:"name"`
	IsClaimed        bool   `json:"is_claimed"`
	AuthorUserId     int    `json:"author_user_id"`
	DateCreated      time.Time
	CharacterClaimId *int              `json:"character_claim_id"`
	CustomFields     CustomFieldEntity `json:"custom_fields" db:"-"`
}
