package rateagreementservice

import "github.com/emoss08/trenova/internal/modules/rating/ratingtest/tarifftest"

func testValidator() *Validator {
	return NewTestValidator(tarifftest.NewTemplates(tarifftest.PerMile()), &tarifftest.Checker{})
}
