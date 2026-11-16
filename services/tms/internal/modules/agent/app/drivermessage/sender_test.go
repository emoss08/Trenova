package drivermessage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/modules/agent/app/drivermessage"
	"github.com/emoss08/trenova/internal/modules/agent/app/playsend/playsendtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func message(workerID pulid.ID) *drivermessage.Message {
	return &drivermessage.Message{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		WorkerID:   workerID,
		Title:      "Call dispatch",
		Text:       "Call when you are parked.",
		ShipmentID: pulid.MustNew("shp_"),
	}
}

// A driver with a login is reached in Dash, and the message opens the load
// it is about.
func TestSenderReachesADriverWithALoginInDash(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	drivers := &playsendtest.Drivers{Known: map[pulid.ID]playsendtest.Driver{
		workerID: {Name: "Laura Wright", OnDash: true},
	}}
	sender := playsendtest.Messages(drivers)
	sent := message(workerID)

	reach, err := sender.Reach(t.Context(), sent)
	require.NoError(t, err)
	assert.Equal(t, drivermessage.ChannelDash, reach.Channel)
	assert.Empty(t, drivers.Notified, "asking how it would reach them sends nothing")

	channel, err := sender.Send(t.Context(), sent)
	require.NoError(t, err)
	assert.Equal(t, drivermessage.ChannelDash, channel)
	require.Len(t, drivers.Notified, 1)
	assert.Equal(t, drivermessage.DispatchEvent, drivers.Notified[0].EventType)
	assert.Equal(t, "/loads/"+sent.ShipmentID.String(), drivers.Notified[0].Link)
}

// A driver with no login is reached by nothing, and Send says so instead of
// pretending.
func TestSenderReachesNobodyWithoutALogin(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	drivers := &playsendtest.Drivers{Known: map[pulid.ID]playsendtest.Driver{
		workerID: {Name: "Theresa Aguilar"},
	}}
	sender := playsendtest.Messages(drivers)

	reach, err := sender.Reach(t.Context(), message(workerID))
	require.NoError(t, err)
	assert.Equal(t, drivermessage.ChannelNone, reach.Channel)

	channel, err := sender.Send(t.Context(), message(workerID))
	require.NoError(t, err)
	assert.Equal(t, drivermessage.ChannelNone, channel)
	assert.Empty(t, drivers.Notified)
}
