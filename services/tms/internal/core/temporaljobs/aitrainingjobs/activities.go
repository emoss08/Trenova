package aitrainingjobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ActivitiesParams struct {
	fx.In

	Runner services.AITrainingExportRunner
	Logger *zap.Logger
}

type Activities struct {
	runner services.AITrainingExportRunner
	l      *zap.Logger
}

func NewActivities(p ActivitiesParams) *Activities {
	return &Activities{
		runner: p.Runner,
		l:      p.Logger.Named("job.aitraining-export"),
	}
}

func (a *Activities) BeginAITrainingExportActivity(ctx context.Context, input *ExportInput) error {
	if _, err := a.runner.Begin(ctx, input.ExportID); err != nil {
		return classify(err)
	}

	return nil
}

func (a *Activities) ListAITrainingOrganizationsActivity(
	ctx context.Context,
	input *ListOrganizationsInput,
) ([]ConsentingOrganization, error) {
	consents, err := a.runner.ListOrganizations(ctx, repositories.ListConsentingOrganizationsRequest{
		AfterOrganizationID: input.AfterOrganizationID,
		AfterBusinessUnitID: input.AfterBusinessUnitID,
		Limit:               input.Limit,
	})
	if err != nil {
		return nil, fmt.Errorf("list consenting organizations: %w", err)
	}

	out := make([]ConsentingOrganization, 0, len(consents))
	for _, consent := range consents {
		out = append(out, ConsentingOrganization{
			OrganizationID: consent.OrganizationID,
			BusinessUnitID: consent.BusinessUnitID,
			GrantedAt:      consent.GrantedAt,
		})
	}

	return out, nil
}

func (a *Activities) ExportAITrainingOrganizationActivity(
	ctx context.Context,
	input *ExportOrganizationInput,
) (*ExportOrganizationOutcome, error) {
	progress, err := a.runner.ExportOrganization(ctx, &services.ExportTrainingOrganizationRequest{
		ExportID: input.ExportID,
		Ordinal:  input.Ordinal,
		Consent: repositories.TrainingConsent{
			OrganizationID: input.Organization.OrganizationID,
			BusinessUnitID: input.Organization.BusinessUnitID,
			Granted:        true,
			GrantedAt:      input.Organization.GrantedAt,
		},
		Heartbeat: activity.RecordHeartbeat,
	})
	if err != nil {
		return nil, classify(err)
	}

	a.l.Debug("training export organization done",
		zap.String("exportId", input.ExportID.String()),
		zap.Int("ordinal", input.Ordinal),
		zap.Int("examples", progress.Examples),
		zap.Bool("consentWithdrawn", progress.ConsentWithdrawn),
	)

	return &ExportOrganizationOutcome{
		Examples:         progress.Examples,
		ConsentWithdrawn: progress.ConsentWithdrawn,
	}, nil
}

func (a *Activities) FinishAITrainingExportActivity(
	ctx context.Context,
	input *ExportInput,
) (*ExportOutcome, error) {
	entity, err := a.runner.Finish(ctx, &services.FinishAITrainingExportRequest{ExportID: input.ExportID})
	if err != nil {
		return nil, fmt.Errorf("finish training export: %w", err)
	}

	return &ExportOutcome{Status: entity.Status.String(), Examples: entity.ExamplesTotal}, nil
}

func (a *Activities) FailAITrainingExportActivity(ctx context.Context, input *FailInput) error {
	if err := a.runner.Fail(ctx, input.ExportID, input.Message); err != nil {
		return fmt.Errorf("fail training export: %w", err)
	}

	return nil
}

func classify(err error) error {
	if errors.Is(err, aitraining.ErrExportInactive) {
		return temporal.NewNonRetryableApplicationError(err.Error(), errorTypeExportInactive, err)
	}

	return err
}
