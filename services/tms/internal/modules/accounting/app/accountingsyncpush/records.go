package accountingsyncpush

import (
	"context"
	"maps"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncenqueue"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (k *Kit) ObjectRecords(
	ctx context.Context,
	sess *Session,
	objectType accountingsync.SyncObjectType,
	objectID pulid.ID,
) ([]*accountingsync.AccountingSyncRecord, error) {
	return k.records.ListByObjects(
		ctx,
		&accountingsyncports.ListAccountingSyncRecordsByObjectsRequest{
			TenantInfo:   sess.Tenant,
			ConnectionID: sess.Conn.ID,
			ObjectTypes:  []accountingsync.SyncObjectType{objectType},
			ObjectIDs:    []pulid.ID{objectID},
		},
	)
}

func LatestOf(
	records []*accountingsync.AccountingSyncRecord,
	operation accountingsync.SyncOperation,
) *accountingsync.AccountingSyncRecord {
	var found *accountingsync.AccountingSyncRecord
	for _, record := range records {
		if record.Operation != operation {
			continue
		}
		found = record
	}
	return found
}

func DocumentRecord(
	records []*accountingsync.AccountingSyncRecord,
) *accountingsync.AccountingSyncRecord {
	var found *accountingsync.AccountingSyncRecord
	for _, record := range records {
		if record.Operation != accountingsync.SyncOperationCreate &&
			record.Operation != accountingsync.SyncOperationRecreate {
			continue
		}
		if found == nil || record.Revision >= found.Revision {
			found = record
		}
	}
	return found
}

func (k *Kit) UpdateTarget(
	ctx context.Context,
	sess *Session,
	record *accountingsync.AccountingSyncRecord,
	label string,
) (*accountingsync.AccountingSyncRecord, error) {
	existing, err := k.ObjectRecords(ctx, sess, record.ObjectType, record.ObjectID)
	if err != nil {
		return nil, err
	}
	sent := DocumentRecord(existing)
	switch {
	case sent == nil:
		return nil, &NoopError{Reason: label + " was never sent, so there is nothing to update"}
	case IsSynced(sent) && sent.SharesProviderDocument():
		return nil, SharedDocumentError(sess, label)
	case IsSynced(sent):
		return sent, nil
	case sent.Status.IsFinal():
		return nil, &NoopError{Reason: label + " was skipped, so its changes are not sent"}
	default:
		k.Kick(ctx, sess)
		return nil, WaitingOn(sent, label)
	}
}

func IsSynced(record *accountingsync.AccountingSyncRecord) bool {
	return record.Status == accountingsync.SyncStatusSynced && record.ExternalID != ""
}

func MergedRefs(records []*accountingsync.AccountingSyncRecord) map[string]string {
	refs := make(map[string]string, 4)
	for _, record := range records {
		if record.Status != accountingsync.SyncStatusSynced {
			continue
		}
		maps.Copy(refs, record.ExternalRefs)
	}
	return refs
}

func (k *Kit) EnqueueDependency(
	ctx context.Context,
	sess *Session,
	req *accountingsyncports.AccountingSyncEnqueueRequest,
) (*accountingsync.AccountingSyncRecord, error) {
	req.TenantInfo = sess.Tenant
	req.SourceEvent = accountingsync.SyncSourceDependencyOf
	record := accountingsyncenqueue.NewRecordFor(sess.Conn, req, timeutils.NowUnix())
	if _, err := k.records.Enqueue(
		ctx,
		[]*accountingsync.AccountingSyncRecord{record},
	); err != nil {
		return nil, err
	}
	existing, err := k.ObjectRecords(ctx, sess, req.ObjectType, req.ObjectID)
	if err != nil {
		return nil, err
	}
	return LatestOf(existing, req.Operation), nil
}

func (k *Kit) SettleUnsent(
	ctx context.Context,
	created *accountingsync.AccountingSyncRecord,
	record *accountingsync.AccountingSyncRecord,
) (bool, error) {
	switch {
	case created == nil:
		return true, &NoopError{Reason: "It never reached the books, so there is nothing to undo"}
	case IsSynced(created):
		return false, nil
	case created.Status == accountingsync.SyncStatusInFlight:
		return true, WaitFor(created.ID, "Waiting for the original to finish sending")
	default:
		if created.Supersede() {
			if _, err := k.records.Update(ctx, created); err != nil {
				return true, err
			}
		}
		if err := k.WithdrawUpdates(ctx, record); err != nil {
			return true, err
		}
		return true, &NoopError{
			Reason: "It never reached the books, so the queued send was withdrawn",
		}
	}
}

const maxRevision = int64(1) << 62

func (k *Kit) WithdrawUpdates(
	ctx context.Context,
	record *accountingsync.AccountingSyncRecord,
) error {
	_, err := k.records.SupersedeOlder(
		ctx,
		&accountingsyncports.SupersedeAccountingSyncRecordsRequest{
			TenantInfo:     record.TenantInfo(),
			ConnectionID:   record.ConnectionID,
			ObjectType:     record.ObjectType,
			ObjectID:       record.ObjectID,
			Operation:      accountingsync.SyncOperationUpdate,
			BeforeRevision: maxRevision,
		},
	)
	return err
}
