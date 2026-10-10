package bankreceiptbatchservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/modules/billing/domain/bankreceiptbatch"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptbatchports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In
	Logger             *zap.Logger
	DB                 kernelports.DBConnection
	Repo               bankreceiptbatchports.BankReceiptBatchRepository
	ReceiptRepo        bankreceiptports.BankReceiptRepository
	BankReceiptService bankreceiptports.BankReceiptService
	AuditService       auditports.AuditService
}
type Service struct {
	l                  *zap.Logger
	db                 kernelports.DBConnection
	repo               bankreceiptbatchports.BankReceiptBatchRepository
	receiptRepo        bankreceiptports.BankReceiptRepository
	bankReceiptService bankreceiptports.BankReceiptService
	auditService       auditports.AuditService
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:                  p.Logger.Named("service.bank-receipt-batch"),
		db:                 p.DB,
		repo:               p.Repo,
		receiptRepo:        p.ReceiptRepo,
		bankReceiptService: p.BankReceiptService,
		auditService:       p.AuditService,
	}
}

func (s *Service) Get(
	ctx context.Context,
	req *bankreceiptbatchports.GetBankReceiptBatchRequest,
) (*bankreceiptbatchports.BankReceiptBatchResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	batch, err := s.repo.GetByID(
		ctx,
		bankreceiptbatchports.GetBankReceiptBatchByIDRequest{
			ID:         req.BatchID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	receipts, err := s.receiptRepo.ListByImportBatchID(
		ctx,
		bankreceiptports.ListBankReceiptsByImportBatchRequest{
			BatchID:    req.BatchID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	return &bankreceiptbatchports.BankReceiptBatchResult{Batch: batch, Receipts: receipts}, nil
}

func (s *Service) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*bankreceiptbatch.BankReceiptBatch, error) {
	return s.repo.List(ctx, tenantInfo)
}

func (s *Service) DistinctSources(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*bankreceiptbatchports.BankReceiptBatchSourceOption], error) {
	return s.repo.DistinctSources(ctx, req)
}

func (s *Service) Import(
	ctx context.Context,
	req *bankreceiptbatchports.ImportBankReceiptBatchRequest,
	actor *permissionports.RequestActor,
) (*bankreceiptbatchports.BankReceiptBatchResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Bank receipt batch import requires an authenticated user",
		)
	}
	if len(req.Receipts) == 0 {
		return nil, errortypes.NewValidationError(
			"receipts",
			errortypes.ErrRequired,
			"At least one receipt is required",
		)
	}
	batch := &bankreceiptbatch.BankReceiptBatch{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Source:         strings.TrimSpace(req.Source),
		Reference:      strings.TrimSpace(req.Reference),
		Status:         bankreceiptbatch.StatusProcessing,
		CreatedByID:    actor.UserID,
		UpdatedByID:    actor.UserID,
	}
	me := errortypes.NewMultiError()
	batch.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	updatedBatch, receipts, err := s.importBatchWithinTx(ctx, req, actor, batch)
	if err != nil {
		return nil, err
	}

	s.logAudit(updatedBatch, nil, actor.UserID, "Bank receipt batch imported")
	return &bankreceiptbatchports.BankReceiptBatchResult{Batch: updatedBatch, Receipts: receipts}, nil
}

func (s *Service) importBatchWithinTx(
	ctx context.Context,
	req *bankreceiptbatchports.ImportBankReceiptBatchRequest,
	actor *permissionports.RequestActor,
	batch *bankreceiptbatch.BankReceiptBatch,
) (*bankreceiptbatch.BankReceiptBatch, []*bankreceipt.BankReceipt, error) {
	var (
		updatedBatch *bankreceiptbatch.BankReceiptBatch
		receipts     []*bankreceipt.BankReceipt
	)

	err := s.db.WithTx(ctx, kernelports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		createdBatch, txErr := s.repo.Create(txCtx, batch)
		if txErr != nil {
			return txErr
		}

		receipts = make([]*bankreceipt.BankReceipt, 0, len(req.Receipts))
		for idx, line := range req.Receipts {
			if line == nil {
				return errortypes.NewValidationError(
					"receipts",
					errortypes.ErrInvalid,
					"Receipt line {0} is required", strconv.Itoa(idx+1),
				)
			}

			receipt, importErr := s.bankReceiptService.Import(
				txCtx,
				&bankreceiptports.ImportBankReceiptRequest{
					ReceiptDate:     line.ReceiptDate,
					AmountMinor:     line.AmountMinor,
					ReferenceNumber: line.ReferenceNumber,
					Memo:            line.Memo,
					BatchID:         createdBatch.ID,
					SkipAudit:       true,
					TenantInfo:      req.TenantInfo,
				},
				actor,
			)
			if importErr != nil {
				return importErr
			}

			receipts = append(receipts, receipt)
			createdBatch.ImportedCount++
			createdBatch.ImportedAmountMinor += receipt.AmountMinor

			switch receipt.Status {
			case bankreceipt.StatusImported:
			case bankreceipt.StatusMatched:
				createdBatch.MatchedCount++
				createdBatch.MatchedAmountMinor += receipt.AmountMinor
			case bankreceipt.StatusException:
				createdBatch.ExceptionCount++
				createdBatch.ExceptionAmountMinor += receipt.AmountMinor
			}
		}

		createdBatch.Status = bankreceiptbatch.StatusCompleted
		createdBatch.UpdatedByID = actor.UserID
		updatedBatch, txErr = s.repo.Update(txCtx, createdBatch)
		return txErr
	})
	if err != nil {
		return nil, nil, err
	}

	return updatedBatch, receipts, nil
}

func (s *Service) logAudit(
	current, previous *bankreceiptbatch.BankReceiptBatch,
	userID pulid.ID,
	comment string,
) {
	params := &auditports.LogActionParams{
		Resource:       permission.ResourceBankReceipt,
		ResourceID:     current.ID.String(),
		Operation:      permission.OpCreate,
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
			"failed to log bank receipt batch audit action",
			zap.Error(err),
			zap.String("batchId", current.ID.String()),
		)
	}
}
