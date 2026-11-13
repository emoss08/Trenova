package playsend

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/drivermessage"
	"github.com/emoss08/trenova/internal/modules/agent/app/requestemail"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerplay"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// ToDriver is the part of a play that messages a driver: how the message
// would reach them, what the person must be allowed to do, and the send.
// Reached is false when nothing reaches the driver, and the playbook says
// why in its own words.
type ToDriver struct {
	Channel watchtowerplay.Channel
	Reached bool
	Needs   []watchtowerports.PlayGrant
	Send    func(ctx context.Context, sending *watchtowerports.PlaySending) error
}

// Driver plans a message from a person to a driver. The send fails, in
// words for the person, when the message reaches nobody: a message a person
// sent must not be dropped quietly the way a notification is.
func Driver(
	ctx context.Context,
	messages *drivermessage.Sender,
	message *drivermessage.Message,
	name string,
) (*ToDriver, error) {
	reached, err := messages.Reach(ctx, message)
	if err != nil {
		return nil, err
	}

	out := &ToDriver{
		Needs: []watchtowerports.PlayGrant{
			{Resource: permission.ResourceDriverMessage, Operation: permission.OpCreate},
		},
	}
	switch reached.Channel {
	case drivermessage.ChannelDash:
		out.Channel, out.Reached = watchtowerplay.ChannelDash, true
	case drivermessage.ChannelText:
		out.Channel, out.Reached = watchtowerplay.ChannelText, true
	case drivermessage.ChannelNone:
		return out, nil
	}

	out.Send = func(ctx context.Context, sending *watchtowerports.PlaySending) error {
		sent := *message
		sent.Text = sending.Message
		channel, sErr := messages.Send(ctx, &sent)
		if sErr != nil {
			return sErr
		}
		if channel == drivermessage.ChannelNone {
			return errortypes.NewBusinessError(
				"The message did not reach {0}: they no longer have a Dash login or a number that can be texted.",
				name,
			)
		}

		return nil
	}

	return out, nil
}

// ByEmail is the part of a play that emails an outside party: who the email
// is from, what the person must be allowed to do, and the send.
type ByEmail struct {
	From  string
	Needs []watchtowerports.PlayGrant
	Send  func(ctx context.Context, sending *watchtowerports.PlaySending) error
}

// Email plans a request email from a person. It renders the request once to
// prove it can go out, and fails, in words for the person, when the
// organization has no email profile or the recipient is on its do-not-email
// list.
func Email(
	ctx context.Context,
	requests *requestemail.Sender,
	request *requestemail.Request,
	name string,
) (*ByEmail, error) {
	composed, err := requests.Compose(ctx, request)
	if err != nil {
		return nil, err
	}
	from, err := requests.From(ctx, composed)
	if err != nil {
		return nil, err
	}

	out := &ByEmail{
		Needs: []watchtowerports.PlayGrant{
			{Resource: permission.ResourceCustomerCommunication, Operation: permission.OpCreate},
		},
	}
	if from != nil {
		if from.Refused {
			return nil, errortypes.NewBusinessError(
				"{0} is on the organization's do-not-email list, so the request would not be sent.",
				name,
			)
		}
		out.From = from.Address()
	}

	out.Send = func(ctx context.Context, sending *watchtowerports.PlaySending) error {
		sent := *request
		sent.Body = sending.Message
		sent.IdempotencyKey = sending.IdempotencyKey
		approved, cErr := requests.Compose(ctx, &sent)
		if cErr != nil {
			return cErr
		}

		return requests.Send(ctx, approved)
	}

	return out, nil
}
