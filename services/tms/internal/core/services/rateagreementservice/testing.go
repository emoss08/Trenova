package rateagreementservice

// NewTestValidator returns a validator with the database-backed checks left
// out, so a test can exercise the business rules without a live connection.
func NewTestValidator() *Validator {
	return &Validator{validator: newBuilder(nil).Build()}
}
