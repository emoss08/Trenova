package gqlscalar

func OptionalInt64(value *int) *int64 {
	if value == nil {
		return nil
	}
	converted := int64(*value)

	return &converted
}

func StrPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func Int64PtrToIntPtr(value *int64) *int {
	if value == nil {
		return nil
	}
	v := int(*value)
	return &v
}

func StringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func Int64Ptr(value *int) *int64 {
	if value == nil {
		return nil
	}
	converted := int64(*value)
	return &converted
}

func Int64Value(value *int) int64 {
	if value == nil {
		return 0
	}
	return int64(*value)
}

func IntValue(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func BoolValue(value *bool) bool {
	if value == nil {
		return false
	}
	return *value
}

func NonEmptyPtr(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func IntPtr(value *int64) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}

func DerefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func DerefInt(v *int) int {
	if v == nil {
		return 0
	}
	return *v
}

func EmptyToNil(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func StringPtrFromValue(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func Int32Ptr(value *int32) *int {
	if value == nil {
		return nil
	}
	out := int(*value)
	return &out
}
