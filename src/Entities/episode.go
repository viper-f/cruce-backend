package Entities

type Episode struct {
	Id           int               `json:"id" db:"id"`
	Topic_Id     int               `json:"topic_id" db:"topic_id"`
	Name         string            `json:"name" db:"name"`
	Characters   []*ShortCharacter `json:"characters" db:"-"`
	Masks        []ShortMask       `json:"masks" db:"-"`
	CustomFields CustomFieldEntity `json:"custom_fields" db:"-"`
	CanEdit      *bool             `json:"can_edit,omitempty" db:"-"`
}

func (e *Episode) GetBaseFields() []string {
	return []string{"topic_id", "name"}
}
