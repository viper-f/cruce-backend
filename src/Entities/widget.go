package Entities

type WidgetType struct {
	Id             int     `json:"id"`
	Name           string  `json:"name"`
	ConfigTemplate *string `json:"config_template"`
	Func           string  `json:"func"`
}

type Widget struct {
	Id         int     `json:"id"`
	Name       string  `json:"name"`
	TemplateId int     `json:"template_id"`
	Config     *string `json:"config"`
}
