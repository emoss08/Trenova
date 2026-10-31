package reviewprompt

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/stretchr/testify/assert"
)

func TestFuelTerms_SaysWhatEachSettingMeansForAFuelCharge(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "Fuel surcharge: none is added automatically; one entered by hand is normal.",
		fuelTerms(customer.FuelSurchargeModeNone), "None never means the payer is not charged fuel")
	assert.Equal(t, "Fuel surcharge: added automatically from the payer's fuel surcharge program.",
		fuelTerms(customer.FuelSurchargeModeProgram))
	assert.Equal(t, "Fuel is included in the rates: a separate fuel charge would bill it twice.",
		fuelTerms(customer.FuelSurchargeModeFuelIncluded))
	assert.Empty(t, fuelTerms(""))
}
