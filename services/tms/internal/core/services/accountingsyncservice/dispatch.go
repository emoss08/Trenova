package accountingsyncservice

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"maps"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

var permissionResource = permission.ResourceAccountingSync.String()

type pushSession struct {
	tenant       pagination.TenantInfo
	conn         *accountingsync.AccountingConnection
	writer       services.AccountingDocumentWriter
	auth         services.AccountingDocumentAuth
	loc          *time.Location
	providerName string
	limits       services.AccountingDocumentLimits
}

type waitError struct {
	reason    *accountingsync.SyncError
	delay     time.Duration
	dependsOn pulid.ID
}

func (w *waitError) Error() string { return w.reason.Message }

func waitFor(dependsOn pulid.ID, message string) *waitError {
	return &waitError{
		reason: &accountingsync.SyncError{
			Category: accountingsync.SyncErrorMapping,
			Message:  message,
		},
		delay:     dependencyWait,
		dependsOn: dependsOn,
	}
}

type noopError struct {
	reason string
}

func (n *noopError) Error() string { return n.reason }

type pushResult struct {
	result *accountingsync.SyncResult
	refs   map[string]string
}

func blocked(
	category accountingsync.SyncErrorCategory,
	message, resolution string,
) *accountingsync.SyncError {
	return &accountingsync.SyncError{
		Category:   category,
		Message:    message,
		Resolution: resolution,
	}
}

func (s *Service) openSession(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
) (*pushSession, error) {
	tenant := tenantOf(conn)
	session, err := s.connService.Session(ctx, tenant, conn.ID)
	if err != nil {
		return nil, err
	}
	writer, ok := session.Connector.(services.AccountingDocumentWriter)
	if !ok {
		return nil, fmt.Errorf(
			"%s cannot receive documents from Trenova",
			accountingsync.ProviderName(conn.IntegrationType),
		)
	}

	loc := time.UTC
	org, err := s.organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}
	if org != nil {
		loc = timeutils.LoadLocation(org.Timezone)
	}

	return &pushSession{
		tenant: tenant,
		conn:   session.Connection,
		writer: writer,
		auth: services.AccountingDocumentAuth{
			RealmID:     session.Connection.ExternalRealmID,
			AccessToken: session.AccessToken,
		},
		loc:          loc,
		providerName: accountingsync.ProviderName(conn.IntegrationType),
		limits:       writer.DocumentLimits(),
	}, nil
}

func (s *Service) Drain(
	ctx context.Context,
	req *services.DrainAccountingSyncRequest,
) (*services.AccountingSyncDrainResult, error) {
	result := new(services.AccountingSyncDrainResult)
	conn, err := s.connectionByID(ctx, req.TenantInfo, req.ConnectionID)
	if err != nil {
		return nil, err
	}
	if !conn.CanDispatch() {
		result.Held = true
		return result, nil
	}

	sess, err := s.openSession(ctx, conn)
	if err != nil {
		s.l.Warn("accounting sync is holding: the connection cannot be used",
			zap.String("connectionId", conn.ID.String()), zap.Error(err))
		result.Held = true
		return result, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultDrainLimit
	}
	claimed, err := s.records.Claim(ctx, &repositories.ClaimAccountingSyncRecordsRequest{
		TenantInfo:   req.TenantInfo,
		ConnectionID: conn.ID,
		Now:          timeutils.NowUnix(),
		Lease:        req.Lease,
		Limit:        limit,
	})
	if err != nil {
		return nil, err
	}
	result.Claimed = len(claimed)

	attention := false
	for _, record := range claimed {
		if req.Heartbeat != nil {
			req.Heartbeat()
		}
		if ctx.Err() != nil {
			return result, ctx.Err()
		}
		outcome, finishErr := s.pushRecord(ctx, sess, record)
		if finishErr != nil {
			if errors.Is(finishErr, repositories.ErrAccountingSyncLeaseLost) {
				result.LeaseLost++
				continue
			}
			return result, finishErr
		}
		tally(result, outcome)
		if outcome != accountingsync.SyncAttemptRetrying &&
			outcome != accountingsync.SyncAttemptWaiting {
			attention = true
		}
	}

	if attention {
		s.refreshAttention(ctx, sess.conn)
	}
	return result, nil
}

func tally(result *services.AccountingSyncDrainResult, outcome accountingsync.SyncAttemptOutcome) {
	switch outcome {
	case accountingsync.SyncAttemptSynced:
		result.Synced++
	case accountingsync.SyncAttemptRetrying:
		result.Retrying++
	case accountingsync.SyncAttemptWaiting:
		result.Waiting++
	case accountingsync.SyncAttemptBlocked:
		result.Blocked++
	case accountingsync.SyncAttemptDeadLettered:
		result.DeadLettered++
	}
}

func (s *Service) pushRecord(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (accountingsync.SyncAttemptOutcome, error) {
	started := time.Now()
	attemptNumber := record.AttemptCount
	pushed, pushErr := s.pushOne(ctx, sess, record)
	finished := time.Now()
	now := finished.Unix()

	if pushed != nil {
		for key, value := range pushed.refs {
			record.SetExternalRef(key, value)
		}
	}

	var outcome accountingsync.SyncAttemptOutcome
	recordAttempt := true
	var wait *waitError
	var noop *noopError
	switch {
	case pushErr == nil:
		record.MarkSynced(pushed.result, now)
		outcome = accountingsync.SyncAttemptSynced
	case errors.As(pushErr, &noop):
		record.MarkSynced(&accountingsync.SyncResult{}, now)
		record.Resolution = noop.reason
		outcome = accountingsync.SyncAttemptSynced
	case errors.As(pushErr, &wait):
		record.Wait(wait.reason, now, wait.delay)
		if !wait.dependsOn.IsNil() {
			record.DependsOnRecordID = wait.dependsOn
		}
		outcome = accountingsync.SyncAttemptWaiting
		recordAttempt = false
	default:
		var classified *accountingsync.SyncError
		if !errors.As(pushErr, &classified) {
			classified = sess.writer.ClassifyDocumentError(pushErr)
		}
		if classified == nil {
			classified = blocked(accountingsync.SyncErrorTransient, pushErr.Error(), "")
		}
		outcome = record.MarkFailed(classified, now)
		if classified.Category == accountingsync.SyncErrorAuth {
			s.connService.ReportCallFailure(ctx, sess.tenant, sess.conn.ID, pushErr)
		}
	}

	var attempt *accountingsync.AccountingSyncAttempt
	if recordAttempt {
		attempt = accountingsync.NewSyncAttempt(&accountingsync.NewSyncAttemptParams{
			Record:        record,
			AttemptNumber: attemptNumber,
			Outcome:       outcome,
			StartedAt:     started,
			FinishedAt:    finished,
		})
	}
	if err := s.records.Finish(ctx, &repositories.FinishAccountingSyncRecordRequest{
		Record:  record,
		Attempt: attempt,
	}); err != nil {
		return outcome, err
	}

	s.publishOutcome(ctx, record, outcome)
	if outcome != accountingsync.SyncAttemptWaiting {
		s.publishInvalidation(ctx, sess.tenant, pulid.Nil, record.ID)
	}
	return outcome, nil
}

func (s *Service) pushOne(
	ctx context.Context,
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
) (*pushResult, error) {
	switch {
	case record.ObjectType == accountingsync.SyncObjectCustomer, record.ObjectType.IsVendor():
		return s.pushParty(ctx, sess, record)
	case record.ObjectType.IsBill():
		if record.Operation == accountingsync.SyncOperationVoid {
			return s.pushBillVoid(ctx, sess, record)
		}
		return s.pushBill(ctx, sess, record)
	case record.ObjectType.IsBillPayment():
		return s.pushBillPayment(ctx, sess, record)
	case record.ObjectType.IsSalesDocument():
		if record.Operation == accountingsync.SyncOperationVoid {
			return s.pushSalesVoid(ctx, sess, record)
		}
		return s.pushSales(ctx, sess, record)
	case record.ObjectType == accountingsync.SyncObjectCustomerPayment:
		if record.Operation == accountingsync.SyncOperationVoid {
			return s.pushPaymentVoid(ctx, sess, record)
		}
		return s.pushPayment(ctx, sess, record)
	case record.ObjectType == accountingsync.SyncObjectCreditApplication:
		if record.Operation == accountingsync.SyncOperationVoid {
			return s.pushCreditApplicationVoid(ctx, sess, record)
		}
		return s.pushCreditApplication(ctx, sess, record)
	default:
		return nil, blocked(
			accountingsync.SyncErrorConfiguration,
			"Trenova does not send "+string(record.ObjectType)+" records",
			"Skip this record",
		)
	}
}

func finishedResult(
	sess *pushSession,
	record *accountingsync.AccountingSyncRecord,
	written *services.AccountingDocumentResult,
	document any,
	mappingIDs []string,
) (*pushResult, error) {
	payload, hash, err := payloadOf(document)
	if err != nil {
		return nil, err
	}
	refs := map[string]string{}
	if written != nil {
		maps.Copy(refs, written.Refs)
	}
	result := &accountingsync.SyncResult{
		PayloadHash:  hash,
		Payload:      payload,
		MappingIDs:   mappingIDs,
		ExternalRefs: refs,
	}
	if written != nil {
		result.ExternalID = written.ExternalID
		result.ExternalDocNumber = written.DocNumber
		result.ExternalURL = sess.writer.DocumentURL(record.ObjectType, written.ExternalID)
	}
	return &pushResult{result: result, refs: refs}, nil
}

func partial(written *services.AccountingDocumentResult) *pushResult {
	if written == nil || len(written.Refs) == 0 {
		return nil
	}
	return &pushResult{refs: written.Refs}
}

func payloadOf(document any) (payload map[string]any, hash string, err error) {
	if document == nil {
		return nil, "", nil
	}
	raw, err := sonic.Marshal(document)
	if err != nil {
		return nil, "", fmt.Errorf("accounting sync: encode payload: %w", err)
	}
	if err = sonic.Unmarshal(raw, &payload); err != nil {
		return nil, "", fmt.Errorf("accounting sync: decode payload: %w", err)
	}
	delete(payload, "Auth")
	delete(payload, "Refs")
	stored, err := sonic.ConfigStd.Marshal(payload)
	if err != nil {
		return nil, "", fmt.Errorf("accounting sync: encode payload: %w", err)
	}
	sum := sha256.Sum256(stored)
	return payload, hex.EncodeToString(sum[:]), nil
}

func (s *Service) objectRecords(
	ctx context.Context,
	sess *pushSession,
	objectType accountingsync.SyncObjectType,
	objectID pulid.ID,
) ([]*accountingsync.AccountingSyncRecord, error) {
	return s.records.ListByObjects(ctx, &repositories.ListAccountingSyncRecordsByObjectsRequest{
		TenantInfo:   sess.tenant,
		ConnectionID: sess.conn.ID,
		ObjectTypes:  []accountingsync.SyncObjectType{objectType},
		ObjectIDs:    []pulid.ID{objectID},
	})
}

func latestOf(
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

func isSynced(record *accountingsync.AccountingSyncRecord) bool {
	return record.Status == accountingsync.SyncStatusSynced && record.ExternalID != ""
}

func mergedRefs(records []*accountingsync.AccountingSyncRecord) map[string]string {
	refs := make(map[string]string, 4)
	for _, record := range records {
		if record.Status != accountingsync.SyncStatusSynced {
			continue
		}
		maps.Copy(refs, record.ExternalRefs)
	}
	return refs
}

func (s *Service) enqueueDependency(
	ctx context.Context,
	sess *pushSession,
	req *services.AccountingSyncEnqueueRequest,
) (*accountingsync.AccountingSyncRecord, error) {
	req.TenantInfo = sess.tenant
	req.SourceEvent = accountingsync.SyncSourceDependencyOf
	record := NewRecordFor(sess.conn, req, timeutils.NowUnix())
	if _, err := s.records.Enqueue(
		ctx,
		[]*accountingsync.AccountingSyncRecord{record},
	); err != nil {
		return nil, err
	}
	existing, err := s.objectRecords(ctx, sess, req.ObjectType, req.ObjectID)
	if err != nil {
		return nil, err
	}
	return latestOf(existing, req.Operation), nil
}

func waitingOn(dependency *accountingsync.AccountingSyncRecord, label string) error {
	if dependency.Status.NeedsAttention() {
		message := label + " has not reached the books"
		if dependency.ErrorMessage != "" {
			message += ": " + dependency.ErrorMessage
		}
		wait := waitFor(dependency.ID, message)
		wait.reason.Resolution = "Fix " + label + " first; this record follows it"
		wait.delay = accountingsync.SyncAuthWait
		return wait
	}
	return waitFor(dependency.ID, "Waiting for "+label+" to reach the books")
}
