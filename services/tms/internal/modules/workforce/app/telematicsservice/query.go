package telematicsservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type Status struct {
	Provider          string `json:"provider"`
	Enabled           bool   `json:"enabled"`
	Configured        bool   `json:"configured"`
	WebhookConfigured bool   `json:"webhookConfigured"`
	LastPolledAt      int64  `json:"lastPolledAt"`
	LastSuccessAt     int64  `json:"lastSuccessAt"`
	FailureCount      int    `json:"failureCount"`
	LastError         string `json:"lastError"`
	MappedTractors    int    `json:"mappedTractors"`
	TotalTractors     int    `json:"totalTractors"`
	MappedWorkers     int    `json:"mappedWorkers"`
}

func (s *Service) ListVehiclePositions(
	ctx context.Context,
	req *telematicsports.ListVehiclePositionsRequest,
) ([]*telematics.VehiclePosition, error) {
	maxAge := req.MaxAgeSeconds
	if maxAge <= 0 {
		maxAge = defaultPositionMaxAgeSeconds
	}
	return s.repo.ListVehiclePositions(ctx, &telematicsports.ListVehiclePositionsRequest{
		TenantInfo:     req.TenantInfo,
		TractorIDs:     req.TractorIDs,
		MaxAgeSeconds:  maxAge,
		IncludeTractor: true,
		IncludeWorker:  true,
	})
}

func (s *Service) GetVehiclePosition(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	tractorID pulid.ID,
) (*telematics.VehiclePosition, error) {
	positions, err := s.repo.ListVehiclePositions(ctx, &telematicsports.ListVehiclePositionsRequest{
		TenantInfo:    tenantInfo,
		TractorIDs:    []pulid.ID{tractorID},
		IncludeWorker: true,
	})
	if err != nil {
		return nil, err
	}
	if len(positions) == 0 {
		return nil, errortypes.NewNotFoundError("The tractor has not reported a position")
	}
	return positions[0], nil
}

func (s *Service) ListWorkerHOSStates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerIDs []pulid.ID,
	limit int,
) ([]*telematics.WorkerHOSState, error) {
	return s.repo.ListWorkerHOSStates(ctx, &telematicsports.ListWorkerHOSStatesRequest{
		TenantInfo:    tenantInfo,
		WorkerIDs:     workerIDs,
		IncludeWorker: true,
		Limit:         limit,
	})
}

func (s *Service) GetWorkerHOSState(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
) (*telematics.WorkerHOSState, error) {
	state, err := s.repo.GetWorkerHOSState(ctx, telematicsports.GetWorkerHOSStateRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return nil, err
	}
	return state, nil
}

func (s *Service) ListWorkerHOSViolations(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	since int64,
	limit int,
) ([]*telematics.WorkerHOSViolation, error) {
	return s.repo.ListWorkerHOSViolations(ctx, &telematicsports.ListWorkerHOSViolationsRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
		Since:      since,
		Limit:      limit,
	})
}

type VehicleInspectionRecord struct {
	*telematics.VehicleInspection
	WorkerName string
}

func (s *Service) ListVehicleInspections(
	ctx context.Context,
	req *telematicsports.ListVehicleInspectionsRequest,
) ([]*VehicleInspectionRecord, error) {
	inspections, err := s.repo.ListVehicleInspections(ctx, req)
	if err != nil {
		return nil, err
	}
	if len(inspections) == 0 {
		return []*VehicleInspectionRecord{}, nil
	}

	names, err := s.workerNamesByID(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	out := make([]*VehicleInspectionRecord, 0, len(inspections))
	for _, inspection := range inspections {
		out = append(out, &VehicleInspectionRecord{
			VehicleInspection: inspection,
			WorkerName:        names[inspection.WorkerID],
		})
	}
	return out, nil
}

func (s *Service) workerNamesByID(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (map[pulid.ID]string, error) {
	mappings, err := s.repo.ListWorkerMappings(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	names := make(map[pulid.ID]string, len(mappings))
	for i := range mappings {
		names[mappings[i].WorkerID] = workerMappingName(&mappings[i])
	}
	return names, nil
}

func workerMappingName(mapping *telematicsports.WorkerTelematicsMapping) string {
	return strings.TrimSpace(
		strings.TrimSpace(mapping.FirstName) + " " + strings.TrimSpace(mapping.LastName),
	)
}

func (s *Service) GetStatus(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*Status, error) {
	status := new(Status)

	providerType := integration.TypeSamsara
	if provider, providerErr := s.resolveProvider(ctx, tenantInfo); providerErr == nil {
		providerType = provider.Type()
	}
	status.Provider = string(providerType)

	runtimeCfg, err := s.integrationService.GetRuntimeConfig(ctx, tenantInfo, providerType)
	switch {
	case err == nil:
		status.Enabled = runtimeCfg.Enabled
		status.Configured = runtimeCfg.Configured && runtimeCfg.Ready
		status.WebhookConfigured = runtimeCfg.Config[integration.WebhookSecretField] != "" &&
			runtimeCfg.Config["webhookToken"] != ""
	case errortypes.IsBusinessError(err):
	default:
		return nil, err
	}

	feedState, err := s.repo.GetFeedState(
		ctx,
		tenantInfo,
		string(providerType),
		telematics.FeedTypeVehicleStats,
	)
	if err != nil && !errortypes.IsNotFoundError(err) {
		return nil, err
	}
	if feedState != nil {
		status.LastPolledAt = feedState.LastPolledAt
		status.LastSuccessAt = feedState.LastSuccessAt
		status.FailureCount = feedState.FailureCount
		status.LastError = feedState.LastError
	}

	tractorMappings, err := s.repo.ListTractorMappings(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	status.TotalTractors = len(tractorMappings)
	for _, mapping := range tractorMappings {
		if mapping.ExternalID != "" {
			status.MappedTractors++
		}
	}

	workerMappings, err := s.repo.ListWorkerMappings(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	status.MappedWorkers = len(workerMappings)

	return status, nil
}

type FormSubmissionRecord struct {
	*telematics.FormSubmission
	WorkerName string
}

func (s *Service) ListShipmentFormSubmissions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) ([]*FormSubmissionRecord, error) {
	submissions, err := s.repo.ListFormSubmissions(ctx, &telematicsports.ListFormSubmissionsRequest{
		TenantInfo: tenantInfo,
		ShipmentID: shipmentID,
	})
	if err != nil {
		return nil, err
	}
	if len(submissions) == 0 {
		return []*FormSubmissionRecord{}, nil
	}

	names, err := s.workerNamesByID(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	out := make([]*FormSubmissionRecord, 0, len(submissions))
	for _, submission := range submissions {
		out = append(out, &FormSubmissionRecord{
			FormSubmission: submission,
			WorkerName:     names[submission.WorkerID],
		})
	}
	return out, nil
}
