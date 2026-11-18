package managedusageservice_test

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUsageIsNilWithoutAStateOrAWindow(t *testing.T) {
	t.Parallel()

	storage := &fakeLevel{meter: platformcatalog.MeterDocumentStorageBytes, level: 7}
	svc := newService(&fakeRepo{}, nil, []managedusageports.LevelReader{storage})
	usage, err := svc.Usage(t.Context())
	require.NoError(t, err)
	assert.Nil(t, usage, "no state yet")

	svc = newService(&fakeRepo{state: &managedusage.State{
		Meters: []managedusage.Meter{{
			Meter:    platformcatalog.MeterDocumentStorageBytes,
			Unit:     managedusage.UnitBytes,
			Quantity: 5,
		}},
	}}, nil, []managedusageports.LevelReader{storage})
	usage, err = svc.Usage(t.Context())
	require.NoError(t, err)
	assert.Nil(t, usage, "a state with no billing period")
}

func TestUsageReadsALevelMeterLiveAndACountedMeterFromTheState(t *testing.T) {
	t.Parallel()

	storage := &fakeLevel{meter: platformcatalog.MeterDocumentStorageBytes, level: 18_400_000_000}
	unnamed := &fakeLevel{
		meter: platformcatalog.MeterOrganizationsTotal,
		err:   errors.New("a meter the state does not name is never read"),
	}
	svc := newService(&fakeRepo{state: &managedusage.State{
		WindowStart: now - hour,
		WindowEnd:   now + hour,
		Meters: []managedusage.Meter{
			{
				Meter:     platformcatalog.MeterDocumentStorageBytes,
				Unit:      managedusage.UnitBytes,
				Quantity:  1,
				Allowance: 100_000_000_000,
			},
			{
				Meter:     platformcatalog.MeterShipmentsTotal,
				Unit:      managedusage.UnitCount,
				Quantity:  312,
				Allowance: 1_500,
			},
		},
	}}, nil, []managedusageports.LevelReader{storage, unnamed})

	usage, err := svc.Usage(t.Context())
	require.NoError(t, err)
	require.NotNil(t, usage)
	assert.Equal(t, now-hour, usage.PeriodStart)
	assert.Equal(t, now+hour, usage.PeriodEnd)
	require.Len(t, usage.Meters, 2)
	assert.Equal(t, platformcatalog.MeterShipmentsTotal, usage.Meters[0].Meter.Key)
	assert.Equal(t, int64(312), usage.Meters[0].Used)
	assert.Equal(t, platformcatalog.MeterDocumentStorageBytes, usage.Meters[1].Meter.Key)
	assert.Equal(
		t,
		int64(18_400_000_000),
		usage.Meters[1].Used,
		"the live level, not the relayed quantity",
	)
	assert.Equal(t, int64(100_000_000_000), usage.Meters[1].Included)
}

func TestUsageReturnsALevelReadFailure(t *testing.T) {
	t.Parallel()

	down := errors.New("database down")
	svc := newService(&fakeRepo{state: &managedusage.State{
		WindowStart: now - hour,
		WindowEnd:   now + hour,
		Meters: []managedusage.Meter{{
			Meter: platformcatalog.MeterOrganizationsTotal,
			Unit:  managedusage.UnitCount,
		}},
	}}, nil, []managedusageports.LevelReader{
		&fakeLevel{meter: platformcatalog.MeterOrganizationsTotal, err: down},
	})

	_, err := svc.Usage(t.Context())
	require.ErrorIs(t, err, down)
}
