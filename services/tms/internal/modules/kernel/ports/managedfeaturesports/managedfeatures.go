package managedfeaturesports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

type ReplaceRequest struct {
	Features []platformcatalog.FeatureKey
	Now      int64
}

type StateRepository interface {
	Get(ctx context.Context) (*managedfeatures.State, error)
	Replace(ctx context.Context, req *ReplaceRequest) (*managedfeatures.Replacement, error)
	History(ctx context.Context, limit int) (*managedfeatures.State, []*managedfeatures.Change, error)
}

type PlanObserver interface {
	PlanChanged(ctx context.Context, replacement *managedfeatures.Replacement) error
}
