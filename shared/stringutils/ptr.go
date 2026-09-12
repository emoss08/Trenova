package stringutils

func Ptr(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}

// FromPtr is the inverse of Ptr: a nil pointer reads as the empty string, which
// is how the domain spells "absent" for a plain string field.
func FromPtr(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
