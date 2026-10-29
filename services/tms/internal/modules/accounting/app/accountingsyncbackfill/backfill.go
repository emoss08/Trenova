package accountingsyncbackfill

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncenqueue"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingbackfill"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingconnection"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) RequestBackfill(
	ctx context.Context,
	req *accountingsyncports.RequestAccountingBackfillRequest,
) (*accountingbackfill.AccountingBackfill, error) {
	if s.dispatcher == nil {
		return nil, errortypes.NewBusinessError(
			"Background work is not available on this server, so a backfill cannot run",
		)
	}
	conn, err := s.effects.SyncingConnection(ctx, req.TenantInfo, req.IntegrationType)
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
	types, err := ObjectTypes(conn, req.ObjectTypes)
	if err != nil {
		return nil, err
	}

	backfill, err := s.backfills.Create(
		ctx,
		accountingbackfill.NewAccountingBackfill(&accountingbackfill.NewBackfillParams{
			TenantInfo:    req.TenantInfo,
			ConnectionID:  conn.ID,
			RangeStart:    rangeStart,
			RangeEnd:      rangeEnd,
			ObjectTypes:   types,
			RequestedByID: req.UserID,
		}),
	)
	if err != nil {
		if errors.Is(err, accountingsyncports.ErrAccountingBackfillActive) {
			return nil, errortypes.NewBusinessError(
				"A backfill is already running for {0}; pause or cancel it first",
				accountingprovider.ProviderName(conn.IntegrationType),
			).WithInternal(err)
		}
		return nil, err
	}

	s.effects.LogAudit(&accountingsynceffects.AuditEntry{
		Resource:   permission.ResourceAccountingSync,
		ResourceID: backfill.ID,
		UserID:     req.UserID,
		Tenant:     req.TenantInfo,
		Current:    backfill,
		Comment:    "Requested a backfill",
	})
	if err = s.dispatcher.StartBackfill(ctx, backfill); err != nil {
		return backfill, err
	}
	return backfill, nil
}

func ObjectTypes(
	conn *accountingconnection.AccountingConnection,
	requested []accountingsync.SyncObjectType,
) ([]accountingsync.SyncObjectType, error) {
	allowed := make(
		[]accountingsync.SyncObjectType,
		0,
		len(accountingbackfill.BackfillObjectTypes()),
	)
	for _, typ := range accountingbackfill.BackfillObjectTypes() {
		if !conn.Backfills(typ) {
			continue
		}
		if typ.NeedsDriverSettlements() && !conn.SyncsDriverSettlements() {
			if slices.Contains(requested, typ) {
				return nil, errortypes.NewValidationError(
					"objectTypes",
					errortypes.ErrInvalid,
					"Owner-operator settlements are not sent to {0}; turn them on first",
					accountingprovider.ProviderName(conn.IntegrationType),
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
	req *accountingsyncports.ChangeAccountingBackfillRequest,
) (*accountingbackfill.AccountingBackfill, error) {
	backfill, err := s.backfills.GetByID(ctx, accountingsyncports.GetAccountingBackfillRequest{
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
	case accountingsyncports.AccountingBackfillPause:
		changed, comment = backfill.Pause(), "Paused the backfill"
	case accountingsyncports.AccountingBackfillResume:
		changed, comment = backfill.Resume(), "Resumed the backfill"
	case accountingsyncports.AccountingBackfillCancel:
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
	s.effects.LogAudit(&accountingsynceffects.AuditEntry{
		Resource:   permission.ResourceAccountingSync,
		ResourceID: updated.ID,
		UserID:     req.UserID,
		Tenant:     req.TenantInfo,
		Current:    updated,
		Previous:   before,
		Comment:    comment,
	})
	if req.Action == accountingsyncports.AccountingBackfillResume && s.dispatcher != nil {
		if err = s.dispatcher.StartBackfill(ctx, updated); err != nil {
			return updated, err
		}
	}
	return updated, nil
}

type Deps struct {
	Records    accountingsyncports.AccountingSyncRecordRepository
	Backfills  accountingsyncports.AccountingBackfillRepository
	Enqueuer   *accountingsyncenqueue.Enqueuer
	Dispatcher accountingsyncports.AccountingSyncDispatcher
	Effects    *accountingsynceffects.Effects
}

type Service struct {
	records    accountingsyncports.AccountingSyncRecordRepository
	backfills  accountingsyncports.AccountingBackfillRepository
	enqueuer   *accountingsyncenqueue.Enqueuer
	dispatcher accountingsyncports.AccountingSyncDispatcher
	effects    *accountingsynceffects.Effects
}

func New(d Deps) *Service {
	return &Service{
		records:    d.Records,
		backfills:  d.Backfills,
		enqueuer:   d.Enqueuer,
		dispatcher: d.Dispatcher,
		effects:    d.Effects,
	}
}

const Page = 200

func (s *Service) BackfillStep(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	backfillID pulid.ID,
) (*accountingsyncports.AccountingBackfillStepResult, error) {
	result := new(accountingsyncports.AccountingBackfillStepResult)
	backfill, err := s.backfills.GetByID(ctx, accountingsyncports.GetAccountingBackfillRequest{
		TenantInfo: tenantInfo,
		ID:         backfillID,
	})
	if err != nil {
		return nil, err
	}
	if !backfill.Status.InProgress() || backfill.Status == accountingbackfill.BackfillStatusPaused {
		result.Stopped = true
		result.Done = backfill.Status == accountingbackfill.BackfillStatusCompleted
		return result, nil
	}

	now := timeutils.NowUnix()
	conn, err := s.effects.ConnectionByID(ctx, tenantInfo, backfill.ConnectionID)
	if err != nil {
		return nil, err
	}
	if !conn.IsSyncing() {
		backfill.Fail(accountingprovider.ProviderName(conn.IntegrationType)+" is not syncing", now)
		if _, err = s.backfills.Update(ctx, backfill); err != nil {
			return nil, err
		}
		result.Stopped = true
		return result, nil
	}
	backfill.Start(now)

	if !conn.Backfills(backfill.Cursor.ObjectType) ||
		(backfill.Cursor.ObjectType.NeedsDriverSettlements() && !conn.SyncsDriverSettlements()) {
		return s.skipBackfillType(ctx, backfill, result, now)
	}
	loc, err := s.effects.OrgLocation(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	undoneBefore := backfill.RangeEnd
	candidates, err := s.records.ListCandidates(
		ctx,
		&accountingsyncports.ListAccountingSyncCandidatesRequest{
			TenantInfo:   tenantInfo,
			ConnectionID: conn.ID,
			ObjectType:   backfill.Cursor.ObjectType,
			Operation:    accountingsync.SyncOperationCreate,
			PostedBefore: &backfill.RangeEnd,
			DatedFrom:    backfill.RangeStart,
			UndoneBefore: &undoneBefore,
			AfterAt:      backfill.Cursor.AfterAt,
			AfterID:      backfill.Cursor.AfterID,
			Limit:        Page,
			Timezone:     loc.String(),
		},
	)
	if err != nil {
		return nil, err
	}

	if len(candidates) > 0 {
		enqueued, enqueueErr := s.enqueuer.QueueCandidates(
			ctx,
			conn,
			candidates,
			accountingsync.SyncSourceBackfill,
			loc,
		)
		if enqueueErr != nil {
			return nil, enqueueErr
		}
		result.Enqueued = len(enqueued.Inserted)
		result.Existing = enqueued.Existing
		last := candidates[len(candidates)-1]
		backfill.Advance(accountingbackfill.BackfillCursor{
			ObjectType: backfill.Cursor.ObjectType,
			AfterAt:    last.PostedAt,
			AfterID:    last.ObjectID,
		}, result.Enqueued, result.Existing)
	}
	if len(candidates) < Page && !backfill.NextObjectType() {
		backfill.Complete(now)
		result.Done = true
	}

	if _, err = s.backfills.Update(ctx, backfill); err != nil {
		if errortypes.IsVersionMismatchError(err) {
			result.Stopped = true
			return result, nil
		}
		return nil, err
	}
	if result.Enqueued > 0 && conn.CanDispatch() {
		s.effects.Kick(ctx, tenantInfo, conn.ID)
	}
	return result, nil
}

func (s *Service) skipBackfillType(
	ctx context.Context,
	backfill *accountingbackfill.AccountingBackfill,
	result *accountingsyncports.AccountingBackfillStepResult,
	now int64,
) (*accountingsyncports.AccountingBackfillStepResult, error) {
	if !backfill.NextObjectType() {
		backfill.Complete(now)
		result.Done = true
	}
	if _, err := s.backfills.Update(ctx, backfill); err != nil {
		if errortypes.IsVersionMismatchError(err) {
			result.Stopped = true
			return result, nil
		}
		return nil, err
	}
	return result, nil
}
