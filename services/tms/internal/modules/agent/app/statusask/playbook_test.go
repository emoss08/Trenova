package statusask

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/modules/agent/app/playsend/playsendtest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerpanel"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerplay"
	"github.com/emoss08/trenova/internal/modules/agent/ports/haulerports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/podchaseports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type haulers struct {
	hauler *haulerports.Hauler
	asked  *haulerports.HaulerRequest
}

func (h *haulers) Hauler(
	_ context.Context,
	req *haulerports.HaulerRequest,
) (*haulerports.Hauler, error) {
	h.asked = req

	return h.hauler, nil
}

func (h *haulers) TenderMove(context.Context, pagination.TenantInfo, pulid.ID) (pulid.ID, error) {
	return pulid.Nil, nil
}

type rig struct {
	playbook *Playbook
	haulers  *haulers
	drivers  *playsendtest.Drivers
	texts    *playsendtest.Texts
	mail     *playsendtest.Mail
}

func newRig(hauler *haulerports.Hauler, plan *playsendtest.Plans, mail *playsendtest.Mail) *rig {
	r := &rig{
		haulers: &haulers{hauler: hauler},
		drivers: &playsendtest.Drivers{Known: map[pulid.ID]playsendtest.Driver{}},
		texts:   &playsendtest.Texts{},
		mail:    mail,
	}
	r.playbook = New(Params{
		Haulers:  r.haulers,
		Messages: playsendtest.Messages(r.drivers, plan, r.texts),
		Requests: mail.Requests(),
	})

	return r
}

func missedDelivery(shipmentID, stopID pulid.ID) *watchtower.Item {
	return &watchtower.Item{
		SourceKind:  watchtower.SourceMissedDelivery,
		SourceID:    stopID.String(),
		SubjectType: agent.SubjectShipment,
		SubjectID:   shipmentID,
		Panel: &watchtowerpanel.Panel{
			Kind: watchtowerpanel.KindStop,
			Stop: &watchtowerpanel.StopPanel{Place: "Hartford DC"},
		},
	}
}

func (r *rig) plan(t *testing.T, item *watchtower.Item) (*watchtowerports.PlannedPlay, error) {
	t.Helper()

	return r.playbook.Plan(t.Context(), &watchtowerports.PlanRequest{
		TenantInfo: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		Item:       item,
	})
}

// A driver on Dash is asked there, about the stop the item names, and the
// load is found by the stop the item is keyed on.
func TestPlaybookAsksTheDriverOnDashAboutTheMissedStop(t *testing.T) {
	t.Parallel()

	shipmentID, stopID, workerID := pulid.MustNew("shp_"), pulid.MustNew("stp_"), pulid.MustNew("wrk_")
	r := newRig(&haulerports.Hauler{
		ShipmentID: shipmentID,
		ProNumber:  "S100",
		Driver:     &podchaseports.Driver{WorkerID: workerID, Name: "Laura Wright", OnDash: true},
	}, &playsendtest.Plans{Texts: true}, playsendtest.NewMail())
	r.drivers.Known[workerID] = playsendtest.Driver{Name: "Laura Wright", OnDash: true}

	planned, err := r.plan(t, missedDelivery(shipmentID, stopID))
	require.NoError(t, err)

	assert.Equal(t, stopID, r.haulers.asked.StopID)
	assert.Equal(t, watchtowerplay.AskForStatus, planned.Kind)
	assert.Equal(t, watchtowerplay.ChannelDash, planned.Channel)
	assert.Equal(t, "Laura Wright", planned.Recipient)
	assert.Contains(t, planned.Message, "Hi Laura,")
	assert.Contains(t, planned.Message, "delivery window at Hartford DC for PRO S100")
	assert.Equal(t, "Asked Laura Wright where PRO S100 is in Dash", planned.Asked)
	assert.Equal(t, []watchtowerports.PlayGrant{
		{Resource: permission.ResourceDriverMessage, Operation: permission.OpCreate},
	}, planned.Needs)

	require.NoError(t, planned.Send(t.Context(), &watchtowerports.PlaySending{Message: "Where are you?"}))
	require.Len(t, r.drivers.Notified, 1)
	words, ok := r.drivers.Notified[0].Context.(notificationtemplate.DriverNotificationContext)
	require.True(t, ok)
	assert.Equal(t, "Where are you?", words.AlertMessage)
}

// A load a carrier hauls has no driver to message, so the carrier is asked
// by email in the status request template, with no greeting in the body:
// the template greets and signs.
func TestPlaybookEmailsTheCarrierThatHasTheLoad(t *testing.T) {
	t.Parallel()

	moveID := pulid.MustNew("smv_")
	mail := playsendtest.NewMail()
	r := newRig(&haulerports.Hauler{
		ShipmentID: pulid.MustNew("shp_"),
		ProNumber:  "S200",
		Carrier:    &podchaseports.Carrier{Name: "Northline Carriers", Email: "dispatch@northline.test"},
	}, &playsendtest.Plans{}, mail)

	planned, err := r.plan(t, &watchtower.Item{
		SourceKind:  watchtower.SourceNoCheckCall,
		SourceID:    moveID.String(),
		SubjectType: agent.SubjectShipmentMove,
		SubjectID:   moveID,
	})
	require.NoError(t, err)

	assert.Equal(t, moveID, r.haulers.asked.MoveID, "a quiet truck is found by its move")
	assert.Equal(t, watchtowerplay.ChannelEmail, planned.Channel)
	assert.Equal(t, "Northline Carriers", planned.Recipient)
	assert.Equal(t, "Status needed: PRO S200", planned.Subject)
	assert.Contains(t, planned.Message, "We have not had a check call on PRO S200")
	assert.NotContains(t, planned.Message, "Hi ")
	assert.NotContains(t, planned.Message, "dispatch@northline.test")
	assert.Equal(t, "Asked Northline Carriers where PRO S200 is by email", planned.Asked)
	assert.Equal(t, []watchtowerports.PlayGrant{
		{Resource: permission.ResourceCustomerCommunication, Operation: permission.OpCreate},
	}, planned.Needs)

	require.NoError(t, planned.Send(t.Context(), &watchtowerports.PlaySending{
		Message: "Please send a check call.", IdempotencyKey: "key-9",
	}))
	require.NotNil(t, mail.Mailer.Sent)
	assert.Equal(t, []string{"dispatch@northline.test"}, mail.Mailer.Sent.To)
	assert.Equal(t, "key-9", mail.Mailer.Sent.IdempotencyKey)
	assert.Equal(t, documenttemplate.KindAgentStatusRequestEmail, mail.Renderer.Req.Kind)
}

// A driver with no login is texted when the plan includes texts, emailed
// when it does not and an address is on file, and otherwise the person is
// told who is missing what.
func TestPlaybookFallsFromDashToTextToEmail(t *testing.T) {
	t.Parallel()

	workerID := pulid.MustNew("wrk_")
	hauler := func() *haulerports.Hauler {
		return &haulerports.Hauler{
			ShipmentID: pulid.MustNew("shp_"),
			ProNumber:  "S300",
			Driver: &podchaseports.Driver{
				WorkerID: workerID, Name: "Theresa Aguilar", HasPhone: true, Email: "theresa@drivers.test",
			},
		}
	}
	item := func(h *haulerports.Hauler) *watchtower.Item {
		return &watchtower.Item{
			SourceKind: watchtower.SourceEtaAtRisk, SubjectType: agent.SubjectShipment, SubjectID: h.ShipmentID,
		}
	}
	known := playsendtest.Driver{Name: "Theresa Aguilar", Phone: "+15555550123"}

	withTexts := newRig(hauler(), &playsendtest.Plans{Texts: true}, playsendtest.NewMail())
	withTexts.drivers.Known[workerID] = known
	planned, err := withTexts.plan(t, item(withTexts.haulers.hauler))
	require.NoError(t, err)
	assert.Equal(t, watchtowerplay.ChannelText, planned.Channel)
	assert.Equal(t, "Asked Theresa Aguilar where PRO S300 is by text", planned.Asked)
	assert.NotContains(t, planned.Message, "5555550123")

	withoutTexts := newRig(hauler(), &playsendtest.Plans{}, playsendtest.NewMail())
	withoutTexts.drivers.Known[workerID] = known
	planned, err = withoutTexts.plan(t, item(withoutTexts.haulers.hauler))
	require.NoError(t, err)
	assert.Equal(t, watchtowerplay.ChannelEmail, planned.Channel)
	assert.Equal(t, "Theresa Aguilar", planned.Recipient)

	bare := hauler()
	bare.Driver.Email, bare.Driver.HasPhone = "", false
	nothing := newRig(bare, &playsendtest.Plans{Texts: true}, playsendtest.NewMail())
	nothing.drivers.Known[workerID] = playsendtest.Driver{Name: "Theresa Aguilar"}
	_, err = nothing.plan(t, item(bare))
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "Call them")
}

// Nobody on the load, or no email profile to send from, is said plainly.
func TestPlaybookSaysWhyNobodyCanBeAsked(t *testing.T) {
	t.Parallel()

	shipmentID := pulid.MustNew("shp_")
	item := &watchtower.Item{
		SourceKind: watchtower.SourceEtaAtRisk, SubjectType: agent.SubjectShipment, SubjectID: shipmentID,
	}

	unassigned := newRig(&haulerports.Hauler{ShipmentID: shipmentID}, &playsendtest.Plans{}, playsendtest.NewMail())
	_, err := unassigned.plan(t, item)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Nobody is assigned to this load yet")

	mail := playsendtest.NewMail()
	mail.NoProfile = true
	noProfile := newRig(&haulerports.Hauler{
		ShipmentID: shipmentID,
		Carrier:    &podchaseports.Carrier{Name: "Northline Carriers", Email: "dispatch@northline.test"},
	}, &playsendtest.Plans{}, mail)
	_, err = noProfile.plan(t, item)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "No email profile is set up to send from")
	assert.Nil(t, mail.Mailer.Sent)
}

// The words never carry a record id: a load with no PRO is "your load" to
// its driver and "this load" to a carrier.
func TestWordsNameALoadWithNoPROWithoutItsID(t *testing.T) {
	t.Parallel()

	hauler := &haulerports.Hauler{
		ShipmentID: pulid.MustNew("shp_"),
		Driver:     &podchaseports.Driver{Name: "Laura Wright"},
	}
	item := &watchtower.Item{SourceKind: watchtower.SourceMissedPickup}

	title, message := driverWords(item, hauler)
	assert.Contains(t, message, "your load")
	assert.NotContains(t, title+message, hauler.ShipmentID.String())

	subject, body := emailWords(item, hauler)
	assert.Contains(t, body, "this load")
	assert.NotContains(t, subject+body, hauler.ShipmentID.String())
	assert.Equal(t, "Asked Laura Wright where the load is by email",
		asked("Laura Wright", hauler, watchtowerplay.ChannelEmail))
}
