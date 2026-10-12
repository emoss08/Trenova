package ratezoneservice

// NewTestValidator returns a validator without the database-backed checks.
func NewTestValidator() *Validator {
	return &Validator{validator: newBuilder(nil).Build()}
}
