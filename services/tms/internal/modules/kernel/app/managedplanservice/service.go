package managedplanservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedfeaturesports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/managedusageports"
	"go.uber.org/fx"
)

type Params struct {
	fx.In

	Config *managedconfig.Config
	Repo   managedfeaturesports.StateRepository
	Usage  managedusageports.UsageReader
	Scopes *managedscope.Resolver
}

type Plan struct {
	Managed    bool
	ManageURL  string
	EndsAt     int64
	BackupDays int
	Usage      *managedusage.Usage
	View       *managedfeatures.PlanView
	Changes    []managedfeatures.HistoryEntry
}

type Service struct {
	managed bool
	repo    managedfeaturesports.StateRepository
	usage   managedusageports.UsageReader
	scopes  *managedscope.Resolver
}

func New(p Params) *Service {
	return &Service{managed: p.Config.Enabled, repo: p.Repo, usage: p.Usage, scopes: p.Scopes}
}

func (s *Service) Plan(ctx context.Context) (*Plan, error) {
	if !s.managed {
		return &Plan{
			BackupDays: managedfeatures.DefaultFinalBackupDays,
			View: &managedfeatures.PlanView{
				Base:      []platformcatalog.Feature{},
				Included:  []managedfeatures.AddOnFeatures{},
				Available: []managedfeatures.AddOnFeatures{},
			},
			Changes: []managedfeatures.HistoryEntry{},
		}, nil
	}

	scope, err := s.scopes.Scope(ctx)
	if err != nil {
		return nil, err
	}
	held, changes, err := s.repo.History(ctx, scope, managedfeatures.PlanHistoryLimit)
	if err != nil {
		return nil, err
	}
	usage, err := s.usage.Usage(ctx, scope)
	if err != nil {
		return nil, err
	}

	var features []platformcatalog.FeatureKey
	var endings []managedfeatures.AddOnEnding
	var manageURL string
	var endsAt int64
	if held != nil {
		features, endings, manageURL = held.Features, held.AddOnEndings, held.PlanURL
		endsAt = held.PlanEndsAt
	}
	backupDays := held.BackupDays()

	return &Plan{
		Managed:    true,
		ManageURL:  manageURL,
		EndsAt:     endsAt,
		BackupDays: backupDays,
		Usage:      usage,
		View:       managedfeatures.View(features, endings),
		Changes:    managedfeatures.History(features, changes),
	}, nil
}
