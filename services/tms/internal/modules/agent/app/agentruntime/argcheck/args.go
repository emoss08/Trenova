package argcheck

func String(arguments map[string]any, key string) string {
	value, _ := arguments[key].(string)

	return value
}
