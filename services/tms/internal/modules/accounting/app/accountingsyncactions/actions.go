package accountingsyncactions

import (
	"context"
	"errors"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsynceffects"
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

const MaxRetryIDs = 500

func validateIDs(ids []pulid.ID) error {
	if len(ids) > MaxRetryIDs {
		return errortypes.NewValidationError(
			"ids",
			errortypes.ErrInvalid,
			"Act on at most {0} records at a time",
			MaxRetryIDs,
		)
	}
	return nil
}

func (s *Service) Retry(
	ctx context.Context,
	req *accountingsyncports.RetryAccountingSyncRequest,
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
	conn, err := s.effects.SyncingConnection(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return 0, err
	}

	requeued, err := s.records.Requeue(
		ctx,
		&accountingsyncports.RequeueAccountingSyncRecordsRequest{
			TenantInfo:      req.TenantInfo,
			ConnectionID:    conn.ID,
			ErrorCategories: req.ErrorCategories,
			IDs:             req.IDs,
			At:              timeutils.NowUnix(),
		},
	)
	if err != nil {
		return 0, err
	}
	if requeued == 0 {
		return 0, nil
	}

	s.effects.LogAudit(&accountingsynceffects.AuditEntry{
		Resource:   permission.ResourceAccountingSync,
		ResourceID: conn.ID,
		UserID:     req.UserID,
		Tenant:     req.TenantInfo,
		Current:    map[string]any{"requeued": requeued, "ids": req.IDs},
		Comment:    "Retried " + strconv.FormatInt(requeued, 10) + " sync records",
	})
	s.effects.RefreshAttention(ctx, conn)
	s.effects.PublishInvalidation(ctx, req.TenantInfo, req.UserID, pulid.Nil)
	s.effects.Kick(ctx, req.TenantInfo, conn.ID)
	return requeued, nil
}

func (s *Service) Release(
	ctx context.Context,
	req *accountingsyncports.ReleaseAccountingSyncRequest,
) (int64, error) {
	if err := validateIDs(req.IDs); err != nil {
		return 0, err
	}
	conn, err := s.effects.SyncingConnection(ctx, req.TenantInfo, req.IntegrationType)
	if err != nil {
		return 0, err
	}

	released, err := s.records.Release(
		ctx,
		&accountingsyncports.ReleaseAccountingSyncRecordsRequest{
			TenantInfo:   req.TenantInfo,
			ConnectionID: conn.ID,
			IDs:          req.IDs,
			ActorID:      req.UserID,
			At:           timeutils.NowUnix(),
		},
	)
	if err != nil {
		return 0, err
	}
	if released == 0 {
		return 0, nil
	}

	s.effects.LogAudit(&accountingsynceffects.AuditEntry{
		Resource:   permission.ResourceAccountingSync,
		ResourceID: conn.ID,
		UserID:     req.UserID,
		Tenant:     req.TenantInfo,
		Current:    map[string]any{"released": released, "ids": req.IDs},
		Comment:    "Released " + strconv.FormatInt(released, 10) + " held documents",
	})
	s.effects.PublishInvalidation(ctx, req.TenantInfo, req.UserID, pulid.Nil)
	s.effects.Kick(ctx, req.TenantInfo, conn.ID)
	return released, nil
}

func (s *Service) PlanRedate(
	ctx context.Context,
	req *accountingsyncports.RedateAccountingSyncRequest,
) (*accountingsyncports.AccountingSyncRedatePlan, error) {
	record, err := s.record(ctx, req.TenantInfo, req.ID)
	if err != nil {
		return nil, err
	}
	conn, err := s.effects.ConnectionByID(ctx, req.TenantInfo, record.ConnectionID)
	if err != nil {
		return nil, err
	}
	provider := accountingprovider.ProviderName(conn.IntegrationType)
	label := accountingsynceffects.DocumentLabel(record.ObjectType, record.ObjectNumber)
	if record.Status != accountingsync.SyncStatusBlocked ||
		record.ErrorCategory != accountingsync.SyncErrorClosedPeriod {
		return nil, errortypes.NewBusinessError(
			"{0} is not held by a closed period, so it keeps its date",
			label,
		)
	}
	redates, err := s.effects.RedatesToNextOpen(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if !redates {
		return nil, errortypes.NewBusinessError(
			"Trenova's closed-period policy keeps a document in its own period; reopen the period in {0}, or set the policy to post to the next open period",
			provider,
		)
	}
	loc, err := s.effects.OrgLocation(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	sentDate, ok := conn.FirstOpenDay(loc)
	if !ok {
		return nil, errortypes.NewBusinessError(
			"{0} no longer reports closed books, so {1} keeps its date; retry it instead",
			provider,
			label,
		)
	}

	after := *record
	if err = after.Redate(req.UserID, sentDate, timeutils.NowUnix()); err != nil {
		return nil, errortypes.NewBusinessError(
			"{0} is not held by a closed period, so it keeps its date",
			label,
		).WithInternal(err)
	}
	return &accountingsyncports.AccountingSyncRedatePlan{
		Connection: conn,
		Before:     record,
		After:      &after,
		SentDate:   sentDate,
		SentDay:    timeutils.FormatCalendarDate(sentDate, loc),
	}, nil
}

func (s *Service) Redate(
	ctx context.Context,
	req *accountingsyncports.RedateAccountingSyncRequest,
) (*accountingsync.AccountingSyncRecord, error) {
	plan, err := s.PlanRedate(ctx, req)
	if err != nil {
		return nil, err
	}
	before := jsonutils.MustToJSON(plan.Before)
	updated, err := s.records.Update(ctx, plan.After)
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
		Comment: "Sent " + accountingsynceffects.DocumentLabel(
			updated.ObjectType,
			updated.ObjectNumber,
		) +
			" dated on the first open day of the books",
	})
	s.effects.RefreshAttention(ctx, plan.Connection)
	s.effects.PublishInvalidation(ctx, req.TenantInfo, req.UserID, updated.ID)
	s.effects.Kick(ctx, req.TenantInfo, plan.Connection.ID)
	return updated, nil
}

func (s *Service) Skip(
	ctx context.Context,
	req *accountingsyncports.SkipAccountingSyncRequest,
) (*accountingsync.AccountingSyncRecord, error) {
	record, err := s.record(ctx, req.TenantInfo, req.ID)
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

	s.effects.LogAudit(&accountingsynceffects.AuditEntry{
		Resource:   permission.ResourceAccountingSync,
		ResourceID: updated.ID,
		UserID:     req.UserID,
		Tenant:     req.TenantInfo,
		Current:    updated,
		Previous:   before,
		Comment: "Skipped " + accountingsynceffects.DocumentLabel(
			updated.ObjectType,
			updated.ObjectNumber,
		) + ": " + updated.SkippedReason,
	})
	if conn, connErr := s.effects.ConnectionByID(
		ctx,
		req.TenantInfo,
		updated.ConnectionID,
	); connErr == nil {
		s.effects.RefreshAttention(ctx, conn)
	}
	s.effects.PublishInvalidation(ctx, req.TenantInfo, req.UserID, updated.ID)
	return updated, nil
}

type Deps struct {
	Records accountingsyncports.AccountingSyncRecordRepository
	Effects *accountingsynceffects.Effects
}

type Service struct {
	records accountingsyncports.AccountingSyncRecordRepository
	effects *accountingsynceffects.Effects
}

func New(d Deps) *Service {
	return &Service{records: d.Records, effects: d.Effects}
}

func (s *Service) record(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
) (*accountingsync.AccountingSyncRecord, error) {
	return s.records.GetByID(ctx, accountingsyncports.GetAccountingSyncRecordRequest{
		TenantInfo: tenantInfo,
		ID:         id,
	})
}
