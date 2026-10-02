package assistantservice

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// turnChecks is everything a question is checked against and read with
// before it is answered.
type turnChecks struct {
	attachments        []conversation.MessageAttachment
	runtimeAttachments []agentdefinition.RuntimeAttachment
	attachmentsErr     error

	definition    *agentdefinition.Definition
	definitionErr error
	pageErr       error
	budgetErr     error
	roomErr       error

	history    []conversation.Message
	historyErr error

	proposals []services.ProposalOutcome
	subject   *agentdefinition.RuntimeSubject
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
	req *services.SendMessageRequest,
	actor *services.RequestActor,
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
		checks.pageErr = s.assertPageTurn(ctx, thread, page, actor)
	})
	wg.Go(func() {
		checks.roomErr = s.assertRoom(ctx, thread, req.TenantInfo)
	})
	// Another agent's steps on a task this one handed it are the thread's to
	// show, not the model's to read again: it only ever saw its own call and
	// the answer that came back.
	wg.Go(func() {
		checks.history, checks.historyErr = s.conversations.ListMessages(
			ctx,
			repositories.ListMessagesRequest{
				ThreadID:     thread.ID,
				TenantInfo:   req.TenantInfo,
				Limit:        historyLimit,
				ExcludeKinds: conversation.ModelHiddenKinds(),
			},
		)
	})
	wg.Go(func() {
		checks.proposals = s.proposalOutcomes(ctx, thread, req.TenantInfo)
	})
	wg.Go(func() {
		checks.subject = s.describeSubject(ctx, thread, actor, req.TenantInfo)
	})
	wg.Wait()

	return checks
}

// usableDefinition is the thread's agent, refused when it is switched off,
// is not a chat agent, or is not one the person may use.
func (s *Service) usableDefinition(
	ctx context.Context,
	thread *conversation.Thread,
	actor *services.RequestActor,
	req *services.SendMessageRequest,
) (*agentdefinition.Definition, error) {
	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         thread.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
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
