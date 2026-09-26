package accountinginboundservice

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type readSession struct {
	tenant pagination.TenantInfo
	conn   *accountingsync.AccountingConnection
	reader services.AccountingChangeReader
	auth   services.AccountingDocumentAuth
	writer services.AccountingDocumentWriter
	loc    *time.Location
}

func (s *Service) location(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*time.Location, error) {
	org, err := s.organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}
	if org == nil {
		return time.UTC, nil
	}
	return timeutils.LoadLocation(org.Timezone), nil
}

func (s *Service) openRead(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
) (*readSession, error) {
	tenant := tenantOf(conn)
	session, err := s.connService.Session(ctx, tenant, conn.ID)
	if err != nil {
		return nil, err
	}
	reader, ok := session.Connector.(services.AccountingChangeReader)
	if !ok {
		return nil, fmt.Errorf(
			"%s does not report its changes to Trenova",
			accountingsync.ProviderName(conn.IntegrationType),
		)
	}
	writer, _ := session.Connector.(services.AccountingDocumentWriter)
	loc, err := s.location(ctx, tenant)
	if err != nil {
		return nil, err
	}
	return &readSession{
		tenant: tenant,
		conn:   session.Connection,
		reader: reader,
		auth: services.AccountingDocumentAuth{
			RealmID:     session.Connection.ExternalRealmID,
			AccessToken: session.AccessToken,
		},
		writer: writer,
		loc:    loc,
	}, nil
}

func (s *Service) PollChanges(
	ctx context.Context,
	req *services.PollAccountingChangesRequest,
) (*services.AccountingChangesPollResult, error) {
	result := new(services.AccountingChangesPollResult)
	conn, err := s.connectionByID(ctx, req.TenantInfo, req.ConnectionID)
	if err != nil {
		return nil, err
	}
	if !conn.ReadsChanges() {
		result.Held = true
		return result, nil
	}

	sess, err := s.openRead(ctx, conn)
	if err != nil {
		s.l.Warn("accounting changes are holding: the connection cannot be used",
			zap.String("connectionId", conn.ID.String()), zap.Error(err))
		result.Held = true
		return result, nil
	}

	cursor, err := s.startCursor(ctx, sess)
	if err != nil {
		return nil, err
	}

	now := s.now()
	page, err := sess.reader.ReadChanges(ctx, &services.ReadAccountingChangesRequest{
		Auth:           sess.auth,
		Cursor:         cursor,
		Now:            now,
		Payments:       true,
		BillPayments:   true,
		Documents:      s.drift != nil,
		ReferenceKinds: accountingsync.AllReferenceKinds(),
	})
	if err != nil {
		s.recordReadFailure(ctx, sess, err)
		return nil, err
	}

	readAt := now.Unix()
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		refs, refErr := s.applyReferences(txCtx, sess, page.References, readAt)
		if refErr != nil {
			return refErr
		}
		result.References = refs
		recorded, recErr := s.recordPayments(txCtx, sess, page.Payments, readAt)
		if recErr != nil {
			return recErr
		}
		result.Recorded = recorded
		_, saveErr := s.connections.SaveChangeFeed(
			txCtx,
			&repositories.SaveAccountingChangeFeedRequest{
				TenantInfo: sess.tenant,
				ID:         sess.conn.ID,
				Cursor:     page.NextCursor,
				ReadAt:     &readAt,
			},
		)
		return saveErr
	})
	if err != nil {
		return nil, err
	}

	result.Payments = len(page.Payments)
	result.Documents = s.recheckDocuments(ctx, sess, page.Documents)
	result.More = page.More
	result.CursorExpired = page.CursorExpired
	if page.CursorExpired && s.refresher != nil {
		if refreshErr := s.refresher.RequestReferenceRefresh(
			ctx,
			sess.tenant,
			sess.conn.ID,
		); refreshErr != nil {
			s.l.Warn("failed to request a reference refresh after an expired change cursor",
				zap.String("connectionId", sess.conn.ID.String()), zap.Error(refreshErr))
		}
	}
	if result.Recorded > 0 {
		s.publishInvalidation(ctx, sess.tenant, sess.conn.ID, sess.conn.ID)
	}
	return result, nil
}

func (s *Service) recheckDocuments(
	ctx context.Context,
	sess *readSession,
	documents []services.AccountingChangedDocument,
) int {
	if s.drift == nil || len(documents) == 0 {
		return 0
	}
	checked, err := s.drift.RecheckDocuments(ctx, &services.RecheckAccountingDriftRequest{
		TenantInfo:   sess.tenant,
		ConnectionID: sess.conn.ID,
		Documents:    documents,
		EventBudget:  driftEventsPerPoll,
	})
	if err != nil {
		s.l.Warn(
			"failed to compare documents the accounting system changed; the nightly check will",
			zap.String("connectionId", sess.conn.ID.String()),
			zap.Error(err),
		)
		return 0
	}
	return checked.Compared
}

func (s *Service) startCursor(ctx context.Context, sess *readSession) (string, error) {
	if sess.conn.ChangeCursor != "" {
		return sess.conn.ChangeCursor, nil
	}
	start := s.now()
	if sess.conn.SyncEnabledAt != nil {
		start = time.Unix(*sess.conn.SyncEnabledAt, 0)
	}
	cursor := sess.reader.ChangeCursorAt(start)
	if _, err := s.connections.SaveChangeFeed(ctx, &repositories.SaveAccountingChangeFeedRequest{
		TenantInfo:  sess.tenant,
		ID:          sess.conn.ID,
		Cursor:      cursor,
		OnlyIfEmpty: true,
	}); err != nil {
		return "", err
	}
	current, err := s.connectionByID(ctx, sess.tenant, sess.conn.ID)
	if err != nil {
		return "", err
	}
	sess.conn = current
	if current.ChangeCursor == "" {
		return cursor, nil
	}
	return current.ChangeCursor, nil
}

func (s *Service) recordReadFailure(ctx context.Context, sess *readSession, readErr error) {
	failure := &accountingsync.SyncError{
		Category: accountingsync.SyncErrorTransient,
		Message:  readErr.Error(),
	}
	if sess.writer != nil {
		if classified := sess.writer.ClassifyDocumentError(readErr); classified != nil {
			failure = classified
		}
	}
	if _, err := s.connections.SaveChangeFeed(ctx, &repositories.SaveAccountingChangeFeedRequest{
		TenantInfo:    sess.tenant,
		ID:            sess.conn.ID,
		ErrorCategory: failure.Category,
		ErrorMessage:  failure.Message,
	}); err != nil {
		s.l.Warn("failed to record an accounting change feed failure", zap.Error(err))
	}
}

func (s *Service) applyReferences(
	ctx context.Context,
	sess *readSession,
	changed []services.AccountingChangedReference,
	seenAt int64,
) (int, error) {
	if len(changed) == 0 {
		return 0, nil
	}
	upserts := make([]*accountingsync.AccountingReferenceObject, 0, len(changed))
	removed := make(
		map[accountingsync.ReferenceKind][]string,
		len(accountingsync.AllReferenceKinds()),
	)
	for _, ref := range changed {
		if ref.Object == nil || ref.Object.ExternalID == "" {
			continue
		}
		if ref.Deleted {
			removed[ref.Object.Kind] = append(removed[ref.Object.Kind], ref.Object.ExternalID)
			continue
		}
		upserts = append(upserts, ref.Object)
	}
	if err := s.references.Upsert(ctx, &repositories.UpsertAccountingReferenceObjectsRequest{
		TenantInfo:   sess.tenant,
		ConnectionID: sess.conn.ID,
		Objects:      upserts,
		SeenAt:       seenAt,
	}); err != nil {
		return 0, err
	}
	for kind, ids := range removed {
		if _, err := s.references.MarkRemoved(
			ctx,
			&repositories.MarkAccountingReferencesRemovedRequest{
				TenantInfo:   sess.tenant,
				ConnectionID: sess.conn.ID,
				Kind:         kind,
				ExternalIDs:  ids,
				At:           seenAt,
			},
		); err != nil {
			return 0, err
		}
	}
	return len(changed), nil
}

func (s *Service) recordPayments(
	ctx context.Context,
	sess *readSession,
	payments []services.AccountingInboundPayment,
	at int64,
) (int, error) {
	recorded := 0
	for _, kind := range accountingsync.AllInboundChangeKinds() {
		count, err := s.recordKind(ctx, sess, kind, paymentsOf(payments, kind), at)
		if err != nil {
			return 0, err
		}
		recorded += count
	}
	return recorded, nil
}

func paymentsOf(
	payments []services.AccountingInboundPayment,
	kind accountingsync.InboundChangeKind,
) []*services.AccountingInboundPayment {
	out := make([]*services.AccountingInboundPayment, 0, len(payments))
	for idx := range payments {
		if payments[idx].Kind == kind && payments[idx].ExternalID != "" {
			out = append(out, &payments[idx])
		}
	}
	return out
}

func (s *Service) recordKind(
	ctx context.Context,
	sess *readSession,
	kind accountingsync.InboundChangeKind,
	payments []*services.AccountingInboundPayment,
	at int64,
) (int, error) {
	if len(payments) == 0 {
		return 0, nil
	}
	ids := make([]string, 0, len(payments))
	for _, payment := range payments {
		ids = append(ids, payment.ExternalID)
	}

	echoes, err := s.echoes(ctx, sess.tenant, sess.conn.ID, kind, ids)
	if err != nil {
		return 0, err
	}
	existing, err := s.changes.ListByExternalIDs(
		ctx,
		&repositories.ListAccountingInboundChangesByExternalIDsRequest{
			TenantInfo:   sess.tenant,
			ConnectionID: sess.conn.ID,
			Kind:         kind,
			ExternalIDs:  ids,
		},
	)
	if err != nil {
		return 0, err
	}
	byID := make(map[string]*accountingsync.AccountingInboundChange, len(existing))
	for _, change := range existing {
		byID[change.ExternalID] = change
	}

	policy := sess.conn.PaymentPolicy()
	recorded := 0
	for _, payment := range payments {
		if echoes[payment.ExternalID] {
			continue
		}
		change := byID[payment.ExternalID]
		changed, recordErr := s.recordOne(ctx, sess, policy, change, payment, at)
		if recordErr != nil {
			return 0, recordErr
		}
		if changed {
			recorded++
		}
	}
	return recorded, nil
}

func (s *Service) echoes(
	ctx context.Context,
	tenant pagination.TenantInfo,
	connectionID pulid.ID,
	kind accountingsync.InboundChangeKind,
	ids []string,
) (map[string]bool, error) {
	records, err := s.records.ListByExternalIDs(
		ctx,
		&repositories.ListAccountingSyncRecordsByExternalIDsRequest{
			TenantInfo:   tenant,
			ConnectionID: connectionID,
			ObjectTypes:  kind.EchoObjectTypes(),
			ExternalIDs:  ids,
		},
	)
	if err != nil {
		return nil, err
	}
	out := make(map[string]bool, len(records))
	for _, record := range records {
		out[record.ExternalID] = true
	}
	return out, nil
}

func (s *Service) recordOne(
	ctx context.Context,
	sess *readSession,
	policy accountingsync.InboundPaymentPolicy,
	change *accountingsync.AccountingInboundChange,
	payment *services.AccountingInboundPayment,
	at int64,
) (bool, error) {
	if payment.Operation != services.AccountingChangeUpsert {
		if change == nil {
			return false, nil
		}
		if !change.Supersede(accountingsync.InboundReasonVoided, removedResolution(sess, payment)) {
			return false, nil
		}
		_, err := s.changes.Update(ctx, change)
		return err == nil, err
	}

	obs, err := s.observation(sess, payment, at)
	if err != nil {
		s.l.Warn("skipping a provider payment Trenova cannot read",
			zap.String("externalId", payment.ExternalID), zap.Error(err))
		return false, nil
	}
	if change == nil {
		if policy == accountingsync.InboundPaymentsOff {
			return false, nil
		}
		_, err = s.changes.Create(ctx, accountingsync.NewAccountingInboundChange(obs))
		return err == nil, err
	}
	if !change.Observe(obs) {
		return false, nil
	}
	_, err = s.changes.Update(ctx, change)
	return err == nil, err
}

func (s *Service) observation(
	sess *readSession,
	payment *services.AccountingInboundPayment,
	at int64,
) (*accountingsync.InboundObservation, error) {
	txnDate, err := timeutils.ParseCalendarDate(payment.TxnDate, sess.loc)
	if err != nil {
		return nil, fmt.Errorf("payment date %q: %w", payment.TxnDate, err)
	}
	currency := payment.CurrencyCode
	if currency == "" {
		currency = sess.conn.ExternalHomeCurrency
	}
	lines := make([]*accountingsync.InboundLine, 0, len(payment.Lines))
	for _, line := range payment.Lines {
		lines = append(lines, &accountingsync.InboundLine{
			DocumentKind:       line.DocumentKind,
			DocumentExternalID: line.DocumentExternalID,
			AmountMinor:        money.MinorUnits(line.Amount),
		})
	}
	var modifiedAt *int64
	if payment.ModifiedAt > 0 {
		modified := payment.ModifiedAt
		modifiedAt = &modified
	}
	externalURL := ""
	if sess.writer != nil {
		externalURL = sess.writer.DocumentURL(payment.Kind.SyncObjectType(), payment.ExternalID)
	}
	return &accountingsync.InboundObservation{
		TenantInfo:         sess.tenant,
		ConnectionID:       sess.conn.ID,
		Kind:               payment.Kind,
		ExternalID:         payment.ExternalID,
		ExternalNumber:     payment.Number,
		ExternalURL:        externalURL,
		ProviderModifiedAt: modifiedAt,
		ProviderModifiedBy: payment.ModifiedBy,
		TxnDate:            txnDate,
		AmountMinor:        money.MinorUnits(payment.Amount),
		CurrencyCode:       currency,
		PartyExternalID:    payment.PartyExternalID,
		PartyName:          payment.PartyName,
		Document: accountingsync.InboundDocument{
			ReferenceNumber:   payment.ReferenceNumber,
			MethodExternalID:  payment.MethodExternalID,
			MethodName:        payment.MethodName,
			AccountExternalID: payment.AccountExternalID,
			UnappliedMinor:    money.MinorUnits(payment.Unapplied),
			Lines:             lines,
		},
		At: at,
	}, nil
}

func removedResolution(sess *readSession, payment *services.AccountingInboundPayment) string {
	provider := accountingsync.ProviderName(sess.conn.IntegrationType)
	if payment.Operation == services.AccountingChangeDelete {
		return "Deleted in " + provider + " before it was applied, so there is nothing to apply."
	}
	return "Voided in " + provider + " before it was applied, so there is nothing to apply."
}
