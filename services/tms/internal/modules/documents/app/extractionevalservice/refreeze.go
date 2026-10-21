package extractionevalservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	freezeReasonNoSourceDocument = "The case has no original document to read again"
	freezeReasonCaseGone         = "The case was deleted before it was frozen again"
)

var (
	errRefreezeUnavailable = errors.New("extraction evaluation re-freeze is not configured")
	errConverterMissing    = errors.New("extraction case converter is not configured")
)

func (s *Service) RefreezeCases(
	ctx context.Context,
	req *extractionevalports.RefreezeExtractionEvalCasesRequest,
	actor *permissionports.RequestActor,
) (*extractionevalports.RefreezeExtractionEvalCasesResult, error) {
	if s.refreezes == nil {
		return nil, errortypes.NewBusinessError(
			"Extraction evaluations cannot run on this installation",
		).WithInternal(errRefreezeUnavailable)
	}

	cases, err := s.refreezeTargets(ctx, req)
	if err != nil {
		return nil, err
	}

	caseIDs := make([]pulid.ID, 0, len(cases))
	for _, item := range cases {
		if item.SourceDocumentID != nil && item.SourceDocumentID.IsNotNil() {
			caseIDs = append(caseIDs, item.ID)
		}
	}
	if len(caseIDs) == 0 {
		return nil, errortypes.NewBusinessError(
			"None of these cases has an original document to read again",
		)
	}

	tenant := req.TenantInfo
	tenant.UserID = actorID(actor, req.TenantInfo)
	started, err := s.refreezes.StartExtractionEvalRefreeze(
		ctx,
		&extractionevalports.ExtractionEvalRefreezeStart{
			TenantInfo: tenant,
			CaseIDs:    caseIDs,
		},
	)
	if err != nil {
		s.l.Error("failed to start extraction evaluation re-freeze", zap.Error(err))
		return nil, fmt.Errorf("start extraction evaluation re-freeze: %w", err)
	}
	if started.AlreadyRunning {
		return s.runningRefreeze(ctx, req.TenantInfo, started.WorkflowID), nil
	}

	s.logAction(actor, &auditports.LogActionParams{
		Resource:   permission.ResourceAgentEvalSuite,
		ResourceID: started.WorkflowID,
		Operation:  permission.OpUpdate,
		CurrentState: jsonutils.MustToJSON(
			map[string]any{"caseIds": caseIDs},
		),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
	}, "Extraction evaluation cases queued to be frozen again through Docling")

	return &extractionevalports.RefreezeExtractionEvalCasesResult{
		Queued:     len(caseIDs),
		WorkflowID: started.WorkflowID,
	}, nil
}

func (s *Service) runningRefreeze(
	ctx context.Context,
	tenant pagination.TenantInfo,
	workflowID string,
) *extractionevalports.RefreezeExtractionEvalCasesResult {
	result := &extractionevalports.RefreezeExtractionEvalCasesResult{
		WorkflowID:     workflowID,
		AlreadyRunning: true,
	}
	progress, err := s.refreezes.ExtractionEvalRefreezeProgress(ctx, tenant)
	switch {
	case err != nil:
		s.l.Warn("failed to read the running extraction evaluation re-freeze", zap.Error(err))
	case progress != nil:
		result.Queued = progress.Total
	}

	return result
}

func (s *Service) RefreezeProgress(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*extractionevalports.ExtractionEvalRefreezeProgress, error) {
	if s.refreezes == nil {
		return nil, nil //nolint:nilnil // no re-freeze can have run on this installation
	}

	return s.refreezes.ExtractionEvalRefreezeProgress(ctx, tenant)
}

func (s *Service) refreezeTargets(
	ctx context.Context,
	req *extractionevalports.RefreezeExtractionEvalCasesRequest,
) ([]*extractioneval.ExtractionCase, error) {
	if len(req.CaseIDs) == 0 {
		return s.cases.ListActive(ctx, extractionevalports.ListActiveExtractionEvalCasesRequest{
			TenantInfo: req.TenantInfo,
			Task:       aicorrection.TaskShipmentDraftExtraction,
			Limit:      extractioneval.MaxCaseLimit,
		})
	}
	if len(req.CaseIDs) > extractioneval.MaxCaseLimit {
		return nil, errortypes.NewValidationError(
			"caseIds",
			errortypes.ErrInvalid,
			"Freeze at most {0} cases at once",
			extractioneval.MaxCaseLimit,
		)
	}

	seen := make(map[pulid.ID]struct{}, len(req.CaseIDs))
	cases := make([]*extractioneval.ExtractionCase, 0, len(req.CaseIDs))
	for _, id := range req.CaseIDs {
		if _, duplicate := seen[id]; duplicate {
			continue
		}
		seen[id] = struct{}{}

		item, err := s.cases.GetByID(ctx, extractionevalports.GetExtractionEvalCaseRequest{
			TenantInfo: req.TenantInfo,
			ID:         id,
		})
		if err != nil {
			return nil, err
		}
		cases = append(cases, item)
	}

	return cases, nil
}

func (s *Service) FreezeDoclingPages(
	ctx context.Context,
	req *extractionevalports.FreezeCasePagesRequest,
) (*extractionevalports.FreezeCasePagesResult, error) {
	evalCase, err := s.cases.GetByID(ctx, extractionevalports.GetExtractionEvalCaseRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.CaseID,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return &extractionevalports.FreezeCasePagesResult{Reason: freezeReasonCaseGone}, nil
		}
		return nil, err
	}
	if evalCase.SourceDocumentID == nil || evalCase.SourceDocumentID.IsNil() {
		return &extractionevalports.FreezeCasePagesResult{Reason: freezeReasonNoSourceDocument}, nil
	}
	if s.converter == nil {
		return nil, errortypes.NewBusinessError(
			"Extraction evaluations cannot run on this installation",
		).WithInternal(errConverterMissing)
	}

	converted, err := s.converter.ConvertCaseDocument(
		ctx,
		req.TenantInfo,
		*evalCase.SourceDocumentID,
	)
	switch {
	case errors.Is(err, kernelports.ErrDocumentNotConvertible), errortypes.IsNotFoundError(err):
		return &extractionevalports.FreezeCasePagesResult{
			Reason: stringutils.TruncateRunes(err.Error(), extractioneval.MaxFailureRunes),
		}, nil
	case err != nil:
		return nil, err
	}

	pages := frozenCasePages(converted.Pages)
	frozenBy := pulid.FirstNotNil(req.TenantInfo.UserID, evalCase.CreatedByID)
	entity := extractioneval.NewCasePages(
		evalCase,
		pages,
		converted.ConverterVersion,
		frozenBy,
	)

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return &extractionevalports.FreezeCasePagesResult{Reason: multiErr.Error()}, nil
	}

	if _, err = s.casePages.Upsert(ctx, entity); err != nil {
		return nil, err
	}

	return &extractionevalports.FreezeCasePagesResult{Frozen: true}, nil
}

func frozenCasePages(pages []extractioneval.Page) []extractioneval.Page {
	out := make([]extractioneval.Page, 0, min(len(pages), extractioneval.MaxPages))
	for _, page := range pages {
		if len(out) == extractioneval.MaxPages {
			break
		}
		page.Text = stringutils.TruncateRunes(page.Text, extractioneval.MaxPageRunes)
		out = append(out, page)
	}

	return out
}
