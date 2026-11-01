package deskcaseresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/deskmapping"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentwait"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/deskcase"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskcaseports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

type waitList struct {
	Items []*agentwait.Wait `json:"items"`
}

type caseBinding struct {
	ThreadID     pulid.ID              `json:"threadId"`
	SubjectType  agent.SubjectType     `json:"subjectType"`
	SubjectID    pulid.ID              `json:"subjectId"`
	SnoozedUntil *int64                `json:"snoozedUntil,omitempty"`
	SnoozeAnchor deskcase.SnoozeAnchor `json:"snoozeAnchor,omitempty"`
	Case         *deskcase.Summary     `json:"case,omitempty"`
}

func bindingOf(thread *conversation.Thread, err error) (any, error) {
	if err != nil {
		return nil, err
	}

	return caseBinding{
		ThreadID:     thread.ID,
		SubjectType:  thread.SubjectType,
		SubjectID:    thread.SubjectID,
		SnoozedUntil: thread.SnoozedUntil,
		SnoozeAnchor: thread.SnoozeAnchor,
		Case:         thread.Case,
	}, nil
}

func caseThread(
	authCtx *authctx.AuthContext,
	threadID string,
) (*deskcaseports.CaseThreadRequest, error) {
	thread, err := deskmapping.ThreadRequest(authCtx, "threadId", threadID)
	if err != nil {
		return nil, err
	}

	return &deskcaseports.CaseThreadRequest{
		ThreadID:   thread.ID,
		TenantInfo: thread.TenantInfo,
		Actor:      actorutil.FromAuthContext(authCtx),
	}, nil
}

func (r *MutationResolver) caseRequest(
	ctx context.Context,
	threadID string,
) (*deskcaseports.CaseThreadRequest, error) {
	authCtx, err := r.RequirePermission(ctx, permission.ResourceAssistant, permission.OpRead)
	if err != nil {
		return nil, err
	}

	return caseThread(authCtx, threadID)
}
