package domaintypes

type SelectOption struct {
	Value string `json:"value"`
	Label string `json:"label"`
	Color string `json:"color,omitempty"`
}
