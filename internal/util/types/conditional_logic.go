package types

// TableChangeAlertCondition represents a condition in the table change alert.
type TableChangeAlertCondition struct {
	ID        int    `json:"id"`
	Column    string `json:"column"`
	Operation string `json:"operation"`
	Value     any    `json:"value"`
	DataType  string `json:"dataType"`
}

// TableChangeAlertConditionalLogic represents the conditional logic for a table change alert.
type TableChangeAlertConditionalLogic struct {
	Name        string                      `json:"name"`
	Description string                      `json:"description"`
	TableName   string                      `json:"tableName"`
	Conditions  []TableChangeAlertCondition `json:"conditions"`
}
