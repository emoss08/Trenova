package workersafetyservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/safetyscorecard"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerdiscipline"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workersafety"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workersafetyports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type EvidenceRequest struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	Discipline bool
	Now        int64
}

type Evidence struct {
	Scorecard *safetyscorecard.SafetyScorecard
	Events    []*workersafety.WorkerSafetyEvent
}

func (s *Service) Evidence(ctx context.Context, req *EvidenceRequest) (*Evidence, error) {
	var actions []*workerdiscipline.WorkerDisciplinaryAction
	if req.Discipline {
		var err error
		actions, err = s.repo.ListActions(
			ctx,
			&workersafetyports.ListWorkerDisciplinaryActionsRequest{
				TenantInfo: req.TenantInfo,
				WorkerID:   req.WorkerID,
			},
		)
		if err != nil {
			return nil, err
		}
	}
	return s.evidence(ctx, req.TenantInfo, req.WorkerID, actions, req.Now)
}

func (s *Service) evidence(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	workerID pulid.ID,
	actions []*workerdiscipline.WorkerDisciplinaryAction,
	now int64,
) (*Evidence, error) {
	events, err := s.repo.ListEvents(ctx, &workersafetyports.ListWorkerSafetyEventsRequest{
		TenantInfo: tenantInfo,
		WorkerID:   workerID,
	})
	if err != nil {
		return nil, err
	}
	recognitions, err := s.repo.ListRecognitions(
		ctx,
		&workersafetyports.ListWorkerRecognitionsRequest{
			TenantInfo: tenantInfo,
			WorkerID:   workerID,
		},
	)
	if err != nil {
		return nil, err
	}
	return &Evidence{
		Scorecard: safetyscorecard.BuildSafetyScorecard(
			workerID,
			events,
			actions,
			recognitions,
			now,
		),
		Events: events,
	}, nil
}
