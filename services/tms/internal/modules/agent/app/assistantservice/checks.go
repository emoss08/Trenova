package assistantservice

import (
	"context"
	"strings"
	"sync"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentturn"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdefinitionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// turnChecks is everything a question is checked against and read with
// before it is answered.
type turnChecks struct {
	attachments        []conversation.MessageAttachment
	runtimeAttachments []agentturn.RuntimeAttachment
	attachmentsErr     error

	definition    *agentdefinition.Definition
	definitionErr error
	pageErr       error
	budgetErr     error
	planErr       error
	allowanceErr  error
	roomErr       error

	directed    *agentruntimeports.DirectedTask
	directedErr error

	history    []conversation.Message
	historyErr error

	proposals    []agentruntimeports.ProposalOutcome
	subject      *agentturn.RuntimeSubject
	anchors      []agentturn.RuntimeAnchor
	earlierFiles map[string]string
}

// err is the first check that failed, in the order the checks were always
// made, so a question that fails two of them is told about the same one it
// always was however the reads happened to finish.
func (c *turnChecks) err() error {
	for _, err := range []error{
		c.attachmentsErr,
		c.definitionErr,
		c.pageErr,
		c.budgetErr,
		c.directedErr,
		c.planErr,
		c.allowanceErr,
		c.roomErr,
		c.historyErr,
	} {
		if err != nil {
			return err
		}
	}

	return nil
}

// checkTurn makes every check a question has to pass, and the reads the turn
// is built from, side by side. None needs another's answer, and run one after
// the other they were a dozen round trips before the model was asked
// anything. Nothing here writes. A failed check does not cancel the others:
// each finishes, so the first failure in the old order is the one reported
// rather than a cancellation another check caused.
func (s *Service) checkTurn(
	ctx context.Context,
	thread *conversation.Thread,
	req *assistantports.SendMessageRequest,
	actor *permissionports.RequestActor,
	page *agent.PageContext,
) *turnChecks {
	checks := &turnChecks{}
	var wg sync.WaitGroup

	wg.Go(func() {
		checks.attachments, checks.runtimeAttachments, checks.attachmentsErr = s.resolveAttachments(
			ctx, thread, req.AttachmentDocumentIDs, actor, req.TenantInfo,
		)
	})
	wg.Go(func() {
		checks.definition, checks.definitionErr = s.usableDefinition(ctx, thread, actor, req)
		if checks.definitionErr == nil {
			checks.budgetErr = s.assertWithinBudget(ctx, checks.definition)
		}
	})
	wg.Go(func() {
		checks.directed, checks.directedErr = s.directedTask(ctx, thread, actor, req)
	})
	wg.Go(func() {
		checks.pageErr = s.assertPageTurn(ctx, thread, page, actor)
	})
	wg.Go(func() {
		checks.roomErr = s.assertRoom(ctx, thread, req.TenantInfo)
	})
	wg.Go(func() {
		checks.allowanceErr = s.assertWithinAllowance(ctx, thread.UserID, req.TenantInfo)
	})
	wg.Go(func() {
		checks.planErr = s.assertWithinPlan(ctx, req.TenantInfo)
	})
	// Another agent's steps on a task this one handed it are the thread's to
	// show, not the model's to read again: it only ever saw its own call and
	// the answer that came back. A compacted stretch is read as its summary.
	wg.Go(func() {
		checks.history, checks.historyErr = s.modelHistory(ctx, thread.ID, req.TenantInfo)
		if checks.historyErr == nil {
			checks.earlierFiles = s.earlierFileTexts(ctx, checks.history, req.TenantInfo)
		}
	})
	wg.Go(func() {
		checks.proposals = s.proposalOutcomes(ctx, thread, req.TenantInfo)
	})
	wg.Go(func() {
		checks.subject = s.describeSubject(ctx, thread, actor, req.TenantInfo)
	})
	wg.Go(func() {
		checks.anchors = s.anchorRecords(ctx, &anchorScope{
			thread:   thread,
			page:     page,
			mentions: req.Mentions,
			actor:    actor,
			tenant:   req.TenantInfo,
		})
	})
	wg.Wait()

	return checks
}

// usableDefinition is the thread's agent, refused when it is switched off,
// is not a chat agent, or is not one the person may use. On a quick question's
// conversation it also holds every read, narrowed to the person's as any
// tool is.
func (s *Service) usableDefinition(
	ctx context.Context,
	thread *conversation.Thread,
	actor *permissionports.RequestActor,
	req *assistantports.SendMessageRequest,
) (*agentdefinition.Definition, error) {
	definition, err := s.usableAgent(ctx, thread.AgentDefinitionID, actor, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if thread.Origin.ReadsEverything() {
		return definition.HoldingReads(s.everyRead()), nil
	}

	return definition, nil
}

// usableAgent is an agent the person may talk to: enabled, a chat agent,
// and one their roles give them.
func (s *Service) usableAgent(
	ctx context.Context,
	id pulid.ID,
	actor *permissionports.RequestActor,
	tenant pagination.TenantInfo,
) (*agentdefinition.Definition, error) {
	definition, err := s.definitions.GetByID(
		ctx,
		agentdefinitionports.GetAgentDefinitionByIDRequest{
			ID:         id,
			TenantInfo: tenant,
		},
	)
	if err != nil {
		return nil, err
	}
	if !definition.Enabled {
		return nil, errortypes.NewBusinessError(
			"Agent {0} is disabled and cannot be used", definition.Name,
		)
	}
	if err = assertChatAgent(definition); err != nil {
		return nil, err
	}
	if err = s.assertMayUseAgent(ctx, actor, definition); err != nil {
		return nil, err
	}

	return definition, nil
}

// directedTask is the agent the person handed the message to, checked as
// one they may talk to and whose budget is not spent. The conversation's
// agent need not list it: the person chose it, not the agent. Nil when the
// message is the conversation's own.
func (s *Service) directedTask(
	ctx context.Context,
	thread *conversation.Thread,
	actor *permissionports.RequestActor,
	req *assistantports.SendMessageRequest,
) (*agentruntimeports.DirectedTask, error) {
	if req.DirectedAgentID.IsNil() {
		return nil, nil //nolint:nilnil // no other agent was chosen
	}
	if req.DirectedAgentID == thread.AgentDefinitionID {
		return nil, errortypes.NewValidationError("directedAgentId", errortypes.ErrInvalid,
			"This conversation's agent takes the message itself; send it without naming one")
	}

	definition, err := s.usableAgent(ctx, req.DirectedAgentID, actor, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if err = s.assertWithinBudget(ctx, definition); err != nil {
		return nil, err
	}

	return &agentruntimeports.DirectedTask{
		Delegate: agentturn.AsDelegate(definition),
		Task:     strings.TrimSpace(req.Content),
	}, nil
}
