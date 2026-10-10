package driverpay

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func validComponent() *PayProfileComponent {
	return &PayProfileComponent{
		ID:             pulid.MustNew("dppc_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		PayProfileID:   pulid.MustNew("dpp_"),
		Kind:           ComponentKindLinehaul,
		Method:         CalcMethodPerLoadedMile,
		Rate:           decimal.NewFromFloat(0.55),
		IsActive:       true,
	}
}

func validPayProfile() PayProfile {
	return PayProfile{
		ID:             pulid.MustNew("dpp_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		OrganizationID: pulid.MustNew("org_"),
		Status:         domaintypes.StatusActive,
		Name:           "OTR Company Driver",
		Classification: PayeeClassificationCompanyDriver,
		CurrencyCode:   "USD",
		Components:     []*PayProfileComponent{validComponent()},
	}
}

func TestPayProfile_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		mutate  func(p *PayProfile)
		wantErr bool
	}{
		{
			name:    "valid profile passes",
			mutate:  func(p *PayProfile) {},
			wantErr: false,
		},
		{
			name:    "missing name fails",
			mutate:  func(p *PayProfile) { p.Name = "" },
			wantErr: true,
		},
		{
			name:    "invalid classification fails",
			mutate:  func(p *PayProfile) { p.Classification = "LeaseOperator" },
			wantErr: true,
		},
		{
			name:    "negative guaranteed minimum fails",
			mutate:  func(p *PayProfile) { p.GuaranteedPeriodMinimumMinor = -1 },
			wantErr: true,
		},
		{
			name:    "negative per diem rate fails",
			mutate:  func(p *PayProfile) { p.PerDiemRatePerMile = decimal.NewFromInt(-1) },
			wantErr: true,
		},
		{
			name:    "negative per diem cap fails",
			mutate:  func(p *PayProfile) { p.PerDiemDailyCapMinor = -1 },
			wantErr: true,
		},
		{
			name:    "no components fails",
			mutate:  func(p *PayProfile) { p.Components = nil },
			wantErr: true,
		},
		{
			name:    "nil component fails",
			mutate:  func(p *PayProfile) { p.Components = []*PayProfileComponent{nil} },
			wantErr: true,
		},
		{
			name: "invalid component surfaces through the profile",
			mutate: func(p *PayProfile) {
				p.Components[0].Method = "PerParsec"
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			entity := validPayProfile()
			tt.mutate(&entity)
			multiErr := errortypes.NewMultiError()
			entity.Validate(multiErr)
			assert.Equal(t, tt.wantErr, multiErr.HasErrors())
		})
	}
}

func TestPayProfileComponent_Validate(t *testing.T) {
	t.Parallel()

	minAmount := int64(10_000)
	maxAmount := int64(5_000)

	tests := []struct {
		name    string
		mutate  func(c *PayProfileComponent)
		wantErr bool
	}{
		{
			name:    "valid per mile component passes",
			mutate:  func(c *PayProfileComponent) {},
			wantErr: false,
		},
		{
			name:    "invalid kind fails",
			mutate:  func(c *PayProfileComponent) { c.Kind = "Parking" },
			wantErr: true,
		},
		{
			name: "custom kind without description fails",
			mutate: func(c *PayProfileComponent) {
				c.Kind = ComponentKindCustom
				c.Description = ""
			},
			wantErr: true,
		},
		{
			name: "custom kind with description passes",
			mutate: func(c *PayProfileComponent) {
				c.Kind = ComponentKindCustom
				c.Description = "Northeast regional premium"
			},
			wantErr: false,
		},
		{
			name: "percent of revenue without basis fails",
			mutate: func(c *PayProfileComponent) {
				c.Method = CalcMethodPercentOfRevenue
				c.Rate = decimal.NewFromInt(25)
			},
			wantErr: true,
		},
		{
			name: "percent of revenue above 100 fails",
			mutate: func(c *PayProfileComponent) {
				c.Method = CalcMethodPercentOfRevenue
				c.RevenueBasis = RevenueBasisLinehaul
				c.Rate = decimal.NewFromInt(101)
			},
			wantErr: true,
		},
		{
			name: "percent of revenue with basis passes",
			mutate: func(c *PayProfileComponent) {
				c.Method = CalcMethodPercentOfRevenue
				c.RevenueBasis = RevenueBasisTotalRevenue
				c.Rate = decimal.NewFromInt(72)
			},
			wantErr: false,
		},
		{
			name:    "negative per mile rate fails",
			mutate:  func(c *PayProfileComponent) { c.Rate = decimal.NewFromInt(-1) },
			wantErr: true,
		},
		{
			name: "negative flat rate fails",
			mutate: func(c *PayProfileComponent) {
				c.Method = CalcMethodFlatPerShipment
				c.Rate = decimal.NewFromInt(-1)
			},
			wantErr: true,
		},
		{
			name: "bands on non per mile method fails",
			mutate: func(c *PayProfileComponent) {
				c.Method = CalcMethodFlatPerShipment
				c.Bands = []MileageBand{
					{
						MinMiles: 0,
						MaxMiles: 100,
						Rate:     decimal.NewFromFloat(0.5),
					},
				}
			},
			wantErr: true,
		},
		{
			name:    "negative free time fails",
			mutate:  func(c *PayProfileComponent) { c.FreeTimeMinutes = -1 },
			wantErr: true,
		},
		{
			name: "detention with non hourly method fails",
			mutate: func(c *PayProfileComponent) {
				c.Kind = ComponentKindDetention
				c.Method = CalcMethodPerDay
			},
			wantErr: true,
		},
		{
			name: "detention with hourly method passes",
			mutate: func(c *PayProfileComponent) {
				c.Kind = ComponentKindDetention
				c.Method = CalcMethodPerHour
				c.Rate = decimal.NewFromInt(50)
			},
			wantErr: false,
		},
		{
			name: "minimum above maximum fails",
			mutate: func(c *PayProfileComponent) {
				c.MinAmountMinor = &minAmount
				c.MaxAmountMinor = &maxAmount
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			component := validComponent()
			tt.mutate(component)
			multiErr := errortypes.NewMultiError()
			component.Validate(multiErr)
			assert.Equal(t, tt.wantErr, multiErr.HasErrors())
		})
	}
}

func TestPayProfileComponent_ValidateBands(t *testing.T) {
	t.Parallel()

	rate := decimal.NewFromFloat(0.5)

	tests := []struct {
		name    string
		bands   []MileageBand
		wantErr bool
	}{
		{
			name: "ascending bands with open end pass",
			bands: []MileageBand{
				{MinMiles: 0, MaxMiles: 500, Rate: rate},
				{MinMiles: 500, MaxMiles: 1000, Rate: rate},
				{MinMiles: 1000, MaxMiles: 0, Rate: rate},
			},
			wantErr: false,
		},
		{
			name:    "negative minimum fails",
			bands:   []MileageBand{{MinMiles: -1, MaxMiles: 100, Rate: rate}},
			wantErr: true,
		},
		{
			name:    "maximum not above minimum fails",
			bands:   []MileageBand{{MinMiles: 100, MaxMiles: 100, Rate: rate}},
			wantErr: true,
		},
		{
			name: "negative band rate fails",
			bands: []MileageBand{
				{MinMiles: 0, MaxMiles: 100, Rate: decimal.NewFromInt(-1)},
			},
			wantErr: true,
		},
		{
			name: "overlapping bands fail",
			bands: []MileageBand{
				{MinMiles: 0, MaxMiles: 500, Rate: rate},
				{MinMiles: 400, MaxMiles: 800, Rate: rate},
			},
			wantErr: true,
		},
		{
			name: "open ended band before the last fails",
			bands: []MileageBand{
				{MinMiles: 0, MaxMiles: 0, Rate: rate},
				{MinMiles: 500, MaxMiles: 1000, Rate: rate},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			component := validComponent()
			component.Bands = tt.bands
			multiErr := errortypes.NewMultiError()
			component.Validate(multiErr)
			assert.Equal(t, tt.wantErr, multiErr.HasErrors())
		})
	}
}

func TestPayProfileComponent_ResolveMileageRate(t *testing.T) {
	t.Parallel()

	fallback := decimal.NewFromFloat(0.40)
	component := &PayProfileComponent{
		Method: CalcMethodPerTotalMile,
		Rate:   fallback,
		Bands: []MileageBand{
			{MinMiles: 0, MaxMiles: 500, Rate: decimal.NewFromFloat(0.50)},
			{MinMiles: 500, MaxMiles: 1000, Rate: decimal.NewFromFloat(0.55)},
			{MinMiles: 1000, MaxMiles: 0, Rate: decimal.NewFromFloat(0.60)},
		},
	}

	tests := []struct {
		name  string
		miles int64
		want  string
	}{
		{name: "first band", miles: 100, want: "0.5"},
		{name: "band boundary belongs to the next band", miles: 500, want: "0.55"},
		{name: "second band", miles: 999, want: "0.55"},
		{name: "open ended band", miles: 5000, want: "0.6"},
		{name: "zero miles uses first band", miles: 0, want: "0.5"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := component.ResolveMileageRate(decimal.NewFromInt(tt.miles))
			assert.Equal(t, tt.want, got.String())
		})
	}

	t.Run("no bands falls back to the component rate", func(t *testing.T) {
		t.Parallel()
		flat := &PayProfileComponent{Rate: fallback}
		assert.Equal(
			t,
			fallback.String(),
			flat.ResolveMileageRate(decimal.NewFromInt(750)).String(),
		)
	})

	t.Run(
		"miles outside every closed band fall back to the component rate",
		func(t *testing.T) {
			t.Parallel()
			closed := &PayProfileComponent{
				Rate: fallback,
				Bands: []MileageBand{
					{
						MinMiles: 100,
						MaxMiles: 200,
						Rate:     decimal.NewFromFloat(0.50),
					},
				},
			}
			assert.Equal(
				t,
				fallback.String(),
				closed.ResolveMileageRate(decimal.NewFromInt(50)).String(),
			)
			assert.Equal(
				t,
				fallback.String(),
				closed.ResolveMileageRate(decimal.NewFromInt(300)).String(),
			)
		},
	)
}
