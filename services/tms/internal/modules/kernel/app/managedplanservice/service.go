package managedplanservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Config *managedconfig.Config
	Repo   managedfeaturesports.StateRepository
}

type Plan struct {
	Managed bool
	View    *managedfeatures.PlanView
	Changes []managedfeatures.HistoryEntry
}

type Service struct {
	managed bool
	repo    managedfeaturesports.StateRepository
}

func New(p Params) *Service {
	return &Service{managed: p.Config.Enabled, repo: p.Repo}
}

func (s *Service) Plan(ctx context.Context) (*Plan, error) {
	if !s.managed {
		return &Plan{
			View: &managedfeatures.PlanView{
				Base:      []platformcatalog.Feature{},
				Included:  []managedfeatures.AddOnFeatures{},
				Available: []managedfeatures.AddOnFeatures{},
			},
			Changes: []managedfeatures.HistoryEntry{},
		}, nil
	}

	held, changes, err := s.repo.History(ctx, managedfeatures.PlanHistoryLimit)
	if err != nil {
		return nil, err
	}
	var features []platformcatalog.FeatureKey
	if held != nil {
		features = held.Features
	}

	return &Plan{
		Managed: true,
		View:    managedfeatures.View(features),
		Changes: managedfeatures.History(features, changes),
	}, nil
}
