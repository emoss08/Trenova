package manualjournalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fiscalperiod"
	"github.com/emoss08/trenova/internal/core/domain/manualjournal"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/seqgen"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger         *zap.Logger
	DB             ports.DBConnection
	Repo           repositories.ManualJournalRepository
	JournalRepo    repositories.JournalPostingRepository
	AccountingRepo repositories.AccountingControlRepository
	Generator      seqgen.Generator
	Validator      *Validator
	AuditService   serviceports.AuditService
}

type Service struct {
	l              *zap.Logger
	db             ports.DBConnection
	repo           repositories.ManualJournalRepository
	journalRepo    repositories.JournalPostingRepository
	accountingRepo repositories.AccountingControlRepository
	generator      seqgen.Generator
	validator      *Validator
	auditService   serviceports.AuditService
}

func New(p Params) *Service {
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
	req *repositories.ListManualJournalRequest,
) (*pagination.ListResult[*manualjournal.Request], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) Get(
	ctx context.Context,
	req *serviceports.GetManualJournalRequest,
) (*manualjournal.Request, error) {
	return s.repo.GetByID(ctx, repositories.GetManualJournalByIDRequest{ID: req.RequestID, TenantInfo: req.TenantInfo})
}

func (s *Service) CreateDraft(
	ctx context.Context,
	req *serviceports.CreateManualJournalRequest,
	actor *serviceports.RequestActor,
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

	if entity.RequestNumber, err = s.generator.GenerateManualJournalRequestNumber(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID, "", ""); err != nil {
		return nil, err
	}

	if multiErr := s.validator.ValidateDraftUpsert(ctx, entity, accountingControl); multiErr != nil {
		return nil, multiErr
	}

	var created *manualjournal.Request
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		created, err = s.repo.Create(txCtx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(permission.OpCreate, created, nil, userID, "Manual journal draft created")
	return created, nil
}

func (s *Service) UpdateDraft(
	ctx context.Context,
	req *serviceports.UpdateManualJournalDraftRequest,
	actor *serviceports.RequestActor,
) (*manualjournal.Request, error) {
	userID, err := requireManualJournalUser(actor)
	if err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetManualJournalByIDRequest{ID: req.RequestID, TenantInfo: req.TenantInfo})
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

	if multiErr := s.validator.ValidateDraftUpsert(ctx, updatedEntity, accountingControl); multiErr != nil {
		return nil, multiErr
	}

	var updated *manualjournal.Request
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		updated, err = s.repo.Update(txCtx, updatedEntity)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(permission.OpUpdate, updated, original, userID, "Manual journal draft updated")
	return updated, nil
}

func (s *Service) Submit(
	ctx context.Context,
	req *serviceports.GetManualJournalRequest,
	actor *serviceports.RequestActor,
) (*manualjournal.Request, error) {
	userID, err := requireManualJournalUser(actor)
	if err != nil {
		return nil, err
	}

	entity, accountingControl, err := s.loadRequestWithAccountingControl(ctx, req.RequestID, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidateSubmit(entity, accountingControl); multiErr != nil {
		return nil, multiErr
	}

	original := cloneRequest(entity)
	entity.UpdatedByID = userID
	if accountingControl.RequireManualJEApproval {
		entity.Status = manualjournal.StatusPendingApproval
		entity.ApprovedAt = nil
		entity.ApprovedByID = pulid.Nil
	} else {
		now := timeutils.NowUnix()
		entity.Status = manualjournal.StatusApproved
		entity.ApprovedAt = &now
		entity.ApprovedByID = userID
	}
	entity.Lines = nil

	var updated *manualjournal.Request
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		updated, err = s.repo.Update(txCtx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(permission.OpSubmit, updated, original, userID, "Manual journal submitted")
	return updated, nil
}

func (s *Service) Approve(
	ctx context.Context,
	req *serviceports.GetManualJournalRequest,
	actor *serviceports.RequestActor,
) (*manualjournal.Request, error) {
	userID, err := requireManualJournalUser(actor)
	if err != nil {
		return nil, err
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetManualJournalByIDRequest{ID: req.RequestID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidateApprove(entity); multiErr != nil {
		return nil, multiErr
	}

	original := cloneRequest(entity)
	now := timeutils.NowUnix()
	entity.Status = manualjournal.StatusApproved
	entity.ApprovedAt = &now
	entity.ApprovedByID = userID
	entity.UpdatedByID = userID
	entity.Lines = nil

	var updated *manualjournal.Request
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		updated, err = s.repo.Update(txCtx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(permission.OpApprove, updated, original, userID, "Manual journal approved")
	return updated, nil
}

func (s *Service) Post(
	ctx context.Context,
	req *serviceports.GetManualJournalRequest,
	actor *serviceports.RequestActor,
) (*manualjournal.Request, error) {
	userID, err := requireManualJournalUser(actor)
	if err != nil {
		return nil, err
	}

	entity, accountingControl, err := s.loadRequestWithAccountingControl(ctx, req.RequestID, req.TenantInfo)
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

	batchNumber, err := s.generator.GenerateJournalBatchNumber(ctx, entity.OrganizationID, entity.BusinessUnitID, "", "")
	if err != nil {
		return nil, err
	}
	entryNumber, err := s.generator.GenerateJournalEntryNumber(ctx, entity.OrganizationID, entity.BusinessUnitID, "", "")
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	batchID := pulid.MustNew("jb_")
	entryID := pulid.MustNew("je_")
	original := cloneRequest(entity)
	entryType := "Standard"
	if postingPeriod.PeriodType == fiscalperiod.PeriodTypeAdjusting {
		entryType = "Adjusting"
	}

	lines := make([]repositories.JournalPostingLine, 0, len(entity.Lines))
	for idx, line := range entity.Lines {
		if line == nil {
			continue
		}
		lines = append(lines, repositories.JournalPostingLine{
			ID:           pulid.MustNew("jel_"),
			GLAccountID:  line.GLAccountID,
			LineNumber:   int16(idx + 1),
			Description:  line.Description,
			DebitAmount:  line.DebitAmount,
			CreditAmount: line.CreditAmount,
			NetAmount:    line.DebitAmount - line.CreditAmount,
			CustomerID:   line.CustomerID,
			LocationID:   line.LocationID,
		})
	}

	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if err := s.journalRepo.CreatePosting(txCtx, repositories.CreateJournalPostingParams{
			BatchID:          batchID,
			OrganizationID:   entity.OrganizationID,
			BusinessUnitID:   entity.BusinessUnitID,
			BatchNumber:      batchNumber,
			BatchType:        "Manual",
			BatchStatus:      "Posted",
			BatchDescription: entity.Description,
			FiscalYearID:     postingPeriod.FiscalYearID,
			FiscalPeriodID:   postingPeriod.ID,
			AccountingDate:   postingDate,
			PostedAt:         &now,
			PostedByID:       userID,
			CreatedByID:      entity.CreatedByID,
			UpdatedByID:      userID,
			EntryID:          entryID,
			EntryNumber:      entryNumber,
			EntryType:        entryType,
			EntryStatus:      "Posted",
			ReferenceNumber:  entity.RequestNumber,
			ReferenceType:    "ManualJournalRequest",
			ReferenceID:      entity.ID.String(),
			EntryDescription: entity.Description,
			TotalDebit:       entity.TotalDebit,
			TotalCredit:      entity.TotalCredit,
			IsPosted:         true,
			IsAutoGenerated:  false,
			RequiresApproval: true,
			IsApproved:       true,
			ApprovedByID:     entity.ApprovedByID,
			ApprovedAt:       entity.ApprovedAt,
			Lines:            lines,
		}); err != nil {
			return err
		}

		entity.Status = manualjournal.StatusPosted
		entity.PostedBatchID = batchID
		entity.UpdatedByID = userID
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

	s.logAudit(permission.OpApprove, entity, original, userID, "Manual journal posted")
	return entity, nil
}

func (s *Service) Reject(
	ctx context.Context,
	req *serviceports.RejectManualJournalRequest,
	actor *serviceports.RequestActor,
) (*manualjournal.Request, error) {
	userID, err := requireManualJournalUser(actor)
	if err != nil {
		return nil, err
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetManualJournalByIDRequest{ID: req.RequestID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidateReject(req.Reason, entity); multiErr != nil {
		return nil, multiErr
	}

	original := cloneRequest(entity)
	now := timeutils.NowUnix()
	entity.Status = manualjournal.StatusRejected
	entity.RejectedAt = &now
	entity.RejectedByID = userID
	entity.RejectionReason = req.Reason
	entity.UpdatedByID = userID
	entity.Lines = nil

	var updated *manualjournal.Request
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		updated, err = s.repo.Update(txCtx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(permission.OpReject, updated, original, userID, "Manual journal rejected")
	return updated, nil
}

func (s *Service) Cancel(
	ctx context.Context,
	req *serviceports.CancelManualJournalRequest,
	actor *serviceports.RequestActor,
) (*manualjournal.Request, error) {
	userID, err := requireManualJournalUser(actor)
	if err != nil {
		return nil, err
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetManualJournalByIDRequest{ID: req.RequestID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	if multiErr := s.validator.ValidateCancel(entity, req.Reason); multiErr != nil {
		return nil, multiErr
	}

	original := cloneRequest(entity)
	now := timeutils.NowUnix()
	entity.Status = manualjournal.StatusCancelled
	entity.CancelledAt = &now
	entity.CancelledByID = userID
	entity.CancelReason = req.Reason
	entity.UpdatedByID = userID
	entity.Lines = nil

	var updated *manualjournal.Request
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		updated, err = s.repo.Update(txCtx, entity)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(permission.OpCancel, updated, original, userID, "Manual journal cancelled")
	return updated, nil
}

func (s *Service) loadRequestWithAccountingControl(
	ctx context.Context,
	requestID pulid.ID,
	tenantInfo pagination.TenantInfo,
) (*manualjournal.Request, *tenant.AccountingControl, error) {
	entity, err := s.repo.GetByID(ctx, repositories.GetManualJournalByIDRequest{ID: requestID, TenantInfo: tenantInfo})
	if err != nil {
		return nil, nil, err
	}
	accountingControl, err := s.accountingRepo.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		return nil, nil, err
	}
	return entity, accountingControl, nil
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

func mapLines(inputs []*serviceports.ManualJournalLineInput) []*manualjournal.Line {
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

func requireManualJournalUser(actor *serviceports.RequestActor) (pulid.ID, error) {
	if actor == nil || actor.UserID.IsNil() {
		return pulid.Nil, errortypes.NewAuthorizationError("Manual journal actions require an authenticated user")
	}
	return actor.UserID, nil
}

func (s *Service) resolvePostingPeriod(
	ctx context.Context,
	entity *manualjournal.Request,
	accountingControl *tenant.AccountingControl,
) (*fiscalperiod.FiscalPeriod, int64, error) {
	period, err := s.validator.fiscalRepo.GetPeriodByDate(ctx, repositories.GetPeriodByDateRequest{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
		Date:  entity.AccountingDate,
	})
	if err != nil {
		return nil, 0, err
	}

	switch period.Status {
	case fiscalperiod.StatusOpen, fiscalperiod.StatusLocked:
		return period, entity.AccountingDate, nil
	case fiscalperiod.StatusClosed, fiscalperiod.StatusPermanentlyClosed:
		if accountingControl.ClosedPeriodPostingPolicy != tenant.ClosedPeriodPostingPolicyPostToNextOpen {
			return nil, 0, errortypes.NewBusinessError("Manual journal cannot be posted to a closed period; reopen the period first")
		}

		periods, listErr := s.validator.fiscalRepo.ListByFiscalYearID(ctx, repositories.ListByFiscalYearIDRequest{
			FiscalYearID: period.FiscalYearID,
			OrgID:        entity.OrganizationID,
			BuID:         entity.BusinessUnitID,
		})
		if listErr != nil {
			return nil, 0, listErr
		}
		for _, candidate := range periods {
			if candidate == nil || candidate.PeriodNumber <= period.PeriodNumber {
				continue
			}
			if candidate.Status == fiscalperiod.StatusOpen || candidate.Status == fiscalperiod.StatusLocked {
				return candidate, candidate.StartDate, nil
			}
		}

		return nil, 0, errortypes.NewBusinessError("No next open fiscal period is available for manual journal posting")
	default:
		return nil, 0, errortypes.NewBusinessError("Manual journal cannot be posted to an inactive fiscal period")
	}
}

func (s *Service) logAudit(
	operation permission.Operation,
	current *manualjournal.Request,
	previous *manualjournal.Request,
	userID pulid.ID,
	comment string,
) {
	params := &serviceports.LogActionParams{
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

	options := []serviceports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		options = append(options, auditservice.WithDiff(previous, current))
	}

	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log manual journal audit action", zap.Error(err), zap.String("requestId", current.ID.String()))
	}
}
