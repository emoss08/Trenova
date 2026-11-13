package drivermessage_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/modules/agent/app/drivermessage"
	"github.com/emoss08/trenova/internal/modules/agent/app/playsend/playsendtest"
	"github.com/emoss08/trenova/internal/modules/identity/jobs/smsjobs/smscontract"
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
		workerID: {Name: "Laura Wright", OnDash: true, Phone: "+15555550100"},
	}}
	texts := &playsendtest.Texts{}
	sender := playsendtest.Messages(drivers, &playsendtest.Plans{Texts: true}, texts)
	sent := message(workerID)

	reach, err := sender.Reach(t.Context(), sent)
	require.NoError(t, err)
	assert.Equal(t, drivermessage.ChannelDash, reach.Channel)
	assert.Empty(t, reach.Text())
	assert.Empty(t, drivers.Notified, "asking how it would reach them sends nothing")

	channel, err := sender.Send(t.Context(), sent)
	require.NoError(t, err)
	assert.Equal(t, drivermessage.ChannelDash, channel)
	require.Len(t, drivers.Notified, 1)
	assert.Equal(t, drivermessage.DispatchEvent, drivers.Notified[0].EventType)
	assert.Equal(t, "/loads/"+sent.ShipmentID.String(), drivers.Notified[0].Link)
	assert.Empty(t, texts.Sent, "a driver on Dash is not also texted")
}

// A driver with no login and a number on file is texted, when the plan
// includes text messages; the number goes to the text and nowhere else.
func TestSenderTextsADriverWithNoLoginWhenThePlanIncludesTexts(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	drivers := &playsendtest.Drivers{Known: map[pulid.ID]playsendtest.Driver{
		workerID: {Name: "Theresa Aguilar", Phone: "+15555550123"},
	}}
	texts := &playsendtest.Texts{}
	sender := playsendtest.Messages(drivers, &playsendtest.Plans{Texts: true}, texts)

	reach, err := sender.Reach(t.Context(), message(workerID))
	require.NoError(t, err)
	assert.Equal(t, drivermessage.ChannelText, reach.Channel)
	assert.Equal(t, "Call dispatch: Call when you are parked.", reach.Text())

	channel, err := sender.Send(t.Context(), message(workerID))
	require.NoError(t, err)
	assert.Equal(t, drivermessage.ChannelText, channel)
	require.Len(t, texts.Sent, 1)
	payload, ok := texts.Sent[0].(*smscontract.SendSMSPayload)
	require.True(t, ok)
	assert.Equal(t, "+15555550123", payload.PhoneNumber)
	assert.Equal(t, "Call dispatch: Call when you are parked.", payload.Message)
	assert.Empty(t, drivers.Notified)
}

// Without text messages in the plan, or with no number, nothing reaches a
// driver with no login, and Send says so instead of pretending.
func TestSenderReachesNobodyWithoutALoginOrATextableNumber(t *testing.T) {
	t.Parallel()

	withNumber, bare := pulid.MustNew("wrk_"), pulid.MustNew("wrk_")
	drivers := &playsendtest.Drivers{Known: map[pulid.ID]playsendtest.Driver{
		withNumber: {Name: "Theresa Aguilar", Phone: "+15555550123"},
		bare:       {Name: "A Driver"},
	}}
	texts := &playsendtest.Texts{}

	noTexts := playsendtest.Messages(drivers, &playsendtest.Plans{}, texts)
	reach, err := noTexts.Reach(t.Context(), message(withNumber))
	require.NoError(t, err)
	assert.Equal(t, drivermessage.ChannelNone, reach.Channel)
	channel, err := noTexts.Send(t.Context(), message(withNumber))
	require.NoError(t, err)
	assert.Equal(t, drivermessage.ChannelNone, channel)

	withTexts := playsendtest.Messages(drivers, &playsendtest.Plans{Texts: true}, texts)
	channel, err = withTexts.Send(t.Context(), message(bare))
	require.NoError(t, err)
	assert.Equal(t, drivermessage.ChannelNone, channel)

	assert.Empty(t, texts.Sent)
	assert.Empty(t, drivers.Notified)
}
