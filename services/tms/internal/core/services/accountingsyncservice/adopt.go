package accountingsyncservice

import (
	"context"
	"maps"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/shopspring/decimal"
)

type createFunc func() (*services.AccountingDocumentResult, error)

func outcomeUnknown(record *accountingsync.AccountingSyncRecord) bool {
	if record.AttemptCount <= 1 {
		return false
	}
	return record.ErrorCategory == "" ||
		record.ErrorCategory == accountingsync.SyncErrorTransient
}

func (s *Service) createAdopting(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
	match *services.AccountingFindDocumentRequest,
	create createFunc,
) (*services.AccountingDocumentResult, error) {
	if sess.limits.IdempotencyWindow <= 0 {
		return create()
	}
	match.Auth = sess.auth
	match.Kind = record.ObjectType
	match.RequestID = record.RequestID

	if outcomeUnknown(record) {
		found, ok, err := sess.writer.FindDocument(ctx, match)
		if err != nil {
			return nil, err
		}
		if ok {
			return adopted(found), nil
		}
	}

	written, err := create()
	if err == nil || !isDuplicate(sess, err) {
		return written, err
	}
	found, ok, findErr := sess.writer.FindDocument(ctx, match)
	if findErr != nil || !ok {
		return written, err
	}
	return adopted(found), nil
}

func isDuplicate(sess *pushSession, err error) bool {
	classified := sess.writer.ClassifyDocumentError(err)
	return classified != nil && classified.Category == accountingsync.SyncErrorDuplicate
}

func adopted(found *services.AccountingDocumentResult) *services.AccountingDocumentResult {
	result := &services.AccountingDocumentResult{
		ExternalID: found.ExternalID,
		DocNumber:  found.DocNumber,
		Refs:       make(map[string]string, len(found.Refs)+1),
	}
	maps.Copy(result.Refs, found.Refs)
	result.Refs[accountingsync.ExternalRefAdopted] = "true"
	return result
}

func salesTotal(lines []services.AccountingDocumentLine) decimal.Decimal {
	total := decimal.Zero
	for idx := range lines {
		total = total.Add(lines[idx].Amount)
	}
	return total
}

func purchaseTotal(lines []services.AccountingPurchaseLine) decimal.Decimal {
	total := decimal.Zero
	for idx := range lines {
		total = total.Add(lines[idx].Amount)
	}
	return total
}

func appliedInvoices(applications []services.AccountingPaymentApplication) []string {
	ids := make([]string, 0, len(applications))
	for idx := range applications {
		ids = append(ids, applications[idx].InvoiceExternalID)
	}
	return ids
}
