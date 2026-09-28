package fuelpurchaseservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

type PurchaseChange struct {
	Before *fuelpurchase.FuelPurchase
	After  *fuelpurchase.FuelPurchase
}

type CardChange struct {
	Before *fuelpurchase.FuelCard
	After  *fuelpurchase.FuelCard
}

type ImportBatchChange struct {
	Before *fuelpurchase.ImportBatch
	After  *fuelpurchase.ImportBatch
}

type CommitPlan struct {
	Batch            *fuelpurchase.ImportBatch
	Purchases        []*fuelpurchase.FuelPurchase
	RowIDByReference map[string]pulid.ID
}

type ResolveRowsPlan struct {
	Batch    *fuelpurchase.ImportBatch
	Settled  []*fuelpurchase.ImportRow
	Worked   []*fuelpurchase.ImportRow
	Summary  *fuelpurchase.ImportSummary
	Reviewed int
	Resolved int
	Queued   int
}

func (s *Service) PlanCreatePurchase(
	ctx context.Context,
	req *CreatePurchaseRequest,
) (*fuelpurchase.FuelPurchase, error) {
	if req.Purchase == nil {
		return nil, errortypes.NewValidationError(
			"purchase",
			errortypes.ErrRequired,
			"A purchase is required",
		)
	}

	purchase := *req.Purchase
	purchase.ID = ""
	purchase.OrganizationID = req.TenantInfo.OrgID
	purchase.BusinessUnitID = req.TenantInfo.BuID
	purchase.Source = fuelpurchase.PurchaseSourceManual
	purchase.ImportBatchID = nil
	purchase.CreatedByID = req.UserID
	purchase.Normalize()

	err := s.resolvePurchaseReferences(ctx, req.TenantInfo, &purchase, req.JurisdictionCode)
	if err != nil {
		return nil, err
	}
	if err = validateEntity(&purchase); err != nil {
		return nil, err
	}

	return &purchase, nil
}

func (s *Service) PlanUpdatePurchase(
	ctx context.Context,
	req *UpdatePurchaseRequest,
) (*PurchaseChange, error) {
	if req.Purchase == nil {
		return nil, errortypes.NewValidationError(
			"purchase",
			errortypes.ErrRequired,
			"A purchase is required",
		)
	}

	stored, err := s.repo.GetPurchaseByID(ctx, &repositories.GetFuelPurchaseByIDRequest{
		ID:         req.Purchase.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	purchase := *req.Purchase
	if purchase.Source != "" && purchase.Source != stored.Source {
		return nil, errortypes.NewValidationError(
			"source",
			errortypes.ErrInvalid,
			"The source of a purchase cannot be changed",
		)
	}
	if purchase.ImportBatchID != nil && !purchase.ImportBatchID.IsNil() &&
		!sameOptionalID(purchase.ImportBatchID, stored.ImportBatchID) {
		return nil, errortypes.NewValidationError(
			"importBatchId",
			errortypes.ErrInvalid,
			"A purchase cannot be moved between imports",
		)
	}

	purchase.OrganizationID = stored.OrganizationID
	purchase.BusinessUnitID = stored.BusinessUnitID
	purchase.Source = stored.Source
	purchase.ImportBatchID = stored.ImportBatchID
	purchase.CreatedByID = stored.CreatedByID
	purchase.CreatedAt = stored.CreatedAt
	purchase.Normalize()

	err = s.resolvePurchaseReferences(ctx, req.TenantInfo, &purchase, req.JurisdictionCode)
	if err != nil {
		return nil, err
	}
	if err = validateEntity(&purchase); err != nil {
		return nil, err
	}

	return &PurchaseChange{Before: stored, After: &purchase}, nil
}

func (s *Service) PlanDeletePurchase(
	ctx context.Context,
	req *DeletePurchaseRequest,
) (*fuelpurchase.FuelPurchase, error) {
	stored, err := s.repo.GetPurchaseByID(ctx, &repositories.GetFuelPurchaseByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if stored.Version != req.Version {
		return nil, dberror.CreateVersionMismatchError("FuelPurchase", stored.ID.String())
	}

	return stored, nil
}

func (s *Service) PlanAssignCard(ctx context.Context, req *AssignCardRequest) (*CardChange, error) {
	stored, err := s.repo.GetCardByID(ctx, &repositories.GetFuelCardByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if stored.Version != req.Version {
		return nil, dberror.CreateVersionMismatchError("FuelCard", stored.ID.String())
	}
	if stored.IsCancelled() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A cancelled card cannot be assigned",
		)
	}

	card := *stored
	card.AssignedTractorID = req.AssignedTractorID
	card.AssignedWorkerID = req.AssignedWorkerID
	if !card.IsUnassigned() && card.Status == fuelpurchase.CardStatusSuspended {
		card.Status = fuelpurchase.CardStatusActive
	}
	card.Normalize()
	if err = validateEntity(&card); err != nil {
		return nil, err
	}

	return &CardChange{Before: stored, After: &card}, nil
}

func (s *Service) loadBatch(
	ctx context.Context,
	req *repositories.GetImportBatchByIDRequest,
	version int64,
) (*fuelpurchase.ImportBatch, error) {
	batch, err := s.repo.GetImportBatchByID(ctx, req)
	if err != nil {
		return nil, err
	}
	if batch.Version != version {
		return nil, dberror.CreateVersionMismatchError("FuelPurchaseImportBatch", batch.ID.String())
	}

	return batch, nil
}

func (s *Service) PlanCommit(ctx context.Context, req *CommitRequest) (*CommitPlan, error) {
	batch, err := s.loadBatch(ctx, &repositories.GetImportBatchByIDRequest{
		ID:          req.BatchID,
		TenantInfo:  req.TenantInfo,
		IncludeRows: true,
	}, req.Version)
	if err != nil {
		return nil, err
	}
	if !batch.CanCommit() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"This import is {0} and cannot be committed", strings.ToLower(batch.Status.Label()),
		)
	}

	plan := &CommitPlan{
		Batch:            batch,
		Purchases:        make([]*fuelpurchase.FuelPurchase, 0, batch.NewRowCount()),
		RowIDByReference: make(map[string]pulid.ID, batch.NewRowCount()),
	}
	multiErr := errortypes.NewMultiError()
	for _, row := range batch.Rows {
		if !row.WillCommit() {
			continue
		}
		purchase := *row.Parsed
		purchase.ID = ""
		purchase.Version = 0
		purchase.OrganizationID = batch.OrganizationID
		purchase.BusinessUnitID = batch.BusinessUnitID
		purchase.Source = fuelpurchase.PurchaseSourceCardImport
		purchase.ImportBatchID = &batch.ID
		purchase.CreatedByID = req.UserID
		if purchase.TransactionReference == "" {
			purchase.TransactionReference = row.TransactionReference
		}
		purchase.Normalize()

		if purchase.TransactionReference == "" {
			multiErr.WithIndex("rows", row.RowNumber).
				Add("transactionReference", errortypes.ErrRequired, "Row has no reference")
			continue
		}
		purchase.Validate(multiErr.WithIndex("rows", row.RowNumber))

		plan.Purchases = append(plan.Purchases, &purchase)
		plan.RowIDByReference[purchase.TransactionReference] = row.ID
	}
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if len(plan.Purchases) == 0 {
		return nil, errortypes.NewValidationError(
			"rows",
			errortypes.ErrInvalid,
			"Nothing in this import is ready to commit",
		)
	}

	return plan, nil
}

func (s *Service) PlanDiscard(ctx context.Context, req *DiscardRequest) (*ImportBatchChange, error) {
	batch, err := s.loadBatch(ctx, &repositories.GetImportBatchByIDRequest{
		ID:         req.BatchID,
		TenantInfo: req.TenantInfo,
	}, req.Version)
	if err != nil {
		return nil, err
	}
	if !batch.CanDiscard() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"This import is {0} and cannot be discarded", strings.ToLower(batch.Status.Label()),
		)
	}

	after := *batch
	after.Status = fuelpurchase.ImportStatusDiscarded

	return &ImportBatchChange{Before: batch, After: &after}, nil
}

func (s *Service) PlanResolveRows(
	ctx context.Context,
	req *ResolveRowsRequest,
) (*ResolveRowsPlan, error) {
	batch, err := s.loadBatch(ctx, &repositories.GetImportBatchByIDRequest{
		ID:          req.BatchID,
		TenantInfo:  req.TenantInfo,
		IncludeRows: true,
	}, req.Version)
	if err != nil {
		return nil, err
	}
	if batch.IsTerminal() && !batch.IsFeed() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"This import is {0} and its rows cannot be worked out again",
			strings.ToLower(batch.Status.Label()),
		)
	}

	settled, pending := partitionRows(batch.Rows)
	if len(pending) == 0 {
		return nil, errortypes.NewValidationError(
			"rows",
			errortypes.ErrInvalid,
			"Nothing in this import is waiting to be worked out",
		)
	}

	staged, err := restage(batch, pending)
	if err != nil {
		return nil, err
	}

	worked, summary, err := s.resolveRows(ctx, batch, staged)
	if err != nil {
		return nil, err
	}

	for _, row := range settled {
		summary.RowCount++
		if row.Status == fuelpurchase.ImportRowStatusAlreadyImported {
			summary.AlreadyImportedCount++
		}
	}

	return &ResolveRowsPlan{
		Batch:    batch,
		Settled:  settled,
		Worked:   worked,
		Summary:  summary,
		Reviewed: len(pending),
		Resolved: summary.NewCount,
		Queued:   queuedRowCount(worked),
	}, nil
}
