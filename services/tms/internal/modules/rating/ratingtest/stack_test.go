package ratingtest_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratequoteports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlatEngineRatesTheShipmentWithOneEvaluation(t *testing.T) {
	t.Parallel()

	stack := ratingtest.NewStack(t, ratingtest.StackOptions{
		Any: ratingtest.FlatTemplate(decimal.NewFromInt(125)),
	})
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	entity := &shipment.Shipment{
		ID:                pulid.MustNew("shp_"),
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		FormulaTemplateID: pulid.MustNew("ft_"),
	}

	rated, err := stack.Engine.RateShipment(t.Context(), &ratequoteports.RateShipmentRequest{
		TenantInfo: tenant,
		Shipment:   entity,
	})
	require.NoError(t, err)
	assert.True(t, rated.Total.Equal(decimal.NewFromInt(125)), rated.Total.String())
	assert.EqualValues(t, 1, stack.Evaluations())
}
