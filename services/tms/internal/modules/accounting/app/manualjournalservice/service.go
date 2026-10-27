//nolint:funlen,gosec // existing legacy workflow/API shape is intentionally kept stable
package manualjournalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/app/journalposting"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/manualjournal"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/manualjournalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	entryTypeStandard  = "Standard"
	entryTypeAdjusting = "Adjusting"
	entryStatusPosted  = "Posted"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	DB             kernelports.Transactor
	Repo           manualjournalports.ManualJournalRepository
	JournalRepo    journalentryports.JournalPostingRepository
	AccountingRepo tenantports.AccountingControlRepository
	Generator      seqgen.Generator
	Validator      *Validator
	AuditService   auditports.AuditService
}

type Service struct {
	l              *zap.Logger
	db             kernelports.Transactor
	repo           manualjournalports.ManualJournalRepository
	journalRepo    journalentryports.JournalPostingRepository
	accountingRepo tenantports.AccountingControlRepository
	generator      seqgen.Generator
	validator      *Validator
	auditService   auditports.AuditService
}

type postingPlan struct {
	change      *manualjournalports.ManualJournalChange
	control     *tenant.AccountingControl
	period      *fiscalperiod.FiscalPeriod
	postingDate int64
	entryType   string
	userID      pulid.ID
}

func New(p Params) *Service { //nolint:gocritic // stable API shape
	return &Service{
		l:              p.Logger.Named("service.manual-journal"),
		db:             p.DB,
		repo:           p.Repo,
		journalRepo:    p.JournalRepo,
		accountingRepo: p.AccountingRepo,
		generator:      p.Generator,
		validator:      p.Validator,
		auditService:   p.AuditService,
	}
}

func (s *Service) List(
	ctx context.Context,
	req *manualjournalports.ListManualJournalRequest,
) (*pagination.ListResult[*manualjournal.Request], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *manualjournalports.ListManualJournalConnectionRequest,
) (*pagination.CursorListResult[*manualjournal.Request], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *manualjournalports.GetManualJournalRequest,
) (*manualjournal.Request, error) {
	return s.repo.GetByID(
		ctx,
		manualjournalports.GetManualJournalByIDRequest{
			ID:         req.RequestID,
			TenantInfo: req.TenantInfo,
		},
	)
}

func (s *Service) PlanCreateDraft(
	ctx context.Context,
	req *manualjournalports.CreateManualJournalRequest,
	actor *permissionports.RequestActor,
) (*manualjournal.Request, error) {
	userID, err := requireManualJournalUser(actor)
	if err != nil {
		return nil, err
	}

	accountingControl, err := s.accountingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}

	entity := &manualjournal.Request{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Status:         manualjournal.StatusDraft,
		Description:    req.Description,
		Reason:         req.Reason,
		AccountingDate: req.AccountingDate,
		CurrencyCode:   req.CurrencyCode,
		CreatedByID:    userID,
		UpdatedByID:    userID,
		Lines:          mapLines(req.Lines),
	}

	if multiErr := s.validator.ValidateDraftUpsert(
		ctx,
		entity,
		accountingControl,
	); multiErr != nil {
		return nil, multiErr
	}
	entity.SyncTotals()

	return entity, nil
}

func (s *Service) CreateDraft(
	ctx context.Context,
	req *manualjournalports.CreateManualJournalRequest,
	actor *permissionports.RequestActor,
) (*manualjournal.Request, error) {
	entity, err := s.PlanCreateDraft(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	if entity.RequestNumber, err = s.generator.GenerateManualJournalRequestNumber(
		ctx,
		req.TenantInfo.OrgID,
		req.TenantInfo.BuID,
		"",
		"",
	); err != nil {
		return nil, err
	}

	var created *manualjournal.Request
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		created, err = s.repo.Create(txCtx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(
		permission.OpCreate,
		created,
		nil,
		entity.CreatedByID,
		"Manual journal draft created",
	)
	return created, nil
}

func (s *Service) PlanUpdateDraft(
	ctx context.Context,
	req *manualjournalports.UpdateManualJournalDraftRequest,
	actor *permissionports.RequestActor,
) (*manualjournalports.ManualJournalChange, error) {
	userID, err := requireManualJournalUser(actor)
	if err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(
		ctx,
		manualjournalports.GetManualJournalByIDRequest{
			ID:         req.RequestID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if !original.Status.IsEditable() {
		return nil, errortypes.NewBusinessError("Only draft manual journals can be edited")
	}

	accountingControl, err := s.accountingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}

	updatedEntity := cloneRequest(original)
	updatedEntity.Description = req.Description
	updatedEntity.Reason = req.Reason
	updatedEntity.AccountingDate = req.AccountingDate
	updatedEntity.CurrencyCode = req.CurrencyCode
	updatedEntity.UpdatedByID = userID
	updatedEntity.Lines = mapLines(req.Lines)

	if multiErr := s.validator.ValidateDraftUpsert(
		ctx,
		updatedEntity,
		accountingControl,
	); multiErr != nil {
		return nil, multiErr
	}
	updatedEntity.SyncTotals()

	return &manualjournalports.ManualJournalChange{Before: original, After: updatedEntity}, nil
}

func (s *Service) UpdateDraft(
	ctx context.Context,
	req *manualjournalports.UpdateManualJournalDraftRequest,
	actor *permissionports.RequestActor,
) (*manualjournal.Request, error) {
	change, err := s.PlanUpdateDraft(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	var updated *manualjournal.Request
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		updated, err = s.repo.Update(txCtx, change.After)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(
		permission.OpUpdate,
		updated,
		change.Before,
		change.After.UpdatedByID,
		"Manual journal draft updated",
	)
	return updated, nil
}

func (s *Service) PlanSubmit(
	ctx context.Context,
	req *manualjournalports.GetManualJournalRequest,
	actor *permissionports.RequestActor,
) (*manualjournalports.ManualJournalChange, error) {
	userID, err := requireManualJournalUser(actor)
	if err != nil {
		return nil, err
	}

	entity, accountingControl, err := s.loadRequestWithAccountingControl(
		ctx,
		req.RequestID,
		req.TenantInfo,
	)
	if err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidateSubmit(entity, accountingControl); multiErr != nil {
		return nil, multiErr
	}

	after := cloneRequest(entity)
	after.UpdatedByID = userID
	if accountingControl.RequireManualJEApproval {
		after.Status = manualjournal.StatusPendingApproval
		after.ApprovedAt = nil
		after.ApprovedByID = pulid.Nil
	} else {
		now := timeutils.NowUnix()
		after.Status = manualjournal.StatusApproved
		after.ApprovedAt = &now
		after.ApprovedByID = userID
	}

	return &manualjournalports.ManualJournalChange{Before: entity, After: after}, nil
}

func (s *Service) Submit(
	ctx context.Context,
	req *manualjournalports.GetManualJournalRequest,
	actor *permissionports.RequestActor,
) (*manualjournal.Request, error) {
	change, err := s.PlanSubmit(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	return s.saveTransition(ctx, change, permission.OpSubmit, "Manual journal submitted")
}

func (s *Service) Approve(
	ctx context.Context,
	req *manualjournalports.GetManualJournalRequest,
	actor *permissionports.RequestActor,
) (*manualjournal.Request, error) {
	userID, err := requireManualJournalUser(actor)
	if err != nil {
		return nil, err
	}

	entity, err := s.repo.GetByID(
		ctx,
		manualjournalports.GetManualJournalByIDRequest{
			ID:         req.RequestID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidateApprove(entity); multiErr != nil {
		return nil, multiErr
	}

	after := cloneRequest(entity)
	now := timeutils.NowUnix()
	after.Status = manualjournal.StatusApproved
	after.ApprovedAt = &now
	after.ApprovedByID = userID
	after.UpdatedByID = userID

	return s.saveTransition(
		ctx,
		&manualjournalports.ManualJournalChange{Before: entity, After: after},
		permission.OpApprove,
		"Manual journal approved",
	)
}

func (s *Service) PlanPost(
	ctx context.Context,
	req *manualjournalports.GetManualJournalRequest,
	actor *permissionports.RequestActor,
) (*manualjournalports.ManualJournalChange, error) {
	plan, err := s.planPost(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	return plan.change, nil
}

func (s *Service) planPost(
	ctx context.Context,
	req *manualjournalports.GetManualJournalRequest,
	actor *permissionports.RequestActor,
) (*postingPlan, error) {
	userID, err := requireManualJournalUser(actor)
	if err != nil {
		return nil, err
	}

	entity, accountingControl, err := s.loadRequestWithAccountingControl(
		ctx,
		req.RequestID,
		req.TenantInfo,
	)
	if err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidatePost(entity); multiErr != nil {
		return nil, multiErr
	}

	postingPeriod, postingDate, err := s.resolvePostingPeriod(ctx, entity, accountingControl)
	if err != nil {
		return nil, err
	}

	entryType := entryTypeStandard
	if postingPeriod.PeriodType == fiscalperiod.PeriodTypeAdjusting {
		entryType = entryTypeAdjusting
	}

	after := cloneRequest(entity)
	after.Status = manualjournal.StatusPosted
	after.UpdatedByID = userID

	return &postingPlan{
		change: &manualjournalports.ManualJournalChange{
			Before:  entity,
			After:   after,
			Journal: journalPreview(entity, postingPeriod.ID, postingDate),
		},
		control:     accountingControl,
		period:      postingPeriod,
		postingDate: postingDate,
		entryType:   entryType,
		userID:      userID,
	}, nil
}

//nolint:govet // existing scoped variable reuse is local and behavior-preserving
func (s *Service) Post(
	ctx context.Context,
	req *manualjournalports.GetManualJournalRequest,
	actor *permissionports.RequestActor,
) (*manualjournal.Request, error) {
	plan, err := s.planPost(ctx, req, actor)
	if err != nil {
		return nil, err
	}
	entity := plan.change.After

	batchNumber, err := s.generator.GenerateJournalBatchNumber(
		ctx,
		entity.OrganizationID,
		entity.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}
	entryNumber, err := s.generator.GenerateJournalEntryNumber(
		ctx,
		entity.OrganizationID,
		entity.BusinessUnitID,
		"",
		"",
	)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	batchID := pulid.MustNew("jb_")
	entryID := pulid.MustNew("je_")

	lines := make([]journalentryports.JournalPostingLine, 0, len(entity.Lines))
	for idx, line := range entity.Lines {
		if line == nil {
			continue
		}
		lines = append(lines, journalentryports.JournalPostingLine{
			ID:           pulid.MustNew("jel_"),
			GLAccountID:  line.GLAccountID,
			LineNumber:   int16(idx + 1),
			DebitAmount:  line.DebitAmount,
			CreditAmount: line.CreditAmount,
			NetAmount:    line.DebitAmount - line.CreditAmount,
			CustomerID:   line.CustomerID,
			LocationID:   line.LocationID,
		})
	}

	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		if err := s.journalRepo.CreatePosting(txCtx, journalentryports.CreateJournalPostingParams{
			BatchID:              batchID,
			OrganizationID:       entity.OrganizationID,
			BusinessUnitID:       entity.BusinessUnitID,
			BatchNumber:          batchNumber,
			BatchType:            "Manual",
			BatchStatus:          entryStatusPosted,
			BatchDescription:     entity.Description,
			FiscalYearID:         plan.period.FiscalYearID,
			FiscalPeriodID:       plan.period.ID,
			AccountingDate:       plan.postingDate,
			PostedAt:             &now,
			PostedByID:           plan.userID,
			CreatedByID:          entity.CreatedByID,
			UpdatedByID:          plan.userID,
			EntryID:              entryID,
			EntryNumber:          entryNumber,
			EntryType:            plan.entryType,
			EntryStatus:          entryStatusPosted,
			ReferenceNumber:      entity.RequestNumber,
			ReferenceType:        "ManualJournalRequest",
			ReferenceID:          entity.ID.String(),
			EntryDescription:     entity.Description,
			TotalDebit:           entity.TotalDebit,
			TotalCredit:          entity.TotalCredit,
			IsPosted:             true,
			IsAutoGenerated:      false,
			RequiresApproval:     plan.control.RequireManualJEApproval,
			IsApproved:           true,
			ApprovedByID:         entity.ApprovedByID,
			ApprovedAt:           entity.ApprovedAt,
			SourceID:             pulid.MustNew("jsrc_"),
			SourceObjectType:     "ManualJournalRequest",
			SourceObjectID:       entity.ID.String(),
			SourceEventType:      "ManualJournalPosted",
			SourceStatus:         entryStatusPosted,
			SourceDocumentNumber: entity.RequestNumber,
			SourceIdempotencyKey: "manual-journal-posted:" + entity.ID.String(),
			Lines:                lines,
		}); err != nil {
			return err
		}

		entity.PostedBatchID = batchID
		entity.Lines = nil

		updated, updateErr := s.repo.Update(txCtx, entity)
		if updateErr != nil {
			return updateErr
		}
		entity = updated
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(
		permission.OpApprove,
		entity,
		plan.change.Before,
		plan.userID,
		"Manual journal posted",
	)
	return entity, nil
}

func (s *Service) Reject(
	ctx context.Context,
	req *manualjournalports.RejectManualJournalRequest,
	actor *permissionports.RequestActor,
) (*manualjournal.Request, error) {
	userID, err := requireManualJournalUser(actor)
	if err != nil {
		return nil, err
	}

	entity, err := s.repo.GetByID(
		ctx,
		manualjournalports.GetManualJournalByIDRequest{
			ID:         req.RequestID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidateReject(req.Reason, entity); multiErr != nil {
		return nil, multiErr
	}

	after := cloneRequest(entity)
	now := timeutils.NowUnix()
	after.Status = manualjournal.StatusRejected
	after.RejectedAt = &now
	after.RejectedByID = userID
	after.RejectionReason = req.Reason
	after.UpdatedByID = userID

	return s.saveTransition(
		ctx,
		&manualjournalports.ManualJournalChange{Before: entity, After: after},
		permission.OpReject,
		"Manual journal rejected",
	)
}

func (s *Service) PlanCancel(
	ctx context.Context,
	req *manualjournalports.CancelManualJournalRequest,
	actor *permissionports.RequestActor,
) (*manualjournalports.ManualJournalChange, error) {
	userID, err := requireManualJournalUser(actor)
	if err != nil {
		return nil, err
	}

	entity, err := s.repo.GetByID(
		ctx,
		manualjournalports.GetManualJournalByIDRequest{
			ID:         req.RequestID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidateCancel(entity, req.Reason); multiErr != nil {
		return nil, multiErr
	}

	after := cloneRequest(entity)
	now := timeutils.NowUnix()
	after.Status = manualjournal.StatusCancelled
	after.CancelledAt = &now
	after.CancelledByID = userID
	after.CancelReason = req.Reason
	after.UpdatedByID = userID

	return &manualjournalports.ManualJournalChange{Before: entity, After: after}, nil
}

func (s *Service) Cancel(
	ctx context.Context,
	req *manualjournalports.CancelManualJournalRequest,
	actor *permissionports.RequestActor,
) (*manualjournal.Request, error) {
	change, err := s.PlanCancel(ctx, req, actor)
	if err != nil {
		return nil, err
	}

	return s.saveTransition(ctx, change, permission.OpCancel, "Manual journal cancelled")
}

func (s *Service) saveTransition(
	ctx context.Context,
	change *manualjournalports.ManualJournalChange,
	operation permission.Operation,
	comment string,
) (*manualjournal.Request, error) {
	entity := cloneRequest(change.After)
	entity.Lines = nil

	var (
		updated *manualjournal.Request
		err     error
	)
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		updated, err = s.repo.Update(txCtx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(operation, updated, change.Before, entity.UpdatedByID, comment)
	return updated, nil
}

func (s *Service) loadRequestWithAccountingControl(
	ctx context.Context,
	requestID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*manualjournal.Request, *tenant.AccountingControl, error) {
	entity, err := s.repo.GetByID(
		ctx,
		manualjournalports.GetManualJournalByIDRequest{ID: requestID, TenantInfo: tenantInfo},
	)
	if err != nil {
		return nil, nil, err
	}
	accountingControl, err := s.accountingRepo.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		return nil, nil, err
	}
	return entity, accountingControl, nil
}

func journalPreview(
	entity *manualjournal.Request,
	periodID pulid.ID,
	postingDate int64,
) *kernelports.JournalPreview {
	lines := make([]kernelports.JournalLinePreview, 0, len(entity.Lines))
	for _, line := range entity.Lines {
		if line == nil {
			continue
		}
		lines = append(lines, kernelports.JournalLinePreview{
			GLAccountID: line.GLAccountID,
			Description: line.Description,
			DebitMinor:  line.DebitAmount,
			CreditMinor: line.CreditAmount,
		})
	}

	return &kernelports.JournalPreview{
		AccountingDate: postingDate,
		FiscalPeriodID: periodID,
		EntryStatus:    entryStatusPosted,
		Lines:          lines,
	}
}

func cloneRequest(src *manualjournal.Request) *manualjournal.Request {
	if src == nil {
		return nil
	}
	clone := *src
	if len(src.Lines) > 0 {
		clone.Lines = make([]*manualjournal.Line, 0, len(src.Lines))
		for _, line := range src.Lines {
			if line == nil {
				clone.Lines = append(clone.Lines, nil)
				continue
			}
			copied := *line
			clone.Lines = append(clone.Lines, &copied)
		}
	}
	return &clone
}

func mapLines(inputs []*manualjournalports.ManualJournalLineInput) []*manualjournal.Line {
	lines := make([]*manualjournal.Line, 0, len(inputs))
	for idx, input := range inputs {
		if input == nil {
			lines = append(lines, nil)
			continue
		}
		lines = append(lines, &manualjournal.Line{
			LineNumber:   idx + 1,
			GLAccountID:  input.GLAccountID,
			Description:  input.Description,
			DebitAmount:  input.DebitAmount,
			CreditAmount: input.CreditAmount,
			CustomerID:   input.CustomerID,
			LocationID:   input.LocationID,
		})
	}
	return lines
}

func requireManualJournalUser(actor *permissionports.RequestActor) (pulid.ID, error) {
	if actor == nil || actor.UserID.IsNil() {
		return pulid.Nil, errortypes.NewAuthorizationError(
			"Manual journal actions require an authenticated user",
		)
	}
	return actor.UserID, nil
}

func (s *Service) resolvePostingPeriod(
	ctx context.Context,
	entity *manualjournal.Request,
	accountingControl *tenant.AccountingControl,
) (*fiscalperiod.FiscalPeriod, int64, error) {
	resolved, err := journalposting.ResolvePeriod(
		ctx,
		s.validator.fiscalRepo,
		&journalposting.ResolvePeriodRequest{
			OrganizationID: entity.OrganizationID,
			BusinessUnitID: entity.BusinessUnitID,
			Date:           entity.AccountingDate,
			Policy:         accountingControl.ClosedPeriodPostingPolicy,
			Subject:        "manual journal",
		},
	)
	if err != nil {
		return nil, 0, err
	}
	return resolved.Period, resolved.AccountingDate, nil
}

func (s *Service) logAudit(
	operation permission.Operation,
	current *manualjournal.Request,
	previous *manualjournal.Request,
	userID pulid.ID,
	comment string,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceManualJournal,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}

	options := []auditports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		options = append(options, auditservice.WithDiff(previous, current))
	}

	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error(
			"failed to log manual journal audit action",
			zap.Error(err),
			zap.String("requestId", current.ID.String()),
		)
	}
}
