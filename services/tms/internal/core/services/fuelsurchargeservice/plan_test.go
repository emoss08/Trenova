package fuelsurchargeservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type planIndexRepo struct {
	repositories.FuelIndexRepository
	index *fuelsurcharge.FuelIndex
}

func (r *planIndexRepo) GetByID(
	_ context.Context,
	req *repositories.GetFuelIndexByIDRequest,
) (*fuelsurcharge.FuelIndex, error) {
	if r.index == nil || r.index.ID != req.FuelIndexID {
		return nil, errortypes.NewNotFoundError("Fuel index not found")
	}
	copied := *r.index

	return &copied, nil
}

type planPriceRepo struct {
	repositories.FuelIndexPriceRepository
	price  *fuelsurcharge.FuelIndexPrice
	writes int
}

func (r *planPriceRepo) GetByID(
	_ context.Context,
	req *repositories.GetFuelIndexPriceByIDRequest,
) (*fuelsurcharge.FuelIndexPrice, error) {
	if r.price == nil || r.price.ID != req.PriceID {
		return nil, errortypes.NewNotFoundError("Fuel price not found")
	}
	copied := *r.price

	return &copied, nil
}

func (r *planPriceRepo) Create(
	_ context.Context,
	entity *fuelsurcharge.FuelIndexPrice,
) (*fuelsurcharge.FuelIndexPrice, error) {
	r.writes++

	return entity, nil
}

func (r *planPriceRepo) Update(
	_ context.Context,
	entity *fuelsurcharge.FuelIndexPrice,
) (*fuelsurcharge.FuelIndexPrice, error) {
	r.writes++

	return entity, nil
}

func planService(
	index *fuelsurcharge.FuelIndex,
	price *fuelsurcharge.FuelIndexPrice,
) (*Service, *planPriceRepo) {
	prices := &planPriceRepo{price: price}

	return &Service{
		l:         zap.NewNop(),
		indexRepo: &planIndexRepo{index: index},
		priceRepo: prices,
	}, prices
}

func customIndex() *fuelsurcharge.FuelIndex {
	return &fuelsurcharge.FuelIndex{
		ID:             pulid.MustNew("fidx_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Source:         fuelsurcharge.IndexSourceCustom,
		Currency:       "USD",
	}
}

func TestPlanAddManualPrice_ChecksTheIndexAndSavesNothing(t *testing.T) {
	t.Parallel()

	index := customIndex()
	svc, prices := planService(index, nil)
	userID := pulid.MustNew("usr_")

	planned, err := svc.PlanAddManualPrice(t.Context(), &fuelsurcharge.FuelIndexPrice{
		OrganizationID: index.OrganizationID,
		BusinessUnitID: index.BusinessUnitID,
		FuelIndexID:    index.ID,
		PriceDate:      "2026-09-28",
		Price:          decimal.RequireFromString("3.899"),
	}, userID)
	require.NoError(t, err)
	assert.True(t, planned.IsManual)
	assert.Equal(t, "USD", planned.Currency)
	require.NotNil(t, planned.EnteredByID)
	assert.Equal(t, userID, *planned.EnteredByID)
	assert.Zero(t, prices.writes)

	index.Source = fuelsurcharge.IndexSourceEIA
	_, err = svc.PlanAddManualPrice(t.Context(), &fuelsurcharge.FuelIndexPrice{
		OrganizationID: index.OrganizationID,
		BusinessUnitID: index.BusinessUnitID,
		FuelIndexID:    index.ID,
		PriceDate:      "2026-09-28",
		Price:          decimal.RequireFromString("3.899"),
	}, userID)
	require.Error(t, err)
}

func TestPlanUpdateManualPrice_ShowsTheChangeAndRefusesAnIngestedPrice(t *testing.T) {
	t.Parallel()

	index := customIndex()
	stored := &fuelsurcharge.FuelIndexPrice{
		ID:             pulid.MustNew("fip_"),
		OrganizationID: index.OrganizationID,
		BusinessUnitID: index.BusinessUnitID,
		FuelIndexID:    index.ID,
		PriceDate:      "2026-09-21",
		Price:          decimal.RequireFromString("3.799"),
		Currency:       "USD",
		IsManual:       true,
	}
	svc, prices := planService(index, stored)

	change, err := svc.PlanUpdateManualPrice(t.Context(), &fuelsurcharge.FuelIndexPrice{
		ID:             stored.ID,
		OrganizationID: index.OrganizationID,
		BusinessUnitID: index.BusinessUnitID,
		PriceDate:      "2026-09-21",
		Price:          decimal.RequireFromString("3.829"),
	}, pulid.MustNew("usr_"))
	require.NoError(t, err)
	assert.Equal(t, "3.799", change.Before.Price.String())
	assert.Equal(t, "3.829", change.After.Price.String())
	assert.Equal(t, index.ID, change.After.FuelIndexID)
	assert.Zero(t, prices.writes)

	stored.IsManual = false
	_, err = svc.PlanUpdateManualPrice(t.Context(), &fuelsurcharge.FuelIndexPrice{
		ID:             stored.ID,
		OrganizationID: index.OrganizationID,
		BusinessUnitID: index.BusinessUnitID,
		PriceDate:      "2026-09-21",
		Price:          decimal.RequireFromString("3.829"),
	}, pulid.MustNew("usr_"))
	require.Error(t, err)
}
