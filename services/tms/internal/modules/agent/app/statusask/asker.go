package statusask

import (
	"context"

	"go.uber.org/fx"

	"github.com/emoss08/trenova/internal/modules/agent/app/drivermessage"
	"github.com/emoss08/trenova/internal/modules/agent/app/playsend"
	"github.com/emoss08/trenova/internal/modules/agent/app/requestemail"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerplay"
	"github.com/emoss08/trenova/internal/modules/agent/ports/haulerports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type Params struct {
	fx.In

	Haulers  haulerports.Reader
	Messages *drivermessage.Sender
	Requests *requestemail.Sender
}

// Playbook is a person asking whoever has a load where it is: the first
// thing a dispatcher does about a missed pickup or delivery, a late
// projection or a check call that never came. Who is asked follows from the
// record: the driver on the load, in Dash, by text or by email, or the
// carrier that has it, by email. What is asked follows from the kind and
// the stop. The person reads the words, changes them if they like and
// sends them under their own name; no agent takes part. Re-covering the
// load, moving the appointment and telling the customer are judgment, and
// stay with the person or an agent they name.
type Playbook struct {
	haulers  haulerports.Reader
	messages *drivermessage.Sender
	requests *requestemail.Sender
}

var _ watchtowerports.Playbook = (*Playbook)(nil)

func New(p Params) *Playbook { //nolint:gocritic // fx requires an fx.In parameter struct by value
	return &Playbook{haulers: p.Haulers, messages: p.Messages, requests: p.Requests}
}

func (p *Playbook) Kind() watchtowerplay.Kind { return watchtowerplay.AskForStatus }

func (p *Playbook) Plan(
	ctx context.Context,
	req *watchtowerports.PlanRequest,
) (*watchtowerports.PlannedPlay, error) {
	item := req.Item
	if item.SubjectID.IsNil() {
		return nil, errortypes.NewBusinessError("This item is not about a load")
	}

	hauler, err := p.haulers.Hauler(ctx, haulerRequest(req))
	if err != nil {
		return nil, err
	}
	if hauler == nil || (hauler.Driver == nil && hauler.Carrier == nil) {
		return nil, errortypes.NewBusinessError(
			"Nobody is assigned to this load yet, so there is nobody to ask. Assign a driver or a carrier first.",
		)
	}

	if driver := hauler.Driver; driver != nil {
		planned, dErr := p.toDriver(ctx, req, hauler)
		if dErr != nil || planned != nil {
			return planned, dErr
		}
	}

	return p.byEmail(ctx, req, hauler)
}

// toDriver asks the driver in Dash or by text. Nil when neither reaches
// them, so the plan falls to email.
func (p *Playbook) toDriver(
	ctx context.Context,
	req *watchtowerports.PlanRequest,
	hauler *haulerports.Hauler,
) (*watchtowerports.PlannedPlay, error) {
	driver := hauler.Driver
	title, text := driverWords(req.Item, hauler)
	sent, err := playsend.Driver(ctx, p.messages, &drivermessage.Message{
		TenantInfo: req.TenantInfo,
		WorkerID:   driver.WorkerID,
		Title:      title,
		Text:       text,
		Priority:   notification.PriorityHigh,
		ShipmentID: hauler.ShipmentID,
	}, driver.Name)
	if err != nil {
		return nil, err
	}
	if !sent.Reached {
		return nil, nil //nolint:nilnil // not reached this way: the plan tries email
	}

	return &watchtowerports.PlannedPlay{
		Play: watchtowerplay.Play{
			Kind:         watchtowerplay.AskForStatus,
			Channel:      sent.Channel,
			Recipient:    driver.Name,
			Message:      text,
			MessageLimit: drivermessage.MaxChars,
		},
		Needs: sent.Needs,
		Asked: asked(driver.Name, hauler, sent.Channel),
		Send:  sent.Send,
	}, nil
}

// byEmail asks the driver by email when that is all that reaches them,
// otherwise the carrier. It fails with who is missing what when neither has
// an address the organization can send to.
func (p *Playbook) byEmail(
	ctx context.Context,
	req *watchtowerports.PlanRequest,
	hauler *haulerports.Hauler,
) (*watchtowerports.PlannedPlay, error) {
	name, address := emailRecipient(hauler)
	if address == "" {
		return nil, unreachable(hauler)
	}
	if !p.requests.CanSend(ctx, req.TenantInfo) {
		return nil, errortypes.NewBusinessError(
			"No email profile is set up to send from, so {0} cannot be asked by email. Call them, or ask an administrator to add one in the email settings.",
			name,
		)
	}

	subject, body := emailWords(req.Item, hauler)
	email, err := playsend.Email(ctx, p.requests, &requestemail.Request{
		TenantInfo:    req.TenantInfo,
		Kind:          documenttemplate.KindAgentStatusRequestEmail,
		To:            []string{address},
		Subject:       subject,
		Body:          body,
		RecipientName: name,
		ProNumber:     hauler.ProNumber,
	}, name)
	if err != nil {
		return nil, err
	}

	return &watchtowerports.PlannedPlay{
		Play: watchtowerplay.Play{
			Kind:         watchtowerplay.AskForStatus,
			Channel:      watchtowerplay.ChannelEmail,
			Recipient:    name,
			Sender:       email.From,
			Subject:      subject,
			Message:      body,
			MessageLimit: watchtowerplay.MaxMessageLength,
		},
		Needs: email.Needs,
		Asked: asked(name, hauler, watchtowerplay.ChannelEmail),
		Send:  email.Send,
	}, nil
}

// emailRecipient is who an email goes to: the driver when they have an
// address, otherwise the carrier.
func emailRecipient(hauler *haulerports.Hauler) (name, address string) {
	if driver := hauler.Driver; driver != nil && driver.Email != "" {
		return driver.Name, driver.Email
	}
	if carrier := hauler.Carrier; carrier != nil && carrier.Email != "" {
		return carrier.Name, carrier.Email
	}

	return "", ""
}

// haulerRequest names the load by what the item's kind keys it on: a quiet
// truck by its move, a missed stop by the stop, a late projection by the
// shipment.
func haulerRequest(req *watchtowerports.PlanRequest) *haulerports.HaulerRequest {
	item := req.Item
	out := &haulerports.HaulerRequest{TenantInfo: req.TenantInfo}
	switch item.SourceKind { //nolint:exhaustive // only the kinds this playbook works
	case watchtower.SourceNoCheckCall:
		out.MoveID = item.SubjectID
	case watchtower.SourceMissedPickup, watchtower.SourceMissedDelivery:
		out.ShipmentID = item.SubjectID
		if stopID, err := pulid.Parse(item.SourceID); err == nil {
			out.StopID = stopID
		}
	default:
		out.ShipmentID = item.SubjectID
	}

	return out
}

// unreachable says why nobody on the load can be asked from here. Each
// message is one literal so it reaches the translation catalog whole.
//
//nolint:lll // a message is one literal
func unreachable(hauler *haulerports.Hauler) *errortypes.BusinessError {
	driver, carrier := hauler.Driver, hauler.Carrier
	switch {
	case driver != nil && carrier != nil:
		return errortypes.NewBusinessError("{0} has no Dash login, phone number that can be texted or email on file, and {1} has no email on file, so neither can be asked from here. Call them, or add a way to reach them.", driver.Name, carrier.Name)
	case driver != nil:
		return errortypes.NewBusinessError("{0} has no Dash login, phone number that can be texted or email on file, so they cannot be asked from here. Call them, or add a way to reach them.", driver.Name)
	default:
		return errortypes.NewBusinessError("{0} has no email on file, so they cannot be asked from here. Call them, or add their email.", carrier.Name)
	}
}
