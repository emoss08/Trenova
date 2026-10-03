package decisioncommitjobs

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Committer services.ApprovalCommitter
	Logger    *zap.Logger
}

type Activities struct {
	committer services.ApprovalCommitter
	l         *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		committer: p.Committer,
		l:         p.Logger.Named("job.decision-commit"),
	}
}

// CommitApprovalActivity carries out an approval whose window has closed. It
// is safe to retry: what it already committed is not in the window any more,
// and what it claimed but did not finish is finished on the next attempt.
func (a *Activities) CommitApprovalActivity(
	ctx context.Context,
	req *services.CommitApprovalRequest,
) error {
	if err := a.committer.CommitApproval(ctx, req); err != nil {
		return fmt.Errorf("commit approval %s: %w", req.WorkflowID, err)
	}

	return nil
}
