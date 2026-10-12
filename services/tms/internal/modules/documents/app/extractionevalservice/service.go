package extractionevalservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentcontentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/app/notificationservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	_ extractionevalports.ExtractionEvalService = (*Service)(nil)
	_ extractionevalports.ExtractionEvalRunner  = (*Service)(nil)
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Cases         extractionevalports.ExtractionEvalCaseRepository
	Runs          extractionevalports.ExtractionEvalRunRepository
	Results       extractionevalports.ExtractionEvalResultRepository
	Corrections   aicorrectionports.AICorrectionRepository
	Documents     documentports.DocumentRepository
	Contents      documentcontentports.DocumentContentRepository
	Providers     agentports.AIProviderRepository
	Audit         auditports.AuditService
	Retention     tenantports.DataRetentionRepository
	Budget        agentqualityports.EvaluationBudget           `optional:"true"`
	Predictor     extractionevalports.ExtractionPredictor      `optional:"true"`
	Starter       extractionevalports.ExtractionEvalRunStarter `optional:"true"`
	Notifications *notificationservice.Service                 `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	cases       extractionevalports.ExtractionEvalCaseRepository
	runs        extractionevalports.ExtractionEvalRunRepository
	results     extractionevalports.ExtractionEvalResultRepository
	corrections aicorrectionports.AICorrectionRepository
	documents   documentports.DocumentRepository
	contents    documentcontentports.DocumentContentRepository
	providers   agentports.AIProviderRepository
	audit       auditports.AuditService
	retention   tenantports.DataRetentionRepository
	budget      agentqualityports.EvaluationBudget
	predictor   extractionevalports.ExtractionPredictor
	starter     extractionevalports.ExtractionEvalRunStarter
	notifier    driftNotifier
	now         func() int64
}

type driftNotifier interface {
	NotifyPermitted(
		ctx context.Context,
		req notificationservice.NotifyPermittedRequest,
	) (int, error)
}

func New(p Params) *Service {
	var notifier driftNotifier
	if p.Notifications != nil {
		notifier = p.Notifications
	}

	return &Service{
		l:           p.Logger.Named("service.extractioneval"),
		cases:       p.Cases,
		runs:        p.Runs,
		results:     p.Results,
		corrections: p.Corrections,
		documents:   p.Documents,
		contents:    p.Contents,
		providers:   p.Providers,
		audit:       p.Audit,
		retention:   p.Retention,
		budget:      p.Budget,
		predictor:   p.Predictor,
		starter:     p.Starter,
		notifier:    notifier,
		now:         timeutils.NowUnix,
	}
}

func AsService(s *Service) extractionevalports.ExtractionEvalService { return s }

func AsRunner(s *Service) extractionevalports.ExtractionEvalRunner { return s }

func AsDriftChecker(s *Service) extractionevalports.ExtractionDriftChecker { return s }

func (s *Service) PromoteCorrection(
	ctx context.Context,
	req *extractionevalports.PromoteCorrectionRequest,
	actor *permissionports.RequestActor,
) (*extractioneval.ExtractionCase, error) {
	correction, err := s.corrections.GetByID(ctx, aicorrectionports.GetAICorrectionRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.CorrectionID,
	})
	if err != nil {
		return nil, err
	}
	if correction.DocumentID == nil || correction.DocumentID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"This correction has no document to build a case from",
		)
	}

	existing, err := s.cases.GetBySourceCorrection(ctx, req.TenantInfo, correction.ID)
	switch {
	case err == nil:
		return nil, errortypes.NewBusinessError(
			"This correction is already in the evaluation set as {0}", existing.Title,
		)
	case !errortypes.IsNotFoundError(err):
		return nil, err
	}

	doc, err := s.documents.GetByID(ctx, documentports.GetDocumentByIDRequest{
		ID:         *correction.DocumentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, errortypes.NewBusinessError(
				"The document behind this correction has been deleted, so its text cannot be frozen",
			)
		}
		return nil, err
	}

	pages, err := s.frozenPages(ctx, doc.ID, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	status := extractioneval.CaseStatusCandidate
	if req.Activate {
		status = extractioneval.CaseStatusActive
	}
	correctionID := correction.ID
	documentID := doc.ID
	entity := &extractioneval.ExtractionCase{
		OrganizationID:      req.TenantInfo.OrgID,
		BusinessUnitID:      req.TenantInfo.BuID,
		Task:                correction.Task,
		Status:              status,
		Title:               caseTitle(req.Title, correction, doc),
		DocumentKind:        correction.DocumentKind,
		DocumentFingerprint: correction.DocumentFingerprint,
		FileName: stringutils.TruncateRunes(
			doc.OriginalName,
			extractioneval.MaxFileNameRunes,
		),
		Pages:              pages,
		Expected:           correction.Confirmed,
		SourceCorrectionID: &correctionID,
		SourceDocumentID:   &documentID,
		CreatedByID:        actorID(actor, req.TenantInfo),
	}
	entity.Normalize()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.cases.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logCase(
		actor,
		created,
		nil,
		permission.OpCreate,
		"Extraction evaluation case added from a correction",
	)

	return created, nil
}

func (s *Service) frozenPages(
	ctx context.Context,
	documentID pulid.ID,
	tenant pagination.TenantInfo,
) ([]extractioneval.Page, error) {
	stored, err := s.contents.ListPagesByDocumentID(ctx, documentID, tenant)
	if err != nil {
		return nil, fmt.Errorf("read document pages: %w", err)
	}

	pages := make([]extractioneval.Page, 0, min(len(stored), extractioneval.MaxPages))
	for _, page := range stored {
		if len(pages) >= extractioneval.MaxPages {
			break
		}
		if page == nil || strings.TrimSpace(page.ExtractedText) == "" {
			continue
		}
		pages = append(pages, frozenPage(page))
	}
	if len(pages) == 0 {
		return nil, errortypes.NewBusinessError(
			"The document has no extracted text to evaluate against; re-extract it first",
		)
	}

	return pages, nil
}

func frozenPage(page *documentcontent.Page) extractioneval.Page {
	return extractioneval.Page{
		Number: page.PageNumber,
		Text:   stringutils.TruncateRunes(page.ExtractedText, extractioneval.MaxPageRunes),
	}
}

func caseTitle(
	requested string,
	correction *aicorrection.Correction,
	doc *document.Document,
) string {
	title := strings.TrimSpace(requested)
	if title == "" {
		parts := stringutils.NonEmptyStrings(
			stringutils.HumanizeCamelCase(correction.DocumentKind),
			strings.TrimSpace(doc.OriginalName),
		)
		title = strings.Join(parts, " · ")
	}
	if title == "" {
		title = "Extraction case"
	}

	return stringutils.TruncateRunes(title, extractioneval.MaxTitleRunes)
}

func (s *Service) UpdateCase(
	ctx context.Context,
	req *extractionevalports.UpdateExtractionCaseRequest,
	actor *permissionports.RequestActor,
) (*extractioneval.ExtractionCase, error) {
	entity, err := s.cases.GetByID(ctx, extractionevalports.GetExtractionEvalCaseRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
		return nil, err
	}
	if req.Version != entity.Version {
		return nil, errortypes.NewConflictError(
			"This case was changed by someone else; reload it and try again",
		)
	}

	previous := *entity
	if req.Title != nil {
		entity.Title = *req.Title
	}
	if req.Notes != nil {
		entity.Notes = *req.Notes
	}
	if req.Status != nil && *req.Status != entity.Status {
		if !entity.Status.CanMoveTo(*req.Status) {
			return nil, errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalid,
				"A case cannot move from {0} to {1}",
				entity.Status.String(),
				req.Status.String(),
			)
		}
		entity.Status = *req.Status
	}
	entity.Normalize()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	updated, err := s.cases.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logCase(actor, updated, &previous, permission.OpUpdate, "Extraction evaluation case updated")

	return updated, nil
}

func (s *Service) DeleteCase(
	ctx context.Context,
	req extractionevalports.GetExtractionEvalCaseRequest,
	actor *permissionports.RequestActor,
) error {
	entity, err := s.cases.GetByID(ctx, req)
	if err != nil {
		return err
	}
	if err = s.cases.Delete(ctx, req); err != nil {
		return err
	}

	s.logCase(actor, entity, entity, permission.OpDelete, "Extraction evaluation case deleted")

	return nil
}

func (s *Service) GetCase(
	ctx context.Context,
	req extractionevalports.GetExtractionEvalCaseRequest,
) (*extractioneval.ExtractionCase, error) {
	return s.cases.GetByID(ctx, req)
}

func (s *Service) ListCases(
	ctx context.Context,
	req *extractionevalports.ListExtractionEvalCaseConnectionRequest,
) (*pagination.CursorListResult[*extractioneval.ExtractionCase], error) {
	return s.cases.ListConnection(ctx, req)
}

func (s *Service) GetCorrection(
	ctx context.Context,
	req aicorrectionports.GetAICorrectionRequest,
) (*aicorrection.Correction, error) {
	return s.corrections.GetByID(ctx, req)
}

func (s *Service) ListCorrections(
	ctx context.Context,
	req *aicorrectionports.ListAICorrectionConnectionRequest,
) (*pagination.CursorListResult[*aicorrection.Correction], error) {
	return s.corrections.ListConnection(ctx, req)
}

func (s *Service) logCase(
	actor *permissionports.RequestActor,
	current, previous *extractioneval.ExtractionCase,
	operation permission.Operation,
	comment string,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceAgentEvalSuite,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		CurrentState:   jsonutils.MustToJSON(auditableCase(current)),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(auditableCase(previous))
	}
	s.logAction(actor, params, comment)
}

func (s *Service) logAction(
	actor *permissionports.RequestActor,
	params *auditports.LogActionParams,
	comment string,
) {
	auditActor := actor.AuditActorOrSystem()
	params.UserID = auditActor.UserID
	params.PrincipalType = auditActor.PrincipalType
	params.PrincipalID = auditActor.PrincipalID
	params.APIKeyID = auditActor.APIKeyID

	if err := s.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log extraction evaluation audit", zap.Error(err))
	}
}

func auditableCase(entity *extractioneval.ExtractionCase) map[string]any {
	return map[string]any{
		"id":                 entity.ID,
		"status":             entity.Status,
		"title":              entity.Title,
		"notes":              entity.Notes,
		"documentKind":       entity.DocumentKind,
		"fileName":           entity.FileName,
		"pageCount":          entity.PageCount,
		"expectedFieldCount": entity.ExpectedFieldCount,
		"sourceCorrectionId": entity.SourceCorrectionID,
		"version":            entity.Version,
	}
}

func actorID(actor *permissionports.RequestActor, tenant pagination.TenantInfo) pulid.ID {
	auditActor := actor.AuditActorOrSystem()
	return pulid.FirstNotNil(auditActor.UserID, auditActor.PrincipalID, tenant.UserID)
}
