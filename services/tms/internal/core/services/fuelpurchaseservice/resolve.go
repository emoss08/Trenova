package fuelpurchaseservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/fuelimport"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type ResolveRowsRequest struct {
	TenantInfo pagination.TenantInfo
	BatchID    pulid.ID
	Version    int64
	UserID     pulid.ID
}

type ResolveRowsResult struct {
	Batch     *fuelpurchase.ImportBatch `json:"batch"`
	Reviewed  int                       `json:"reviewed"`
	Resolved  int                       `json:"resolved"`
	Committed int                       `json:"committed"`
	Queued    int                       `json:"queued"`
}

// ResolveRows reads the rows a batch is still holding and works them out again
// against the tenant's current cards, tractors and jurisdictions.
//
// This is what makes a discovered card worth assigning. A feed cannot guess which
// truck a card belongs to, so its rows wait; once somebody assigns the card, adds
// the missing tractor, or registers the jurisdiction, the same rows resolve. They
// are parsed from the cells exactly as they were first read, so nothing about the
// original statement is reinterpreted — only what it is matched against changes.
//
// Rows already committed are left untouched: they are purchases now, and a second
// pass must never disturb them.
func (s *Service) ResolveRows(
	ctx context.Context,
	req *ResolveRowsRequest,
) (*ResolveRowsResult, error) {
	log := s.l.With(
		zap.String("operation", "ResolveRows"),
		zap.String("batchId", req.BatchID.String()),
	)

	batch, err := s.repo.GetImportBatchByID(ctx, &repositories.GetImportBatchByIDRequest{
		ID:          req.BatchID,
		TenantInfo:  req.TenantInfo,
		IncludeRows: true,
	})
	if err != nil {
		return nil, err
	}
	if batch.Version != req.Version {
		return nil, dberror.CreateVersionMismatchError("FuelPurchaseImportBatch", batch.ID.String())
	}
	if batch.IsTerminal() && !batch.IsFeed() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"This import is "+strings.ToLower(batch.Status.Label())+
				" and its rows cannot be worked out again",
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

	resolved, summary, err := s.resolveRows(ctx, batch, staged)
	if err != nil {
		log.Error("failed to resolve held rows", zap.Error(err))
		return nil, err
	}

	// The settled rows keep their identity and their purchase, and their counts
	// have to be folded back in or the batch would look like it had shrunk.
	for _, row := range settled {
		summary.RowCount++
		if row.Status == fuelpurchase.ImportRowStatusAlreadyImported {
			summary.AlreadyImportedCount++
		}
	}

	if err = s.repo.ReplaceImportRows(ctx, batch, append(settled, resolved...)); err != nil {
		return nil, err
	}

	stagedAt := s.now()
	batch.Summary = summary
	batch.RowCount = summary.RowCount
	batch.ErrorCount = summary.ErrorCount
	batch.StagedAt = &stagedAt
	batch.Error = ""
	if batch.IsTerminal() {
		batch.Status = fuelpurchase.ImportStatusParsed
	}
	if err = validateEntity(batch); err != nil {
		return nil, err
	}
	if batch, err = s.repo.UpdateImportBatch(ctx, batch); err != nil {
		return nil, err
	}

	result := &ResolveRowsResult{
		Reviewed: len(pending),
		Resolved: summary.NewCount,
		Queued:   queuedRowCount(resolved),
	}

	// A feed posts what it can on its own, exactly as the sync that opened the
	// batch did. An upload waits for the person to commit it, as it always has.
	if batch.IsFeed() && summary.NewCount > 0 {
		commit, cErr := s.commitFeedRows(ctx, &SyncFeedRequest{
			TenantInfo: req.TenantInfo,
			Provider:   batch.Provider,
			UserID:     req.UserID,
		}, batch, resolved)
		if cErr != nil {
			return nil, cErr
		}
		if commit != nil {
			result.Committed = commit.Committed
		}
	}

	result.Batch = batch

	s.audit(&auditParams{
		resource:   permission.ResourceFuelPurchaseImport,
		resourceID: batch.ID.String(),
		operation:  permission.OpImport,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    batch,
		comment: "Worked out " + strconv.Itoa(result.Reviewed) + " held rows: " +
			strconv.Itoa(result.Resolved) + " resolved, " +
			strconv.Itoa(result.Queued) + " still waiting",
	})
	s.publish(ctx, req.TenantInfo, realtimeImport, permission.OpImport, batch.ID, req.UserID)
	if result.Committed > 0 {
		s.publish(ctx, req.TenantInfo, realtimePurchase, permission.OpImport, batch.ID, req.UserID)
	}

	return result, nil
}

// partitionRows splits the rows that are finished with from the rows still worth
// another look. A committed row is a purchase and an already-imported row points
// at one somebody else made; neither is ours to redo.
func partitionRows(rows []*fuelpurchase.ImportRow) (settled, pending []*fuelpurchase.ImportRow) {
	settled = make([]*fuelpurchase.ImportRow, 0, len(rows))
	pending = make([]*fuelpurchase.ImportRow, 0, len(rows))

	for _, row := range rows {
		switch row.Status {
		case fuelpurchase.ImportRowStatusCommitted,
			fuelpurchase.ImportRowStatusAlreadyImported:
			settled = append(settled, row)
		default:
			pending = append(pending, row)
		}
	}

	return settled, pending
}

// restage reads the held rows back out of their stored cells. The batch's mapping
// says which column is which, and the unit is taken from what the rows parsed to
// the first time, so a statement measured in litres is not silently re-read as
// gallons.
func restage(
	batch *fuelpurchase.ImportBatch,
	pending []*fuelpurchase.ImportRow,
) (*fuelimport.StageResult, error) {
	mapping, err := fuelimport.MappingFromStrings(batch.Mapping)
	if err != nil {
		return nil, errortypes.NewValidationError("mapping", errortypes.ErrInvalid, err.Error())
	}
	if len(mapping) == 0 {
		return nil, errortypes.NewValidationError(
			"mapping",
			errortypes.ErrInvalid,
			"This import has no column mapping, so its rows cannot be read again",
		)
	}

	raw := make([]fuelimport.RawRow, 0, len(pending))
	for _, row := range pending {
		raw = append(raw, fuelimport.RawRow{RowNumber: row.RowNumber, Cells: row.Cells})
	}

	return &fuelimport.StageResult{
		Mapping: mapping,
		Rows: fuelimport.StageRows(raw, mapping, fuelimport.ParseOptions{
			Provider:        batch.Provider,
			DefaultFuelType: batch.DefaultFuelType,
			DefaultCurrency: batch.DefaultCurrency,
			DefaultUnit:     storedUnit(pending),
		}),
	}, nil
}

func storedUnit(rows []*fuelpurchase.ImportRow) fuelpurchase.QuantityUnit {
	for _, row := range rows {
		if row.Parsed != nil && row.Parsed.QuantityUnit != "" {
			return row.Parsed.QuantityUnit
		}
	}

	return ""
}
