package accountinginboundservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/customerpayment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	maxReferenceNumber = 100
	paymentMemoPrefix  = "Recorded in "
)

type blockedError struct {
	reason     accountingsync.InboundChangeReason
	resolution string
}

func (b *blockedError) Error() string { return b.resolution }

func (s *Service) Evaluate(
	ctx context.Context,
	req *services.EvaluateAccountingInboundRequest,
) (*services.AccountingInboundEvaluation, error) {
	result := new(services.AccountingInboundEvaluation)
	conn, err := s.connectionByID(ctx, req.TenantInfo, req.ConnectionID)
	if err != nil {
		return nil, err
	}
	if !conn.IsSyncing() {
		return result, nil
	}

	inFlight, err := s.records.CountInFlight(ctx, &repositories.CountAccountingSyncInFlightRequest{
		TenantInfo:   req.TenantInfo,
		ConnectionID: conn.ID,
		ObjectTypes:  echoTypes(),
	})
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
		&repositories.ListDetectedAccountingInboundChangesRequest{
			TenantInfo:     req.TenantInfo,
			ConnectionID:   conn.ID,
			DetectedBefore: s.now().Add(-accountingsync.InboundEvaluationSettle).Unix(),
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
		case accountingsync.InboundStatusApplied:
			result.Applied++
		case accountingsync.InboundStatusProposed:
			result.Proposed++
			if firstProposed.IsNil() {
				firstProposed = row.ID
			}
		case accountingsync.InboundStatusIgnored:
			result.Ignored++
		case accountingsync.InboundStatusSuperseded:
			result.Superseded++
		case accountingsync.InboundStatusDetected:
		}
	}
	result.More = len(rows) == limit

	if result.Evaluated > 0 {
		if !firstProposed.IsNil() {
			services.PublishAgentEvent(ctx, s.publisher, services.AgentEvent{
				Kind:       agent.EventAccountingPaymentProposed,
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
	for _, kind := range accountingsync.AllInboundChangeKinds() {
		out = append(out, kind.EchoObjectTypes()...)
	}
	return out
}

func (s *Service) evaluateOne(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
	id pulid.ID,
) (accountingsync.InboundChangeStatus, error) {
	autoApply := false
	var outcome accountingsync.InboundChangeStatus
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		change, err := s.changes.GetByID(txCtx, repositories.GetAccountingInboundChangeRequest{
			TenantInfo: tenantOf(conn),
			ID:         id,
			ForUpdate:  true,
		})
		if err != nil {
			return err
		}
		if change.Status != accountingsync.InboundStatusDetected {
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
		case conn.PaymentPolicy() == accountingsync.InboundPaymentsApply:
			autoApply = true
			outcome = accountingsync.InboundStatusDetected
			return nil
		default:
			change.Propose(accountingsync.InboundReasonPolicyPropose, proposeText(p))
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
) (*services.RequestActor, error) {
	user, err := s.users.GetSystemUser(ctx, "id")
	if err != nil {
		return nil, err
	}
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeSystem,
		PrincipalID:    user.ID,
		UserID:         user.ID,
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
	}, nil
}

func (s *Service) proposeAfterFailure(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
	id pulid.ID,
	applyErr error,
) (*accountingsync.AccountingInboundChange, error) {
	reason := accountingsync.InboundReasonApplyFailed
	resolution := applyErr.Error()
	var blocked *blockedError
	switch {
	case errors.As(applyErr, &blocked):
		reason = blocked.reason
		resolution = blocked.resolution
	case !isBusinessError(applyErr):
		return nil, applyErr
	}

	var updated *accountingsync.AccountingInboundChange
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		change, err := s.changes.GetByID(txCtx, repositories.GetAccountingInboundChangeRequest{
			TenantInfo: tenantOf(conn),
			ID:         id,
			ForUpdate:  true,
		})
		if err != nil {
			return err
		}
		if !change.Status.IsOpen() {
			updated = change
			return nil
		}
		if reason == accountingsync.InboundReasonNotTrenovaDocument ||
			reason == accountingsync.InboundReasonSentFromTrenova {
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
	req *services.DecideAccountingInboundChangeRequest,
	actor *services.RequestActor,
) (*accountingsync.AccountingInboundChange, error) {
	if actor == nil || actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError(
			"Applying a payment from the accounting system requires an authenticated user",
		)
	}
	current, err := s.changes.GetByID(ctx, repositories.GetAccountingInboundChangeRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
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

func (s *Service) applyChange(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
	id pulid.ID,
	actor *services.RequestActor,
	byPerson bool,
) (*accountingsync.AccountingInboundChange, error) {
	links := s.linkTarget(ctx, conn)

	var updated *accountingsync.AccountingInboundChange
	var previous map[string]any
	err := s.db.WithTx(ctx, ports.TxOptions{}, func(txCtx context.Context, _ bun.Tx) error {
		change, err := s.changes.GetByID(txCtx, repositories.GetAccountingInboundChangeRequest{
			TenantInfo: tenantOf(conn),
			ID:         id,
			ForUpdate:  true,
		})
		if err != nil {
			return err
		}
		previous = jsonutils.MustToJSON(change)
		if byPerson {
			if err = change.CanApply(); err != nil {
				return decisionError(err)
			}
		} else if !change.Status.IsOpen() {
			return decisionError(accountingsync.ErrInboundChangeClosed)
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
	actor *services.RequestActor,
) ([]accountingsync.AppliedObject, error) {
	change := p.change
	tenant := change.TenantInfo()
	objects := make([]accountingsync.AppliedObject, 0, p.objectCount())

	if change.Kind == accountingsync.InboundBillPayment {
		for _, settlement := range p.settlements {
			req := &services.MarkSettlementPaidRequest{
				TenantInfo:       tenant,
				SettlementID:     settlement.id,
				PaymentMethod:    p.method,
				PaymentReference: referenceOf(change),
				PaidAt:           p.paidAt,
			}
			var err error
			appliedType := accountingsync.AppliedCarrierSettlement
			if settlement.kind == repositories.PayableDriver {
				appliedType = accountingsync.AppliedDriverSettlement
				_, err = s.driverPayer.MarkPaid(ctx, req, actor)
			} else {
				_, err = s.carrierPayer.MarkPaid(ctx, req, actor)
			}
			if err != nil {
				return nil, err
			}
			objects = append(
				objects,
				accountingsync.AppliedObject{Type: appliedType, ID: settlement.id},
			)
		}
		return objects, nil
	}

	if change.AmountMinor > 0 {
		payment, err := s.customerPayments.PostAndApply(ctx, &services.PostCustomerPaymentRequest{
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
		}, actor)
		if err != nil {
			return nil, err
		}
		objects = append(objects, accountingsync.AppliedObject{
			Type: accountingsync.AppliedCustomerPayment,
			ID:   payment.ID,
		})
	}
	for _, credit := range p.credits {
		applications, err := s.customerPayments.ApplyCreditMemo(
			ctx,
			&services.ApplyCreditMemoRequest{
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
			objects = append(objects, accountingsync.AppliedObject{
				Type: accountingsync.AppliedCreditMemoApplication,
				ID:   application.ID,
			})
		}
	}
	return objects, nil
}

func referenceOf(change *accountingsync.AccountingInboundChange) string {
	reference := change.Document.ReferenceNumber
	if reference == "" {
		reference = change.ExternalNumber
	}
	return stringutils.TruncateRunes(reference, maxReferenceNumber)
}

func syncTypeOf(applied accountingsync.AppliedObjectType) accountingsync.SyncObjectType {
	switch applied {
	case accountingsync.AppliedCustomerPayment:
		return accountingsync.SyncObjectCustomerPayment
	case accountingsync.AppliedCreditMemoApplication:
		return accountingsync.SyncObjectCreditApplication
	case accountingsync.AppliedCarrierSettlement:
		return accountingsync.SyncObjectCarrierBillPay
	case accountingsync.AppliedDriverSettlement:
		return accountingsync.SyncObjectDriverBillPay
	default:
		return ""
	}
}

type linkTarget struct {
	urlOf func(kind accountingsync.SyncObjectType, externalID string) string
}

func (s *Service) linkTarget(
	ctx context.Context,
	conn *accountingsync.AccountingConnection,
) *linkTarget {
	target := &linkTarget{urlOf: func(accountingsync.SyncObjectType, string) string { return "" }}
	session, err := s.connService.Session(ctx, tenantOf(conn), conn.ID)
	if err != nil {
		s.l.Debug("linking without provider links: the connection cannot be used", zap.Error(err))
		return target
	}
	if writer, ok := session.Connector.(services.AccountingDocumentWriter); ok {
		target.urlOf = writer.DocumentURL
	}
	return target
}

func (s *Service) linkRecords(
	ctx context.Context,
	p *postingPlan,
	objects []accountingsync.AppliedObject,
	target *linkTarget,
) error {
	change := p.change
	combined := len(objects) > 1
	for _, object := range objects {
		objectType := syncTypeOf(object.Type)
		records, err := s.records.ListByObjects(
			ctx,
			&repositories.ListAccountingSyncRecordsByObjectsRequest{
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
				ExternalID:  change.ExternalID,
				ExternalURL: target.urlOf(objectType, change.ExternalID),
				Resolution:  linkedText(p),
				Combined:    combined,
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
	req *services.GetAccountingInboundChangeRequest,
) (*services.AccountingInboundApplyPreview, error) {
	change, err := s.changes.GetByID(ctx, repositories.GetAccountingInboundChangeRequest{
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

	preview := &services.AccountingInboundApplyPreview{
		Change:    change,
		PaidAt:    p.paidAt,
		CashMinor: p.cashMinor,
		Lines:     p.lines,
	}
	if change.Kind == accountingsync.InboundCustomerPayment {
		preview.UnappliedMinor = max(change.AmountMinor-p.cashMinor, 0)
	}
	switch {
	case !change.Status.IsOpen():
		preview.Blocker = accountingsync.ErrInboundChangeClosed.Error()
	case p.blocked():
		preview.Blocker = p.resolution
	default:
		preview.CanApply = true
	}
	return preview, nil
}
