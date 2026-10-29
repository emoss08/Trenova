package turnlimits

import (
	"context"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/conversationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

const (
	// MaxThreadMessages is where a conversation must be continued in a new
	// one. Every message past the model's history window is history the
	// client still has to render.
	MaxThreadMessages = 400
)

// AssertWithinBudget refuses a turn once the agent's monthly budget is
// spent. The person asking is told which cap, because the alternative is a
// conversation that stops answering without saying why.
func (s *Limits) AssertWithinBudget(
	ctx context.Context,
	definition *agentdefinition.Definition,
) error {
	if s.Budgets == nil {
		return nil
	}

	refusal, err := s.Budgets.CheckRun(ctx, definition)
	if err != nil {
		return err
	}
	if refusal.Refused() {
		refused := errortypes.NewBusinessError(refusal.Message(definition.Name))
		// The Desk says which cap, when it lifts and who can raise it, so it
		// gets the figures as well as the sentence.
		refused.Params = map[string]string{
			"code":     "agent_budget",
			"cap":      refusal.Cap,
			"spent":    refusal.Spent,
			"limit":    refusal.Limit,
			"resetsAt": strconv.FormatInt(refusal.ResetsAt, 10),
		}
		return refused
	}

	return nil
}

// AssertRoom refuses a turn on a thread that has reached its length. The
// thread is what the client renders and what every turn replays a slice of;
// one that grows without end slows the page it lives on and buries the
// question under its own history. The limit is generous for a conversation
// and the refusal says what to do instead.
func (s *Limits) AssertRoom(
	ctx context.Context,
	thread *conversation.Thread,
	tenantInfo pagination.TenantInfo,
) error {
	count, err := s.Conversations.CountMessages(ctx, conversationports.CountMessagesRequest{
		ThreadID:   thread.ID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if count < MaxThreadMessages {
		return nil
	}

	return errortypes.NewBusinessError(
		"This conversation has reached its limit of {0} messages. Start a new conversation to continue",
		MaxThreadMessages,
	)
}
