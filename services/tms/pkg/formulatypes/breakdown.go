package formulatypes

import (
	"regexp"
	"strconv"

	"github.com/emoss08/trenova/pkg/errortypes"
)

const MaxBreakdownDefinitions = 20

var breakdownNamePattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*$`)

type BreakdownDefinition struct {
	Name       string `json:"name"`
	Label      string `json:"label"`
	Expression string `json:"expression"`
}

func ValidateBreakdownDefinitions(
	definitions []*BreakdownDefinition,
	multiErr *errortypes.MultiError,
) {
	if len(definitions) > MaxBreakdownDefinitions {
		multiErr.Add(
			"breakdownDefinitions",
			errortypes.ErrInvalid,
			"A template cannot have more than 20 breakdown definitions",
		)
		return
	}

	seen := make(map[string]struct{}, len(definitions))
	for i, def := range definitions {
		if def == nil {
			continue
		}

		fieldPrefix := "breakdownDefinitions[" + strconv.Itoa(i) + "]"

		if def.Name == "" {
			multiErr.Add(fieldPrefix+".name", errortypes.ErrRequired, "Name is required")
		} else if !breakdownNamePattern.MatchString(def.Name) {
			multiErr.Add(
				fieldPrefix+".name",
				errortypes.ErrInvalid,
				"Name must start with a letter and contain only letters, digits, and underscores",
			)
		}

		if _, dup := seen[def.Name]; dup && def.Name != "" {
			multiErr.Add(
				fieldPrefix+".name",
				errortypes.ErrInvalid,
				"Breakdown names must be unique",
			)
		}
		seen[def.Name] = struct{}{}

		if def.Expression == "" {
			multiErr.Add(
				fieldPrefix+".expression",
				errortypes.ErrRequired,
				"Expression is required",
			)
		}
	}
}
