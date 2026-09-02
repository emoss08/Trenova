package rateagreementservice

import "github.com/emoss08/trenova/internal/core/ports/repositories"

// NewTestValidator returns a validator with the database-backed checks left
// out, so a test can exercise the business rules without a live connection.
func NewTestValidator() *Validator {
	return NewTestValidatorWithTemplateRepo(nil)
}

// NewTestValidatorWithTemplateRepo lets a test supply a stub formula template
// repository so the rule-reference checks can be exercised without a database.
func NewTestValidatorWithTemplateRepo(
	formulaTemplateRepo repositories.FormulaTemplateRepository,
) *Validator {
	return &Validator{
		validator:           newBuilder(nil, formulaTemplateRepo).Build(),
		formulaTemplateRepo: formulaTemplateRepo,
	}
}
