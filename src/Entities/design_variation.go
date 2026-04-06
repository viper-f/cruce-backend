package Entities

type DesignVariation struct {
	Id        int     `json:"id"`
	ClassName *string `json:"class_name"`
	Name      *string `json:"name"`
}
