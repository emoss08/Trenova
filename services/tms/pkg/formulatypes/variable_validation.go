package formulatypes

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
)

var variableNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_]*$`)

func (t VariableValueType) Canonical() VariableValueType {
	for _, known := range []VariableValueType{
		VariableValueTypeNumber,
		VariableValueTypeString,
		VariableValueTypeBoolean,
		VariableValueTypeDate,
		VariableValueTypeArray,
		VariableValueTypeObject,
		VariableValueTypeAny,
	} {
		if strings.EqualFold(string(t), string(known)) {
			return known
		}
	}
	return t
}

func (t VariableValueType) IsValid() bool {
	switch t.Canonical() {
	case VariableValueTypeNumber,
		VariableValueTypeString,
		VariableValueTypeBoolean,
		VariableValueTypeDate,
		VariableValueTypeArray,
		VariableValueTypeObject,
		VariableValueTypeAny:
		return true
	default:
		return false
	}
}

func ValidateVariableDefinitions(
	definitions []*VariableDefinition,
	multiErr *errortypes.MultiError,
) {
	seen := make(map[string]int, len(definitions))
	for i, def := range definitions {
		field := "variableDefinitions[" + strconv.Itoa(i) + "]"
		if def == nil {
			multiErr.Add(field, errortypes.ErrRequired, "Variable definition is required")
			continue
		}

		switch {
		case def.Name == "":
			multiErr.Add(field+".name", errortypes.ErrRequired, "Name is required")
		case !variableNamePattern.MatchString(def.Name):
			multiErr.Add(
				field+".name",
				errortypes.ErrInvalid,
				"Name must start with a letter and contain only letters, digits, and underscores",
			)
		default:
			if first, dup := seen[def.Name]; dup {
				multiErr.Add(
					field+".name",
					errortypes.ErrDuplicate,
					"Input {0} is declared twice; the first is input {1}",
					def.Name,
					first+1,
				)
			} else {
				seen[def.Name] = i
			}
		}

		if !def.Type.IsValid() {
			multiErr.Add(
				field+".type",
				errortypes.ErrInvalid,
				"Type must be Number, String, Boolean, Date, Array, Object or Any",
			)
			continue
		}

		if def.DefaultValue != nil && !defaultMatchesType(def.Type.Canonical(), def.DefaultValue) {
			multiErr.Add(
				field+".defaultValue",
				errortypes.ErrInvalid,
				"Default value must be a {0} to match the input's type",
				string(def.Type.Canonical()),
			)
		}
	}
}

func defaultMatchesType(valueType VariableValueType, value any) bool {
	switch valueType {
	case VariableValueTypeNumber:
		switch value.(type) {
		case float64, float32, int, int64, int32, int16, int8, uint, uint64, uint32, uint16, uint8:
			return true
		}
		return false
	case VariableValueTypeString:
		_, ok := value.(string)
		return ok
	case VariableValueTypeBoolean:
		_, ok := value.(bool)
		return ok
	case VariableValueTypeDate:
		switch value.(type) {
		case string, float64, int64, int:
			return true
		}
		return false
	case VariableValueTypeArray:
		_, ok := value.([]any)
		return ok
	case VariableValueTypeObject:
		_, ok := value.(map[string]any)
		return ok
	case VariableValueTypeAny:
		return true
	default:
		return true
	}
}
