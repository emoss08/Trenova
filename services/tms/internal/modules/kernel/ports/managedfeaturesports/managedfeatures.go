package managedfeaturesports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
)

type ReplaceRequest struct {
	Received *managedfeatures.Received
	Now      int64
}

type StateRepository interface {
	Get(ctx context.Context) (*managedfeatures.State, error)
	Replace(ctx context.Context, req *ReplaceRequest) (*managedfeatures.Replacement, error)
	RemindPlanEnd(ctx context.Context, now int64) (*managedfeatures.Replacement, error)
	History(ctx context.Context, limit int) (*managedfeatures.State, []*managedfeatures.Change, error)
}

type PlanObserver interface {
	PlanChanged(ctx context.Context, replacement *managedfeatures.Replacement) error
}

type PlanEndReminderResult struct {
	Reminded bool  `json:"reminded"`
	EndsAt   int64 `json:"endsAt"`
}

type PlanEndReminder interface {
	RemindPlanEnd(ctx context.Context) (*PlanEndReminderResult, error)
}
