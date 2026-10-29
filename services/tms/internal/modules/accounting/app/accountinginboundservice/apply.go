package accountinginboundservice

import (
	"context"
	"errors"
	"slices"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountinginbound"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountinginboundports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingpayablesports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/customerpayment"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/carriersettlementports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"go.uber.org/zap"
)

const (
	maxReferenceNumber = 100
	paymentMemoPrefix  = "Recorded in "
)

type blockedError struct {
	reason     accountinginbound.InboundChangeReason
	resolution string
}

func (b *blockedError) Error() string { return b.resolution }

func (s *Service) Evaluate(
	ctx context.Context,
	req *accountinginboundports.EvaluateAccountingInboundRequest,
) (*accountinginboundports.AccountingInboundEvaluation, error) {
	result := new(accountinginboundports.AccountingInboundEvaluation)
	conn, err := s.connectionByID(ctx, req.TenantInfo, req.ConnectionID)
	if err != nil {
		return nil, err
	}
	if !conn.IsSyncing() {
		return result, nil
	}

	inFlight, err := s.records.CountInFlight(
		ctx,
		&accountingsyncports.CountAccountingSyncInFlightRequest{
			TenantInfo:   req.TenantInfo,
			ConnectionID: conn.ID,
			ObjectTypes:  echoTypes(),
		},
	)
	if err != nil {
		return nil, err
	}
	if inFlight > 0 {
		result.Waiting = true
		return result, nil
	}

	limit := req.Limit
	if limit <= 0 {
		limit = defaultEvaluateLimit
	}
	rows, err := s.changes.ListDetected(
		ctx,
		&accountinginboundports.ListDetectedAccountingInboundChangesRequest{
			TenantInfo:     req.TenantInfo,
			ConnectionID:   conn.ID,
			DetectedBefore: s.now().Add(-accountinginbound.InboundEvaluationSettle).Unix(),
			Limit:          limit,
		},
	)
	if err != nil {
		return nil, err
	}

	var firstProposed pulid.ID
	for _, row := range rows {
		outcome, evalErr := s.evaluateOne(ctx, conn, row.ID)
		if evalErr != nil {
			return nil, evalErr
		}
		result.Evaluated++
		switch outcome {
		case accountinginbound.InboundStatusApplied:
			result.Applied++
		case accountinginbound.InboundStatusProposed:
			result.Proposed++
			if firstProposed.IsNil() {
				firstProposed = row.ID
			}
		case accountinginbound.InboundStatusIgnored:
			result.Ignored++
		case accountinginbound.InboundStatusSuperseded:
			result.Superseded++
		case accountinginbound.InboundStatusDetected:
		}
	}
	result.More = len(rows) == limit

	if result.Evaluated > 0 {
		if !firstProposed.IsNil() {
			agentports.PublishAgentEvent(ctx, s.publisher, agentports.AgentEvent{
				Kind:       agenttrigger.EventAccountingPaymentProposed,
				SubjectID:  firstProposed,
				TenantInfo: req.TenantInfo,
			})
		}
		s.refreshAttention(ctx, conn)
		s.publishInvalidation(ctx, req.TenantInfo, pulid.Nil, conn.ID)
	}
	return result, nil
}

func echoTypes() []accountingsync.SyncObjectType {
	out := make([]accountingsync.SyncObjectType, 0, 4)
	for _, kind := range accountinginbound.AllInboundChangeKinds() {
		out = append(out, kind.EchoObjectTypes()...)
	}
	return out
}

func (s *Service) evaluateOne(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
	id pulid.ID,
) (accountinginbound.InboundChangeStatus, error) {
	autoApply := false
	var outcome accountinginbound.InboundChangeStatus
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		change, err := s.changes.GetByID(
			txCtx,
			accountinginboundports.GetAccountingInboundChangeQuery{
				TenantInfo: tenantOf(conn),
				ID:         id,
				ForUpdate:  true,
			},
		)
		if err != nil {
			return err
		}
		if change.Status != accountinginbound.InboundStatusDetected {
			outcome = change.Status
			return nil
		}
		p, err := s.plan(txCtx, conn, change)
		if err != nil {
			return err
		}
		switch {
		case p.ignorable():
			change.Ignore(p.reason, p.resolution, pulid.Nil, s.nowUnix())
		case p.blocked():
			change.Propose(p.reason, p.resolution)
		case conn.PaymentPolicy() == accountingconnection.InboundPaymentsApply:
			autoApply = true
			outcome = accountinginbound.InboundStatusDetected
			return nil
		default:
			change.Propose(accountinginbound.InboundReasonPolicyPropose, proposeText(p))
		}
		outcome = change.Status
		_, err = s.changes.Update(txCtx, change)
		return err
	})
	if err != nil || !autoApply {
		return outcome, err
	}

	actor, err := s.systemActor(ctx, tenantOf(conn))
	if err != nil {
		return outcome, err
	}
	applied, err := s.applyChange(ctx, conn, id, actor, false)
	if err == nil {
		return applied.Status, nil
	}
	proposed, proposeErr := s.proposeAfterFailure(ctx, conn, id, err)
	if proposeErr != nil {
		return outcome, proposeErr
	}
	return proposed.Status, nil
}

func (s *Service) systemActor(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*permissionports.RequestActor, error) {
	user, err := s.users.GetSystemUser(ctx, "id")
	if err != nil {
		return nil, err
	}
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeSystem,
		PrincipalID:    user.ID,
		UserID:         user.ID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}, nil
}

func (s *Service) proposeAfterFailure(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
	id pulid.ID,
	applyErr error,
) (*accountinginbound.AccountingInboundChange, error) {
	reason := accountinginbound.InboundReasonApplyFailed
	resolution := applyErr.Error()
	var blocked *blockedError
	switch {
	case errors.As(applyErr, &blocked):
		reason = blocked.reason
		resolution = blocked.resolution
	case !isBusinessError(applyErr):
		return nil, applyErr
	}

	var updated *accountinginbound.AccountingInboundChange
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		change, err := s.changes.GetByID(
			txCtx,
			accountinginboundports.GetAccountingInboundChangeQuery{
				TenantInfo: tenantOf(conn),
				ID:         id,
				ForUpdate:  true,
			},
		)
		if err != nil {
			return err
		}
		if !change.Status.IsOpen() {
			updated = change
			return nil
		}
		if reason == accountinginbound.InboundReasonNotTrenovaDocument ||
			reason == accountinginbound.InboundReasonSentFromTrenova {
			change.Ignore(reason, resolution, pulid.Nil, s.nowUnix())
		} else {
			change.Propose(reason, resolution)
		}
		updated, err = s.changes.Update(txCtx, change)
		return err
	})
	return updated, err
}

func isBusinessError(err error) bool {
	return errortypes.IsError(err) || errortypes.IsBusinessError(err) ||
		errortypes.IsNotFoundError(err)
}

func (s *Service) Apply(
	ctx context.Context,
	req *accountinginboundports.DecideAccountingInboundChangeRequest,
	actor *permissionports.RequestActor,
) (*accountinginbound.AccountingInboundChange, error) {
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Applying a payment from the accounting system requires an authenticated user",
		)
	}
	current, err := s.changes.GetByID(ctx, accountinginboundports.GetAccountingInboundChangeQuery{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
		return nil, err
	}
	if err = s.authorizeApply(ctx, current, actor); err != nil {
		return nil, err
	}
	conn, err := s.connectionByID(ctx, req.TenantInfo, current.ConnectionID)
	if err != nil {
		return nil, err
	}

	applied, err := s.applyChange(ctx, conn, req.ID, actor, true)
	if err == nil {
		return applied, nil
	}
	var blocked *blockedError
	if !errors.As(err, &blocked) {
		return nil, err
	}
	if _, proposeErr := s.proposeAfterFailure(ctx, conn, req.ID, err); proposeErr != nil {
		return nil, proposeErr
	}
	s.afterDecision(ctx, current, actor.UserID)
	return nil, errortypes.NewValidationError("id", errortypes.ErrInvalid, blocked.resolution)
}

type requiredPermission struct {
	resource  permission.Resource
	operation permission.Operation
}

func applyPermissions(change *accountinginbound.AccountingInboundChange) []requiredPermission {
	if change.Kind == accountinginbound.InboundCustomerPayment {
		return []requiredPermission{{permission.ResourceCustomerPayment, permission.OpCreate}}
	}
	out := make([]requiredPermission, 0, 2)
	for _, line := range change.Document.Lines {
		var needed requiredPermission
		switch line.ObjectType { //nolint:exhaustive // only bills are paid by a bill payment
		case accountingsync.SyncObjectCarrierBill:
			needed = requiredPermission{permission.ResourceCarrierSettlement, permission.OpUpdate}
		case accountingsync.SyncObjectDriverBill:
			needed = requiredPermission{permission.ResourceDriverSettlement, permission.OpUpdate}
		default:
			continue
		}
		if !slices.Contains(out, needed) {
			out = append(out, needed)
		}
	}
	return out
}

func (s *Service) authorizeApply(
	ctx context.Context,
	change *accountinginbound.AccountingInboundChange,
	actor *permissionports.RequestActor,
) error {
	for _, needed := range applyPermissions(change) {
		result, err := s.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
			PrincipalType:  actor.PrincipalType,
			PrincipalID:    actor.PrincipalID,
			UserID:         actor.UserID,
			APIKeyID:       actor.APIKeyID,
			BusinessUnitID: actor.BusinessUnitID,
			OrganizationID: actor.OrganizationID,
			Resource:       needed.resource.String(),
			Operation:      needed.operation,
		})
		if err != nil {
			return err
		}
		if result == nil || !result.Allowed {
			return errortypes.NewAuthorizationError(
				"Applying this payment posts to {0}, which you do not have permission to change",
				needed.resource.String(),
			)
		}
	}
	return nil
}

func (s *Service) applyChange(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
	id pulid.ID,
	actor *permissionports.RequestActor,
	byPerson bool,
) (*accountinginbound.AccountingInboundChange, error) {
	links := s.linkTarget(ctx, conn)

	var updated *accountinginbound.AccountingInboundChange
	var previous map[string]any
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		change, err := s.changes.GetByID(
			txCtx,
			accountinginboundports.GetAccountingInboundChangeQuery{
				TenantInfo: tenantOf(conn),
				ID:         id,
				ForUpdate:  true,
			},
		)
		if err != nil {
			return err
		}
		previous = jsonutils.MustToJSON(change)
		if byPerson {
			if err = change.CanApply(); err != nil {
				return decisionError(err)
			}
		} else if !change.Status.IsOpen() {
			return decisionError(accountinginbound.ErrInboundChangeClosed)
		}

		p, err := s.plan(txCtx, conn, change)
		if err != nil {
			return err
		}
		if p.blocked() {
			return &blockedError{reason: p.reason, resolution: p.resolution}
		}

		objects, err := s.execute(txCtx, p, actor)
		if err != nil {
			return err
		}
		if err = s.linkRecords(txCtx, p, objects, links); err != nil {
			return err
		}
		change.MarkApplied(objects, actor.UserID, appliedText(p), s.nowUnix())
		updated, err = s.changes.Update(txCtx, change)
		return err
	})
	if err != nil {
		return nil, err
	}

	s.logAudit(updated, actor.UserID, previous, "Applied a payment recorded in the books")
	s.afterDecision(ctx, updated, actor.UserID)
	return updated, nil
}

func (s *Service) execute(
	ctx context.Context,
	p *postingPlan,
	actor *permissionports.RequestActor,
) ([]accountinginbound.AppliedObject, error) {
	change := p.change
	tenant := change.TenantInfo()
	objects := make([]accountinginbound.AppliedObject, 0, p.objectCount())

	if change.Kind == accountinginbound.InboundBillPayment {
		for _, settlement := range p.settlements {
			req := &carriersettlementports.MarkSettlementPaidRequest{
				TenantInfo:       tenant,
				SettlementID:     settlement.id,
				PaymentMethod:    p.method,
				PaymentReference: referenceOf(change),
				PaidAt:           p.paidAt,
			}
			var err error
			appliedType := accountinginbound.AppliedCarrierSettlement
			if settlement.kind == accountingpayablesports.PayableDriver {
				appliedType = accountinginbound.AppliedDriverSettlement
				_, err = s.driverPayer.MarkPaid(ctx, req, actor)
			} else {
				_, err = s.carrierPayer.MarkPaid(ctx, req, actor)
			}
			if err != nil {
				return nil, err
			}
			objects = append(
				objects,
				accountinginbound.AppliedObject{Type: appliedType, ID: settlement.id},
			)
		}
		return objects, nil
	}

	if change.AmountMinor > 0 {
		payment, err := s.customerPayments.PostAndApply(
			ctx,
			&customerpaymentports.PostCustomerPaymentRequest{
				CustomerID:      p.customerID,
				PaymentDate:     p.paidAt,
				AccountingDate:  p.paidAt,
				AmountMinor:     change.AmountMinor,
				PaymentMethod:   customerpayment.Method(p.method),
				ReferenceNumber: referenceOf(change),
				Memo:            paymentMemoPrefix + p.provider + " as " + paymentName(p),
				CurrencyCode:    change.CurrencyCode,
				Applications:    p.cash,
				TenantInfo:      tenant,
			},
			actor,
		)
		if err != nil {
			return nil, err
		}
		objects = append(objects, accountinginbound.AppliedObject{
			Type: accountinginbound.AppliedCustomerPayment,
			ID:   payment.ID,
		})
	}
	for _, credit := range p.credits {
		applications, err := s.customerPayments.ApplyCreditMemo(
			ctx,
			&customerpaymentports.ApplyCreditMemoRequest{
				CreditMemoID:   credit.memoID,
				AccountingDate: p.paidAt,
				Applications:   credit.applications,
				TenantInfo:     tenant,
			},
			actor,
		)
		if err != nil {
			return nil, err
		}
		for _, application := range applications {
			objects = append(objects, accountinginbound.AppliedObject{
				Type: accountinginbound.AppliedCreditMemoApplication,
				ID:   application.ID,
			})
		}
	}
	return objects, nil
}

func referenceOf(change *accountinginbound.AccountingInboundChange) string {
	reference := change.Document.ReferenceNumber
	if reference == "" {
		reference = change.ExternalNumber
	}
	return stringutils.TruncateRunes(reference, maxReferenceNumber)
}

func syncTypeOf(applied accountinginbound.AppliedObjectType) accountingsync.SyncObjectType {
	switch applied {
	case accountinginbound.AppliedCustomerPayment:
		return accountingsync.SyncObjectCustomerPayment
	case accountinginbound.AppliedCreditMemoApplication:
		return accountingsync.SyncObjectCreditApplication
	case accountinginbound.AppliedCarrierSettlement:
		return accountingsync.SyncObjectCarrierBillPay
	case accountinginbound.AppliedDriverSettlement:
		return accountingsync.SyncObjectDriverBillPay
	default:
		return ""
	}
}

type linkTarget struct {
	urlOf func(link accountingconnectorports.AccountingDocumentLink) string
}

func (s *Service) linkTarget(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
) *linkTarget {
	target := &linkTarget{
		urlOf: func(accountingconnectorports.AccountingDocumentLink) string { return "" },
	}
	session, err := s.connService.Session(ctx, tenantOf(conn), conn.ID)
	if err != nil {
		s.l.Debug("linking without provider links: the connection cannot be used", zap.Error(err))
		return target
	}
	if writer, ok := session.Connector.(accountingconnectorports.AccountingDocumentWriter); ok {
		auth := accountingconnectorports.DocumentAuthFor(session.Connection, session.AccessToken)
		target.urlOf = func(link accountingconnectorports.AccountingDocumentLink) string {
			return writer.DocumentURL(auth, link)
		}
	}
	return target
}

func (s *Service) linkRecords(
	ctx context.Context,
	p *postingPlan,
	objects []accountinginbound.AppliedObject,
	target *linkTarget,
) error {
	change := p.change
	combined := len(objects) > 1
	for _, object := range objects {
		objectType := syncTypeOf(object.Type)
		records, err := s.records.ListByObjects(
			ctx,
			&accountingsyncports.ListAccountingSyncRecordsByObjectsRequest{
				TenantInfo:   change.TenantInfo(),
				ConnectionID: change.ConnectionID,
				ObjectTypes:  []accountingsync.SyncObjectType{objectType},
				ObjectIDs:    []pulid.ID{object.ID},
			},
		)
		if err != nil {
			return err
		}
		for _, record := range records {
			if record.Operation != accountingsync.SyncOperationCreate {
				continue
			}
			if !record.Link(&accountingsync.SyncLink{
				ExternalID: change.ExternalID,
				ExternalURL: target.urlOf(accountingconnectorports.AccountingDocumentLink{
					Kind:       objectType,
					ExternalID: change.ExternalID,
					DocNumber:  change.ExternalNumber,
				}),
				Resolution: linkedText(p),
				Combined:   combined,
			}, s.nowUnix()) {
				continue
			}
			if _, err = s.records.Update(ctx, record); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) PreviewApply(
	ctx context.Context,
	req *accountinginboundports.GetAccountingInboundChangeRequest,
) (*accountinginboundports.AccountingInboundApplyPreview, error) {
	change, err := s.changes.GetByID(ctx, accountinginboundports.GetAccountingInboundChangeQuery{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
		return nil, err
	}
	conn, err := s.connectionByID(ctx, req.TenantInfo, change.ConnectionID)
	if err != nil {
		return nil, err
	}
	p, err := s.plan(ctx, conn, change)
	if err != nil {
		return nil, err
	}

	preview := &accountinginboundports.AccountingInboundApplyPreview{
		Change:    change,
		PaidAt:    p.paidAt,
		CashMinor: p.cashMinor,
		Lines:     p.lines,
	}
	if change.Kind == accountinginbound.InboundCustomerPayment {
		preview.UnappliedMinor = max(change.AmountMinor-p.cashMinor, 0)
	}
	switch {
	case !change.Status.IsOpen():
		preview.Blocker = accountinginbound.ErrInboundChangeClosed.Error()
	case p.blocked():
		preview.Blocker = p.resolution
	default:
		preview.CanApply = true
	}
	return preview, nil
}
