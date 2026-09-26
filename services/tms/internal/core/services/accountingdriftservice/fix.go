package accountingdriftservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type fixPlan struct {
	finding     *accountingsync.AccountingDriftFinding
	conn        *accountingsync.AccountingConnection
	direction   accountingsync.DriftDirection
	fixObject   accountingsync.DriftFixObject
	operation   accountingsync.SyncOperation
	amountMinor int64
	document    *accountingsync.AccountingSyncRecord
	revision    int64
	invoice     *invoice.Invoice
	payment     *customerpayment.Payment
}

type requiredPermission struct {
	resource  permission.Resource
	operation permission.Operation
}

func (s *Service) authorize(
	ctx context.Context,
	actor *services.RequestActor,
	needed []requiredPermission,
) error {
	if actor == nil || actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError(
			"Fixing a difference with the accounting system requires an authenticated user",
		)
	}
	for _, perm := range needed {
		result, err := s.permissions.Check(
			ctx,
			actor.PermissionCheck(perm.resource, perm.operation),
		)
		if err != nil {
			return err
		}
		if result == nil || !result.Allowed {
			return errortypes.NewAuthorizationError(
				"This fix changes {0}, which you do not have permission to change",
				perm.resource.String(),
			)
		}
	}
	return nil
}

func fixPermissions(
	finding *accountingsync.AccountingDriftFinding,
	direction accountingsync.DriftDirection,
) []requiredPermission {
	needed := []requiredPermission{{permission.ResourceAccountingSync, permission.OpUpdate}}
	if direction != accountingsync.DriftAdjustTrenova {
		return needed
	}
	if finding.ObjectType == accountingsync.SyncObjectCustomerPayment {
		return append(
			needed,
			requiredPermission{permission.ResourceCustomerPayment, permission.OpUpdate},
		)
	}
	return append(needed, requiredPermission{permission.ResourceInvoice, permission.OpUpdate})
}

func fixError(err error) error {
	switch {
	case errors.Is(err, accountingsync.ErrDriftClosed):
		return errortypes.NewValidationError("status", errortypes.ErrInvalidOperation, err.Error())
	case errors.Is(err, accountingsync.ErrDriftFixUnavailable):
		return errortypes.NewValidationError("direction", errortypes.ErrInvalid, err.Error())
	case errors.Is(err, accountingsync.ErrDriftNoteRequired):
		return errortypes.NewValidationError("note", errortypes.ErrRequired, err.Error())
	default:
		return err
	}
}

func (s *Service) findingByID(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	forUpdate bool,
) (*accountingsync.AccountingDriftFinding, error) {
	return s.findings.GetByID(ctx, repositories.GetAccountingDriftFindingRequest{
		TenantInfo: tenant,
		ID:         id,
		ForUpdate:  forUpdate,
	})
}

func (s *Service) PreviewResolve(
	ctx context.Context,
	req *services.ResolveAccountingDriftRequest,
	actor *services.RequestActor,
) (*services.AccountingDriftFixPreview, error) {
	finding, err := s.findingByID(ctx, req.TenantInfo, req.ID, false)
	if err != nil {
		return nil, err
	}
	if err = s.authorize(ctx, actor, fixPermissions(finding, req.Direction)); err != nil {
		return nil, err
	}
	plan, err := s.plan(ctx, finding, req.Direction)
	if err != nil {
		return nil, err
	}
	return s.preview(ctx, plan)
}

func (s *Service) Resolve(
	ctx context.Context,
	req *services.ResolveAccountingDriftRequest,
	actor *services.RequestActor,
) (*accountingsync.AccountingDriftFinding, error) {
	current, err := s.findingByID(ctx, req.TenantInfo, req.ID, false)
	if err != nil {
		return nil, err
	}
	if err = s.authorize(ctx, actor, fixPermissions(current, req.Direction)); err != nil {
		return nil, err
	}

	var updated *accountingsync.AccountingDriftFinding
	var previous map[string]any
	kick := false
	err = s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		finding, txErr := s.findingByID(txCtx, req.TenantInfo, req.ID, true)
		if txErr != nil {
			return txErr
		}
		previous = jsonutils.MustToJSON(finding)
		plan, txErr := s.plan(txCtx, finding, req.Direction)
		if txErr != nil {
			return txErr
		}
		if plan.direction == accountingsync.DriftPushTrenovaValue {
			kick, txErr = s.push(txCtx, plan, actor.UserID)
		} else {
			txErr = s.adjust(txCtx, plan, actor)
		}
		if txErr != nil {
			return txErr
		}
		updated, txErr = s.findings.Update(txCtx, plan.finding)
		return txErr
	})
	if err != nil {
		return nil, err
	}

	if kick {
		s.kickDispatcher(ctx, updated)
	}
	s.logAudit(updated, actor.UserID, previous, resolveComment(req.Direction))
	s.refreshAttentionFor(ctx, updated)
	s.publishInvalidation(ctx, req.TenantInfo, actor.UserID, updated.ID)
	return updated, nil
}

func resolveComment(direction accountingsync.DriftDirection) string {
	if direction == accountingsync.DriftAdjustTrenova {
		return "Adjusted Trenova to match the accounting system"
	}
	return "Sent Trenova's value to the accounting system"
}

func (s *Service) plan(
	ctx context.Context,
	finding *accountingsync.AccountingDriftFinding,
	direction accountingsync.DriftDirection,
) (*fixPlan, error) {
	if err := finding.CanFix(direction); err != nil {
		return nil, fixError(err)
	}
	conn, err := s.connectionByID(ctx, finding.TenantInfo(), finding.ConnectionID)
	if err != nil {
		return nil, err
	}
	if !conn.IsSyncing() {
		return nil, errortypes.NewBusinessError(
			"{0} is not syncing, so nothing can be sent to it or matched with it",
			accountingsync.ProviderName(conn.IntegrationType),
		)
	}
	records, err := s.records.ListByObjects(
		ctx,
		&repositories.ListAccountingSyncRecordsByObjectsRequest{
			TenantInfo:   finding.TenantInfo(),
			ConnectionID: finding.ConnectionID,
			ObjectTypes:  []accountingsync.SyncObjectType{finding.ObjectType},
			ObjectIDs:    []pulid.ID{finding.ObjectID},
		},
	)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		if !record.Status.IsFinal() {
			return nil, errortypes.NewBusinessError(
				"{0} already has a change on its way to {1}. Fix it once that change is sent",
				finding.ObjectNumber,
				accountingsync.ProviderName(conn.IntegrationType),
			)
		}
	}

	plan := &fixPlan{finding: finding, conn: conn, direction: direction}
	if direction == accountingsync.DriftPushTrenovaValue {
		return plan, planPush(plan, records)
	}
	return plan, s.planAdjust(ctx, plan)
}

func pushOperation(kind accountingsync.DriftKind) accountingsync.SyncOperation {
	switch kind {
	case accountingsync.DriftDeletedInProvider, accountingsync.DriftVoidedInProvider:
		return accountingsync.SyncOperationRecreate
	case accountingsync.DriftStatusMismatch:
		return accountingsync.SyncOperationVoid
	case accountingsync.DriftAmountMismatch, accountingsync.DriftCustomerBalanceMismatch:
		return accountingsync.SyncOperationUpdate
	default:
		return accountingsync.SyncOperationUpdate
	}
}

func planPush(plan *fixPlan, records []*accountingsync.AccountingSyncRecord) error {
	documents := latestDocumentRecords(records)
	if len(documents) == 0 {
		return errortypes.NewBusinessError(
			"{0} was never sent to {1}, so there is nothing to send again",
			plan.finding.ObjectNumber,
			accountingsync.ProviderName(plan.conn.IntegrationType),
		)
	}
	plan.document = documents[0]
	for _, record := range records {
		plan.revision = max(plan.revision, record.Revision)
	}
	plan.revision++
	plan.operation = pushOperation(plan.finding.Kind)
	plan.fixObject = accountingsync.DriftFixSyncRecord
	return nil
}

func (s *Service) planAdjust(ctx context.Context, plan *fixPlan) error {
	finding := plan.finding
	if finding.ObjectType == accountingsync.SyncObjectCustomerPayment {
		payment, err := s.payments.Get(ctx, &services.GetCustomerPaymentRequest{
			PaymentID:  finding.ObjectID,
			TenantInfo: finding.TenantInfo(),
		})
		if err != nil {
			return err
		}
		if payment.Status != customerpayment.StatusPosted {
			return errortypes.NewBusinessError(
				"Payment {0} is already reversed",
				finding.ObjectNumber,
			)
		}
		plan.payment = payment
		plan.fixObject = accountingsync.DriftFixPaymentReversal
		plan.amountMinor = payment.AmountMinor
		return nil
	}

	inv, err := s.invoiceRepo.GetByID(ctx, repositories.GetInvoiceByIDRequest{
		ID:         finding.ObjectID,
		TenantInfo: finding.TenantInfo(),
	})
	if err != nil {
		return err
	}
	if inv.Status != invoice.StatusPosted {
		return errortypes.NewBusinessError("Invoice {0} is not posted", inv.Number)
	}
	plan.invoice = inv
	if finding.Kind.Gone() {
		if inv.AppliedAmountMinor > 0 {
			return errortypes.NewBusinessError(
				"Invoice {0} has payments or credits applied. Unapply them before voiding it",
				inv.Number,
			)
		}
		plan.fixObject = accountingsync.DriftFixInvoiceVoid
		plan.amountMinor = inv.TotalAmountMinor
		return nil
	}

	if finding.DifferenceMinor == nil || *finding.DifferenceMinor == 0 {
		return fixError(accountingsync.ErrDriftFixUnavailable)
	}
	difference := *finding.DifferenceMinor
	plan.fixObject = accountingsync.DriftFixDebitMemo
	plan.amountMinor = difference
	if difference < 0 {
		plan.fixObject = accountingsync.DriftFixCreditMemo
		plan.amountMinor = -difference
	}
	return nil
}

func (s *Service) push(
	ctx context.Context,
	plan *fixPlan,
	actorID pulid.ID,
) (bool, error) {
	now := s.now().Unix()
	finding := plan.finding
	record := accountingsync.NewAccountingSyncRecord(&accountingsync.NewSyncRecord{
		TenantInfo:   finding.TenantInfo(),
		ConnectionID: finding.ConnectionID,
		Key: accountingsync.SyncRecordKey{
			ObjectType: finding.ObjectType,
			ObjectID:   finding.ObjectID,
			Operation:  plan.operation,
			Revision:   plan.revision,
		},
		ObjectNumber: plan.document.ObjectNumber,
		SourceEvent:  accountingsync.SyncSourceDriftResolved,
		DocumentDate: plan.document.DocumentDate,
		At:           now,
	})
	if !plan.conn.AutoSync {
		record.ReleasedByID = actorID
	}
	result, err := s.records.Enqueue(ctx, []*accountingsync.AccountingSyncRecord{record})
	if err != nil {
		return false, err
	}
	if len(result.Inserted) == 0 {
		return false, errortypes.NewConflictError(
			"{0} is already queued to be sent again",
			finding.ObjectNumber,
		)
	}
	if plan.operation == accountingsync.SyncOperationUpdate {
		if _, err = s.records.SupersedeOlder(
			ctx,
			&repositories.SupersedeAccountingSyncRecordsRequest{
				TenantInfo:     finding.TenantInfo(),
				ConnectionID:   finding.ConnectionID,
				ObjectType:     finding.ObjectType,
				ObjectID:       finding.ObjectID,
				Operation:      plan.operation,
				BeforeRevision: plan.revision,
			},
		); err != nil {
			return false, err
		}
	}
	finding.MarkPushed(result.Inserted[0].ID, actorID)
	return true, nil
}

func (s *Service) adjust(
	ctx context.Context,
	plan *fixPlan,
	actor *services.RequestActor,
) error {
	collectCtx, collector := services.CollectAccountingSync(ctx)
	fixID, err := s.runAdjustment(collectCtx, plan, actor)
	if err != nil {
		return err
	}
	if err = s.reflect(ctx, plan, collector.Records(), actor.UserID); err != nil {
		return err
	}
	plan.finding.MarkAdjusted(plan.fixObject, fixID, actor.UserID, s.now().Unix())
	return nil
}

func (s *Service) adjustmentReason(plan *fixPlan) string {
	provider := accountingsync.ProviderName(plan.conn.IntegrationType)
	switch plan.fixObject {
	case accountingsync.DriftFixInvoiceVoid:
		return "Voided to match " + provider + ", where " + plan.finding.ObjectNumber +
			" was " + goneWord(plan.finding.Kind)
	case accountingsync.DriftFixPaymentReversal:
		return "Reversed to match " + provider + ", where payment " + plan.finding.ObjectNumber +
			" was " + goneWord(plan.finding.Kind)
	case accountingsync.DriftFixCreditMemo, accountingsync.DriftFixDebitMemo:
		return "Brings " + plan.finding.ObjectNumber + " to its total in " + provider + ": " +
			money.FormatMinor(*plan.finding.ProviderMinor, plan.finding.CurrencyCode) +
			" instead of " + money.FormatMinor(*plan.finding.TrenovaMinor, plan.finding.CurrencyCode)
	case accountingsync.DriftFixSyncRecord:
		return ""
	default:
		return ""
	}
}

func goneWord(kind accountingsync.DriftKind) string {
	if kind == accountingsync.DriftVoidedInProvider {
		return "voided"
	}
	return "deleted"
}

func (s *Service) runAdjustment(
	ctx context.Context,
	plan *fixPlan,
	actor *services.RequestActor,
) (pulid.ID, error) {
	tenant := plan.finding.TenantInfo()
	reason := s.adjustmentReason(plan)
	switch plan.fixObject {
	case accountingsync.DriftFixPaymentReversal:
		reversed, err := s.payments.Reverse(ctx, &services.ReverseCustomerPaymentRequest{
			PaymentID:      plan.payment.ID,
			AccountingDate: s.now().Unix(),
			Reason:         reason,
			TenantInfo:     tenant,
		}, actor)
		if err != nil {
			return pulid.Nil, err
		}
		return reversed.ID, nil
	case accountingsync.DriftFixInvoiceVoid:
		voided, err := s.invoices.VoidInvoice(ctx, &services.VoidInvoiceRequest{
			InvoiceID:   plan.invoice.ID,
			TenantInfo:  tenant,
			Reason:      reason,
			Disposition: invoice.VoidDispositionDoNotRebill,
		}, actor)
		if err != nil {
			return pulid.Nil, err
		}
		if voided.PendingApproval {
			return pulid.Nil, errortypes.NewBusinessError(
				"Voiding {0} needs an approver, so it cannot be done in one step here. Void it from the invoice, then dismiss this finding",
				plan.invoice.Number,
			)
		}
		return plan.invoice.ID, nil
	case accountingsync.DriftFixCreditMemo, accountingsync.DriftFixDebitMemo:
		return s.postMemo(ctx, plan, reason, actor)
	case accountingsync.DriftFixSyncRecord:
		return pulid.Nil, fixError(accountingsync.ErrDriftFixUnavailable)
	default:
		return pulid.Nil, fixError(accountingsync.ErrDriftFixUnavailable)
	}
}

func (s *Service) postMemo(
	ctx context.Context,
	plan *fixPlan,
	reason string,
	actor *services.RequestActor,
) (pulid.ID, error) {
	tenant := plan.finding.TenantInfo()
	billType := billingqueue.BillTypeDebitMemo
	if plan.fixObject == accountingsync.DriftFixCreditMemo {
		billType = billingqueue.BillTypeCreditMemo
	}
	memo, err := s.invoices.CreateMemo(ctx, &services.CreateMemoRequest{
		TenantInfo: tenant,
		CustomerID: plan.invoice.CustomerID,
		BillType:   billType,
		Lines: []*services.CreateMemoLineInput{{
			Description: reason,
			Amount:      money.DecimalFromMinor(plan.amountMinor),
		}},
		ReferenceInvoiceID: plan.invoice.ID,
		Reason:             reason,
		MemoKind:           invoice.MemoKindManual,
		AutoPost:           true,
	}, actor)
	if err != nil {
		return pulid.Nil, err
	}
	applied := min(plan.amountMinor, plan.invoice.BalanceDueMinor)
	if billType != billingqueue.BillTypeCreditMemo || applied <= 0 {
		return memo.ID, nil
	}
	if _, err = s.payments.ApplyCreditMemo(ctx, &services.ApplyCreditMemoRequest{
		CreditMemoID:   memo.ID,
		AccountingDate: s.now().Unix(),
		Applications: []*services.CreditMemoApplicationInput{{
			InvoiceID:          plan.invoice.ID,
			AppliedAmountMinor: applied,
		}},
		TenantInfo: tenant,
	}, actor); err != nil {
		return pulid.Nil, err
	}
	return memo.ID, nil
}

func (s *Service) reflect(
	ctx context.Context,
	plan *fixPlan,
	enqueued []*accountingsync.AccountingSyncRecord,
	actorID pulid.ID,
) error {
	ids := make([]pulid.ID, 0, len(enqueued))
	for _, record := range enqueued {
		if record.ConnectionID == plan.finding.ConnectionID {
			ids = append(ids, record.ID)
		}
	}
	if len(ids) == 0 {
		return nil
	}
	current, err := s.records.GetByIDs(ctx, repositories.GetAccountingSyncRecordsByIDsRequest{
		TenantInfo: plan.finding.TenantInfo(),
		IDs:        ids,
	})
	if err != nil {
		return err
	}
	reason := "Made in Trenova to match " + accountingsync.ProviderName(plan.conn.IntegrationType) +
		"; already there"
	for _, record := range current {
		if !record.Reflect(actorID, plan.finding.ExternalID, reason) {
			continue
		}
		if _, err = s.records.Update(ctx, record); err != nil {
			return err
		}
	}
	return nil
}

func (s *Service) kickDispatcher(
	ctx context.Context,
	finding *accountingsync.AccountingDriftFinding,
) {
	if s.dispatcher == nil {
		return
	}
	if err := s.dispatcher.Kick(ctx, finding.TenantInfo(), finding.ConnectionID); err != nil {
		s.l.Warn("failed to wake the accounting dispatcher; the schedule will pick it up",
			zap.String("connectionId", finding.ConnectionID.String()), zap.Error(err))
	}
}
