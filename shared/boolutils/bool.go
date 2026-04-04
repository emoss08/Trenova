package boolutils

func BooleanValue(v any) bool {
	value, _ := v.(bool)
	return value
}
