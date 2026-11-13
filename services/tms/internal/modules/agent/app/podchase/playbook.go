package podchase

import (
	"context"

	"go.uber.org/fx"

	"github.com/emoss08/trenova/internal/modules/agent/app/drivermessage"
	"github.com/emoss08/trenova/internal/modules/agent/app/playsend"
	"github.com/emoss08/trenova/internal/modules/agent/app/requestemail"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerplay"
	"github.com/emoss08/trenova/internal/modules/agent/ports/podchaseports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type PlaybookParams struct {
	fx.In

	Facts         podchaseports.FactsReader
	Messages      *drivermessage.Sender
	Requests      *requestemail.Sender
	Organizations tenantports.OrganizationRepository
}

// Playbook is a person asking for a delivered shipment's signed proof of
// delivery from the Watchtower. Who is asked and how comes from the record
// (planAsk); the person reads the words, changes them if they like and
// sends them under their own name. No agent takes part.
type Playbook struct {
	facts         podchaseports.FactsReader
	messages      *drivermessage.Sender
	requests      *requestemail.Sender
	organizations tenantports.OrganizationRepository
}

var _ watchtowerports.Playbook = (*Playbook)(nil)

func NewPlaybook(
	p PlaybookParams, //nolint:gocritic // fx requires an fx.In parameter struct by value
) *Playbook {
	return &Playbook{
		facts:         p.Facts,
		messages:      p.Messages,
		requests:      p.Requests,
		organizations: p.Organizations,
	}
}

func (p *Playbook) Kind() watchtowerplay.Kind { return watchtowerplay.AskForPOD }

func (p *Playbook) Plan(
	ctx context.Context,
	req *watchtowerports.PlanRequest,
) (*watchtowerports.PlannedPlay, error) {
	item := req.Item
	if item.SubjectType != agent.SubjectShipment || item.SubjectID.IsNil() {
		return nil, errortypes.NewBusinessError("This item is not about a shipment")
	}

	facts, err := p.facts.ListPODChaseFacts(ctx, &podchaseports.ListFactsRequest{
		TenantInfo:      req.TenantInfo,
		ShipmentIDs:     []pulid.ID{item.SubjectID},
		DeliveredBefore: timeutils.NowUnix(),
	})
	if err != nil {
		return nil, err
	}
	if len(facts) == 0 {
		return nil, errortypes.NewBusinessError(
			"A proof of delivery is already on file for this shipment",
		)
	}
	fact := facts[0]

	org, err := p.organizations.GetByID(
		ctx,
		tenantports.GetOrganizationByIDRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, err
	}
	can := reach{
		email: p.requests.CanSend(ctx, req.TenantInfo),
		texts: p.messages.Texts(ctx, req.TenantInfo),
	}
	planned := planAsk(fact, can, org.Timezone)
	if planned == nil {
		return nil, unreachable(fact, can)
	}

	if planned.channel == watchtowerplay.ChannelEmail {
		return p.byEmail(ctx, req.TenantInfo, fact, planned)
	}

	return p.toDriver(ctx, req.TenantInfo, fact, planned)
}

func (p *Playbook) toDriver(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fact *podchaseports.Facts,
	planned *ask,
) (*watchtowerports.PlannedPlay, error) {
	driver, err := playsend.Driver(ctx, p.messages, &drivermessage.Message{
		TenantInfo: tenantInfo,
		WorkerID:   planned.workerID,
		Title:      planned.title,
		Text:       planned.message,
		ShipmentID: fact.ShipmentID,
	}, planned.name)
	if err != nil {
		return nil, err
	}
	if !driver.Reached {
		return nil, errortypes.NewBusinessError(
			"{0} cannot be reached in Dash or by text right now. Upload the proof of delivery here, or add a way to reach them.",
			planned.name,
		)
	}
	planned.channel = driver.Channel

	return &watchtowerports.PlannedPlay{
		Play: watchtowerplay.Play{
			Kind:         watchtowerplay.AskForPOD,
			Channel:      driver.Channel,
			Recipient:    planned.name,
			Message:      planned.message,
			MessageLimit: drivermessage.MaxChars,
		},
		Needs: driver.Needs,
		Asked: planned.asked(),
		Send:  driver.Send,
	}, nil
}

func (p *Playbook) byEmail(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	fact *podchaseports.Facts,
	planned *ask,
) (*watchtowerports.PlannedPlay, error) {
	email, err := playsend.Email(ctx, p.requests, &requestemail.Request{
		TenantInfo:         tenantInfo,
		Kind:               documenttemplate.KindAgentRequestMissingDocsEmail,
		To:                 []string{planned.email},
		Subject:            planned.subject,
		Body:               planned.message,
		RecipientName:      planned.name,
		ProNumber:          fact.ProNumber,
		RequestedDocuments: []string{requestedDocument},
	}, planned.name)
	if err != nil {
		return nil, err
	}

	return &watchtowerports.PlannedPlay{
		Play: watchtowerplay.Play{
			Kind:         watchtowerplay.AskForPOD,
			Channel:      watchtowerplay.ChannelEmail,
			Recipient:    planned.name,
			Sender:       email.From,
			Subject:      planned.subject,
			Message:      planned.message,
			MessageLimit: watchtowerplay.MaxMessageLength,
		},
		Needs: email.Needs,
		Asked: planned.asked(),
		Send:  email.Send,
	}, nil
}
