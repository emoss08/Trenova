package formulatemplatetypes

type SchemaVariableInfo struct {
	Name        string   `json:"name"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	Category    string   `json:"category"`
	Nullable    bool     `json:"nullable"`
	Computed    bool     `json:"computed"`
	Enum        []string `json:"enum,omitempty"`
}

type SchemaFunctionInfo struct {
	Name        string `json:"name"`
	Signature   string `json:"signature"`
	Description string `json:"description"`
	Example     string `json:"example"`
	Category    string `json:"category"`
}

type SchemaDescription struct {
	SchemaID  string               `json:"schemaId"`
	Variables []SchemaVariableInfo `json:"variables"`
	Functions []SchemaFunctionInfo `json:"functions"`
}
