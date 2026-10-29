package accountingsyncpush

import (
	"context"
	"maps"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
)

type CreateFunc func() (*accountingconnectorports.AccountingDocumentResult, error)

func outcomeUnknown(record *accountingsync.AccountingSyncRecord) bool {
	if record.AttemptCount <= 1 {
		return false
	}
	return record.ErrorCategory == "" ||
		record.ErrorCategory == accountingsync.SyncErrorTransient
}

func (k *Kit) CreateAdopting(
	ctx context.Context,
	sess *Session,
	record *accountingsync.AccountingSyncRecord,
	match *accountingconnectorports.AccountingFindDocumentRequest,
	create CreateFunc,
) (*accountingconnectorports.AccountingDocumentResult, error) {
	if sess.Limits.IdempotencyWindow <= 0 {
		return create()
	}
	match.Auth = sess.Auth
	match.Kind = record.ObjectType
	match.RequestID = record.RequestID

	if outcomeUnknown(record) {
		found, ok, err := sess.Writer.FindDocument(ctx, match)
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
	found, ok, findErr := sess.Writer.FindDocument(ctx, match)
	if findErr != nil || !ok {
		return written, err
	}
	return adopted(found), nil
}

func isDuplicate(sess *Session, err error) bool {
	classified := sess.Writer.ClassifyDocumentError(err)
	return classified != nil && classified.Category == accountingsync.SyncErrorDuplicate
}

func adopted(
	found *accountingconnectorports.AccountingDocumentResult,
) *accountingconnectorports.AccountingDocumentResult {
	result := &accountingconnectorports.AccountingDocumentResult{
		ExternalID: found.ExternalID,
		DocNumber:  found.DocNumber,
		Refs:       make(map[string]string, len(found.Refs)+1),
	}
	maps.Copy(result.Refs, found.Refs)
	result.Refs[accountingsync.ExternalRefAdopted] = "true"
	return result
}
