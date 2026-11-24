package driversettlementresolver

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/settlementreleaseports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

func blockersRequest(batchID *string, limit *int) (*settlementreleaseports.BlockedRequest, error) {
	req := &settlementreleaseports.BlockedRequest{Limit: settlementreleaseports.DefaultBlocked}
	if limit != nil {
		if *limit < 1 || *limit > settlementreleaseports.MaxBlocked {
			return nil, errortypes.NewValidationError(
				"limit", errortypes.ErrInvalid, "Limit must be from 1 to 100",
			)
		}
		req.Limit = *limit
	}
	if batchID != nil && strings.TrimSpace(*batchID) != "" {
		id, err := pulid.MustParse(*batchID)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"batchId", errortypes.ErrInvalid, "Invalid batch",
			)
		}
		req.BatchID = id
	}

	return req, nil
}

func minorOf(amount int64) *int {
	value := int(amount)

	return &value
}

func blockerMessage(ctx context.Context, row *settlementreleaseports.Settlement) string {
	if row.Block == driversettlement.ReleaseBlockCriticalException {
		if exception, ok := row.CriticalException(); ok {
			return strings.TrimSpace(exception.Message)
		}
	}
	if row.Block == driversettlement.ReleaseBlockDisputeOpen {
		return settlementreleaseports.DisputeAbout(ctx, row.DisputeCategory)
	}

	return ""
}

func blockersToModel(
	ctx context.Context,
	rows []*settlementreleaseports.Settlement,
) []*gqlmodel.DriverSettlementBlocker {
	out := make([]*gqlmodel.DriverSettlementBlocker, 0, len(rows))
	for _, row := range rows {
		model := &gqlmodel.DriverSettlementBlocker{
			ID:                   row.ID.String(),
			SettlementNumber:     row.SettlementNumber,
			Status:               row.Status,
			WorkerID:             row.WorkerID.String(),
			DriverName:           row.Driver,
			BatchName:            row.BatchName,
			PayDate:              int(row.PayDate),
			NetPayMinor:          minorOf(row.NetPayMinor),
			GrossEarningsMinor:   minorOf(row.GrossEarningsMinor),
			DeductionsMinor:      minorOf(row.DeductionsMinor),
			CarryForwardOutMinor: minorOf(row.CarryForwardOutMinor),
			CurrencyCode:         row.Currency,
			Reason:               settlementreleaseports.BlockWords(ctx, row.Block),
			Message:              blockerMessage(ctx, row),
		}
		if row.BatchID != nil {
			batch := row.BatchID.String()
			model.BatchID = &batch
		}
		if row.Block == driversettlement.ReleaseBlockHeld {
			reason := settlementreleaseports.HoldReasonWords(ctx, row.HoldReason)
			note := row.HoldNote
			model.HoldReason = &reason
			model.HoldNote = &note
		}
		out = append(out, model)
	}

	return out
}
