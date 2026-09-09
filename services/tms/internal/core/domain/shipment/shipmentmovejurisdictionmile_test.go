package shipment

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func validJurisdictionMile() *ShipmentMoveJurisdictionMile {
	return &ShipmentMoveJurisdictionMile{
		OrganizationID:   pulid.MustNew("org_"),
		BusinessUnitID:   pulid.MustNew("bu_"),
		ShipmentMoveID:   pulid.MustNew("sm_"),
		ShipmentID:       pulid.MustNew("shp_"),
		CountryCode:      "US",
		JurisdictionCode: "TX",
		Sequence:         0,
		Distance:         120.5,
		DistanceUnits:    JurisdictionDistanceUnitsMiles,
		Loaded:           true,
		Source:           JurisdictionMileSourceRouteCalculation,
		Provider:         "PCMiler",
		DataVersion:      "Current",
		CalculatedAt:     1_700_000_000,
	}
}

func TestJurisdictionMileSource_IsValid(t *testing.T) {
	t.Parallel()

	assert.True(t, JurisdictionMileSourceRouteCalculation.IsValid())
	assert.True(t, JurisdictionMileSourceManual.IsValid())
	assert.False(t, JurisdictionMileSource("").IsValid())
	assert.False(t, JurisdictionMileSource("Telematics").IsValid())
	assert.Equal(t, "Manual", JurisdictionMileSourceManual.String())
}

func TestShipmentMoveJurisdictionMile_Validate(t *testing.T) {
	t.Parallel()

	negative := -1.0

	tests := []struct {
		name    string
		mutate  func(m *ShipmentMoveJurisdictionMile)
		wantErr bool
	}{
		{name: "valid miles row", mutate: func(*ShipmentMoveJurisdictionMile) {}},
		{
			name: "valid kilometers row",
			mutate: func(m *ShipmentMoveJurisdictionMile) {
				m.DistanceUnits = JurisdictionDistanceUnitsKilometers
			},
		},
		{
			name: "valid manual row with toll and ferry",
			mutate: func(m *ShipmentMoveJurisdictionMile) {
				m.Source = JurisdictionMileSourceManual
				toll, ferry := 10.0, 0.0
				m.TollDistance = &toll
				m.FerryDistance = &ferry
			},
		},
		{
			name: "valid canadian province",
			mutate: func(m *ShipmentMoveJurisdictionMile) {
				m.CountryCode = "CA"
				m.JurisdictionCode = "ON"
			},
		},
		{
			name: "valid zero distance",
			mutate: func(m *ShipmentMoveJurisdictionMile) {
				m.Distance = 0
			},
		},
		{
			name:    "missing move",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.ShipmentMoveID = "" },
			wantErr: true,
		},
		{
			name:    "missing shipment",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.ShipmentID = "" },
			wantErr: true,
		},
		{
			name:    "blank country",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.CountryCode = "" },
			wantErr: true,
		},
		{
			name:    "lowercase country",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.CountryCode = "us" },
			wantErr: true,
		},
		{
			name:    "three letter country",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.CountryCode = "USA" },
			wantErr: true,
		},
		{
			name:    "blank jurisdiction code",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.JurisdictionCode = "" },
			wantErr: true,
		},
		{
			name:    "lowercase jurisdiction code",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.JurisdictionCode = "tx" },
			wantErr: true,
		},
		{
			name:    "jurisdiction code too long",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.JurisdictionCode = "ABCDEF" },
			wantErr: true,
		},
		{
			name:    "negative sequence",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.Sequence = -1 },
			wantErr: true,
		},
		{
			name:    "negative distance",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.Distance = -0.5 },
			wantErr: true,
		},
		{
			name:    "blank units",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.DistanceUnits = "" },
			wantErr: true,
		},
		{
			name:    "unknown units",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.DistanceUnits = "NauticalMiles" },
			wantErr: true,
		},
		{
			name:    "negative toll distance",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.TollDistance = &negative },
			wantErr: true,
		},
		{
			name:    "negative ferry distance",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.FerryDistance = &negative },
			wantErr: true,
		},
		{
			name:    "blank source",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.Source = "" },
			wantErr: true,
		},
		{
			name:    "unknown source",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.Source = "Telematics" },
			wantErr: true,
		},
		{
			name:    "zero calculated at",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.CalculatedAt = 0 },
			wantErr: true,
		},
		{
			name:    "negative calculated at",
			mutate:  func(m *ShipmentMoveJurisdictionMile) { m.CalculatedAt = -5 },
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			row := validJurisdictionMile()
			tt.mutate(row)
			multiErr := errortypes.NewMultiError()

			row.Validate(multiErr)

			assert.Equal(t, tt.wantErr, multiErr.HasErrors(), multiErr.Error())
		})
	}
}

func TestShipmentMoveJurisdictionMile_Key(t *testing.T) {
	t.Parallel()

	row := validJurisdictionMile()
	assert.Equal(t, "US_TX", row.Key())

	row.CountryCode = "CA"
	row.JurisdictionCode = "ON"
	assert.Equal(t, "CA_ON", row.Key())
}

func TestShipmentMoveJurisdictionMile_DistanceInMiles(t *testing.T) {
	t.Parallel()

	miles := validJurisdictionMile()
	assert.InDelta(t, 120.5, miles.DistanceInMiles(), 0.000001)

	kilometers := validJurisdictionMile()
	kilometers.Distance = 160.9344
	kilometers.DistanceUnits = JurisdictionDistanceUnitsKilometers
	assert.InDelta(t, 100.0, kilometers.DistanceInMiles(), 0.000001)

	lowercase := validJurisdictionMile()
	lowercase.Distance = 1.609344
	lowercase.DistanceUnits = "kilometers"
	assert.InDelta(t, 1.0, lowercase.DistanceInMiles(), 0.000001)
}

func TestShipmentMoveJurisdictionMile_BeforeAppendModelDefaults(t *testing.T) {
	t.Parallel()

	row := &ShipmentMoveJurisdictionMile{}
	require.NoError(t, row.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil)))

	assert.False(t, row.ID.IsNil())
	assert.Equal(t, JurisdictionMileSourceRouteCalculation, row.Source)
	assert.Equal(t, JurisdictionDistanceUnitsMiles, row.DistanceUnits)
	assert.NotZero(t, row.CalculatedAt)
	assert.NotZero(t, row.CreatedAt)
	assert.Equal(t, row.CreatedAt, row.UpdatedAt)

	explicit := &ShipmentMoveJurisdictionMile{
		Source:        JurisdictionMileSourceManual,
		DistanceUnits: JurisdictionDistanceUnitsKilometers,
		CalculatedAt:  42,
	}
	require.NoError(t, explicit.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil)))
	assert.Equal(t, JurisdictionMileSourceManual, explicit.Source)
	assert.Equal(t, JurisdictionDistanceUnitsKilometers, explicit.DistanceUnits)
	assert.Equal(t, int64(42), explicit.CalculatedAt)
}

func TestShipmentMove_JurisdictionMilesSumMiles(t *testing.T) {
	t.Parallel()

	var nilMove *ShipmentMove
	assert.Zero(t, nilMove.JurisdictionMilesSumMiles())

	empty := &ShipmentMove{}
	assert.Zero(t, empty.JurisdictionMilesSumMiles())

	move := &ShipmentMove{
		JurisdictionMiles: []*ShipmentMoveJurisdictionMile{
			{CountryCode: "US", JurisdictionCode: "TX", Distance: 100, DistanceUnits: JurisdictionDistanceUnitsMiles},
			nil,
			{CountryCode: "US", JurisdictionCode: "OK", Distance: 160.9344, DistanceUnits: JurisdictionDistanceUnitsKilometers},
		},
	}
	assert.InDelta(t, 200.0, move.JurisdictionMilesSumMiles(), 0.000001)
}
