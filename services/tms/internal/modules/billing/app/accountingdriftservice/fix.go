package accountingdriftservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingdrift"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type fixPlan struct {
	finding     *accountingdrift.AccountingDriftFinding
	conn        *accountingconnection.AccountingConnection
	direction   accountingdrift.DriftDirection
	fixObject   accountingdrift.DriftFixObject
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
	actor *permissionports.RequestActor,
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
	finding *accountingdrift.AccountingDriftFinding,
	direction accountingdrift.DriftDirection,
) []requiredPermission {
	needed := []requiredPermission{{permission.ResourceAccountingSync, permission.OpUpdate}}
	if direction != accountingdrift.DriftAdjustTrenova {
		return needed
	}
	if finding.ObjectType == accountingsync.SyncObjectCustomerPayment {
		return append(
			needed,
			requiredPermission{permission.ResourceCustomerPayment, permission.OpUpdate},
		)
	}
	return append(
		needed,
		requiredPermission{permission.ResourceInvoice, permission.OpUpdate},
		requiredPermission{permission.ResourceInvoice, permission.OpPost},
	)
}

func fixError(err error) error {
	switch {
	case errors.Is(err, accountingdrift.ErrDriftClosed):
		return errortypes.NewValidationError("status", errortypes.ErrInvalidOperation, err.Error())
	case errors.Is(err, accountingdrift.ErrDriftFixUnavailable),
		errors.Is(err, accountingdrift.ErrDriftExplainOnly):
		return errortypes.NewValidationError("direction", errortypes.ErrInvalid, err.Error())
	case errors.Is(err, accountingdrift.ErrDriftNoteRequired):
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
) (*accountingdrift.AccountingDriftFinding, error) {
	return s.findings.GetByID(ctx, accountingdriftports.GetAccountingDriftFindingQuery{
		TenantInfo: tenant,
		ID:         id,
		ForUpdate:  forUpdate,
	})
}

func (s *Service) PreviewResolve(
	ctx context.Context,
	req *accountingdriftports.ResolveAccountingDriftRequest,
	actor *permissionports.RequestActor,
) (*accountingdriftports.AccountingDriftFixPreview, error) {
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
	req *accountingdriftports.ResolveAccountingDriftRequest,
	actor *permissionports.RequestActor,
) (*accountingdrift.AccountingDriftFinding, error) {
	current, err := s.findingByID(ctx, req.TenantInfo, req.ID, false)
	if err != nil {
		return nil, err
	}
	if err = s.authorize(ctx, actor, fixPermissions(current, req.Direction)); err != nil {
		return nil, err
	}

	var updated *accountingdrift.AccountingDriftFinding
	var previous map[string]any
	kick := false
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		finding, txErr := s.findingByID(txCtx, req.TenantInfo, req.ID, true)
		if txErr != nil {
			return txErr
		}
		previous = jsonutils.MustToJSON(finding)
		plan, txErr := s.plan(txCtx, finding, req.Direction)
		if txErr != nil {
			return txErr
		}
		if plan.direction == accountingdrift.DriftPushTrenovaValue {
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

func resolveComment(direction accountingdrift.DriftDirection) string {
	if direction == accountingdrift.DriftAdjustTrenova {
		return "Adjusted Trenova to match the accounting system"
	}
	return "Sent Trenova's value to the accounting system"
}

func (s *Service) plan(
	ctx context.Context,
	finding *accountingdrift.AccountingDriftFinding,
	direction accountingdrift.DriftDirection,
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
			accountingprovider.ProviderName(conn.IntegrationType),
		)
	}
	records, err := s.records.ListByObjects(
		ctx,
		&accountingsyncports.ListAccountingSyncRecordsByObjectsRequest{
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
				accountingprovider.ProviderName(conn.IntegrationType),
			)
		}
	}

	plan := &fixPlan{finding: finding, conn: conn, direction: direction}
	if direction == accountingdrift.DriftPushTrenovaValue {
		return plan, planPush(plan, records)
	}
	return plan, s.planAdjust(ctx, plan)
}

func pushOperation(kind accountingdrift.DriftKind) accountingsync.SyncOperation {
	switch kind {
	case accountingdrift.DriftDeletedInProvider, accountingdrift.DriftVoidedInProvider:
		return accountingsync.SyncOperationRecreate
	case accountingdrift.DriftStatusMismatch:
		return accountingsync.SyncOperationVoid
	case accountingdrift.DriftAmountMismatch, accountingdrift.DriftCustomerBalanceMismatch:
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
			accountingprovider.ProviderName(plan.conn.IntegrationType),
		)
	}
	plan.document = documents[0]
	for _, record := range records {
		plan.revision = max(plan.revision, record.Revision)
	}
	plan.revision++
	plan.operation = pushOperation(plan.finding.Kind)
	plan.fixObject = accountingdrift.DriftFixSyncRecord
	return nil
}

func (s *Service) planAdjust(ctx context.Context, plan *fixPlan) error {
	finding := plan.finding
	if finding.ObjectType == accountingsync.SyncObjectCustomerPayment {
		payment, err := s.payments.Get(ctx, &customerpaymentports.GetCustomerPaymentRequest{
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
		plan.fixObject = accountingdrift.DriftFixPaymentReversal
		plan.amountMinor = payment.AmountMinor
		return nil
	}

	inv, err := s.invoiceRepo.GetByID(ctx, invoiceports.GetInvoiceByIDRequest{
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
		plan.fixObject = accountingdrift.DriftFixInvoiceVoid
		plan.amountMinor = inv.TotalAmountMinor
		return nil
	}

	if finding.DifferenceMinor == nil || *finding.DifferenceMinor == 0 {
		return fixError(accountingdrift.ErrDriftFixUnavailable)
	}
	difference := *finding.DifferenceMinor
	plan.fixObject = accountingdrift.DriftFixDebitMemo
	plan.amountMinor = difference
	if difference < 0 {
		plan.fixObject = accountingdrift.DriftFixCreditMemo
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
			&accountingsyncports.SupersedeAccountingSyncRecordsRequest{
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
	actor *permissionports.RequestActor,
) error {
	collectCtx, collector := accountingsyncports.CollectAccountingSync(ctx)
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
	provider := accountingprovider.ProviderName(plan.conn.IntegrationType)
	switch plan.fixObject {
	case accountingdrift.DriftFixInvoiceVoid:
		return "Voided to match " + provider + ", where " + plan.finding.ObjectNumber +
			" was " + goneWord(plan.finding.Kind)
	case accountingdrift.DriftFixPaymentReversal:
		return "Reversed to match " + provider + ", where payment " + plan.finding.ObjectNumber +
			" was " + goneWord(plan.finding.Kind)
	case accountingdrift.DriftFixCreditMemo, accountingdrift.DriftFixDebitMemo:
		return "Brings " + plan.finding.ObjectNumber + " to its total in " + provider + ": " +
			money.FormatMinor(*plan.finding.ProviderMinor, plan.finding.CurrencyCode) +
			" instead of " + money.FormatMinor(*plan.finding.TrenovaMinor, plan.finding.CurrencyCode)
	case accountingdrift.DriftFixSyncRecord:
		return ""
	default:
		return ""
	}
}

func goneWord(kind accountingdrift.DriftKind) string {
	if kind == accountingdrift.DriftVoidedInProvider {
		return "voided"
	}
	return "deleted"
}

func (s *Service) runAdjustment(
	ctx context.Context,
	plan *fixPlan,
	actor *permissionports.RequestActor,
) (pulid.ID, error) {
	tenant := plan.finding.TenantInfo()
	reason := s.adjustmentReason(plan)
	switch plan.fixObject {
	case accountingdrift.DriftFixPaymentReversal:
		reversed, err := s.payments.Reverse(
			ctx,
			&customerpaymentports.ReverseCustomerPaymentRequest{
				PaymentID:      plan.payment.ID,
				AccountingDate: s.now().Unix(),
				Reason:         reason,
				TenantInfo:     tenant,
			},
			actor,
		)
		if err != nil {
			return pulid.Nil, err
		}
		return reversed.ID, nil
	case accountingdrift.DriftFixInvoiceVoid:
		voided, err := s.invoices.VoidInvoice(ctx, &invoiceports.VoidInvoiceRequest{
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
	case accountingdrift.DriftFixCreditMemo, accountingdrift.DriftFixDebitMemo:
		return s.postMemo(ctx, plan, reason, actor)
	case accountingdrift.DriftFixSyncRecord:
		return pulid.Nil, fixError(accountingdrift.ErrDriftFixUnavailable)
	default:
		return pulid.Nil, fixError(accountingdrift.ErrDriftFixUnavailable)
	}
}

func (s *Service) postMemo(
	ctx context.Context,
	plan *fixPlan,
	reason string,
	actor *permissionports.RequestActor,
) (pulid.ID, error) {
	tenant := plan.finding.TenantInfo()
	billType := billingqueue.BillTypeDebitMemo
	if plan.fixObject == accountingdrift.DriftFixCreditMemo {
		billType = billingqueue.BillTypeCreditMemo
	}
	memo, err := s.invoices.CreateMemo(ctx, &invoiceports.CreateMemoRequest{
		TenantInfo: tenant,
		CustomerID: plan.invoice.CustomerID,
		BillType:   billType,
		Lines: []*invoiceports.CreateMemoLineInput{{
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
	if _, err = s.payments.ApplyCreditMemo(ctx, &customerpaymentports.ApplyCreditMemoRequest{
		CreditMemoID:   memo.ID,
		AccountingDate: s.now().Unix(),
		Applications: []*customerpaymentports.CreditMemoApplicationInput{{
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
	current, err := s.records.GetByIDs(
		ctx,
		accountingsyncports.GetAccountingSyncRecordsByIDsRequest{
			TenantInfo: plan.finding.TenantInfo(),
			IDs:        ids,
		},
	)
	if err != nil {
		return err
	}
	reason := "Made in Trenova to match " + accountingprovider.ProviderName(
		plan.conn.IntegrationType,
	) +
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
	finding *accountingdrift.AccountingDriftFinding,
) {
	if s.dispatcher == nil {
		return
	}
	if err := s.dispatcher.Kick(ctx, finding.TenantInfo(), finding.ConnectionID); err != nil {
		s.l.Warn("failed to wake the accounting dispatcher; the schedule will pick it up",
			zap.String("connectionId", finding.ConnectionID.String()), zap.Error(err))
	}
}
