package journalreversalservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalreversal"
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

	Logger              *zap.Logger
	DB                  ports.DBConnection
	JournalEntryRepo    repositories.JournalEntryRepository
	JournalReversalRepo repositories.JournalReversalRepository
	JournalPostingRepo  repositories.JournalPostingRepository
	AccountingRepo      repositories.AccountingControlRepository
	SequenceGenerator   seqgen.Generator
	Validator           *Validator
	AuditService        serviceports.AuditService
}

type Service struct {
	l                   *zap.Logger
	db                  ports.DBConnection
	journalEntryRepo    repositories.JournalEntryRepository
	journalReversalRepo repositories.JournalReversalRepository
	journalPostingRepo  repositories.JournalPostingRepository
	accountingRepo      repositories.AccountingControlRepository
	sequenceGenerator   seqgen.Generator
	validator           *Validator
	auditService        serviceports.AuditService
}

func New(p Params) *Service {
	return &Service{l: p.Logger.Named("service.journal-reversal"), db: p.DB, journalEntryRepo: p.JournalEntryRepo, journalReversalRepo: p.JournalReversalRepo, journalPostingRepo: p.JournalPostingRepo, accountingRepo: p.AccountingRepo, sequenceGenerator: p.SequenceGenerator, validator: p.Validator, auditService: p.AuditService}
}

func (s *Service) List(ctx context.Context, req *repositories.ListJournalReversalsRequest) (*pagination.ListResult[*journalreversal.Reversal], error) {
	return s.journalReversalRepo.List(ctx, req)
}
func (s *Service) Get(ctx context.Context, req *serviceports.GetJournalReversalRequest) (*journalreversal.Reversal, error) {
	return s.journalReversalRepo.GetByID(ctx, repositories.GetJournalReversalByIDRequest{ID: req.ReversalID, TenantInfo: req.TenantInfo})
}

func (s *Service) Create(ctx context.Context, req *serviceports.CreateJournalReversalRequest, actor *serviceports.RequestActor) (*journalreversal.Reversal, error) {
	userID, err := requireUser(actor)
	if err != nil {
		return nil, err
	}
	entry, err := s.journalEntryRepo.GetByID(ctx, repositories.GetJournalEntryByIDRequest{ID: req.OriginalJournalEntryID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	if me := s.validator.ValidateCreate(ctx, entry, req.RequestedAccountingDate, req.ReasonCode, req.ReasonText); me != nil {
		return nil, me
	}
	control, err := s.accountingRepo.GetByOrgID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, err
	}
	if control.JournalReversalPolicy == tenant.JournalReversalPolicyDisallow {
		return nil, errortypes.NewBusinessError("Journal reversals are disabled by accounting policy")
	}
	period, postingDate, me := s.validator.ResolvePostingPeriod(ctx, req.TenantInfo.OrgID, req.TenantInfo.BuID, req.RequestedAccountingDate, control)
	if me != nil {
		return nil, me
	}
	now := timeutils.NowUnix()
	entity := &journalreversal.Reversal{ID: pulid.MustNew("jrev_"), OrganizationID: req.TenantInfo.OrgID, BusinessUnitID: req.TenantInfo.BuID, OriginalJournalEntryID: req.OriginalJournalEntryID, Status: journalreversal.StatusApproved, RequestedAccountingDate: postingDate, ResolvedFiscalYearID: period.FiscalYearID, ResolvedFiscalPeriodID: period.ID, ReasonCode: req.ReasonCode, ReasonText: req.ReasonText, RequestedByID: userID, ApprovedByID: userID, ApprovedAt: &now}
	if control.RequireManualJEApproval {
		entity.Status = journalreversal.StatusPendingApproval
		entity.ApprovedByID = pulid.Nil
		entity.ApprovedAt = nil
	}
	created, err := s.journalReversalRepo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAudit(permission.OpCreate, created, nil, userID, "Journal reversal requested")
	return created, nil
}

func (s *Service) Approve(ctx context.Context, req *serviceports.GetJournalReversalRequest, actor *serviceports.RequestActor) (*journalreversal.Reversal, error) {
	userID, err := requireUser(actor)
	if err != nil {
		return nil, err
	}
	entity, err := s.journalReversalRepo.GetByID(ctx, repositories.GetJournalReversalByIDRequest{ID: req.ReversalID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	if me := s.validator.ValidateApprove(entity); me != nil {
		return nil, me
	}
	original := *entity
	now := timeutils.NowUnix()
	entity.Status = journalreversal.StatusApproved
	entity.ApprovedByID = userID
	entity.ApprovedAt = &now
	updated, err := s.journalReversalRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAudit(permission.OpApprove, updated, &original, userID, "Journal reversal approved")
	return updated, nil
}

func (s *Service) Reject(ctx context.Context, req *serviceports.RejectJournalReversalRequest, actor *serviceports.RequestActor) (*journalreversal.Reversal, error) {
	userID, err := requireUser(actor)
	if err != nil {
		return nil, err
	}
	entity, err := s.journalReversalRepo.GetByID(ctx, repositories.GetJournalReversalByIDRequest{ID: req.ReversalID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	if me := s.validator.ValidateReject(entity, req.Reason); me != nil {
		return nil, me
	}
	original := *entity
	now := timeutils.NowUnix()
	entity.Status = journalreversal.StatusRejected
	entity.RejectedByID = userID
	entity.RejectedAt = &now
	entity.RejectionReason = req.Reason
	updated, err := s.journalReversalRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAudit(permission.OpReject, updated, &original, userID, "Journal reversal rejected")
	return updated, nil
}

func (s *Service) Cancel(ctx context.Context, req *serviceports.CancelJournalReversalRequest, actor *serviceports.RequestActor) (*journalreversal.Reversal, error) {
	userID, err := requireUser(actor)
	if err != nil {
		return nil, err
	}
	entity, err := s.journalReversalRepo.GetByID(ctx, repositories.GetJournalReversalByIDRequest{ID: req.ReversalID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	if me := s.validator.ValidateCancel(entity, req.Reason); me != nil {
		return nil, me
	}
	original := *entity
	now := timeutils.NowUnix()
	entity.Status = journalreversal.StatusCancelled
	entity.CancelledByID = userID
	entity.CancelledAt = &now
	entity.CancelReason = req.Reason
	updated, err := s.journalReversalRepo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.logAudit(permission.OpCancel, updated, &original, userID, "Journal reversal cancelled")
	return updated, nil
}

func (s *Service) Post(ctx context.Context, req *serviceports.GetJournalReversalRequest, actor *serviceports.RequestActor) (*journalreversal.Reversal, error) {
	userID, err := requireUser(actor)
	if err != nil {
		return nil, err
	}
	entity, err := s.journalReversalRepo.GetByID(ctx, repositories.GetJournalReversalByIDRequest{ID: req.ReversalID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	if me := s.validator.ValidatePost(entity); me != nil {
		return nil, me
	}
	originalEntry, err := s.journalEntryRepo.GetByID(ctx, repositories.GetJournalEntryByIDRequest{ID: entity.OriginalJournalEntryID, TenantInfo: req.TenantInfo})
	if err != nil {
		return nil, err
	}
	if !originalEntry.ReversedByID.IsNil() {
		return nil, errortypes.NewBusinessError("Journal entry has already been reversed")
	}
	batchNumber, err := s.sequenceGenerator.GenerateJournalBatchNumber(ctx, entity.OrganizationID, entity.BusinessUnitID, "", "")
	if err != nil {
		return nil, err
	}
	entryNumber, err := s.sequenceGenerator.GenerateJournalEntryNumber(ctx, entity.OrganizationID, entity.BusinessUnitID, "", "")
	if err != nil {
		return nil, err
	}
	now := timeutils.NowUnix()
	batchID := pulid.MustNew("jb_")
	reversalEntryID := pulid.MustNew("je_")
	sourceID := pulid.MustNew("jsrc_")
	lines := make([]repositories.JournalPostingLine, 0, len(originalEntry.Lines))
	for _, line := range originalEntry.Lines {
		if line == nil {
			continue
		}
		lines = append(lines, repositories.JournalPostingLine{ID: pulid.MustNew("jel_"), GLAccountID: line.GLAccountID, LineNumber: line.LineNumber, Description: line.Description, DebitAmount: line.CreditAmount, CreditAmount: line.DebitAmount, NetAmount: -line.NetAmount, CustomerID: line.CustomerID, LocationID: line.LocationID})
	}
	original := *entity
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		if err := s.journalPostingRepo.CreatePosting(txCtx, repositories.CreateJournalPostingParams{BatchID: batchID, OrganizationID: entity.OrganizationID, BusinessUnitID: entity.BusinessUnitID, BatchNumber: batchNumber, BatchType: "Reversal", BatchStatus: "Posted", BatchDescription: "Journal reversal", FiscalYearID: entity.ResolvedFiscalYearID, FiscalPeriodID: entity.ResolvedFiscalPeriodID, AccountingDate: entity.RequestedAccountingDate, PostedAt: &now, PostedByID: userID, CreatedByID: userID, UpdatedByID: userID, EntryID: reversalEntryID, EntryNumber: entryNumber, EntryType: "Reversal", EntryStatus: "Posted", ReferenceNumber: originalEntry.EntryNumber, ReferenceType: "JournalReversal", ReferenceID: entity.ID.String(), EntryDescription: originalEntry.Description, TotalDebit: originalEntry.TotalCredit, TotalCredit: originalEntry.TotalDebit, IsPosted: true, IsAutoGenerated: true, IsReversal: true, ReversalOfID: originalEntry.ID, ReversalDate: &now, ReversalReason: entity.ReasonText, RequiresApproval: false, IsApproved: true, ApprovedByID: userID, ApprovedAt: &now, SourceID: sourceID, SourceObjectType: "JournalEntry", SourceObjectID: originalEntry.ID.String(), SourceEventType: "JournalReversalCreated", SourceStatus: "Posted", SourceDocumentNumber: originalEntry.EntryNumber, SourceIdempotencyKey: "journal-reversal:" + entity.ID.String(), Lines: lines}); err != nil {
			return err
		}
		if err := s.journalEntryRepo.MarkReversed(txCtx, repositories.MarkJournalEntryReversedRequest{OriginalEntryID: originalEntry.ID, ReversalEntryID: reversalEntryID, OrganizationID: entity.OrganizationID, BusinessUnitID: entity.BusinessUnitID, ReversalDate: entity.RequestedAccountingDate, ReversalReason: entity.ReasonText, UpdatedByID: userID}); err != nil {
			return err
		}
		entity.Status = journalreversal.StatusPosted
		entity.PostedByID = userID
		entity.PostedAt = &now
		entity.ReversalJournalEntryID = reversalEntryID
		entity.PostedBatchID = batchID
		updated, updateErr := s.journalReversalRepo.Update(txCtx, entity)
		if updateErr != nil {
			return updateErr
		}
		entity = updated
		return nil
	})
	if err != nil {
		return nil, err
	}
	s.logAudit(permission.OpApprove, entity, &original, userID, "Journal reversal posted")
	return entity, nil
}

func requireUser(actor *serviceports.RequestActor) (pulid.ID, error) {
	if actor == nil || actor.UserID.IsNil() {
		return pulid.Nil, errortypes.NewAuthorizationError("Journal reversal actions require an authenticated user")
	}
	return actor.UserID, nil
}

func (s *Service) logAudit(op permission.Operation, current *journalreversal.Reversal, previous *journalreversal.Reversal, userID pulid.ID, comment string) {
	params := &serviceports.LogActionParams{Resource: permission.ResourceJournalReversal, ResourceID: current.ID.String(), Operation: op, UserID: userID, CurrentState: jsonutils.MustToJSON(current), OrganizationID: current.OrganizationID, BusinessUnitID: current.BusinessUnitID}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}
	options := []serviceports.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		options = append(options, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, options...); err != nil {
		s.l.Error("failed to log journal reversal audit action", zap.Error(err), zap.String("reversalId", current.ID.String()))
	}
}
