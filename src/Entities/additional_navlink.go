package Entities

import "encoding/json"

type AdditionalNavlinkType int

const (
	LinkAdditionalNavlink  AdditionalNavlinkType = 0
	LoginAdditionalNavlink AdditionalNavlinkType = 1
)

type AdditionalNavlink struct {
	Id         int                   `json:"id"`
	Title      string                `json:"title"`
	Type       AdditionalNavlinkType `json:"type"`
	Config     json.RawMessage       `json:"config"`
	Roles      []string              `json:"roles" db:"-"`
	IsInactive bool                  `json:"is_inactive"`
}
