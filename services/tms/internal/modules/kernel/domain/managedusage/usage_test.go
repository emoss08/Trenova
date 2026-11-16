package managedusage_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsageOfIsNilWithoutAStateOrAWindow(t *testing.T) {
	t.Parallel()

	assert.Nil(t, managedusage.UsageOf(nil, nil))
	assert.Nil(t, managedusage.UsageOf(&managedusage.State{
		Meters: []managedusage.Meter{{Meter: platformcatalog.MeterShipmentsTotal}},
	}, nil))
}

func TestUsageOfNamesKnownMetersInCatalogueOrderAndTakesLiveLevels(t *testing.T) {
	t.Parallel()

	usage := managedusage.UsageOf(&managedusage.State{
		WindowStart: 100,
		WindowEnd:   200,
		Meters: []managedusage.Meter{
			{
				Meter:     platformcatalog.MeterOrganizationsTotal,
				Unit:      managedusage.UnitCount,
				Quantity:  3,
				Allowance: 1,
				HardCap:   true,
			},
			{Meter: "teleport.jumps", Unit: managedusage.UnitCount, Quantity: 9},
			{
				Meter:     platformcatalog.MeterShipmentsTotal,
				Unit:      managedusage.UnitCount,
				Quantity:  1_600,
				Allowance: 1_500,
				Stopped:   true,
			},
		},
	}, map[platformcatalog.MeterKey]int64{platformcatalog.MeterOrganizationsTotal: 1})

	require.NotNil(t, usage)
	assert.Equal(t, int64(100), usage.PeriodStart)
	assert.Equal(t, int64(200), usage.PeriodEnd)
	require.Len(t, usage.Meters, 2, "a meter the catalogue cannot name is left out")
	assert.Equal(t, "Shipments", usage.Meters[0].Meter.Label)
	assert.Equal(t, int64(1_600), usage.Meters[0].Used)
	assert.True(t, usage.Meters[0].Stopped)
	assert.Equal(t, "Organizations", usage.Meters[1].Meter.Label)
	assert.Equal(t, int64(1), usage.Meters[1].Used, "a level meter is read live")
	assert.Equal(t, int64(1), usage.Meters[1].Included)
	assert.True(t, usage.Meters[1].HardCap)
}
