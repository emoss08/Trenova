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

// NilIfEmpty is Ptr for named string types such as enums: an unset value
// becomes nil, which a database driver writes as NULL rather than "" and so
// satisfies a CHECK that lists the valid options.
func NilIfEmpty[T ~string](v T) *T {
	if v == "" {
		return nil
	}

	return &v
}
