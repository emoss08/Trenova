package formulatemplatetypes

import "github.com/emoss08/trenova/pkg/rules"

type SchemaDescription struct {
	SchemaID  string                  `json:"schemaId"`
	Variables []rules.FormulaVariable `json:"variables"`
}
