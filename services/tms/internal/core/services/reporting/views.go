package reporting

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/report"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type SaveViewRequest struct {
	Request

	ViewID       pulid.ID
	DefinitionID pulid.ID
	Name         string
	Description  string
	Params       map[string]any
	Shared       bool
	Pinned       bool
	Format       report.Format
	Version      int64
}

type GetViewRequest struct {
	Request

	ViewID pulid.ID
}

type ListViewsRequest struct {
	Request

	DefinitionID pulid.ID
	Limit        int
	Offset       int
}

// validateViewParams rejects anything the report does not declare. A view is a
// set of answers to a report's questions, so an unknown key is either a stale
// preset left behind by a definition edit or an attempt to smuggle a value the
// compiler never validated — neither should be stored.
func (s *Service) validateViewParams(ctx context.Context, req *SaveViewRequest) error {
	definition, err := s.GetDefinition(ctx, &GetDefinitionRequest{
		Request:      req.Request,
		DefinitionID: req.DefinitionID,
	})
	if err != nil {
		return err
	}

	if len(req.Params) == 0 {
		return nil
	}

	multiErr := errortypes.NewMultiError()
	for name := range req.Params {
		if _, ok := definition.Definition.Parameter(name); !ok {
			multiErr.Add("params."+name, errortypes.ErrInvalid,
				"This report has no parameter by that name")
		}
	}
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) CreateView(
	ctx context.Context,
	req *SaveViewRequest,
) (*report.ReportView, error) {
	if err := s.validateViewParams(ctx, req); err != nil {
		return nil, err
	}

	entity := &report.ReportView{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		DefinitionID:   req.DefinitionID,
		OwnerID:        req.TenantInfo.UserID,
		Name:           req.Name,
		Description:    req.Description,
		Params:         req.Params,
		Shared:         req.Shared,
		Pinned:         req.Pinned,
		Format:         req.Format,
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.viewRepo.Create(ctx, entity)
	if err != nil {
		s.l.Error("failed to create report view", zap.Error(err))
		return nil, err
	}

	return created, nil
}

func (s *Service) UpdateView(
	ctx context.Context,
	req *SaveViewRequest,
) (*report.ReportView, error) {
	existing, err := s.viewRepo.GetByID(ctx, &repositories.GetReportViewRequest{
		TenantInfo: req.TenantInfo,
		ViewID:     req.ViewID,
	})
	if err != nil {
		return nil, err
	}
	if existing.OwnerID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError(
			"Only the view owner can modify this view",
		)
	}

	// The report a view answers is fixed at creation: repointing it would keep
	// the name and silently change what the numbers mean.
	req.DefinitionID = existing.DefinitionID
	if err = s.validateViewParams(ctx, req); err != nil {
		return nil, err
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.Params = req.Params
	existing.Shared = req.Shared
	existing.Pinned = req.Pinned
	existing.Format = req.Format
	existing.Version = req.Version

	multiErr := errortypes.NewMultiError()
	existing.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.viewRepo.Update(ctx, existing)
}

func (s *Service) GetView(
	ctx context.Context,
	req *GetViewRequest,
) (*report.ReportView, error) {
	view, err := s.viewRepo.GetByID(ctx, &repositories.GetReportViewRequest{
		TenantInfo: req.TenantInfo,
		ViewID:     req.ViewID,
	})
	if err != nil {
		return nil, err
	}

	// A private view belongs to its owner alone, and reading the report it
	// points at is a separate question the definition check answers.
	if !view.Shared && view.OwnerID != req.TenantInfo.UserID {
		return nil, errortypes.NewAuthorizationError("You do not have access to this view")
	}
	if _, err = s.GetDefinition(ctx, &GetDefinitionRequest{
		Request:      req.Request,
		DefinitionID: view.DefinitionID,
	}); err != nil {
		return nil, err
	}

	return view, nil
}

func (s *Service) ListViews(
	ctx context.Context,
	req *ListViewsRequest,
) ([]*report.ReportView, error) {
	if !req.DefinitionID.IsNil() {
		if _, err := s.GetDefinition(ctx, &GetDefinitionRequest{
			Request:      req.Request,
			DefinitionID: req.DefinitionID,
		}); err != nil {
			return nil, err
		}
	}

	return s.viewRepo.List(ctx, &repositories.ListReportViewsRequest{
		TenantInfo:   req.TenantInfo,
		DefinitionID: req.DefinitionID,
		VisibleToID:  req.TenantInfo.UserID,
		Limit:        req.Limit,
		Offset:       req.Offset,
	})
}

func (s *Service) DeleteView(ctx context.Context, req *GetViewRequest) error {
	existing, err := s.viewRepo.GetByID(ctx, &repositories.GetReportViewRequest{
		TenantInfo: req.TenantInfo,
		ViewID:     req.ViewID,
	})
	if err != nil {
		return err
	}
	if existing.OwnerID != req.TenantInfo.UserID {
		return errortypes.NewAuthorizationError("Only the view owner can delete this view")
	}

	return s.viewRepo.Delete(ctx, &repositories.GetReportViewRequest{
		TenantInfo: req.TenantInfo,
		ViewID:     req.ViewID,
	})
}

// RecordViewRun stamps usage after a run is queued from a view. It is
// best-effort by design: losing the stamp costs nothing but picker ordering,
// and must never fail the run the caller actually asked for.
func (s *Service) RecordViewRun(ctx context.Context, req *GetViewRequest) {
	if req.ViewID.IsNil() {
		return
	}
	err := s.viewRepo.RecordRun(ctx, &repositories.RecordReportViewRunRequest{
		TenantInfo: req.TenantInfo,
		ViewID:     req.ViewID,
		RanAt:      timeutils.NowUnix(),
	})
	if err != nil {
		s.l.Warn("failed to record report view run",
			zap.String("viewId", req.ViewID.String()), zap.Error(err))
	}
}
