package reporting

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/reportports"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type DefinitionChange struct {
	Before *report.ReportDefinition
	After  *report.ReportDefinition
}

type RunChange struct {
	Before *report.ReportRun
	After  *report.ReportRun
}

func (s *Service) PlanDeleteDefinition(
	ctx context.Context,
	req *GetDefinitionRequest,
) (*report.ReportDefinition, error) {
	existing, err := s.defRepo.GetByID(ctx, &reportports.GetReportDefinitionRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: req.DefinitionID,
	})
	if err != nil {
		return nil, err
	}
	if existing.OwnerID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError("Only the report owner can delete this report")
	}

	return existing, nil
}

func (s *Service) PlanResetCannedFork(
	ctx context.Context,
	req *GetDefinitionRequest,
) (*DefinitionChange, error) {
	existing, err := s.defRepo.GetByID(ctx, &reportports.GetReportDefinitionRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: req.DefinitionID,
	})
	if err != nil {
		return nil, err
	}
	if existing.OwnerID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the report owner can reset this report",
		)
	}
	if existing.Kind != report.DefinitionKindCannedFork || existing.CannedKey == "" {
		return nil, errortypes.NewBusinessError(
			"Only reports customized from a canned report can be reset",
		)
	}

	entry, err := s.GetCanned(existing.CannedKey)
	if err != nil {
		return nil, err
	}

	reset := *existing
	reset.Definition = entry.Definition
	reset.CannedVersion = entry.Version
	reset.CatalogVersion = s.CatalogVersion()
	reset.Status = report.DefinitionStatusActive
	reset.Diagnostics = nil

	return &DefinitionChange{Before: existing, After: &reset}, nil
}

func (s *Service) PlanCancelRun(ctx context.Context, req *GetRunRequest) (*RunChange, error) {
	run, err := s.GetRun(ctx, req)
	if err != nil {
		return nil, err
	}

	if run.Status.IsTerminal() {
		return nil, errortypes.NewBusinessError("This report run has already finished")
	}

	canceled := *run
	canceled.Status = report.RunStatusCanceled
	canceled.Error = &report.RunError{Code: "CANCELED", Message: "The run was canceled"}

	return &RunChange{Before: run, After: &canceled}, nil
}
