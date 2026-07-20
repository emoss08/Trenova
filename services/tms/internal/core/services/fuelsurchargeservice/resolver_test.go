package fuelsurchargeservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/fuelsurcharge"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubCustomerRepo struct {
	repositories.CustomerRepository
	profile *customer.CustomerBillingProfile
	err     error
}

func (s *stubCustomerRepo) GetBillingProfile(
	context.Context,
	pulid.ID,
) (*customer.CustomerBillingProfile, error) {
	return s.profile, s.err
}

type stubProgramRepo struct {
	repositories.FuelSurchargeProgramRepository
	program *fuelsurcharge.FuelSurchargeProgram
	err     error
}

func (s *stubProgramRepo) GetByID(
	context.Context,
	*repositories.GetFuelSurchargeProgramByIDRequest,
) (*fuelsurcharge.FuelSurchargeProgram, error) {
	return s.program, s.err
}

type stubPriceRepo struct {
	repositories.FuelIndexPriceRepository
	prices []*fuelsurcharge.FuelIndexPrice
	err    error
}

func (s *stubPriceRepo) GetLatestOnOrBefore(
	context.Context,
	*repositories.GetLatestFuelPricesRequest,
) ([]*fuelsurcharge.FuelIndexPrice, error) {
	return s.prices, s.err
}

type stubOrgCacheRepo struct {
	repositories.OrganizationCacheRepository
	org *tenant.Organization
}

func (s *stubOrgCacheRepo) GetByID(context.Context, pulid.ID) (*tenant.Organization, error) {
	if s.org == nil {
		return nil, errors.New("not found")
	}
	return s.org, nil
}

type resolverFixture struct {
	programID pulid.ID
	chargeID  pulid.ID
	indexID   pulid.ID
	service   *Service
	shipment  *shipment.Shipment
	program   *fuelsurcharge.FuelSurchargeProgram
}

func newResolverFixture(t *testing.T) *resolverFixture {
	t.Helper()

	programID := pulid.MustNew("fsp_")
	chargeID := pulid.MustNew("acc_")
	indexID := pulid.MustNew("fidx_")

	program := stepProgram()
	program.ID = programID
	program.AccessorialChargeID = chargeID
	program.FuelIndexID = indexID
	program.FuelIndex = &fuelsurcharge.FuelIndex{
		ID:          indexID,
		Code:        "DOE_US",
		Source:      fuelsurcharge.IndexSourceEIA,
		EIASeriesID: "EMD_EPD2D_PTE_NUS_DPG",
	}

	distance := 1000.0
	entity := &shipment.Shipment{
		ID:             pulid.MustNew("shp_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		CustomerID:     pulid.MustNew("cus_"),
		ServiceTypeID:  pulid.MustNew("st_"),
		ShipmentTypeID: pulid.MustNew("sht_"),
		CreatedAt:      date("2026-07-16").Unix(),
		Moves: []*shipment.ShipmentMove{
			{Distance: &distance},
		},
	}

	svc := &Service{
		l: zap.NewNop(),
		customerRepo: &stubCustomerRepo{
			profile: &customer.CustomerBillingProfile{
				FuelSurchargeMode:      customer.FuelSurchargeModeProgram,
				FuelSurchargeProgramID: &programID,
			},
		},
		programRepo: &stubProgramRepo{program: program},
		priceRepo: &stubPriceRepo{prices: []*fuelsurcharge.FuelIndexPrice{
			price("2026-07-13", "3.70"),
			price("2026-07-06", "3.60"),
		}},
		orgCacheRepo: &stubOrgCacheRepo{},
		now:          func() int64 { return date("2026-07-16").Unix() },
	}

	return &resolverFixture{
		programID: programID,
		chargeID:  chargeID,
		indexID:   indexID,
		service:   svc,
		shipment:  entity,
		program:   program,
	}
}

func (f *resolverFixture) resolve(ctx context.Context) (*services.ResolvedFuelSurcharge, error) {
	return f.service.ResolveShipmentCharge(ctx, &services.ResolveShipmentChargeRequest{
		Shipment: f.shipment,
		Linehaul: decimal.NewFromInt(2500),
	})
}

func TestResolveShipmentCharge_PerMileStep(t *testing.T) {
	t.Parallel()

	f := newResolverFixture(t)

	resolved, err := f.resolve(t.Context())

	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, f.programID, resolved.ProgramID)
	assert.Equal(t, f.chargeID, resolved.AccessorialChargeID)
	assert.True(t, dec("500").Equal(resolved.Amount), "got %s", resolved.Amount.String())

	require.NotNil(t, resolved.Detail)
	assert.Equal(t, "2026-07-13", resolved.Detail.PriceDate)
	assert.Equal(t, "DOE_US", resolved.Detail.IndexCode)
	assert.Equal(t, "EMD_EPD2D_PTE_NUS_DPG", resolved.Detail.EIASeriesID)
	assert.False(t, resolved.Detail.UsedFallback)
	assert.False(t, resolved.Detail.Stale)
	require.NotNil(t, resolved.Detail.RatePerMile)
	assert.InDelta(t, 0.5, *resolved.Detail.RatePerMile, 0.0001)
	require.NotNil(t, resolved.Detail.Miles)
	assert.InDelta(t, 1000, *resolved.Detail.Miles, 0.0001)
}

func TestResolveShipmentCharge_NoProgramAssigned(t *testing.T) {
	t.Parallel()

	f := newResolverFixture(t)
	f.service.customerRepo = &stubCustomerRepo{
		profile: &customer.CustomerBillingProfile{},
	}

	resolved, err := f.resolve(t.Context())

	require.NoError(t, err)
	assert.Nil(t, resolved)
}

func TestResolveShipmentCharge_ModeGatesApplication(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mode    customer.FuelSurchargeMode
		wantNil bool
	}{
		{"none mode skips even with program assigned", customer.FuelSurchargeModeNone, true},
		{"fuel included skips even with program assigned", customer.FuelSurchargeModeFuelIncluded, true},
		{"program mode applies", customer.FuelSurchargeModeProgram, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newResolverFixture(t)
			f.service.customerRepo = &stubCustomerRepo{
				profile: &customer.CustomerBillingProfile{
					FuelSurchargeMode:      tt.mode,
					FuelSurchargeProgramID: &f.programID,
				},
			}

			resolved, err := f.resolve(t.Context())

			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, resolved)
			} else {
				assert.NotNil(t, resolved)
			}
		})
	}
}

func TestResolveShipmentCharge_BillingProfileErrorIsSwallowed(t *testing.T) {
	t.Parallel()

	f := newResolverFixture(t)
	f.service.customerRepo = &stubCustomerRepo{err: errors.New("boom")}

	resolved, err := f.resolve(t.Context())

	require.NoError(t, err)
	assert.Nil(t, resolved)
}

func TestResolveShipmentCharge_InactiveProgram(t *testing.T) {
	t.Parallel()

	f := newResolverFixture(t)
	f.program.Status = fuelsurcharge.ProgramStatusInactive

	resolved, err := f.resolve(t.Context())

	require.NoError(t, err)
	assert.Nil(t, resolved)
}

func TestResolveShipmentCharge_OutsideEffectiveWindow(t *testing.T) {
	t.Parallel()

	f := newResolverFixture(t)
	end := date("2026-07-01").Unix()
	f.program.EffectiveEndDate = &end

	resolved, err := f.resolve(t.Context())

	require.NoError(t, err)
	assert.Nil(t, resolved)
}

func TestResolveShipmentCharge_ApplicabilityFilters(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		configure func(f *resolverFixture)
		wantNil   bool
	}{
		{
			name: "matching service type",
			configure: func(f *resolverFixture) {
				f.program.ServiceTypeIDs = []pulid.ID{f.shipment.ServiceTypeID}
			},
			wantNil: false,
		},
		{
			name: "non matching service type",
			configure: func(f *resolverFixture) {
				f.program.ServiceTypeIDs = []pulid.ID{pulid.MustNew("st_")}
			},
			wantNil: true,
		},
		{
			name: "non matching shipment type",
			configure: func(f *resolverFixture) {
				f.program.ShipmentTypeIDs = []pulid.ID{pulid.MustNew("sht_")}
			},
			wantNil: true,
		},
		{
			name: "equipment filter with nil shipment equipment",
			configure: func(f *resolverFixture) {
				f.program.TractorTypeIDs = []pulid.ID{pulid.MustNew("et_")}
			},
			wantNil: true,
		},
		{
			name: "matching tractor type",
			configure: func(f *resolverFixture) {
				tractorType := pulid.MustNew("et_")
				f.shipment.TractorTypeID = tractorType
				f.program.TractorTypeIDs = []pulid.ID{tractorType}
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			f := newResolverFixture(t)
			tt.configure(f)

			resolved, err := f.resolve(t.Context())

			require.NoError(t, err)
			if tt.wantNil {
				assert.Nil(t, resolved)
			} else {
				assert.NotNil(t, resolved)
			}
		})
	}
}

func TestResolveShipmentCharge_MissingPriceSkip(t *testing.T) {
	t.Parallel()

	f := newResolverFixture(t)
	f.program.MissingPriceFallback = fuelsurcharge.FallbackSkip
	f.service.priceRepo = &stubPriceRepo{}

	resolved, err := f.resolve(t.Context())

	require.NoError(t, err)
	assert.Nil(t, resolved)
}

func TestResolveShipmentCharge_FallbackFlagged(t *testing.T) {
	t.Parallel()

	f := newResolverFixture(t)
	f.service.priceRepo = &stubPriceRepo{prices: []*fuelsurcharge.FuelIndexPrice{
		price("2026-07-06", "3.60"),
	}}

	resolved, err := f.resolve(t.Context())

	require.NoError(t, err)
	require.NotNil(t, resolved)
	require.NotNil(t, resolved.Detail)
	assert.True(t, resolved.Detail.UsedFallback)
}

func TestResolveShipmentCharge_TenderDateBasis(t *testing.T) {
	t.Parallel()

	f := newResolverFixture(t)
	f.program.DateBasis = fuelsurcharge.DateBasisTenderDate
	f.shipment.CreatedAt = date("2026-07-14").Unix()

	resolved, err := f.resolve(t.Context())

	require.NoError(t, err)
	require.NotNil(t, resolved)
	assert.Equal(t, "2026-07-06", resolved.Detail.PriceDate)
	assert.False(t, resolved.Detail.UsedFallback)
}
