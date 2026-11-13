package watchtowerservice

import (
	"context"
	"strings"

	"go.uber.org/fx"
	"go.uber.org/zap"

	"github.com/emoss08/trenova/internal/modules/agent/app/requestwording"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerplay"
	"github.com/emoss08/trenova/internal/modules/agent/ports/caseownerports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type PlaysParams struct {
	fx.In

	Logger      *zap.Logger
	Exceptions  *Exceptions
	Permissions permissionports.PermissionEngine
	Owners      caseownerports.Service
	Playbooks   []watchtowerports.Playbook `group:"watchtower_playbooks"`
	Wording     *requestwording.Service
	Users       tenantports.UserRepository
}

// Plays is a person's first move on a Watchtower item. The kind of item
// says what the move is (watchtower.SourceKind.Play) and its playbook plans
// it from the record: who it goes to, how, and the words. The person reads
// that, changes the words if they like, and sends it under their own name
// with their own permissions. Nothing here asks a model what to do or
// depends on an agent being set up, enabled or out of shadow; the only
// model call is the rewording a person asks for, which changes words and
// nothing else.
//
// It is a type of its own rather than part of Service because playbooks
// reach the mailer, the tender service and the driver notification path,
// and the read face is held by tools those must not close a loop with.
type Plays struct {
	l           *zap.Logger
	exceptions  *Exceptions
	permissions permissionports.PermissionEngine
	access      kindAccess
	playbooks   map[watchtowerplay.Kind]watchtowerports.Playbook
	wording     *requestwording.Service
	users       tenantports.UserRepository
}

var _ watchtowerports.PlayService = (*Plays)(nil)

func NewPlays(p PlaysParams) *Plays { //nolint:gocritic // fx params are passed by value
	playbooks := make(map[watchtowerplay.Kind]watchtowerports.Playbook, len(p.Playbooks))
	for _, playbook := range p.Playbooks {
		if playbook != nil {
			playbooks[playbook.Kind()] = playbook
		}
	}

	return &Plays{
		l:           p.Logger.Named("service.watchtower.plays"),
		exceptions:  p.Exceptions,
		permissions: p.Permissions,
		access:      kindAccess{permissions: p.Permissions, owners: p.Owners},
		playbooks:   playbooks,
		wording:     p.Wording,
		users:       p.Users,
	}
}

func AsPlayService(p *Plays) watchtowerports.PlayService { return p }

// Preview plans the play and returns it as the person would send it. It
// fails with the reason, in words for the person, when the play cannot be
// made or is not theirs to make.
func (s *Plays) Preview(
	ctx context.Context,
	req *watchtowerports.PlayTarget,
	actor *permissionports.RequestActor,
) (*watchtowerports.PlayPreview, error) {
	item, planned, err := s.plan(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	return &watchtowerports.PlayPreview{
		ItemID:       item.ID,
		Kind:         planned.Kind,
		Channel:      planned.Channel,
		Recipient:    planned.Recipient,
		Sender:       planned.Sender,
		Subject:      planned.Subject,
		Message:      planned.Message,
		MessageLimit: planned.MessageLimit,
		Summary:      planned.Summary,
		CanReword:    planned.Message != "" && s.wording.Available(ctx, req.TenantInfo),
	}, nil
}

// Reword has Nova rewrite the words in the box the way the person asked.
// Who the play goes to is not Nova's to change and is not sent to it.
func (s *Plays) Reword(
	ctx context.Context,
	req *watchtowerports.RewordPlayRequest,
	actor *permissionports.RequestActor,
) (string, error) {
	_, planned, err := s.plan(ctx, &req.PlayTarget, actor)
	if err != nil {
		return "", err
	}
	if planned.Message == "" {
		return "", errortypes.NewBusinessError("This sends no message, so there is nothing to reword")
	}

	message := strings.TrimSpace(req.Message)
	if message == "" {
		message = planned.Message
	}

	return s.wording.Word(ctx, &requestwording.Request{
		TenantInfo: req.TenantInfo,
		UserID:     actor.UserID,
		Message:    message,
		Guidance:   req.Guidance,
		MaxBytes:   planned.MessageLimit,
	})
}

// Send plans the play again from the record as it is now, sends it with the
// words the person approved, and notes on the item who asked what of whom.
// The plan is never taken from the client: only the words are theirs.
func (s *Plays) Send(
	ctx context.Context,
	req *watchtowerports.SendPlayRequest,
	actor *permissionports.RequestActor,
) (*watchtower.Item, error) {
	item, planned, err := s.plan(ctx, &req.PlayTarget, actor)
	if err != nil {
		return nil, err
	}

	message, err := approvedMessage(planned, req.Message)
	if err != nil {
		return nil, err
	}
	if err = planned.Send(ctx, &watchtowerports.PlaySending{
		Message:        message,
		IdempotencyKey: req.IdempotencyKey,
	}); err != nil {
		return nil, err
	}

	saved, err := s.record(ctx, req, actor, planned.Asked)
	if err != nil {
		// What was asked has gone out; failing here would have the person
		// send it twice. The item is returned as it was read.
		s.l.Error("a play was sent but could not be noted on its item",
			zap.String("item", item.ID.String()), zap.Error(err))

		return item, nil
	}

	return saved, nil
}

// approvedMessage is the words to send: the person's, when they left any,
// held to the channel's bound.
func approvedMessage(planned *watchtowerports.PlannedPlay, typed string) (string, error) {
	if planned.Message == "" {
		return "", nil
	}
	message := strings.TrimSpace(typed)
	if message == "" {
		message = planned.Message
	}
	if limit := planned.MessageLimit; limit > 0 && len(message) > limit {
		return "", errortypes.NewValidationError(
			"message",
			errortypes.ErrInvalid,
			"The message is too long for how it is sent. Shorten it and send again.",
		)
	}

	return message, nil
}

// plan reads the item for the person, refuses one that is not theirs to
// work, and has the kind's playbook plan the play. It then holds the plan
// to what the person may do.
func (s *Plays) plan(
	ctx context.Context,
	req *watchtowerports.PlayTarget,
	actor *permissionports.RequestActor,
) (*watchtower.Item, *watchtowerports.PlannedPlay, error) {
	item, err := s.exceptions.GetException(ctx, watchtowerports.GetWatchtowerItemRequest{
		ID:         req.ItemID,
		TenantInfo: req.TenantInfo,
	}, actor)
	if err != nil {
		return nil, nil, err
	}
	if err = s.access.assertMayWork(ctx, actor, item); err != nil {
		return nil, nil, err
	}
	if item.IsResolved() {
		return nil, nil, errortypes.NewBusinessError("This is already resolved")
	}

	playbook, ok := s.playbooks[watchtowerplay.Of(item.SourceKind)]
	if !ok {
		return nil, nil, errortypes.NewBusinessError(
			"Nothing can be sent from here for this kind of item",
		)
	}
	planned, err := playbook.Plan(ctx, &watchtowerports.PlanRequest{
		TenantInfo: req.TenantInfo,
		Item:       item,
	})
	if err != nil {
		return nil, nil, err
	}
	if err = s.assertMaySend(ctx, actor, planned); err != nil {
		return nil, nil, err
	}

	return item, planned, nil
}

// assertMaySend holds a play to the permission the page for the same act
// asks: messaging a driver, emailing an outside party, tendering a load.
func (s *Plays) assertMaySend(
	ctx context.Context,
	actor *permissionports.RequestActor,
	planned *watchtowerports.PlannedPlay,
) error {
	for _, grant := range planned.Needs {
		result, err := s.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
			PrincipalType:  actor.PrincipalType,
			PrincipalID:    actor.PrincipalID,
			UserID:         actor.UserID,
			APIKeyID:       actor.APIKeyID,
			BusinessUnitID: actor.BusinessUnitID,
			OrganizationID: actor.OrganizationID,
			Resource:       grant.Resource.String(),
			Operation:      grant.Operation,
		})
		if err != nil {
			return err
		}
		if !result.Allowed {
			return notAllowed(grant.Resource)
		}
	}

	return nil
}

// notAllowed says which access the person is missing, in the words of what
// they were trying to do.
//
//nolint:lll // a message is one literal
func notAllowed(resource permission.Resource) error {
	switch resource { //nolint:exhaustive // only what a play asks for
	case permission.ResourceDriverMessage:
		return errortypes.NewAuthorizationError("Your role does not let you message drivers, so this cannot be sent from here. Ask an administrator for that access.")
	case permission.ResourceTender:
		return errortypes.NewAuthorizationError("Your role does not let you tender loads, so this cannot be sent from here. Ask an administrator for that access.")
	default:
		return errortypes.NewAuthorizationError("Your role does not let you email customers and carriers, so this cannot be sent from here. Ask an administrator for that access.")
	}
}

// record notes the request on the item, in its history and in the audit
// log, and puts the item in progress.
func (s *Plays) record(
	ctx context.Context,
	req *watchtowerports.SendPlayRequest,
	actor *permissionports.RequestActor,
	asked string,
) (*watchtower.Item, error) {
	name := s.nameOf(ctx, actor)

	return s.exceptions.applyException(ctx, &exceptionChange{
		target: &watchtowerports.ExceptionTarget{
			TenantInfo: req.TenantInfo,
			ItemID:     req.ItemID,
		},
		actor: actor,
		mutate: func(item *watchtower.Item, now int64) ([]*watchtower.Event, error) {
			if err := item.RecordRequest(actorUserID(actor), name, asked, now); err != nil {
				return nil, err
			}

			return []*watchtower.Event{
				watchtower.NewEvent(item, watchtower.EventRequested, actorUserID(actor)).
					WithNote(asked),
			}, nil
		},
	})
}

// nameOf is the person's name as the item keeps it, so the case can say who
// asked without reading the user again or showing an id.
func (s *Plays) nameOf(ctx context.Context, actor *permissionports.RequestActor) string {
	userID := actorUserID(actor)
	if userID.IsNil() {
		return ""
	}
	user, err := s.users.GetByID(ctx, tenantports.GetUserByIDRequest{
		TenantInfo:   actor.TenantInfo(),
		LookupUserID: userID,
	})
	if err != nil || user == nil {
		return ""
	}

	return strings.TrimSpace(user.Name)
}
