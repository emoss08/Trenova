package telematicsservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

const formsLookbackSeconds = int64(48 * 3600)

func (s *Service) syncForms(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider services.TelematicsProvider,
	drivers *sweepDrivers,
	result *TenantSweepResult,
) error {
	if len(drivers.externalIDs) == 0 {
		return nil
	}

	mappingsByTemplate, err := s.formMappingsByTemplate(ctx, tenantInfo)
	if err != nil {
		return err
	}

	providerType := string(provider.Type())
	now := timeutils.NowUnix()
	startAt := now - formsLookbackSeconds

	submissions, listErr := provider.ListFormSubmissions(ctx, drivers.externalIDs, startAt, now)
	if _, ok := partialDriverFailures(listErr); !ok {
		return listErr
	}

	upserted := 0
	for i := range submissions {
		submission := &submissions[i]
		ingestErr := s.ingestFormSubmission(
			ctx,
			tenantInfo,
			drivers.workersByExternalID,
			mappingsByTemplate,
			&ingestFormInput{
				Provider:     providerType,
				SubmissionID: submission.ID,
				TemplateID:   submission.TemplateID,
				TemplateName: submission.TemplateName,
				DriverID:     submission.DriverID,
				RouteStopID:  submission.RouteStopID,
				SubmittedAt:  submission.SubmittedAt,
				Fields:       submission.Fields,
			},
		)
		if ingestErr != nil {
			return ingestErr
		}
		upserted++
	}

	result.FormsUpserted = upserted
	return listErr
}

func (s *Service) formMappingsByTemplate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (map[string]*telematics.FormMapping, error) {
	enabled := true
	mappings, err := s.repo.ListFormMappings(ctx, &repositories.ListFormMappingsRequest{
		TenantInfo: tenantInfo,
		Enabled:    &enabled,
	})
	if err != nil {
		return nil, err
	}
	byTemplate := make(map[string]*telematics.FormMapping, len(mappings))
	for _, mapping := range mappings {
		byTemplate[mapping.TemplateID] = mapping
	}
	return byTemplate, nil
}
