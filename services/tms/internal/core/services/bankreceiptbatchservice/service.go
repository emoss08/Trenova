package bankreceiptbatchservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/bankreceiptbatch"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
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
	DB                 ports.DBConnection
	Repo               repositories.BankReceiptBatchRepository
	ReceiptRepo        repositories.BankReceiptRepository
	BankReceiptService serviceports.BankReceiptService
	AuditService       serviceports.AuditService
}
type Service struct {
	l                  *zap.Logger
	db                 ports.DBConnection
	repo               repositories.BankReceiptBatchRepository
	receiptRepo        repositories.BankReceiptRepository
	bankReceiptService serviceports.BankReceiptService
	auditService       serviceports.AuditService
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
	req *serviceports.GetBankReceiptBatchRequest,
) (*serviceports.BankReceiptBatchResult, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"request",
			errortypes.ErrRequired,
			"Request is required",
		)
	}

	batch, err := s.repo.GetByID(
		ctx,
		repositories.GetBankReceiptBatchByIDRequest{
			ID:         req.BatchID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	receipts, err := s.receiptRepo.ListByImportBatchID(
		ctx,
		repositories.ListBankReceiptsByImportBatchRequest{
			BatchID:    req.BatchID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	return &serviceports.BankReceiptBatchResult{Batch: batch, Receipts: receipts}, nil
}

func (s *Service) List(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*bankreceiptbatch.Batch, error) {
	return s.repo.List(ctx, tenantInfo)
}

func (s *Service) DistinctSources(
	ctx context.Context,
	req *pagination.SelectQueryRequest,
) (*pagination.ListResult[*bankreceiptbatch.SourceOption], error) {
	return s.repo.DistinctSources(ctx, req)
}

func (s *Service) Import(
	ctx context.Context,
	req *serviceports.ImportBankReceiptBatchRequest,
	actor *serviceports.RequestActor,
) (*serviceports.BankReceiptBatchResult, error) {
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
	batch := &bankreceiptbatch.Batch{
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
	return &serviceports.BankReceiptBatchResult{Batch: updatedBatch, Receipts: receipts}, nil
}

func (s *Service) importBatchWithinTx(
	ctx context.Context,
	req *serviceports.ImportBankReceiptBatchRequest,
	actor *serviceports.RequestActor,
	batch *bankreceiptbatch.Batch,
) (*bankreceiptbatch.Batch, []*bankreceipt.Receipt, error) {
	var (
		updatedBatch *bankreceiptbatch.Batch
		receipts     []*bankreceipt.Receipt
	)

	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		createdBatch, txErr := s.repo.Create(txCtx, batch)
		if txErr != nil {
			return txErr
		}

		receipts = make([]*bankreceipt.Receipt, 0, len(req.Receipts))
		for idx, line := range req.Receipts {
			if line == nil {
				return errortypes.NewValidationError(
					"receipts",
					errortypes.ErrInvalid,
					"Receipt line "+strconv.Itoa(idx+1)+" is required",
				)
			}

			receipt, importErr := s.bankReceiptService.Import(
				txCtx,
				&serviceports.ImportBankReceiptRequest{
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
	current, previous *bankreceiptbatch.Batch,
	userID pulid.ID,
	comment string,
) {
	params := &serviceports.LogActionParams{
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
	options := []serviceports.LogOption{auditservice.WithComment(comment)}
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
