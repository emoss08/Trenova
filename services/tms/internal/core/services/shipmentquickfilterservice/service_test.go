package shipmentquickfilterservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/costingservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubControls struct {
	repositories.ShipmentControlRepository
	control *tenant.ShipmentControl
	calls   int
}

func (s *stubControls) Get(
	context.Context,
	repositories.GetShipmentControlRequest,
) (*tenant.ShipmentControl, error) {
	s.calls++
	return s.control, nil
}

type stubCosts struct {
	profile *costingservice.ResolvedCostProfile
	calls   int
}

func (s *stubCosts) ResolveCostProfile(
	context.Context,
	pagination.TenantInfo,
	time.Time,
) (*costingservice.ResolvedCostProfile, error) {
	s.calls++
	return s.profile, nil
}

func newTestService(controls *stubControls, costs *stubCosts) *Service {
	fixed := time.Date(2026, 10, 6, 20, 0, 0, 0, time.UTC)
	return NewWithDependencies(controls, costs, func() time.Time { return fixed })
}

func TestResolveRejectsUnknownZone(t *testing.T) {
	t.Parallel()

	svc := newTestService(&stubControls{}, &stubCosts{})
	_, err := svc.Resolve(t.Context(), &services.ResolveShipmentQuickFilterBasisRequest{
		Timezone: "Mars/Olympus",
	})

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
}

func TestResolveLoadsOnlyWhatIsNeeded(t *testing.T) {
	t.Parallel()

	threshold := int16(45)
	controls := &stubControls{control: &tenant.ShipmentControl{
		DetentionThreshold:       &threshold,
		UseDetentionPolicyEngine: true,
	}}
	costs := &stubCosts{profile: &costingservice.ResolvedCostProfile{
		TotalCPM:             decimal.RequireFromString("2.10"),
		IncludeDeadheadMiles: true,
	}}
	svc := newTestService(controls, costs)

	basis, err := svc.Resolve(t.Context(), &services.ResolveShipmentQuickFilterBasisRequest{
		Timezone: "America/New_York",
	})
	require.NoError(t, err)
	assert.Nil(t, basis.Margin)
	assert.Nil(t, basis.Detention)
	assert.Equal(t, 0, controls.calls+costs.calls)
	assert.Equal(t, "America/New_York", basis.Location.String())
	assert.Equal(t, 16, basis.Now.Hour())

	basis, err = svc.Resolve(t.Context(), &services.ResolveShipmentQuickFilterBasisRequest{
		Timezone:  "UTC",
		Margin:    true,
		Detention: true,
	})
	require.NoError(t, err)
	assert.True(t, basis.Margin.CostPerMile.Equal(decimal.RequireFromString("2.10")))
	assert.True(t, basis.Margin.TargetMarginPercent.Equal(costingcontrol.DefaultTargetMarginPercent))
	assert.Equal(t, int64(45), basis.Detention.ThresholdMinutes)
	assert.True(t, basis.Detention.UsePolicyEngine)
}

func TestPrepareValidatesAndAttachesTheBasis(t *testing.T) {
	t.Parallel()

	svc := newTestService(&stubControls{control: &tenant.ShipmentControl{}}, &stubCosts{})

	opts := &repositories.ShipmentOptions{}
	require.NoError(t, svc.Prepare(t.Context(), pagination.TenantInfo{}, opts))
	assert.Nil(t, opts.QuickFilterBasis)

	opts = &repositories.ShipmentOptions{
		QuickFilters: []shipment.QuickFilterSpec{{Filter: shipment.QuickFilterDeliveryHour}},
		Timezone:     "UTC",
	}
	err := svc.Prepare(t.Context(), pagination.TenantInfo{}, opts)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)

	opts = &repositories.ShipmentOptions{
		QuickFilters: []shipment.QuickFilterSpec{shipment.Quick(shipment.QuickFilterDetention)},
		Timezone:     "UTC",
	}
	require.NoError(t, svc.Prepare(t.Context(), pagination.TenantInfo{}, opts))
	require.NotNil(t, opts.QuickFilterBasis)
	assert.Equal(t, int64(30), opts.QuickFilterBasis.Detention.ThresholdMinutes)
}
