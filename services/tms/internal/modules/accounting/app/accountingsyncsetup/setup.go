package accountingsyncsetup

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncbackfill"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncenqueue"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingmappingports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const systemNoteMaxLength = 500

func (s *Service) UpdateSettings(
	ctx context.Context,
	req *accountingsyncports.UpdateAccountingSyncSettingsRequest,
) (*accountingconnection.AccountingConnection, error) {
	conn, err := s.effects.Connection(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	if conn.SetupStep != accountingconnection.SetupStepComplete {
		return nil, errortypes.NewBusinessError(
			"Finish setting up {0} before changing how documents are sent",
			accountingprovider.ProviderName(conn.IntegrationType),
		)
	}

	if req.InboundPayments != "" && !req.InboundPayments.IsValid() {
		return nil, errortypes.NewValidationError(
			"inboundPayments",
			errortypes.ErrInvalid,
			"Choose whether payments recorded in {0} are ignored, proposed or applied",
			accountingprovider.ProviderName(conn.IntegrationType),
		)
	}

	if req.InboundPayments != "" &&
		req.InboundPayments != accountingconnection.InboundPaymentsOff &&
		!conn.ReadsPayments() {
		return nil, errortypes.NewValidationError(
			"inboundPayments",
			errortypes.ErrInvalid,
			accountingprovider.MustProfile(conn.IntegrationType).InboundUnavailableReason,
		)
	}

	before := jsonutils.MustToJSON(conn)
	previous := settingsSnapshot{
		sendingDrivers: conn.SyncsDriverSettlements(),
		policy:         conn.PaymentPolicy(),
	}
	conn.AutoSync = req.AutoSync
	conn.SetDriverSettlements(req.DriverSettlements, timeutils.NowUnix())
	conn.SetInboundPayments(req.InboundPayments)
	updated, err := s.connections.Update(ctx, conn)
	if err != nil {
		return nil, err
	}

	s.effects.LogAudit(&accountingsynceffects.AuditEntry{
		Resource:   permission.ResourceAccountingIntegration,
		ResourceID: updated.ID,
		UserID:     req.UserID,
		Tenant:     req.TenantInfo,
		Current:    updated,
		Previous:   before,
		Comment:    settingsComment(previous, updated),
	})
	s.effects.PublishInvalidation(ctx, req.TenantInfo, req.UserID, updated.ID)
	return updated, nil
}

type settingsSnapshot struct {
	sendingDrivers bool
	policy         accountingconnection.InboundPaymentPolicy
}

func settingsComment(
	previous settingsSnapshot,
	conn *accountingconnection.AccountingConnection,
) string {
	provider := accountingprovider.ProviderName(conn.IntegrationType)
	switch {
	case !previous.sendingDrivers && conn.SyncsDriverSettlements():
		return "Started sending owner-operator settlements to " + provider
	case previous.sendingDrivers && !conn.SyncsDriverSettlements():
		return "Stopped sending owner-operator settlements to " + provider
	case previous.policy != conn.PaymentPolicy():
		return inboundPolicyComment(provider, conn.PaymentPolicy())
	default:
		return "Changed how documents are sent to " + provider
	}
}

func inboundPolicyComment(
	provider string,
	policy accountingconnection.InboundPaymentPolicy,
) string {
	switch policy {
	case accountingconnection.InboundPaymentsApply:
		return "Payments recorded in " + provider + " are now applied in Trenova automatically"
	case accountingconnection.InboundPaymentsOff:
		return "Payments recorded in " + provider + " are no longer brought into Trenova"
	case accountingconnection.InboundPaymentsPropose:
		return "Payments recorded in " + provider + " now wait for someone to apply them"
	default:
		return "Changed how payments recorded in " + provider + " are handled"
	}
}

func (s *Service) EnableSync(
	ctx context.Context,
	req *accountingsyncports.EnableAccountingSyncRequest,
) (*accountingconnection.AccountingConnection, error) {
	conn, err := s.effects.Connection(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	provider := accountingprovider.ProviderName(conn.IntegrationType)
	if !conn.IsActive() {
		return nil, errortypes.NewBusinessError(
			"{0} is not connected. A person must connect it from the integrations page.",
			provider,
		)
	}
	switch conn.SetupStep {
	case accountingconnection.SetupStepMode:
		return nil, errortypes.NewBusinessError(
			"Choose whether to send documents or journal entries first",
		)
	case accountingconnection.SetupStepMappings:
		return nil, errortypes.NewBusinessError("Confirm the required mappings first")
	case accountingconnection.SetupStepStartDate, accountingconnection.SetupStepComplete:
	}
	if req.OpeningBalances && !conn.SendsLedger() {
		return nil, errortypes.NewValidationError(
			"openingBalances",
			errortypes.ErrInvalid,
			accountingconnection.ErrOpeningBalancesNeedLedger.Error(),
		)
	}
	now := timeutils.NowUnix()
	if req.StartDate <= 0 {
		return nil, errortypes.NewValidationError(
			"startDate",
			errortypes.ErrRequired,
			"Choose the first day documents are sent from",
		)
	}
	if req.StartDate > now {
		return nil, errortypes.NewValidationError(
			"startDate",
			errortypes.ErrInvalid,
			"The start date cannot be in the future",
		)
	}
	if err = conn.CanMoveStartDate(req.StartDate); err != nil {
		return nil, errortypes.NewValidationError(
			"startDate",
			errortypes.ErrInvalid,
			"The start date cannot move once opening balances are sent, because they hold every balance up to it",
		)
	}

	before := jsonutils.MustToJSON(conn)
	hadOpeningBalances := conn.SentOpeningBalances()
	conn.EnableSync(accountingconnection.SyncSettings{
		StartDate:         req.StartDate,
		AutoSync:          req.AutoSync,
		DriverSettlements: req.DriverSettlements,
		OpeningBalances:   req.OpeningBalances,
	}, now)
	multiErr := errortypes.NewMultiError()
	conn.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	var updated *accountingconnection.AccountingConnection
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		var txErr error
		if updated, txErr = s.connections.Update(txCtx, conn); txErr != nil {
			return txErr
		}
		if hadOpeningBalances || !updated.SentOpeningBalances() {
			return nil
		}
		return s.enqueueOpeningBalances(txCtx, updated, now)
	})
	if err != nil {
		return nil, err
	}

	s.effects.LogAudit(&accountingsynceffects.AuditEntry{
		Resource:   permission.ResourceAccountingIntegration,
		ResourceID: updated.ID,
		UserID:     req.UserID,
		Tenant:     req.TenantInfo,
		Current:    updated,
		Previous:   before,
		Comment:    "Started sending " + sentWhat(updated) + " to " + provider,
	})
	s.effects.PublishInvalidation(ctx, req.TenantInfo, req.UserID, updated.ID)

	if req.Backfill {
		if _, err = s.backfills.RequestBackfill(ctx, &accountingsyncports.RequestAccountingBackfillRequest{
			TenantInfo:      req.TenantInfo,
			UserID:          req.UserID,
			IntegrationType: req.IntegrationType,
		}); err != nil &&
			!errors.Is(err, accountingsyncports.ErrAccountingBackfillActive) {
			return updated, err
		}
	}
	return updated, nil
}

func (s *Service) Pause(
	ctx context.Context,
	req *accountingsyncports.PauseAccountingSyncRequest,
) (*accountingconnection.AccountingConnection, error) {
	conn, err := s.effects.SyncingConnection(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	if conn.IsPaused() {
		return conn, nil
	}
	reason := stringutils.OneLine(req.Reason, systemNoteMaxLength)
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Say why sync is paused so others know when to resume it",
		)
	}

	before := jsonutils.MustToJSON(conn)
	conn.Pause(req.UserID, reason, timeutils.NowUnix())
	updated, err := s.connections.Update(ctx, conn)
	if err != nil {
		return nil, err
	}
	s.effects.LogAudit(&accountingsynceffects.AuditEntry{
		Resource:   permission.ResourceAccountingIntegration,
		ResourceID: updated.ID,
		UserID:     req.UserID,
		Tenant:     req.TenantInfo,
		Current:    updated,
		Previous:   before,
		Comment:    "Paused sync: " + reason,
	})
	s.effects.RefreshPaused(ctx, updated)
	s.effects.PublishInvalidation(ctx, req.TenantInfo, req.UserID, updated.ID)
	return updated, nil
}

func (s *Service) Resume(
	ctx context.Context,
	req *accountingsyncports.PauseAccountingSyncRequest,
) (*accountingconnection.AccountingConnection, error) {
	conn, err := s.effects.SyncingConnection(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	if !conn.IsPaused() {
		return conn, nil
	}

	before := jsonutils.MustToJSON(conn)
	conn.Resume()
	updated, err := s.connections.Update(ctx, conn)
	if err != nil {
		return nil, err
	}
	s.effects.LogAudit(&accountingsynceffects.AuditEntry{
		Resource:   permission.ResourceAccountingIntegration,
		ResourceID: updated.ID,
		UserID:     req.UserID,
		Tenant:     req.TenantInfo,
		Current:    updated,
		Previous:   before,
		Comment:    "Resumed sync",
	})
	s.effects.RefreshPaused(ctx, updated)
	s.effects.PublishInvalidation(ctx, req.TenantInfo, req.UserID, updated.ID)
	s.effects.Kick(ctx, req.TenantInfo, updated.ID)
	return updated, nil
}

func (s *Service) ChooseMode(
	ctx context.Context,
	req *accountingsyncports.ChooseAccountingSyncModeRequest,
) (*accountingconnection.AccountingConnection, error) {
	conn, err := s.effects.Connection(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	provider := accountingprovider.ProviderName(conn.IntegrationType)
	if !conn.IsActive() {
		return nil, errortypes.NewBusinessError(
			"{0} is not connected. A person must connect it from the integrations page.",
			provider,
		)
	}

	before := jsonutils.MustToJSON(conn)
	if err = conn.ChooseMode(req.Mode, req.Granularity); err != nil {
		if errors.Is(err, accountingconnection.ErrModeUnavailable) {
			return nil, errortypes.NewValidationError(
				"mode",
				errortypes.ErrInvalid,
				accountingprovider.MustProfile(conn.IntegrationType).LedgerUnavailableReason,
			)
		}
		return nil, modeError(err)
	}
	multiErr := errortypes.NewMultiError()
	conn.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	updated, err := s.connections.Update(ctx, conn)
	if err != nil {
		return nil, err
	}
	if _, rescoreErr := s.scorer.Rescore(
		ctx,
		req.TenantInfo,
		updated.ID,
	); rescoreErr != nil {
		s.l.Warn("the mode was saved but its mappings were not rebuilt; the next refresh will",
			zap.String("connectionId", updated.ID.String()), zap.Error(rescoreErr))
	}

	comment := "Chose to send documents to " + provider
	if updated.SendsLedger() {
		comment = "Chose to send journal entries to " + provider
		if updated.SumsByDay() {
			comment += ", summed by day"
		}
	}
	s.effects.LogAudit(&accountingsynceffects.AuditEntry{
		Resource:   permission.ResourceAccountingIntegration,
		ResourceID: updated.ID,
		UserID:     req.UserID,
		Tenant:     req.TenantInfo,
		Current:    updated,
		Previous:   before,
		Comment:    comment,
	})
	s.effects.PublishInvalidation(ctx, req.TenantInfo, req.UserID, updated.ID)
	return updated, nil
}

func modeError(err error) error {
	switch {
	case errors.Is(err, accountingconnection.ErrModeNotRecognized):
		return errortypes.NewValidationError("mode", errortypes.ErrInvalid, err.Error())
	case errors.Is(err, accountingconnection.ErrGranularityRequired):
		return errortypes.NewValidationError("granularity", errortypes.ErrRequired, err.Error())
	default:
		return errortypes.NewBusinessError(err.Error())
	}
}

func sentWhat(conn *accountingconnection.AccountingConnection) string {
	if conn.SendsLedger() {
		return "journal entries"
	}
	return "documents"
}

func (s *Service) enqueueOpeningBalances(
	ctx context.Context,
	conn *accountingconnection.AccountingConnection,
	now int64,
) error {
	loc, err := s.effects.OrgLocation(ctx, accountingsynceffects.TenantOf(conn))
	if err != nil {
		return err
	}
	start := *conn.SyncStartDate
	dated := timeutils.PreviousDayStart(start, loc)
	req := &accountingsyncports.AccountingSyncEnqueueRequest{
		TenantInfo:   accountingsynceffects.TenantOf(conn),
		ObjectType:   accountingsync.SyncObjectJournalSummary,
		ObjectID:     pulid.ID(accountingsync.JournalOpeningID(start, loc)),
		ObjectNumber: timeutils.FormatCalendarDate(dated, loc),
		Operation:    accountingsync.SyncOperationCreate,
		Revision:     1,
		SourceEvent:  accountingsync.SyncSourceOpeningBalances,
		DocumentDate: dated,
	}
	_, err = s.enqueuer.EnqueueRecords(
		ctx,
		req.TenantInfo,
		[]*accountingsync.AccountingSyncRecord{accountingsyncenqueue.NewRecordFor(conn, req, now)},
	)
	return err
}

type Params struct {
	fx.In

	Logger      *zap.Logger
	DB          kernelports.Transactor
	Connections accountingconnectionports.AccountingConnectionRepository
	Scorer      accountingmappingports.AccountingMappingScorer
	Enqueuer    *accountingsyncenqueue.Enqueuer
	Effects     *accountingsynceffects.Effects
	Backfills   *accountingsyncbackfill.Service
}

type Service struct {
	l           *zap.Logger
	db          kernelports.Transactor
	connections accountingconnectionports.AccountingConnectionRepository
	scorer      accountingmappingports.AccountingMappingScorer
	enqueuer    *accountingsyncenqueue.Enqueuer
	effects     *accountingsynceffects.Effects
	backfills   *accountingsyncbackfill.Service
}

//nolint:gocritic // dependency injection
func New(p Params) *Service {
	return &Service{
		l:           p.Logger.Named("service.accounting-sync"),
		db:          p.DB,
		connections: p.Connections,
		scorer:      p.Scorer,
		enqueuer:    p.Enqueuer,
		effects:     p.Effects,
		backfills:   p.Backfills,
	}
}

var _ accountingsyncports.AccountingSyncSetupService = (*Service)(nil)
