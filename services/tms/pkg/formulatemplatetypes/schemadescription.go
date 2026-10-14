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
	Operator    bool   `json:"operator"`
	MinArgs     int    `json:"minArgs"`
	MaxArgs     int    `json:"maxArgs"`
	Returns     string `json:"returns"`
	NumericArgs bool   `json:"numericArgs"`
	Predicate   bool   `json:"predicate"`
}

type SchemaDisabledFunction struct {
	Name        string `json:"name"`
	Reason      string `json:"reason"`
	Replacement string `json:"replacement,omitempty"`
}

type SchemaDescription struct {
	SchemaID  string                   `json:"schemaId"`
	Variables []SchemaVariableInfo     `json:"variables"`
	Functions []SchemaFunctionInfo     `json:"functions"`
	Builtins  []SchemaFunctionInfo     `json:"builtins"`
	Disabled  []SchemaDisabledFunction `json:"disabled"`
}
