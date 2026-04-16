package Entities

type Reaction struct {
	Id       int    `json:"id"`
	Url      string `json:"url"`
	IsActive bool   `json:"is_active"`
}
