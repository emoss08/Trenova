package rateresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/reference/domain/accessorialcharge"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRateAgreementCard_NamesEachAccessorialAndTheFuelProgram(t *testing.T) {
	t.Parallel()

	agreement := &rateagreement.RateAgreement{
		ID:   pulid.MustNew("ra_"),
		Code: "RA-2041",
		Name: "Acme 2026",
		Accessorials: []*rateagreement.RateAgreementAccessorial{{
			ID:                  pulid.MustNew("raa_"),
			AccessorialChargeID: pulid.MustNew("acc_"),
			Amount:              decimal.NewFromInt(75),
			AccessorialCharge:   &accessorialcharge.AccessorialCharge{Code: "DET", Description: "Detention"},
		}, {
			ID:                  pulid.MustNew("raa_"),
			AccessorialChargeID: pulid.MustNew("acc_"),
			Amount:              decimal.NewFromInt(250),
		}},
		FuelBinding: &rateagreement.RateAgreementFuelBinding{
			ID:                     pulid.MustNew("rafb_"),
			FuelSurchargeProgramID: pulid.MustNew("fsp_"),
			Program:                &fuelsurcharge.FuelSurchargeProgram{Name: "DOE weekly"},
		},
	}

	model, err := rateAgreementCardToModel(agreement, nil, false)

	require.NoError(t, err)
	require.Len(t, model.Accessorials, 2)
	assert.Equal(t, "DET", model.Accessorials[0].ChargeCode)
	assert.Equal(t, "Detention", model.Accessorials[0].ChargeDescription)
	assert.Empty(t, model.Accessorials[1].ChargeCode, "a charge that was not loaded names nothing")
	assert.Equal(t, "DOE weekly", model.FuelBinding.ProgramName)
}
