package formutil

import (
	"encoding/json"
)

// FormType ...
type FormType string

const (
	FormTypeMenu   FormType = "form"
	FormTypeModal  FormType = "modal"
	FormTypeCustom FormType = "custom_form"
)

// MenuForm ...
type MenuForm struct {
	Type     FormType        `json:"type"`
	Title    string          `json:"title"`
	Content  string          `json:"content"`
	Elements []ButtonElement `json:"elements"`
}

// ButtonElement ...
type ButtonElement struct {
	Type string `json:"type"`

	Text  string      `json:"text"`
	Image ButtonImage `json:"image,omitempty"`
}

// ButtonImage ...
type ButtonImage struct {
	Type string `json:"type"`
	Data string `json:"data"`
}

// SetImageURL ...
func (b *ButtonElement) SetImageURL(url string) {
	b.Image = ButtonImage{
		Type: "url",
		Data: url,
	}
}

// ParseMenuForm ...
func ParseMenuForm(data []byte) (*MenuForm, error) {
	var form MenuForm
	if err := json.Unmarshal(data, &form); err != nil {
		return nil, err
	}
	if form.Type != FormTypeMenu {
		// We only care about menu forms, for now.
		return nil, nil
	}
	return &form, nil
}

// Marshal ...
func (m *MenuForm) Marshal() ([]byte, error) {
	return json.Marshal(m)
}
