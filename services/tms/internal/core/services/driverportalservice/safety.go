package driverportalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// MySafetyScorecard is the driver's own record, the same numbers the office
// sees so there is nothing to argue about.
func (s *Service) MySafetyScorecard(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*worker.SafetyScorecard, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	return s.safety.Scorecard(ctx, tenantInfo, wrk.ID)
}

// MyRecognitions lists the praise the office chose to share with the driver.
func (s *Service) MyRecognitions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.WorkerRecognition, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	return s.safety.ListRecognitions(ctx, tenantInfo, wrk.ID, true)
}

// MyDisciplinaryActions lists the driver's own actions, rescinded ones
// included so the history reads honestly.
func (s *Service) MyDisciplinaryActions(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*worker.WorkerDisciplinaryAction, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	return s.safety.ListActions(ctx, tenantInfo, wrk.ID)
}

// AcknowledgeMyDisciplinaryAction is the driver confirming they have read an
// action, with an optional response kept on the record.
func (s *Service) AcknowledgeMyDisciplinaryAction(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	actionID pulid.ID,
	comment string,
) (*worker.WorkerDisciplinaryAction, error) {
	wrk, err := s.ResolveWorker(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	saved, err := s.safety.AcknowledgeAction(ctx, tenantInfo, actionID, wrk.ID, comment)
	if err != nil {
		return nil, err
	}
	s.notifyDispatch(
		ctx,
		tenantInfo,
		"disciplinary_acknowledged",
		"Disciplinary action acknowledged",
		wrk.FirstName+" "+wrk.LastName+" acknowledged their "+saved.Level.String()+".",
		"/hr/workers?tab=safety",
		map[string]any{"workerId": wrk.ID.String(), "disciplinaryActionId": saved.ID.String()},
	)
	return saved, nil
}
