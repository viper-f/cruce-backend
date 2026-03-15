package Entities

type CharacterProfile struct {
	Id            int               `json:"id"`
	CharacterId   *int              `json:"character_id"`
	CharacterName string            `json:"character_name"`
	MaskName      *string           `json:"mask_name"`
	Avatar        *string           `json:"avatar"`
	CustomFields  CustomFieldEntity `json:"custom_fields"`
	CanEdit       *bool             `json:"can_edit,omitempty" db:"-"`
	UserId        *int              `json:"user_id"`
	IsMask        *bool             `json:"is_mask"`
}

func (cp *CharacterProfile) GetBaseFields() []string {
	return []string{"character_id", "avatar", "mask_name", "is_mask", "user_id"}
}
