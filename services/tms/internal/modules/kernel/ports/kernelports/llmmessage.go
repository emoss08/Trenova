package kernelports

// ToolSpec describes a tool to the model.
type ToolSpec struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  map[string]any `json:"parameters"`
	Deferred    bool           `json:"deferred,omitempty"`
	Searches    bool           `json:"searches,omitempty"`
}
