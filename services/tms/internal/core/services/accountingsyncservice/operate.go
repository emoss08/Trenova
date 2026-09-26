package accountingsyncservice

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/zap"
)

type auditEntry struct {
	resource   permission.Resource
	resourceID pulid.ID
	userID     pulid.ID
	tenant     pagination.TenantInfo
	current    any
	previous   map[string]any
	comment    string
}

func (s *Service) logAudit(entry *auditEntry) {
	params := &services.LogActionParams{
		Resource:       entry.resource,
		ResourceID:     entry.resourceID.String(),
		Operation:      permission.OpUpdate,
		UserID:         entry.userID,
		CurrentState:   jsonutils.MustToJSON(entry.current),
		PreviousState:  entry.previous,
		OrganizationID: entry.tenant.OrgID,
		BusinessUnitID: entry.tenant.BuID,
	}
	if err := s.audit.LogAction(params, auditservice.WithComment(entry.comment)); err != nil {
		s.l.Error("failed to log accounting sync audit", zap.Error(err))
	}
}

func (s *Service) Summary(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	integrationType integration.Type,
) (*services.AccountingSyncSummary, error) {
	summary := &services.AccountingSyncSummary{
		IntegrationType: integrationType,
		ProviderName:    accountingsync.ProviderName(integrationType),
		Counts:          []repositories.AccountingSyncStatusCount{},
		Attention:       []repositories.AccountingSyncAttentionGroup{},
	}
	conn, err := s.connectionFor(ctx, tenantInfo, integrationType)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return summary, nil
		}
		return nil, err
	}
	summary.Connection = conn
	ref := repositories.AccountingSyncConnectionRequest{
		TenantInfo:   tenantInfo,
		ConnectionID: conn.ID,
	}

	if summary.Counts, err = s.records.CountByStatus(ctx, ref); err != nil {
		return nil, err
	}
	if summary.Attention, err = s.records.ListAttention(
		ctx,
		repositories.ListAccountingSyncAttentionRequest{
			TenantInfo:   tenantInfo,
			ConnectionID: conn.ID,
			Limit:        attentionGroups,
		},
	); err != nil {
		return nil, err
	}
	active, err := s.backfills.GetActive(ctx, ref)
	switch {
	case err == nil:
		summary.ActiveBackfill = active
	case !errortypes.IsNotFoundError(err):
		return nil, err
	}
	return summary, nil
}

func (s *Service) ListRecords(
	ctx context.Context,
	req *services.ListAccountingSyncRecordsRequest,
) (*pagination.CursorListResult[*accountingsync.AccountingSyncRecord], error) {
	conn, err := s.connectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	filter := req.Filter
	if filter == nil {
		filter = &pagination.QueryOptions{}
	}
	filter.TenantInfo = req.TenantInfo
	return s.records.ListConnection(ctx, &repositories.ListAccountingSyncRecordsConnectionRequest{
		Filter:          filter,
		Cursor:          req.Cursor,
		ConnectionID:    conn.ID,
		Statuses:        req.Statuses,
		ObjectTypes:     req.ObjectTypes,
		ErrorCategories: req.ErrorCategories,
		ObjectID:        req.ObjectID,
		Search:          req.Search,
	})
}

func (s *Service) GetRecord(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*accountingsync.AccountingSyncRecord, error) {
	return s.records.GetByID(ctx, repositories.GetAccountingSyncRecordRequest{
		TenantInfo: tenantInfo,
		ID:         id,
	})
}

func (s *Service) ListAttempts(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	recordID pulid.ID,
) ([]*accountingsync.AccountingSyncAttempt, error) {
	if _, err := s.GetRecord(ctx, tenantInfo, recordID); err != nil {
		return nil, err
	}
	return s.records.ListAttempts(ctx, repositories.ListAccountingSyncAttemptsRequest{
		TenantInfo:   tenantInfo,
		SyncRecordID: recordID,
		Limit:        maxAttemptsListed,
	})
}

func stateRank(record *accountingsync.AccountingSyncRecord) int {
	switch {
	case record.Status.NeedsAttention():
		return 4
	case record.Status == accountingsync.SyncStatusInFlight ||
		record.Status.Dispatchable() ||
		record.Status == accountingsync.SyncStatusAwaitingApproval:
		return 3
	case record.Status == accountingsync.SyncStatusSynced:
		return 2
	case record.Status == accountingsync.SyncStatusSkipped:
		return 1
	default:
		return 0
	}
}

func outranks(record, current *accountingsync.AccountingSyncRecord) bool {
	rank, currentRank := stateRank(record), stateRank(current)
	if rank != currentRank {
		return rank > currentRank
	}
	return record.QueuedAt > current.QueuedAt
}

func (s *Service) ObjectStates(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	objectIDs []pulid.ID,
) (map[pulid.ID]*services.AccountingSyncObjectState, error) {
	states := make(map[pulid.ID]*services.AccountingSyncObjectState, len(objectIDs))
	if len(objectIDs) == 0 {
		return states, nil
	}
	if len(objectIDs) > maxStateObjects {
		objectIDs = objectIDs[:maxStateObjects]
	}

	conns, err := s.connections.ListByTenant(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	providers := make(map[pulid.ID]string, len(conns))
	for _, conn := range conns {
		if conn.IsSyncing() {
			providers[conn.ID] = accountingsync.ProviderName(conn.IntegrationType)
		}
	}
	if len(providers) == 0 {
		return states, nil
	}

	records, err := s.records.ListByObjects(
		ctx,
		&repositories.ListAccountingSyncRecordsByObjectsRequest{
			TenantInfo: tenantInfo,
			ObjectIDs:  objectIDs,
		},
	)
	if err != nil {
		return nil, err
	}
	for _, record := range records {
		provider, ok := providers[record.ConnectionID]
		if !ok || record.Status == accountingsync.SyncStatusSuperseded {
			continue
		}
		current, seen := states[record.ObjectID]
		if seen && !outranks(record, current.Record) {
			continue
		}
		states[record.ObjectID] = &services.AccountingSyncObjectState{
			ObjectType:   record.ObjectType,
			ObjectID:     record.ObjectID,
			ProviderName: provider,
			Record:       record,
		}
	}
	return states, nil
}

func (s *Service) ListBackfills(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	integrationType integration.Type,
) ([]*accountingsync.AccountingBackfill, error) {
	conn, err := s.connectionFor(ctx, tenantInfo, integrationType)
	if err != nil {
		return nil, err
	}
	return s.backfills.ListByConnection(ctx, repositories.ListAccountingBackfillsRequest{
		TenantInfo:   tenantInfo,
		ConnectionID: conn.ID,
		Limit:        maxBackfillsListed,
	})
}

func (s *Service) UpdateSettings(
	ctx context.Context,
	req *services.UpdateAccountingSyncSettingsRequest,
) (*accountingsync.AccountingConnection, error) {
	conn, err := s.connectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	if conn.SetupStep != accountingsync.SetupStepComplete {
		return nil, errortypes.NewBusinessError(
			"Finish setting up {0} before changing how documents are sent",
			accountingsync.ProviderName(conn.IntegrationType),
		)
	}

	if req.InboundPayments != "" && !req.InboundPayments.IsValid() {
		return nil, errortypes.NewValidationError(
			"inboundPayments",
			errortypes.ErrInvalid,
			"Choose whether payments recorded in {0} are ignored, proposed or applied",
			accountingsync.ProviderName(conn.IntegrationType),
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

	s.logAudit(&auditEntry{
		resource:   permission.ResourceAccountingIntegration,
		resourceID: updated.ID,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   before,
		comment:    settingsComment(previous, updated),
	})
	s.publishInvalidation(ctx, req.TenantInfo, req.UserID, updated.ID)
	return updated, nil
}

type settingsSnapshot struct {
	sendingDrivers bool
	policy         accountingsync.InboundPaymentPolicy
}

func settingsComment(previous settingsSnapshot, conn *accountingsync.AccountingConnection) string {
	provider := accountingsync.ProviderName(conn.IntegrationType)
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

func inboundPolicyComment(provider string, policy accountingsync.InboundPaymentPolicy) string {
	switch policy {
	case accountingsync.InboundPaymentsApply:
		return "Payments recorded in " + provider + " are now applied in Trenova automatically"
	case accountingsync.InboundPaymentsOff:
		return "Payments recorded in " + provider + " are no longer brought into Trenova"
	case accountingsync.InboundPaymentsPropose:
		return "Payments recorded in " + provider + " now wait for someone to apply them"
	default:
		return "Changed how payments recorded in " + provider + " are handled"
	}
}

func (s *Service) EnableSync(
	ctx context.Context,
	req *services.EnableAccountingSyncRequest,
) (*accountingsync.AccountingConnection, error) {
	conn, err := s.connectionFor(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}
	provider := accountingsync.ProviderName(conn.IntegrationType)
	if !conn.IsActive() {
		return nil, errortypes.NewBusinessError(
			"{0} is not connected. A person must connect it from the integrations page.",
			provider,
		)
	}
	if conn.SetupStep == accountingsync.SetupStepMappings {
		return nil, errortypes.NewBusinessError("Confirm the required mappings first")
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

	before := jsonutils.MustToJSON(conn)
	conn.EnableSync(accountingsync.SyncSettings{
		StartDate:         req.StartDate,
		AutoSync:          req.AutoSync,
		DriverSettlements: req.DriverSettlements,
	}, now)
	multiErr := errortypes.NewMultiError()
	conn.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	updated, err := s.connections.Update(ctx, conn)
	if err != nil {
		return nil, err
	}

	s.logAudit(&auditEntry{
		resource:   permission.ResourceAccountingIntegration,
		resourceID: updated.ID,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   before,
		comment:    "Started sending documents to " + provider,
	})
	s.publishInvalidation(ctx, req.TenantInfo, req.UserID, updated.ID)

	if req.Backfill {
		if _, err = s.RequestBackfill(ctx, &services.RequestAccountingBackfillRequest{
			TenantInfo:      req.TenantInfo,
			UserID:          req.UserID,
			IntegrationType: req.IntegrationType,
		}); err != nil && !errors.Is(err, repositories.ErrAccountingBackfillActive) {
			return updated, err
		}
	}
	return updated, nil
}

func (s *Service) Pause(
	ctx context.Context,
	req *services.PauseAccountingSyncRequest,
) (*accountingsync.AccountingConnection, error) {
	conn, err := s.syncingConnection(ctx, req.TenantInfo, req.IntegrationType)
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
	s.logAudit(&auditEntry{
		resource:   permission.ResourceAccountingIntegration,
		resourceID: updated.ID,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   before,
		comment:    "Paused sync: " + reason,
	})
	s.refreshPaused(ctx, updated)
	s.publishInvalidation(ctx, req.TenantInfo, req.UserID, updated.ID)
	return updated, nil
}

func (s *Service) Resume(
	ctx context.Context,
	req *services.PauseAccountingSyncRequest,
) (*accountingsync.AccountingConnection, error) {
	conn, err := s.syncingConnection(ctx, req.TenantInfo, req.IntegrationType)
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
	s.logAudit(&auditEntry{
		resource:   permission.ResourceAccountingIntegration,
		resourceID: updated.ID,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   before,
		comment:    "Resumed sync",
	})
	s.refreshPaused(ctx, updated)
	s.publishInvalidation(ctx, req.TenantInfo, req.UserID, updated.ID)
	s.kick(ctx, req.TenantInfo, updated.ID)
	return updated, nil
}

func (s *Service) syncingConnection(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	integrationType integration.Type,
) (*accountingsync.AccountingConnection, error) {
	conn, err := s.connectionFor(ctx, tenantInfo, integrationType)
	if err != nil {
		return nil, err
	}
	if !conn.IsSyncing() {
		return nil, errortypes.NewBusinessError(
			"{0} is not sending documents yet; finish its setup first",
			accountingsync.ProviderName(conn.IntegrationType),
		)
	}
	return conn, nil
}

func validateIDs(ids []pulid.ID) error {
	if len(ids) > maxRetryIDs {
		return errortypes.NewValidationError(
			"ids",
			errortypes.ErrInvalid,
			"Act on at most {0} records at a time",
			maxRetryIDs,
		)
	}
	return nil
}

func (s *Service) Retry(
	ctx context.Context,
	req *services.RetryAccountingSyncRequest,
) (int64, error) {
	if err := validateIDs(req.IDs); err != nil {
		return 0, err
	}
	for _, category := range req.ErrorCategories {
		if !category.IsValid() {
			return 0, errortypes.NewValidationError(
				"errorCategories",
				errortypes.ErrInvalid,
				"{0} is not an error category",
				string(category),
			)
		}
	}
	conn, err := s.syncingConnection(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return 0, err
	}

	requeued, err := s.records.Requeue(ctx, &repositories.RequeueAccountingSyncRecordsRequest{
		TenantInfo:      req.TenantInfo,
		ConnectionID:    conn.ID,
		ErrorCategories: req.ErrorCategories,
		IDs:             req.IDs,
		At:              timeutils.NowUnix(),
	})
	if err != nil {
		return 0, err
	}
	if requeued == 0 {
		return 0, nil
	}

	s.logAudit(&auditEntry{
		resource:   permission.ResourceAccountingSync,
		resourceID: conn.ID,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    map[string]any{"requeued": requeued, "ids": req.IDs},
		comment:    "Retried " + strconv.FormatInt(requeued, 10) + " sync records",
	})
	s.refreshAttention(ctx, conn)
	s.publishInvalidation(ctx, req.TenantInfo, req.UserID, pulid.Nil)
	s.kick(ctx, req.TenantInfo, conn.ID)
	return requeued, nil
}

func (s *Service) Release(
	ctx context.Context,
	req *services.ReleaseAccountingSyncRequest,
) (int64, error) {
	if err := validateIDs(req.IDs); err != nil {
		return 0, err
	}
	conn, err := s.syncingConnection(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return 0, err
	}

	released, err := s.records.Release(ctx, &repositories.ReleaseAccountingSyncRecordsRequest{
		TenantInfo:   req.TenantInfo,
		ConnectionID: conn.ID,
		IDs:          req.IDs,
		ActorID:      req.UserID,
		At:           timeutils.NowUnix(),
	})
	if err != nil {
		return 0, err
	}
	if released == 0 {
		return 0, nil
	}

	s.logAudit(&auditEntry{
		resource:   permission.ResourceAccountingSync,
		resourceID: conn.ID,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    map[string]any{"released": released, "ids": req.IDs},
		comment:    "Released " + strconv.FormatInt(released, 10) + " held documents",
	})
	s.publishInvalidation(ctx, req.TenantInfo, req.UserID, pulid.Nil)
	s.kick(ctx, req.TenantInfo, conn.ID)
	return released, nil
}

func (s *Service) Skip(
	ctx context.Context,
	req *services.SkipAccountingSyncRequest,
) (*accountingsync.AccountingSyncRecord, error) {
	record, err := s.GetRecord(ctx, req.TenantInfo, req.ID)
	if err != nil {
		return nil, err
	}
	before := jsonutils.MustToJSON(record)
	if err = record.Skip(req.UserID, req.Reason); err != nil {
		switch {
		case errors.Is(err, accountingsync.ErrSkipReasonRequired):
			return nil, errortypes.NewValidationError(
				"reason",
				errortypes.ErrRequired,
				"Say why this record is skipped",
			)
		case errors.Is(err, accountingsync.ErrSyncRecordNotSkippable):
			return nil, errortypes.NewBusinessError(
				"This record was already sent, skipped or is being sent now",
			).WithInternal(err)
		default:
			return nil, err
		}
	}
	updated, err := s.records.Update(ctx, record)
	if err != nil {
		return nil, err
	}

	s.logAudit(&auditEntry{
		resource:   permission.ResourceAccountingSync,
		resourceID: updated.ID,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   before,
		comment: "Skipped " + documentLabel(
			updated.ObjectType,
			updated.ObjectNumber,
		) + ": " + updated.SkippedReason,
	})
	if conn, connErr := s.connectionByID(
		ctx,
		req.TenantInfo,
		updated.ConnectionID,
	); connErr == nil {
		s.refreshAttention(ctx, conn)
	}
	s.publishInvalidation(ctx, req.TenantInfo, req.UserID, updated.ID)
	return updated, nil
}

func (s *Service) RequestBackfill(
	ctx context.Context,
	req *services.RequestAccountingBackfillRequest,
) (*accountingsync.AccountingBackfill, error) {
	if s.dispatcher == nil {
		return nil, errortypes.NewBusinessError(
			"Background work is not available on this server, so a backfill cannot run",
		)
	}
	conn, err := s.syncingConnection(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return nil, err
	}

	rangeStart := *conn.SyncStartDate
	if req.RangeStart != nil {
		rangeStart = *req.RangeStart
	}
	rangeEnd := *conn.SyncEnabledAt
	if req.RangeEnd != nil {
		rangeEnd = *req.RangeEnd
	}
	if rangeStart < *conn.SyncStartDate {
		return nil, errortypes.NewValidationError(
			"rangeStart",
			errortypes.ErrInvalid,
			"A backfill cannot reach before the start date; move the start date first",
		)
	}
	if rangeEnd > *conn.SyncEnabledAt || rangeEnd < rangeStart {
		return nil, errortypes.NewValidationError(
			"rangeEnd",
			errortypes.ErrInvalid,
			"A backfill covers documents posted between the start date and when sync began",
		)
	}
	types, err := BackfillTypes(conn, req.ObjectTypes)
	if err != nil {
		return nil, err
	}

	backfill, err := s.backfills.Create(
		ctx,
		accountingsync.NewAccountingBackfill(&accountingsync.NewBackfillParams{
			TenantInfo:    req.TenantInfo,
			ConnectionID:  conn.ID,
			RangeStart:    rangeStart,
			RangeEnd:      rangeEnd,
			ObjectTypes:   types,
			RequestedByID: req.UserID,
		}),
	)
	if err != nil {
		if errors.Is(err, repositories.ErrAccountingBackfillActive) {
			return nil, errortypes.NewBusinessError(
				"A backfill is already running for {0}; pause or cancel it first",
				accountingsync.ProviderName(conn.IntegrationType),
			).WithInternal(err)
		}
		return nil, err
	}

	s.logAudit(&auditEntry{
		resource:   permission.ResourceAccountingSync,
		resourceID: backfill.ID,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    backfill,
		comment:    "Requested a backfill",
	})
	if err = s.dispatcher.StartBackfill(ctx, backfill); err != nil {
		return backfill, err
	}
	return backfill, nil
}

func BackfillTypes(
	conn *accountingsync.AccountingConnection,
	requested []accountingsync.SyncObjectType,
) ([]accountingsync.SyncObjectType, error) {
	allowed := make([]accountingsync.SyncObjectType, 0, len(accountingsync.BackfillObjectTypes()))
	for _, typ := range accountingsync.BackfillObjectTypes() {
		if typ.NeedsDriverSettlements() && !conn.SyncsDriverSettlements() {
			if slices.Contains(requested, typ) {
				return nil, errortypes.NewValidationError(
					"objectTypes",
					errortypes.ErrInvalid,
					"Owner-operator settlements are not sent to {0}; turn them on first",
					accountingsync.ProviderName(conn.IntegrationType),
				)
			}
			continue
		}
		allowed = append(allowed, typ)
	}
	if len(requested) == 0 {
		return allowed, nil
	}
	types := make([]accountingsync.SyncObjectType, 0, len(allowed))
	for _, typ := range allowed {
		if slices.Contains(requested, typ) {
			types = append(types, typ)
		}
	}
	for _, typ := range requested {
		if !slices.Contains(allowed, typ) {
			return nil, errortypes.NewValidationError(
				"objectTypes",
				errortypes.ErrInvalid,
				"{0} records are not backfilled; choose from {1}",
				string(typ),
				joinTypes(allowed),
			)
		}
	}
	return types, nil
}

func joinTypes(types []accountingsync.SyncObjectType) string {
	names := make([]string, 0, len(types))
	for _, typ := range types {
		names = append(names, string(typ))
	}
	return strings.Join(names, ", ")
}

func (s *Service) ChangeBackfill(
	ctx context.Context,
	req *services.ChangeAccountingBackfillRequest,
) (*accountingsync.AccountingBackfill, error) {
	backfill, err := s.backfills.GetByID(ctx, repositories.GetAccountingBackfillRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
	})
	if err != nil {
		return nil, err
	}
	before := jsonutils.MustToJSON(backfill)

	var changed bool
	var comment string
	switch req.Action {
	case services.AccountingBackfillPause:
		changed, comment = backfill.Pause(), "Paused the backfill"
	case services.AccountingBackfillResume:
		changed, comment = backfill.Resume(), "Resumed the backfill"
	case services.AccountingBackfillCancel:
		changed, comment = backfill.Cancel(timeutils.NowUnix()), "Cancelled the backfill"
	default:
		return nil, errortypes.NewValidationError(
			"action",
			errortypes.ErrInvalid,
			"Pause, resume or cancel a backfill",
		)
	}
	if !changed {
		return nil, errortypes.NewBusinessError(
			"A {0} backfill cannot be changed that way",
			strings.ToLower(string(backfill.Status)),
		)
	}

	updated, err := s.backfills.Update(ctx, backfill)
	if err != nil {
		return nil, err
	}
	s.logAudit(&auditEntry{
		resource:   permission.ResourceAccountingSync,
		resourceID: updated.ID,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    updated,
		previous:   before,
		comment:    comment,
	})
	if req.Action == services.AccountingBackfillResume && s.dispatcher != nil {
		if err = s.dispatcher.StartBackfill(ctx, updated); err != nil {
			return updated, err
		}
	}
	return updated, nil
}
