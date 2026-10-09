// Package assistantqueueservice keeps what a person says to a conversation
// while its agent is still working: a word that steers the reply under way at
// its next step, and messages queued to be sent, in order, as each reply ends.
// The queue lives on the server, so it survives a reload, another tab and a
// restart of anything that was working on the reply.
package assistantqueueservice

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentruntime"
	"github.com/emoss08/trenova/internal/core/services/assistantturnservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/assistantjobs"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/serviceerror"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const QueueResource = "assistant_queue"

type Params struct {
	fx.In

	Logger        *zap.Logger
	Queue         repositories.AssistantQueueRepository
	Conversations repositories.ConversationRepository
	Turns         *assistantturnservice.Service
	Workflows     serviceports.WorkflowStarter
	Realtime      serviceports.RealtimeService `optional:"true"`
}

// turnStarter is the part of the turn service the queue needs: whether a
// reply is under way, and starting one.
type turnStarter interface {
	Active(
		ctx context.Context,
		req repositories.ActiveAssistantTurnRequest,
	) (*conversation.AssistantTurn, error)
	StartTurn(
		ctx context.Context,
		req assistantturnservice.StartRequest,
		start func(turn *conversation.AssistantTurn) (string, error),
	) (*conversation.AssistantTurn, error)
}

type threadReader interface {
	GetThread(ctx context.Context, req repositories.GetThreadRequest) (*conversation.Thread, error)
}

type Service struct {
	l             *zap.Logger
	queue         repositories.AssistantQueueRepository
	conversations threadReader
	turns         turnStarter
	workflows     serviceports.WorkflowStarter
	realtime      serviceports.RealtimeService
}

var _ serviceports.AssistantQueueSettler = (*Service)(nil)

func New(p Params) *Service { //nolint:gocritic // fx param structs are passed by value
	return &Service{
		l:             p.Logger.Named("service.assistantqueue"),
		queue:         p.Queue,
		conversations: p.Conversations,
		turns:         p.Turns,
		workflows:     p.Workflows,
		realtime:      p.Realtime,
	}
}

func NewSettler(s *Service) serviceports.AssistantQueueSettler { return s }

type Scope struct {
	ThreadID pulid.ID
	Actor    serviceports.RequestActor
}

func (s *Scope) tenant() pagination.TenantInfo {
	return pagination.TenantInfo{
		OrgID:  s.Actor.OrganizationID,
		BuID:   s.Actor.BusinessUnitID,
		UserID: s.Actor.UserID,
	}
}

func (s *Scope) repo() *repositories.AssistantQueueScope {
	return &repositories.AssistantQueueScope{
		ThreadID:   s.ThreadID,
		UserID:     s.Actor.UserID,
		TenantInfo: s.tenant(),
	}
}

type EnqueueRequest struct {
	Scope   *Scope
	Content string
	Request conversation.QueuedRequest
	Steer   bool
}

type EditRequest struct {
	Scope   *Scope
	ID      pulid.ID
	Content string
	Version int64
}

type ItemRequest struct {
	Scope *Scope
	ID    pulid.ID
}

type ReorderRequest struct {
	Scope *Scope
	IDs   []pulid.ID
}

type Outcome struct {
	Item *conversation.QueuedMessage `json:"item,omitempty"`
	// Steering says the message was handed to the reply under way, which
	// reads it at its next step.
	Steering bool `json:"steering"`
	// Turn is the reply the message started, when the conversation was free.
	Turn *conversation.AssistantTurn `json:"-"`
	// Held says why the message is waiting rather than sent, when it could
	// not be sent although the conversation was free.
	Held string `json:"held,omitempty"`
}

func (s *Service) List(ctx context.Context, scope *Scope) ([]*conversation.QueuedMessage, error) {
	if err := s.ownThread(ctx, scope); err != nil {
		return nil, err
	}

	return s.queue.List(ctx, scope.repo())
}

func (s *Service) Enqueue(ctx context.Context, req *EnqueueRequest) (*Outcome, error) {
	if err := s.ownThread(ctx, req.Scope); err != nil {
		return nil, err
	}

	request := req.Request
	item := &conversation.QueuedMessage{
		OrganizationID: req.Scope.Actor.OrganizationID,
		BusinessUnitID: req.Scope.Actor.BusinessUnitID,
		ThreadID:       req.Scope.ThreadID,
		UserID:         req.Scope.Actor.UserID,
		Content:        req.Content,
		Request:        &request,
		Steer:          req.Steer,
	}
	item.Normalize()
	multiErr := errortypes.NewMultiError()
	item.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	active, err := s.active(ctx, req.Scope)
	if err != nil {
		return nil, err
	}
	item.Steer = req.Steer && steerable(active)

	created, err := s.queue.Insert(ctx, item)
	if err != nil {
		if errors.Is(err, repositories.ErrQueueFull) {
			return nil, errortypes.NewBusinessError(fmt.Sprintf(
				"Up to %d messages can wait in a conversation. Send or remove one first.",
				conversation.MaxQueuedPerThread,
			))
		}
		return nil, err
	}
	defer s.announce(ctx, &req.Scope.Actor, req.Scope.ThreadID)

	if created.Steer && s.steer(ctx, active, created) {
		return &Outcome{Item: created, Steering: true}, nil
	}
	if active != nil {
		return &Outcome{Item: created}, nil
	}

	return s.sendWhenFree(ctx, req.Scope, created)
}

func (s *Service) Edit(ctx context.Context, req *EditRequest) (*conversation.QueuedMessage, error) {
	item, err := s.editable(ctx, req.Scope, req.ID)
	if err != nil {
		return nil, err
	}

	multiErr := errortypes.NewMultiError()
	conversation.ValidateQueuedContent("content", req.Content, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	item.Content = req.Content
	item.Version = req.Version
	item.Normalize()
	updated, err := s.queue.Update(ctx, item)
	if err != nil {
		return nil, err
	}
	s.announce(ctx, &req.Scope.Actor, req.Scope.ThreadID)

	return updated, nil
}

func (s *Service) Remove(ctx context.Context, req *ItemRequest) error {
	if _, err := s.editable(ctx, req.Scope, req.ID); err != nil {
		return err
	}
	if err := s.queue.Delete(ctx, &repositories.QueuedMessageRequest{
		ID:    req.ID,
		Scope: *req.Scope.repo(),
	}); err != nil {
		return err
	}
	s.announce(ctx, &req.Scope.Actor, req.Scope.ThreadID)

	return nil
}

func (s *Service) Reorder(
	ctx context.Context,
	req *ReorderRequest,
) ([]*conversation.QueuedMessage, error) {
	if err := s.ownThread(ctx, req.Scope); err != nil {
		return nil, err
	}

	current, err := s.queue.List(ctx, req.Scope.repo())
	if err != nil {
		return nil, err
	}
	if !sameItems(current, req.IDs) {
		return nil, errortypes.NewValidationError(
			"ids",
			errortypes.ErrInvalid,
			"The order must name every waiting message once. Reload the conversation and try again.",
		)
	}

	ordered, err := s.queue.Reorder(ctx, &repositories.ReorderQueuedMessagesRequest{
		IDs:   req.IDs,
		Scope: *req.Scope.repo(),
	})
	if err != nil {
		return nil, err
	}
	s.announce(ctx, &req.Scope.Actor, req.Scope.ThreadID)
	slices.SortFunc(ordered, func(a, b *conversation.QueuedMessage) int {
		return cmp.Compare(a.Position, b.Position)
	})

	return ordered, nil
}

func (s *Service) SendNow(ctx context.Context, req *ItemRequest) (*Outcome, error) {
	if err := s.ownThread(ctx, req.Scope); err != nil {
		return nil, err
	}

	active, err := s.active(ctx, req.Scope)
	if err != nil {
		return nil, err
	}
	key := &repositories.QueuedMessageRequest{ID: req.ID, Scope: *req.Scope.repo()}
	defer s.announce(ctx, &req.Scope.Actor, req.Scope.ThreadID)

	if active != nil {
		return s.steerNow(ctx, active, key)
	}

	item, err := s.queue.Claim(ctx, key)
	if errors.Is(err, repositories.ErrQueueEmpty) {
		return nil, errortypes.NewNotFoundError("{0} not found within your organization",
			"Queued message")
	}
	if err != nil {
		return nil, err
	}
	turn, err := s.start(ctx, item)
	if err != nil {
		s.restore(ctx, item)
		return &Outcome{Item: item, Held: heldReason(err)}, nil
	}

	return &Outcome{Turn: turn}, nil
}

// steerNow hands a waiting message to the reply under way, which reads it at
// its next step. A message with files waits for the reply to end, and one the
// reply can no longer take is sent once it does.
func (s *Service) steerNow(
	ctx context.Context,
	active *conversation.AssistantTurn,
	key *repositories.QueuedMessageRequest,
) (*Outcome, error) {
	if !steerable(active) {
		return nil, errortypes.NewBusinessError(
			"The conversation is being compacted. The message is sent once that finishes.",
		)
	}
	waiting, err := s.queue.Get(ctx, key)
	if err != nil {
		return nil, err
	}
	if waiting.Request != nil && len(waiting.Request.AttachmentDocumentIDs) > 0 {
		return nil, errortypes.NewBusinessError(
			"A message with files is sent once the reply under way ends; it cannot steer it.",
		)
	}
	item, err := s.queue.MarkSteer(ctx, key)
	if err != nil {
		return nil, err
	}
	if s.steer(ctx, active, item) {
		return &Outcome{Item: item, Steering: true}, nil
	}

	return &Outcome{Item: item}, nil
}

func (s *Service) SettleQueue(
	ctx context.Context,
	req *serviceports.SettleQueueRequest,
) *serviceports.QueuedTurn {
	tenant := pagination.TenantInfo{OrgID: req.TenantInfo.OrgID, BuID: req.TenantInfo.BuID}
	ctx = dbscope.WithValidTenant(context.WithoutCancel(ctx), tenant.DBTenant())
	scope := &repositories.AssistantQueueScope{ThreadID: req.ThreadID, TenantInfo: tenant}
	owner := serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    req.UserID,
		UserID:         req.UserID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}

	changed := false
	if len(req.Read) > 0 {
		if err := s.queue.DeleteMany(ctx, &repositories.DeleteQueuedMessagesRequest{
			IDs:   req.Read,
			Scope: *scope,
		}); err != nil {
			s.l.Warn("could not clear the messages a reply read",
				zap.String("thread", req.ThreadID.String()),
				zap.Error(err),
			)
		}
		changed = true
	}
	if !req.Dispatch {
		if changed {
			s.announce(ctx, &owner, req.ThreadID)
		}
		return nil
	}

	turn, item := s.dispatchNext(ctx, scope)
	if changed || item != nil {
		s.announce(ctx, &owner, req.ThreadID)
	}
	if turn == nil {
		return nil
	}

	return &serviceports.QueuedTurn{TurnID: turn.ID, QueuedID: item.ID, Input: turn.Input}
}

func (s *Service) sendWhenFree(
	ctx context.Context,
	scope *Scope,
	created *conversation.QueuedMessage,
) (*Outcome, error) {
	turn, item := s.dispatchNext(ctx, scope.repo())
	if turn != nil {
		if item.ID == created.ID {
			return &Outcome{Turn: turn}, nil
		}
		return &Outcome{Item: created, Turn: turn}, nil
	}
	if item != nil {
		return &Outcome{Item: created, Held: item.heldReason}, nil
	}

	return &Outcome{Item: created}, nil
}

type dispatched struct {
	*conversation.QueuedMessage
	heldReason string
}

func (s *Service) dispatchNext(
	ctx context.Context,
	scope *repositories.AssistantQueueScope,
) (*conversation.AssistantTurn, *dispatched) {
	item, err := s.queue.ClaimNext(ctx, scope)
	if errors.Is(err, repositories.ErrQueueEmpty) {
		return nil, nil
	}
	if err != nil {
		s.l.Warn("could not take the next queued message",
			zap.String("thread", scope.ThreadID.String()),
			zap.Error(err),
		)
		return nil, nil
	}
	turn, err := s.start(ctx, item)
	if err != nil {
		s.restore(ctx, item)
		s.l.Info("a queued message waits: the conversation could not start it",
			zap.String("thread", scope.ThreadID.String()),
			zap.Error(err),
		)
		return nil, &dispatched{QueuedMessage: item, heldReason: heldReason(err)}
	}

	return turn, &dispatched{QueuedMessage: item}
}

func (s *Service) start(
	ctx context.Context,
	item *conversation.QueuedMessage,
) (*conversation.AssistantTurn, error) {
	request := item.Request
	if request == nil {
		request = &conversation.QueuedRequest{}
	}
	actor := serviceports.RequestActor{
		PrincipalType:  serviceports.PrincipalTypeUser,
		PrincipalID:    item.UserID,
		UserID:         item.UserID,
		OrganizationID: item.OrganizationID,
		BusinessUnitID: item.BusinessUnitID,
	}

	return s.turns.StartTurn(ctx, assistantturnservice.StartRequest{
		ThreadID: item.ThreadID,
		UserID:   item.UserID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  item.OrganizationID,
			BuID:   item.BusinessUnitID,
			UserID: item.UserID,
		},
		Origin: conversation.AssistantTurnOriginPerson,
		Input:  item.Content,
	}, func(turn *conversation.AssistantTurn) (string, error) {
		run, err := assistantjobs.StartTurnWorkflow(ctx, s.workflows, turn,
			assistantjobs.TurnStart{
				Actor:   actor,
				Content: item.Content,
				Request: assistantjobs.AssistantTurnRequest{
					Page:                  request.Page,
					Surface:               request.Surface,
					Mentions:              request.Mentions,
					AttachmentDocumentIDs: request.AttachmentDocumentIDs,
					PreferredProviderID:   request.ProviderID,
					ProviderChosen:        request.ProviderChosen,
				},
			})
		if err != nil {
			return "", err
		}

		return run.GetID(), nil
	})
}

func (s *Service) restore(ctx context.Context, item *conversation.QueuedMessage) {
	if err := s.queue.Restore(ctx, item); err != nil {
		s.l.Error("a queued message could not be put back after it failed to start",
			zap.String("message", item.ID.String()),
			zap.Error(err),
		)
	}
}

// steer hands the message to the reply under way. It reports false when no
// execution carries that reply any more, in which case the message waits in
// the queue like any other and is sent as the reply ends.
func (s *Service) steer(
	ctx context.Context,
	active *conversation.AssistantTurn,
	item *conversation.QueuedMessage,
) bool {
	if active == nil {
		return false
	}
	var mentions []agent.EntityRef
	if item.Request != nil {
		mentions = item.Request.Mentions
	}
	err := s.workflows.SignalWorkflow(ctx, active.ExecutionID(), "", agentflow.SteerSignal,
		agentruntime.Steer{ID: item.ID, Content: item.Content, Mentions: mentions})
	if err == nil {
		return true
	}

	var gone *serviceerror.NotFound
	if !errors.As(err, &gone) {
		s.l.Warn("could not hand a message to the reply under way; it waits in the queue",
			zap.String("turn", active.ID.String()),
			zap.Error(err),
		)
	}

	return false
}

func (s *Service) editable(
	ctx context.Context,
	scope *Scope,
	id pulid.ID,
) (*conversation.QueuedMessage, error) {
	if err := s.ownThread(ctx, scope); err != nil {
		return nil, err
	}
	item, err := s.queue.Get(ctx, &repositories.QueuedMessageRequest{ID: id, Scope: *scope.repo()})
	if err != nil {
		return nil, err
	}
	if !item.Steer {
		return item, nil
	}

	active, err := s.active(ctx, scope)
	if err != nil {
		return nil, err
	}
	if steerable(active) {
		return nil, errortypes.NewBusinessError(
			"This message has been handed to the reply under way. Stop the reply to take it back.",
		)
	}
	item.Steer = false

	return item, nil
}

func (s *Service) ownThread(ctx context.Context, scope *Scope) error {
	_, err := s.conversations.GetThread(ctx, repositories.GetThreadRequest{
		ID:         scope.ThreadID,
		UserID:     scope.Actor.UserID,
		TenantInfo: scope.tenant(),
	})

	return err
}

func (s *Service) active(ctx context.Context, scope *Scope) (*conversation.AssistantTurn, error) {
	return s.turns.Active(ctx, repositories.ActiveAssistantTurnRequest{
		ThreadID:   scope.ThreadID,
		UserID:     scope.Actor.UserID,
		TenantInfo: scope.tenant(),
	})
}

func (s *Service) announce(
	ctx context.Context,
	actor *serviceports.RequestActor,
	threadID pulid.ID,
) {
	if s.realtime == nil {
		return
	}
	if err := s.realtime.PublishResourceInvalidation(
		context.WithoutCancel(ctx),
		&serviceports.PublishResourceInvalidationRequest{
			OrganizationID: actor.OrganizationID,
			BusinessUnitID: actor.BusinessUnitID,
			AudienceUserID: actor.UserID,
			Resource:       QueueResource,
			Action:         "updated",
			RecordID:       threadID,
			Entity:         map[string]string{"threadId": threadID.String()},
		},
	); err != nil {
		s.l.Debug("could not announce a change to a conversation's queue",
			zap.String("thread", threadID.String()),
			zap.Error(err),
		)
	}
}

func steerable(active *conversation.AssistantTurn) bool {
	return active != nil && active.Origin != conversation.AssistantTurnOriginCompaction
}

func sameItems(current []*conversation.QueuedMessage, ids []pulid.ID) bool {
	if len(current) != len(ids) {
		return false
	}
	seen := make(map[pulid.ID]struct{}, len(ids))
	for _, id := range ids {
		if _, dup := seen[id]; dup {
			return false
		}
		seen[id] = struct{}{}
	}
	for _, item := range current {
		if _, ok := seen[item.ID]; !ok {
			return false
		}
	}

	return true
}

func heldReason(err error) string {
	switch {
	case errortypes.IsBusinessError(err),
		errortypes.IsQuotaExceededError(err),
		errortypes.IsPlanRestrictionError(err):
		return err.Error()
	default:
		return "The message could not be sent just now. It waits at the front of the queue."
	}
}
