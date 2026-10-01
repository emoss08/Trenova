package dispatchconsoleresolver

func int16Ptr(value *int16) *int {
	if value == nil {
		return nil
	}
	converted := int(*value)
	return &converted
}
