package toolschema

import (
	"strings"
)

// KeepEmpty marks a parameter whose empty value means something other than
// leaving it out, so it reaches the tool as sent: an update's field that an
// empty value clears, or a list whose absence means "every one" and whose
// empty value means "none" (the carrier intelligence fields to apply, the pay
// events to pay now). Every other optional parameter sent as "" or an empty
// list is read as not sent (see toolcoerce.Apply). It returns the property it marks.
func KeepEmpty(property map[string]any) map[string]any {
	property[KeyKeepEmpty] = true

	return property
}

func Clamped(property map[string]any) map[string]any {
	property[KeyClamp] = true

	return property
}

// JoinPath writes a nested field's path the way the validator writes one.
func JoinPath(path, name string) string {
	if path == "" {
		return name
	}

	return path + "." + name
}

// NameShape is a parameter name with case and separators taken out, so
// shipment_id, shipmentID and ShipmentId read as the same name.
func NameShape(name string) string {
	return strings.ToLower(strings.NewReplacer("_", "", "-", "").Replace(name))
}

// DeclaredType is the one type a property declares, the first non-null one
// when it declares several.
func DeclaredType(property map[string]any) string {
	if definition := localDefinition(property); definition != nil {
		return DeclaredType(definition)
	}
	switch typed := property["type"].(type) {
	case string:
		return typed
	case []string:
		for _, kind := range typed {
			if kind != "null" {
				return kind
			}
		}
	case []any:
		for _, kind := range typed {
			if name, ok := kind.(string); ok && name != "null" {
				return name
			}
		}
	}

	return ""
}

// localDefinition is the definition a property's $ref names among its own
// $defs, for a property that is a whole schema of its own, as a report
// definition is: its root is a $ref with no type beside it. Without it a
// report sent as JSON text was refused as "got string, want object" rather
// than read as the object it holds.
func localDefinition(property map[string]any) map[string]any {
	if _, typed := property["type"]; typed {
		return nil
	}
	ref, _ := property["$ref"].(string)
	name, local := strings.CutPrefix(ref, "#/$defs/")
	if !local {
		return nil
	}
	definitions, _ := property["$defs"].(map[string]any)
	definition, _ := definitions[name].(map[string]any)
	if _, nested := definition["$ref"]; nested {
		return nil
	}

	return definition
}

// EnumValues is the values a property allows, whichever way the schema lists them.
func EnumValues(property map[string]any) []string {
	switch typed := property["enum"].(type) {
	case []string:
		return typed
	case []any:
		values := make([]string, 0, len(typed))
		for _, value := range typed {
			if text, ok := value.(string); ok {
				values = append(values, text)
			}
		}

		return values
	default:
		return nil
	}
}
