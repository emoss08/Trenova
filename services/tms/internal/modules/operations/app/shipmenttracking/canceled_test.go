package shipmenttracking

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuild_SaysWhoCanceledAShipmentWhenAndWhy(t *testing.T) {
	t.Parallel()

	sp, _ := inTransitShipment()
	sp.Status = shipment.StatusCanceled
	sp.CanceledAt = ptr(now - 600)
	sp.CancelReason = "  Range pulled the load by phone "
	sp.CanceledBy = &tenant.User{Name: "Admin"}

	snapshot := Build(Input{Shipment: sp, Now: now, Timezone: "UTC"})

	require.NotNil(t, snapshot.Canceled)
	assert.Equal(t, CanceledSnapshot{
		At: now - 600, By: "Admin", Reason: "Range pulled the load by phone",
	}, *snapshot.Canceled)

	var canceled *Fact
	for _, fact := range snapshot.Facts() {
		if fact.Name == "canceled" {
			canceled = &fact
		}
	}
	require.NotNil(t, canceled)
	assert.Equal(t, "by Admin: Range pulled the load by phone", canceled.Value)
	assert.EqualValues(t, now-600, canceled.SeenAt)
}

func TestBuild_ALiveShipmentCarriesNoCancellation(t *testing.T) {
	t.Parallel()

	sp, _ := inTransitShipment()
	sp.CancelReason = "left over from an earlier cancel"

	snapshot := Build(Input{Shipment: sp, Now: now, Timezone: "UTC"})

	assert.Nil(t, snapshot.Canceled)
	for _, fact := range snapshot.Facts() {
		assert.NotEqual(t, "canceled", fact.Name)
	}
}

func TestCanceledText_SaysWhatIsKnown(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "by Admin, no reason given", (&CanceledSnapshot{By: "Admin"}).text())
	assert.Equal(t, "Range pulled it", (&CanceledSnapshot{Reason: "Range pulled it"}).text())
	assert.Equal(t, "no reason given", (&CanceledSnapshot{}).text())
}
