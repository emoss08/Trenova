package typeutils

func DefaultValueForType(varType string) any {
	switch varType {
	case "Number":
		return 0.0
	case "String":
		return ""
	case "Boolean":
		return false
	case "Array":
		return []any{}
	case "Object":
		return map[string]any{}
	default:
		return nil
	}
}
