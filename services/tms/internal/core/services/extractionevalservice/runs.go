package extractionevalservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const canceledReason = "Canceled before every case ran"

var errRunnerUnavailable = errors.New("extraction evaluation runner is not configured")

func (s *Service) StartRun(
	ctx context.Context,
	req *services.StartExtractionEvalRunRequest,
	actor *services.RequestActor,
) (*extractioneval.ExtractionRun, error) {
	if s.starter == nil {
		return nil, errortypes.NewBusinessError(
			"Extraction evaluations cannot run on this installation",
		).WithInternal(errRunnerUnavailable)
	}

	provider, err := s.evaluatedProvider(ctx, req)
	if err != nil {
		return nil, err
	}

	limit := req.CaseLimit
	if limit <= 0 {
		limit = extractioneval.DefaultCaseLimit
	}
	run := &extractioneval.ExtractionRun{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Task:           aicorrection.TaskShipmentDraftExtraction,
		Status:         extractioneval.RunStatusQueued,
		ProviderID:     provider.ID,
		ProviderName:   stringutils.TruncateRunes(provider.Name, extractioneval.MaxProviderNameRunes),
		ProviderModel:  stringutils.TruncateRunes(provider.Model, extractioneval.MaxModelRunes),
		CaseLimit:      limit,
		RequestedByID:  actorID(actor, req.TenantInfo),
	}

	multiErr := errortypes.NewMultiError()
	run.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	cases, err := s.cases.ListActive(ctx, repositories.ListActiveExtractionEvalCasesRequest{
		TenantInfo: req.TenantInfo,
		Task:       run.Task,
		Limit:      limit,
	})
	if err != nil {
		return nil, err
	}
	if len(cases) == 0 {
		return nil, errortypes.NewBusinessError(
			"There are no active cases to evaluate; activate a case in the evaluation set first",
		)
	}

	results := make([]*extractioneval.ExtractionResult, 0, len(cases))
	for idx, item := range cases {
		results = append(results, &extractioneval.ExtractionResult{
			CaseID:    item.ID,
			CaseTitle: item.Title,
			Ordinal:   idx + 1,
			Status:    extractioneval.ResultStatusPending,
		})
	}
	run.CasesTotal = len(results)

	created, err := s.runs.Create(ctx, run, results)
	if err != nil {
		return nil, err
	}

	workflowID, err := s.starter.StartExtractionEvalRun(ctx, &services.ExtractionEvalRunStart{
		TenantInfo: req.TenantInfo,
		RunID:      created.ID,
	})
	if err != nil {
		s.l.Error("failed to start extraction evaluation run", zap.Error(err))
		if failErr := s.FailRun(
			ctx,
			repositories.GetExtractionEvalRunRequest{TenantInfo: req.TenantInfo, ID: created.ID},
			"The evaluation could not be started",
		); failErr != nil {
			s.l.Error("failed to mark unstarted extraction evaluation run", zap.Error(failErr))
		}

		return nil, fmt.Errorf("start extraction evaluation run: %w", err)
	}

	created.WorkflowID = workflowID
	updated, err := s.runs.Update(ctx, created)
	if err != nil {
		return nil, err
	}

	s.logRun(actor, updated, permission.OpCreate, "Extraction evaluation started")

	return updated, nil
}

func (s *Service) evaluatedProvider(
	ctx context.Context,
	req *services.StartExtractionEvalRunRequest,
) (*aiprovider.Provider, error) {
	if req.ProviderID.IsNil() {
		return nil, errortypes.NewValidationError(
			"providerId", errortypes.ErrRequired, "Choose the AI provider to evaluate",
		)
	}

	provider, err := s.providers.GetByID(ctx, repositories.GetAIProviderByIDRequest{
		ID:         req.ProviderID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if ok, reason := provider.CanServeTask(aiprovider.TaskDocumentExtraction); !ok {
		return nil, errortypes.NewValidationError(
			"providerId",
			errortypes.ErrInvalid,
			"{0} cannot run document extraction: {1}",
			provider.Name,
			reason,
		)
	}

	return provider, nil
}

func (s *Service) CancelRun(
	ctx context.Context,
	req repositories.GetExtractionEvalRunRequest,
	actor *services.RequestActor,
) (*extractioneval.ExtractionRun, error) {
	run, err := s.runs.GetByID(ctx, req)
	if err != nil {
		return nil, err
	}
	if !run.Status.IsActive() {
		return nil, errortypes.NewBusinessError("This evaluation has already finished")
	}

	finished, err := s.finish(ctx, run, extractioneval.RunStatusCanceled, canceledReason)
	if err != nil {
		return nil, err
	}

	s.logRun(actor, finished, permission.OpUpdate, "Extraction evaluation canceled")

	return finished, nil
}

func (s *Service) GetRun(
	ctx context.Context,
	req repositories.GetExtractionEvalRunRequest,
) (*extractioneval.ExtractionRun, error) {
	return s.runs.GetByID(ctx, req)
}

func (s *Service) ListRuns(
	ctx context.Context,
	req *repositories.ListExtractionEvalRunConnectionRequest,
) (*pagination.CursorListResult[*extractioneval.ExtractionRun], error) {
	return s.runs.ListConnection(ctx, req)
}

func (s *Service) ListResults(
	ctx context.Context,
	req *repositories.ListExtractionEvalResultConnectionRequest,
) (*pagination.CursorListResult[*extractioneval.ExtractionResult], error) {
	if _, err := s.runs.GetByID(ctx, repositories.GetExtractionEvalRunRequest{
		TenantInfo: req.Filter.TenantInfo,
		ID:         req.RunID,
	}); err != nil {
		return nil, err
	}

	return s.results.ListConnection(ctx, req)
}

func (s *Service) GetResult(
	ctx context.Context,
	req repositories.GetExtractionEvalResultRequest,
) (*extractioneval.ExtractionResult, error) {
	return s.results.GetByID(ctx, req)
}

func (s *Service) logRun(
	actor *services.RequestActor,
	run *extractioneval.ExtractionRun,
	operation permission.Operation,
	comment string,
) {
	s.logAction(actor, &services.LogActionParams{
		Resource:   permission.ResourceAgentEvalSuite,
		ResourceID: run.ID.String(),
		Operation:  operation,
		CurrentState: jsonutils.MustToJSON(map[string]any{
			"id":           run.ID,
			"status":       run.Status,
			"providerId":   run.ProviderID,
			"providerName": run.ProviderName,
			"model":        run.ProviderModel,
			"caseLimit":    run.CaseLimit,
			"casesTotal":   run.CasesTotal,
		}),
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
	}, comment)
}
