package managedfeaturesports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

type ReplaceRequest struct {
	Scope    managedscope.Scope
	Received *managedfeatures.Received
	Now      int64
}

type LifecycleRequest struct {
	Scope     managedscope.Scope
	Lifecycle managedfeatures.Lifecycle
}

type DueRemindersRequest struct {
	Now   int64
	Limit int
	After *DueReminder
}

type DueReminder struct {
	Scope  managedscope.Scope
	EndsAt int64
}

type StateRepository interface {
	Get(ctx context.Context, scope managedscope.Scope) (*managedfeatures.State, error)
	Replace(ctx context.Context, req *ReplaceRequest) (*managedfeatures.Replacement, error)
	SetLifecycle(ctx context.Context, req *LifecycleRequest) (*managedfeatures.Replacement, error)
	DuePlanEndReminders(ctx context.Context, req *DueRemindersRequest) ([]DueReminder, error)
	RemindPlanEnd(
		ctx context.Context,
		scope managedscope.Scope,
		now int64,
	) (*managedfeatures.Replacement, error)
	History(
		ctx context.Context,
		scope managedscope.Scope,
		limit int,
	) (*managedfeatures.State, []*managedfeatures.Change, error)
	AnyActiveScope(ctx context.Context, feature *platformcatalog.FeatureKey) (bool, error)
}

type PlanObserver interface {
	PlanChanged(ctx context.Context, replacement *managedfeatures.Replacement) error
}

type PlanEndReminderResult struct {
	Reminded int `json:"reminded"`
}

type PlanEndReminder interface {
	RemindPlanEnd(ctx context.Context) (*PlanEndReminderResult, error)
}
