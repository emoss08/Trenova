package shipmentquickfilterservice

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/app/costingservice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/costingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/quickfilter"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubControls struct {
	tenantports.ShipmentControlRepository
	control *shipmentcontrol.ShipmentControl
	calls   int
}

func (s *stubControls) Get(
	context.Context,
	tenantports.GetShipmentControlRequest,
) (*shipmentcontrol.ShipmentControl, error) {
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
	_, err := svc.Resolve(t.Context(), &shipmentports.ResolveShipmentQuickFilterBasisRequest{
		Timezone: "Mars/Olympus",
	})

	var validation *errortypes.Error
	require.ErrorAs(t, err, &validation)
}

func TestResolveLoadsOnlyWhatIsNeeded(t *testing.T) {
	t.Parallel()

	threshold := int16(45)
	controls := &stubControls{control: &shipmentcontrol.ShipmentControl{
		DetentionThreshold:       &threshold,
		UseDetentionPolicyEngine: true,
	}}
	costs := &stubCosts{profile: &costingservice.ResolvedCostProfile{
		TotalCPM:             decimal.RequireFromString("2.10"),
		IncludeDeadheadMiles: true,
	}}
	svc := newTestService(controls, costs)

	basis, err := svc.Resolve(t.Context(), &shipmentports.ResolveShipmentQuickFilterBasisRequest{
		Timezone: "America/New_York",
	})
	require.NoError(t, err)
	assert.Nil(t, basis.Margin)
	assert.Nil(t, basis.Detention)
	assert.Equal(t, 0, controls.calls+costs.calls)
	assert.Equal(t, "America/New_York", basis.Location.String())
	assert.Equal(t, 16, basis.Now.Hour())

	basis, err = svc.Resolve(t.Context(), &shipmentports.ResolveShipmentQuickFilterBasisRequest{
		Timezone:  "UTC",
		Margin:    true,
		Detention: true,
	})
	require.NoError(t, err)
	assert.True(t, basis.Margin.CostPerMile.Equal(decimal.RequireFromString("2.10")))
	assert.True(
		t,
		basis.Margin.TargetMarginPercent.Equal(costingcontrol.DefaultTargetMarginPercent),
	)
	assert.Equal(t, int64(45), basis.Detention.ThresholdMinutes)
	assert.True(t, basis.Detention.UsePolicyEngine)
}

func TestPrepareValidatesAndAttachesTheBasis(t *testing.T) {
	t.Parallel()

	svc := newTestService(&stubControls{control: &shipmentcontrol.ShipmentControl{}}, &stubCosts{})

	opts := &shipmentports.ShipmentOptions{}
	require.NoError(t, svc.Prepare(t.Context(), pagination.TenantInfo{}, opts))
	assert.Nil(t, opts.QuickFilterBasis)

	opts = &shipmentports.ShipmentOptions{
		QuickFilters: []quickfilter.QuickFilterSpec{{Filter: quickfilter.QuickFilterDeliveryHour}},
		Timezone:     "UTC",
	}
	err := svc.Prepare(t.Context(), pagination.TenantInfo{}, opts)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)

	opts = &shipmentports.ShipmentOptions{
		QuickFilters: []quickfilter.QuickFilterSpec{
			quickfilter.Quick(quickfilter.QuickFilterDetention),
		},
		Timezone: "UTC",
	}
	require.NoError(t, svc.Prepare(t.Context(), pagination.TenantInfo{}, opts))
	require.NotNil(t, opts.QuickFilterBasis)
	assert.Equal(t, int64(30), opts.QuickFilterBasis.Detention.ThresholdMinutes)
}
