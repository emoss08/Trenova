package podchase

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/modules/agent/app/playsend/playsendtest"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerplay"
	"github.com/emoss08/trenova/internal/modules/agent/ports/podchaseports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/domain/notificationtemplate"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type facts struct {
	rows []*podchaseports.Facts
}

func (f *facts) ListPODChaseFacts(
	context.Context,
	*podchaseports.ListFactsRequest,
) ([]*podchaseports.Facts, error) {
	return f.rows, nil
}

type organizations struct {
	tenantports.OrganizationRepository
}

func (organizations) GetByID(
	context.Context,
	tenantports.GetOrganizationByIDRequest,
) (*tenant.Organization, error) {
	return &tenant.Organization{Timezone: "UTC"}, nil
}

type rig struct {
	playbook *Playbook
	drivers  *playsendtest.Drivers
	texts    *playsendtest.Texts
	mail     *playsendtest.Mail
	tenant   pagination.TenantInfo
}

func newRig(fact *podchaseports.Facts, plan *playsendtest.Plans) *rig {
	r := &rig{
		drivers: &playsendtest.Drivers{Known: map[pulid.ID]playsendtest.Driver{}},
		texts:   &playsendtest.Texts{},
		mail:    playsendtest.NewMail(),
		tenant:  pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
	}
	rows := []*podchaseports.Facts{}
	if fact != nil {
		rows = append(rows, fact)
	}
	r.playbook = NewPlaybook(PlaybookParams{
		Facts:         &facts{rows: rows},
		Messages:      playsendtest.Messages(r.drivers, plan, r.texts),
		Requests:      r.mail.Requests(),
		Organizations: organizations{},
	})

	return r
}

func (r *rig) plan(t *testing.T, shipmentID pulid.ID) (*watchtowerports.PlannedPlay, error) {
	t.Helper()

	return r.playbook.Plan(t.Context(), &watchtowerports.PlanRequest{
		TenantInfo: r.tenant,
		Item: &watchtower.Item{
			SourceKind:  watchtower.SourcePodMissing,
			SubjectType: agent.SubjectShipment,
			SubjectID:   shipmentID,
		},
	})
}

func deliveredFact() *podchaseports.Facts {
	return &podchaseports.Facts{
		ShipmentID:   pulid.MustNew("shp_"),
		ProNumber:    "S26080375057552",
		CustomerName: "Simpson & Jones Industries",
		DeliveredAt:  time.Date(2026, 8, 15, 18, 0, 0, 0, time.UTC).Unix(),
	}
}

// A driver on Dash is shown as the recipient, the play needs the right to
// message drivers, and sending delivers the words the person approved, not
// the standard ones.
func TestPlaybookSendsADashMessageWithThePersonsWords(t *testing.T) {
	t.Parallel()

	fact := deliveredFact()
	workerID := pulid.MustNew("wrk_")
	fact.Driver = &podchaseports.Driver{WorkerID: workerID, Name: "Laura Wright", OnDash: true}
	r := newRig(fact, &playsendtest.Plans{Texts: true})
	r.drivers.Known[workerID] = playsendtest.Driver{Name: "Laura Wright", OnDash: true}

	planned, err := r.plan(t, fact.ShipmentID)
	require.NoError(t, err)

	assert.Equal(t, watchtowerplay.AskForPOD, planned.Kind)
	assert.Equal(t, watchtowerplay.ChannelDash, planned.Channel)
	assert.Equal(t, "Laura Wright", planned.Recipient)
	assert.Contains(t, planned.Message, "PRO S26080375057552")
	assert.Equal(t, []watchtowerports.PlayGrant{
		{Resource: permission.ResourceDriverMessage, Operation: permission.OpCreate},
	}, planned.Needs)
	assert.Equal(t, "Asked Laura Wright for the signed proof of delivery in Dash", planned.Asked)
	assert.Empty(t, r.drivers.Notified, "planning sends nothing")

	require.NoError(t, planned.Send(t.Context(), &watchtowerports.PlaySending{
		Message: "Laura, third ask: we need the signed POD today.",
	}))
	require.Len(t, r.drivers.Notified, 1)
	assert.Equal(t, workerID, r.drivers.Notified[0].WorkerID)
	assert.Contains(t,
		fmtContext(r.drivers.Notified[0].Context), "third ask: we need the signed POD today")
}

// A driver reached only by text is texted when the plan includes texts, and
// the number never appears in what the person is shown.
func TestPlaybookTextsADriverWithOnlyANumber(t *testing.T) {
	t.Parallel()

	fact := deliveredFact()
	workerID := pulid.MustNew("wrk_")
	fact.Driver = &podchaseports.Driver{WorkerID: workerID, Name: "Theresa Aguilar", HasPhone: true}
	r := newRig(fact, &playsendtest.Plans{Texts: true})
	r.mail.NoProfile = true
	r.playbook.requests = r.mail.Requests()
	r.drivers.Known[workerID] = playsendtest.Driver{Name: "Theresa Aguilar", Phone: "+15555550123"}

	planned, err := r.plan(t, fact.ShipmentID)
	require.NoError(t, err)

	assert.Equal(t, watchtowerplay.ChannelText, planned.Channel)
	assert.NotContains(t, planned.Message, "5555550123")
	assert.NotContains(t, planned.Recipient, "5555550123")
	assert.NotContains(t, planned.Asked, "5555550123")

	require.NoError(t, planned.Send(t.Context(), &watchtowerports.PlaySending{Message: "Please call in."}))
	assert.Len(t, r.texts.Sent, 1)
	assert.Empty(t, r.drivers.Notified)
}

// A carrier is asked by email in the missing documents template, from the
// organization's own profile, and the person's words are the body.
func TestPlaybookEmailsTheCarrierInTheOrganizationsTemplate(t *testing.T) {
	t.Parallel()

	fact := deliveredFact()
	fact.Carrier = &podchaseports.Carrier{Name: "Acme Freight", Email: "ops@acme.test"}
	r := newRig(fact, &playsendtest.Plans{})

	planned, err := r.plan(t, fact.ShipmentID)
	require.NoError(t, err)

	assert.Equal(t, watchtowerplay.ChannelEmail, planned.Channel)
	assert.Equal(t, "Acme Freight", planned.Recipient)
	assert.NotContains(t, planned.Recipient, "@")
	assert.NotEmpty(t, planned.Sender)
	assert.Equal(t, "Signed proof of delivery for PRO S26080375057552", planned.Subject)
	assert.Equal(t, []watchtowerports.PlayGrant{
		{Resource: permission.ResourceCustomerCommunication, Operation: permission.OpCreate},
	}, planned.Needs)
	assert.Nil(t, r.mail.Mailer.Sent, "planning sends nothing")

	require.NoError(t, planned.Send(t.Context(), &watchtowerports.PlaySending{
		Message:        "We need the signed POD by Friday.",
		IdempotencyKey: "key-1",
	}))
	require.NotNil(t, r.mail.Mailer.Sent)
	assert.Equal(t, []string{"ops@acme.test"}, r.mail.Mailer.Sent.To)
	assert.Equal(t, "key-1", r.mail.Mailer.Sent.IdempotencyKey)
	assert.Equal(t, documenttemplate.KindAgentRequestMissingDocsEmail, r.mail.Renderer.Req.Kind)
}

// The play fails with a reason a person can act on, and sends nothing, when
// the paperwork is already on file, nobody can be reached, or the recipient
// is on the do-not-email list.
func TestPlaybookSaysWhyItCannotBeSent(t *testing.T) {
	t.Parallel()

	onFile := newRig(nil, &playsendtest.Plans{})
	_, err := onFile.plan(t, pulid.MustNew("shp_"))
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	assert.Contains(t, err.Error(), "already on file")

	fact := deliveredFact()
	fact.Driver = &podchaseports.Driver{WorkerID: pulid.MustNew("wrk_"), Name: "A Driver"}
	unreachable := newRig(fact, &playsendtest.Plans{Texts: true})
	_, err = unreachable.plan(t, fact.ShipmentID)
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))

	suppressed := deliveredFact()
	suppressed.Carrier = &podchaseports.Carrier{Name: "Acme Freight", Email: "ops@acme.test"}
	refused := newRig(suppressed, &playsendtest.Plans{})
	refused.mail.Refused = true
	refused.playbook.requests = refused.mail.Requests()
	_, err = refused.plan(t, suppressed.ShipmentID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "do-not-email")
	assert.Nil(t, refused.mail.Mailer.Sent)
}

func fmtContext(value any) string {
	if words, ok := value.(notificationtemplate.DriverNotificationContext); ok {
		return words.AlertTitle + " " + words.AlertMessage
	}

	return ""
}
