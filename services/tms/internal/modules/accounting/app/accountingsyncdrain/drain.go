package accountingsyncdrain

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncledger"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncparty"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpayables"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncpush"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncsales"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectorports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/accountingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const defaultDrainLimit = 25

type Params struct {
	fx.In

	Logger   *zap.Logger
	Records  accountingsyncports.AccountingSyncRecordRepository
	Sessions accountingconnectionports.AccountingSessionProvider
	Controls tenantports.AccountingControlRepository
	Effects  *accountingsynceffects.Effects
	Parties  *accountingsyncparty.Pusher
	Sales    *accountingsyncsales.Pusher
	Payables *accountingsyncpayables.Pusher
	Ledger   *accountingsyncledger.Pusher
}

type Service struct {
	l        *zap.Logger
	records  accountingsyncports.AccountingSyncRecordRepository
	sessions accountingconnectionports.AccountingSessionProvider
	controls tenantports.AccountingControlRepository
	effects  *accountingsynceffects.Effects
	parties  *accountingsyncparty.Pusher
	sales    *accountingsyncsales.Pusher
	payables *accountingsyncpayables.Pusher
	ledger   *accountingsyncledger.Pusher
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:        p.Logger.Named("service.accounting-sync"),
		records:  p.Records,
		sessions: p.Sessions,
		controls: p.Controls,
		effects:  p.Effects,
		parties:  p.Parties,
		sales:    p.Sales,
		payables: p.Payables,
		ledger:   p.Ledger,
	}
}

var _ accountingsyncports.AccountingSyncDrainer = (*Service)(nil)

func (s *Service) openSession(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
) (*accountingsyncpush.Session, error) {
	tenantInfo := accountingsynceffects.TenantOf(conn)
	session, err := s.sessions.Session(ctx, tenantInfo, conn.ID)
	if err != nil {
		return nil, err
	}
	writer, ok := session.Connector.(accountingconnectorports.AccountingDocumentWriter)
	if !ok {
		return nil, fmt.Errorf(
			"%s cannot receive documents from Trenova",
			accountingprovider.ProviderName(conn.IntegrationType),
		)
	}

	journals, _ := session.Connector.(accountingconnectorports.AccountingJournalWriter)

	loc, err := s.effects.OrgLocation(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	control, err := s.controls.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		return nil, err
	}

	return &accountingsyncpush.Session{
		Tenant:   tenantInfo,
		Conn:     session.Connection,
		Writer:   writer,
		Journals: journals,
		Auth: accountingconnectorports.DocumentAuthFor(
			session.Connection,
			session.AccessToken,
		),
		Loc:               loc,
		ProviderName:      accountingprovider.ProviderName(conn.IntegrationType),
		Profile:           accountingprovider.MustProfile(conn.IntegrationType),
		Limits:            writer.DocumentLimits(),
		Control:           control,
		RedatesToNextOpen: control.ClosedPeriodPostingPolicy == accountingcontrol.ClosedPeriodPostingPolicyPostToNextOpen,
	}, nil
}

func (s *Service) Drain(
	ctx context.Context,
	req *accountingsyncports.DrainAccountingSyncRequest,
) (*accountingsyncports.AccountingSyncDrainResult, error) {
	result := new(accountingsyncports.AccountingSyncDrainResult)
	conn, err := s.effects.ConnectionByID(ctx, req.TenantInfo, req.ConnectionID)
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
	claimed, err := s.records.Claim(ctx, &accountingsyncports.ClaimAccountingSyncRecordsRequest{
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
			if errors.Is(finishErr, accountingsyncports.ErrAccountingSyncLeaseLost) {
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
		s.effects.RefreshAttention(ctx, sess.Conn)
	}
	return result, nil
}

func tally(
	result *accountingsyncports.AccountingSyncDrainResult,
	outcome accountingsync.SyncAttemptOutcome,
) {
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
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (accountingsync.SyncAttemptOutcome, error) {
	started := time.Now()
	attemptNumber := record.AttemptCount
	pushed, pushErr := s.pushOne(ctx, sess, record)
	finished := time.Now()
	now := finished.Unix()

	if pushed != nil {
		for key, value := range pushed.Refs {
			record.SetExternalRef(key, value)
		}
	}

	var outcome accountingsync.SyncAttemptOutcome
	recordAttempt := true
	var wait *accountingsyncpush.WaitError
	var noop *accountingsyncpush.NoopError
	switch {
	case pushErr == nil:
		record.MarkSynced(pushed.Sync, now)
		outcome = accountingsync.SyncAttemptSynced
	case errors.As(pushErr, &noop):
		record.MarkSynced(&accountingsync.SyncResult{}, now)
		record.Resolution = noop.Reason
		outcome = accountingsync.SyncAttemptSynced
	case errors.As(pushErr, &wait):
		record.Wait(wait.Reason, now, wait.Delay)
		if !wait.DependsOn.IsNil() {
			record.DependsOnRecordID = wait.DependsOn
		}
		outcome = accountingsync.SyncAttemptWaiting
		recordAttempt = false
	default:
		var classified *accountingsync.SyncError
		if !errors.As(pushErr, &classified) {
			classified = sess.Writer.ClassifyDocumentError(pushErr)
		}
		if classified == nil {
			classified = accountingsyncpush.Blocked(
				accountingsync.SyncErrorTransient,
				pushErr.Error(),
				"",
			)
		}
		outcome = record.MarkFailed(classified, now)
		if classified.Category == accountingsync.SyncErrorAuth {
			s.sessions.ReportCallFailure(ctx, sess.Tenant, sess.Conn.ID, pushErr)
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
	if err := s.records.Finish(ctx, &accountingsyncports.FinishAccountingSyncRecordRequest{
		Record:  record,
		Attempt: attempt,
	}); err != nil {
		return outcome, err
	}

	s.effects.PublishOutcome(ctx, record, outcome)
	if outcome != accountingsync.SyncAttemptWaiting {
		s.effects.PublishInvalidation(ctx, sess.Tenant, pulid.Nil, record.ID)
	}
	return outcome, nil
}

func (s *Service) pushOne(
	ctx context.Context,
	sess *accountingsyncpush.Session,
	record *accountingsync.AccountingSyncRecord,
) (*accountingsyncpush.Result, error) {
	switch {
	case record.ObjectType == accountingsync.SyncObjectCustomer, record.ObjectType.IsVendor():
		return s.parties.Push(ctx, sess, record)
	case record.ObjectType.IsBill():
		if record.Operation == accountingsync.SyncOperationVoid {
			return s.payables.PushBillVoid(ctx, sess, record)
		}
		return s.payables.PushBill(ctx, sess, record)
	case record.ObjectType.IsBillPayment():
		return s.payables.PushBillPayment(ctx, sess, record)
	case record.ObjectType.IsSalesDocument():
		if record.Operation == accountingsync.SyncOperationVoid {
			return s.sales.PushSalesVoid(ctx, sess, record)
		}
		return s.sales.PushSales(ctx, sess, record)
	case record.ObjectType == accountingsync.SyncObjectCustomerPayment:
		if record.Operation == accountingsync.SyncOperationVoid {
			return s.sales.PushPaymentVoid(ctx, sess, record)
		}
		return s.sales.PushPayment(ctx, sess, record)
	case record.ObjectType == accountingsync.SyncObjectCreditApplication:
		if record.Operation == accountingsync.SyncOperationVoid {
			return s.sales.PushCreditApplicationVoid(ctx, sess, record)
		}
		return s.sales.PushCreditApplication(ctx, sess, record)
	case record.ObjectType.IsLedger():
		return s.ledger.Push(ctx, sess, record)
	default:
		return nil, accountingsyncpush.Blocked(
			accountingsync.SyncErrorConfiguration,
			"Trenova does not send "+string(record.ObjectType)+" records",
			"Skip this record",
		)
	}
}
