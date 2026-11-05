package shipmentcommercial

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
)

func newCalculator(t *testing.T, p Params) *Calculator {
	t.Helper()

	if p.Hydrator == nil {
		p.Hydrator = ratingtest.NoHydration{}
	}
	if p.BillingControlRepo == nil {
		p.BillingControlRepo = ratingtest.NoBillingControl{}
	}

	if p.FuelSurcharge == nil {
		p.FuelSurcharge = ratingtest.NoFuelSurcharge{}
	}

	return New(p)
}
