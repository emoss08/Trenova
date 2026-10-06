package assistantservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/pkg/errortypes"
)

// raiseNotifier tells everyone who holds a permission about one thing.
type raiseNotifier interface {
	NotifyPermitted(
		ctx context.Context,
		req notificationservice.NotifyPermittedRequest,
	) (int, error)
}

const (
	// EventAssistantRaiseRequested is a person asking for more room with an
	// agent: access, allowance or budget.
	EventAssistantRaiseRequested = "assistant.raise_requested"
	raiseSource                  = "assistant"
	// maxRaiseNotices keeps one request from paging a whole company.
	maxRaiseNotices = 10
)

// RequestMore asks the people who run AI Control for what a person ran out
// of in a conversation: access to its agent, more of their own allowance, or
// more of the agent's budget or daily requests. The same request from the same
// person about the same agent goes out once a day.
func (s *Service) RequestMore(
	ctx context.Context,
	req services.RequestMoreRequest,
) (*services.RequestMoreResult, error) {
	if s.notifier == nil {
		return nil, errortypes.NewBusinessError("Requests cannot be sent from here yet")
	}
	thread, err := s.conversations.GetThread(ctx, req.Thread)
	if err != nil {
		return nil, err
	}
	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         thread.AgentDefinitionID,
		TenantInfo: req.Thread.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	who := s.personName(ctx, &req.Thread.UserID, req.Thread.TenantInfo)
	if who == "" {
		who = "Someone"
	}
	title, message, err := raiseWording(req.Kind, who, definition.Name)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	correlation := fmt.Sprintf("%s:%s:%s:%s",
		req.Kind, req.Thread.UserID, definition.ID, now.Format("2006-01-02"))
	sent, err := s.notifier.NotifyPermitted(ctx, notificationservice.NotifyPermittedRequest{
		Tenant:      req.Thread.TenantInfo,
		Resource:    permission.ResourceAgentControl,
		Operation:   permission.OpUpdate,
		Limit:       maxRaiseNotices,
		Now:         now.Unix(),
		DedupeSince: now.Add(-24 * time.Hour).Unix(),
		Notification: notification.Notification{
			EventType:     EventAssistantRaiseRequested,
			Priority:      notification.PriorityMedium,
			Title:         title,
			Message:       message,
			Source:        raiseSource,
			CorrelationID: &correlation,
			Data: map[string]any{
				"link":              "/admin/agent-control",
				"kind":              req.Kind,
				"agentDefinitionId": definition.ID.String(),
				"threadId":          thread.ID.String(),
			},
		},
	})
	if err != nil {
		return nil, err
	}

	return &services.RequestMoreResult{Sent: sent}, nil
}

func raiseWording(kind, who, agent string) (title, message string, err error) {
	switch kind {
	case services.RequestMoreAccess:
		return who + " asks for access to " + agent,
			who + " can no longer use " + agent + " and asked for access in AI Control.", nil
	case services.RequestMoreAllowance:
		return who + " asks for more AI allowance",
			who + " has used this month's AI allowance and asked for more.", nil
	case services.RequestMoreBudget:
		return agent + " has spent its monthly budget",
			who + " asked for " + agent + "'s monthly budget to be raised.", nil
	case services.RequestMoreDailyRuns:
		return agent + " reached its daily request limit",
			who + " asked for " + agent + "'s daily request limit to be raised.", nil
	default:
		return "", "", errortypes.NewValidationError(
			"kind", errortypes.ErrInvalid, "Ask for access, allowance, budget or daily requests",
		)
	}
}
