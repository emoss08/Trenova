package stringutils

func Ptr(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}
