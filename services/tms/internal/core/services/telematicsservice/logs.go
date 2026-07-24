package telematicsservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type WorkerHOSLogEntry = services.ProviderHOSLogEntry

type WorkerHOSDailyLog = services.ProviderHOSDailyLog

type WorkerFormSubmission = services.ProviderFormSubmission

type HOSCertificationSummary struct {
	WorkerID        pulid.ID `json:"workerId"`
	WorkerName      string   `json:"workerName"`
	UncertifiedDays int      `json:"uncertifiedDays"`
	TotalDays       int      `json:"totalDays"`
}

func (s *Service) workerExternalID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (string, error) {
	mappings, err := s.repo.ListWorkerMappings(ctx, tenantInfo)
	if err != nil {
		return "", err
	}
	for _, mapping := range mappings {
		if mapping.WorkerID == workerID {
			return mapping.ExternalID, nil
		}
	}
	return "", nil
}

func (s *Service) GetWorkerHOSLogs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	startAt int64,
	endAt int64,
) ([]*WorkerHOSLogEntry, error) {
	externalID, err := s.workerExternalID(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}
	if externalID == "" {
		return []*WorkerHOSLogEntry{}, nil
	}

	provider, err := s.resolveProvider(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	entries, err := provider.ListHOSLogs(ctx, externalID, startAt, endAt)
	if err != nil {
		return nil, err
	}

	out := make([]*WorkerHOSLogEntry, 0, len(entries))
	for i := range entries {
		out = append(out, &entries[i])
	}
	return out, nil
}

func (s *Service) GetWorkerHOSDailyLogs(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	startDate string,
	endDate string,
) ([]*WorkerHOSDailyLog, error) {
	externalID, err := s.workerExternalID(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}
	if externalID == "" {
		return []*WorkerHOSDailyLog{}, nil
	}

	provider, err := s.resolveProvider(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	days, err := provider.ListHOSDailyLogs(ctx, externalID, startDate, endDate)
	if err != nil {
		return nil, err
	}

	out := make([]*WorkerHOSDailyLog, 0, len(days))
	for i := range days {
		out = append(out, &days[i])
	}
	return out, nil
}

func (s *Service) GetWorkerFormSubmissions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	startAt int64,
	endAt int64,
) ([]*WorkerFormSubmission, error) {
	externalID, err := s.workerExternalID(ctx, tenantInfo, workerID)
	if err != nil {
		return nil, err
	}
	if externalID == "" {
		return []*WorkerFormSubmission{}, nil
	}

	provider, err := s.resolveProvider(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	submissions, err := provider.ListFormSubmissions(ctx, externalID, startAt, endAt)
	if err != nil {
		return nil, err
	}

	out := make([]*WorkerFormSubmission, 0, len(submissions))
	for i := range submissions {
		out = append(out, &submissions[i])
	}
	return out, nil
}

func (s *Service) GetHOSCertificationSummary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	startDate string,
	endDate string,
) ([]*HOSCertificationSummary, error) {
	mappings, err := s.repo.ListWorkerMappings(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	if len(mappings) == 0 {
		return []*HOSCertificationSummary{}, nil
	}

	provider, err := s.resolveProvider(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	out := make([]*HOSCertificationSummary, 0, len(mappings))
	for i := range mappings {
		mapping := &mappings[i]
		if mapping.ExternalID == "" {
			continue
		}

		days, dayErr := provider.ListHOSDailyLogs(ctx, mapping.ExternalID, startDate, endDate)
		if dayErr != nil {
			return nil, dayErr
		}

		uncertified := 0
		for j := range days {
			if !days[j].IsCertified {
				uncertified++
			}
		}
		if uncertified == 0 {
			continue
		}

		out = append(out, &HOSCertificationSummary{
			WorkerID:        mapping.WorkerID,
			WorkerName:      workerMappingName(mapping),
			UncertifiedDays: uncertified,
			TotalDays:       len(days),
		})
	}
	return out, nil
}
