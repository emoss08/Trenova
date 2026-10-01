package base

func StrPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}
