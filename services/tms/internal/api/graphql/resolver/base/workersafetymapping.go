package base

func Int32Ptr(value *int32) *int {
	if value == nil {
		return nil
	}
	out := int(*value)
	return &out
}
