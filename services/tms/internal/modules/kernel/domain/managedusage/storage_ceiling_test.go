package managedusage_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/stretchr/testify/assert"
)

func sharedStorage(level int64, stopped bool) managedusage.Meter {
	return managedusage.Meter{
		Meter:     platformcatalog.MeterDocumentStorageBytes,
		Unit:      managedusage.UnitBytes,
		Quantity:  level,
		Allowance: 25_000_000_000,
		Ceiling:   250_000_000_000,
		Stopped:   stopped,
	}
}

func TestStorageAtItsCeilingRefusesAnUploadAndNamesTheCeiling(t *testing.T) {
	t.Parallel()

	full := sharedStorage(250_000_000_000, true)
	d := managedusage.DecideAt(state(true, full), full.Meter, 250_000_000_000, 1)
	assert.False(t, d.Allowed)
	assert.Equal(t, managedusage.ReasonCeiling, d.Reason)
	assert.True(t, d.CeilingReached())

	err := managedusage.CeilingRefusal(full.Meter, d.Ceiling, d.Used, "managed")
	assert.Equal(t, "Your organization's document storage is full at {0}, the most your plan "+
		"can hold. Delete documents you no longer need, or move to a larger plan from your "+
		"Trenova account, to keep uploading.", err.Message)
	assert.Equal(t, []any{"250 GB"}, err.Args)
	assert.NotContains(t, err.Message, "billing period", "storage is a level, not a count per period")
}

func TestStorageStoppedAtItsCeilingIsAcceptedAgainOnceSpaceIsFreed(t *testing.T) {
	t.Parallel()

	stopped := sharedStorage(250_000_000_000, true)
	d := managedusage.DecideAt(state(true, stopped), stopped.Meter, 249_000_000_000, 500_000_000)
	assert.True(t, d.Allowed, "the stop was the ceiling, and the live level is below it now")
	assert.Equal(t, managedusage.ReasonWithin, d.Reason)

	d = managedusage.DecideAt(state(true, stopped), stopped.Meter, 249_000_000_000, 2_000_000_000)
	assert.False(t, d.Allowed, "an upload that would cross the ceiling is refused before it is stored")
	assert.Equal(t, managedusage.ReasonCeiling, d.Reason)
}

func TestAStopThatIsNotTheCeilingIsHonouredWhateverTheLevel(t *testing.T) {
	t.Parallel()

	stopped := sharedStorage(1_000, true)
	d := managedusage.DecideAt(state(true, stopped), stopped.Meter, 10, 1)
	assert.False(t, d.Allowed)
	assert.Equal(t, managedusage.ReasonStopped, d.Reason)
}

func TestAStorageMeterWithoutACeilingIsUnaffected(t *testing.T) {
	t.Parallel()

	dedicated := managedusage.Meter{
		Meter:     platformcatalog.MeterDocumentStorageBytes,
		Unit:      managedusage.UnitBytes,
		Quantity:  900_000_000_000,
		Allowance: 25_000_000_000,
	}
	d := managedusage.DecideAt(state(true, dedicated), dedicated.Meter, 900_000_000_000, 1_000)
	assert.True(t, d.Allowed)
	assert.False(t, d.CeilingReached())
}

func TestDecimalBytesSaysAPlanAmountTheWayThePlanDoes(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "250 GB", managedusage.DecimalBytes(250_000_000_000))
	assert.Equal(t, "25 GB", managedusage.DecimalBytes(25_000_000_000))
	assert.Equal(t, "1.5 TB", managedusage.DecimalBytes(1_500_000_000_000))
	assert.Equal(t, "999 bytes", managedusage.DecimalBytes(999))
}
