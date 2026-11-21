package supportlogservice

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/managedconfig"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportlog"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportlogports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	RealtimeResource = "support_log_request"
	ActionRequested  = "support_logs.requested"
	ActionSent       = "support_logs.sent"
	ActionFailed     = "support_logs.failed"
	noticeTimeout    = 10 * time.Second
	overdueBatch     = 50
)

var (
	errNotAvailable = errortypes.NewNotFoundError(
		"Sending logs to Trenova support is available only on a Managed Trenova instance",
	)
	errSharedServer = errortypes.NewBusinessError(
		"Logs can't be sent to Trenova support from this server: it runs many companies, and its logs hold every company's activity",
	)
)

type Params struct {
	fx.In

	Repo      supportlogports.Repository
	Validator *Validator
	Auditor   permissionports.SecurityAuditor
	Notifier  supportlogports.Notifier
	Realtime  realtimeports.RealtimeService
	Managed   *managedconfig.Config
	Clock     timeutils.Clock
	Logger    *zap.Logger
}

type Service struct {
	repo      supportlogports.Repository
	validator *Validator
	auditor   permissionports.SecurityAuditor
	notifier  supportlogports.Notifier
	realtime  realtimeports.RealtimeService
	enabled   bool
	shared    bool
	clock     timeutils.Clock
	l         *zap.Logger
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) *Service {
	return &Service{
		repo:      p.Repo,
		validator: p.Validator,
		auditor:   p.Auditor,
		notifier:  p.Notifier,
		realtime:  p.Realtime,
		enabled:   p.Managed.Enabled,
		shared:    p.Managed.SharedTenancy(),
		clock:     p.Clock,
		l:         p.Logger.Named("service.support-logs"),
	}
}

type Overview struct {
	Managed   bool
	Available bool
	InFlight  bool
	Remaining int
	Requests  []*supportlog.LogRequest
}

func (s *Service) Overview(ctx context.Context) (*Overview, error) {
	if !s.enabled {
		return &Overview{Requests: []*supportlog.LogRequest{}}, nil
	}
	if s.shared {
		return &Overview{Managed: true, Requests: []*supportlog.LogRequest{}}, nil
	}

	now := s.clock()
	requests, err := s.repo.Recent(ctx, supportlog.RecentLimit)
	if err != nil {
		return nil, err
	}
	load, err := s.repo.InstanceLoad(ctx, now-int64(supportlog.DailyWindow/time.Second))
	if err != nil {
		return nil, err
	}
	return &Overview{
		Managed:   true,
		Available: true,
		InFlight:  load.InFlight,
		Remaining: load.Remaining(),
		Requests:  requests,
	}, nil
}

type RequestLogsRequest struct {
	TenantInfo pagination.TenantInfo
	Period     supportlog.Period
}

func (s *Service) Request(
	ctx context.Context,
	req *RequestLogsRequest,
) (*supportlog.LogRequest, error) {
	if !s.enabled {
		return nil, errNotAvailable
	}
	if s.shared {
		return nil, errSharedServer
	}
	if req.TenantInfo.UserID.IsNil() || supportaccess.IsPrincipalID(req.TenantInfo.UserID) {
		return nil, errortypes.NewAuthorizationError(
			"Only a person in this organization may send logs to Trenova support",
		)
	}

	requester, err := s.repo.Requester(ctx, req.TenantInfo.UserID)
	if err != nil {
		return nil, err
	}

	entity := supportlog.New(&supportlog.NewRequest{
		RequestedByID:  requester.UserID,
		RequesterName:  requester.Name,
		RequesterEmail: requester.Email,
		Period:         req.Period,
		Now:            s.clock(),
	})
	if err = s.validator.ValidateCreate(ctx, entity); err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.record(ctx, &entry{
		tenantInfo: req.TenantInfo,
		actor:      permissionports.UserActor(req.TenantInfo).AuditActor(),
		request:    created,
		operation:  permission.OpCreate,
		action:     ActionRequested,
		comment: "Asked for the instance's logs from " + periodPhrase(created.Period) +
			" to be sent to Trenova support",
	})
	s.publish(ctx, created, "created")
	return created, nil
}

func (s *Service) Waiting(ctx context.Context) ([]*supportlog.LogRequest, error) {
	if !s.enabled {
		return nil, errNotAvailable
	}
	return s.repo.Waiting(ctx, supportlog.WaitingLimit)
}

func (s *Service) Report(ctx context.Context, id pulid.ID, report *supportlog.Report) error {
	if !s.enabled {
		return errNotAvailable
	}

	held, err := s.repo.ForOperator(ctx, id)
	if err != nil {
		return err
	}
	before := *held
	changed, err := held.Apply(report, s.clock())
	if err != nil || !changed {
		return err
	}

	saved, err := s.repo.SaveReport(ctx, held)
	if err != nil {
		return err
	}
	s.announce(ctx, &before, saved)
	return nil
}

type SweepResult struct {
	Failed int `json:"failed"`
}

func (s *Service) FailOverdue(ctx context.Context, now int64) (*SweepResult, error) {
	result := &SweepResult{}
	if !s.enabled {
		return result, nil
	}

	rows, err := s.repo.Overdue(ctx, &supportlogports.OverdueRequest{
		RequestedBefore:  now - int64(supportlog.PickUpTimeout/time.Second),
		CollectingBefore: now - int64(supportlog.CollectTimeout/time.Second),
		Limit:            overdueBatch,
	})
	if err != nil {
		return nil, err
	}

	for _, row := range rows {
		before := *row
		if !row.FailOverdue(now) {
			continue
		}
		saved, saveErr := s.repo.SaveReport(ctx, row)
		if errors.Is(saveErr, supportlog.ErrChangedMeanwhile) {
			continue
		}
		if saveErr != nil {
			return result, saveErr
		}
		s.announce(ctx, &before, saved)
		result.Failed++
	}

	if result.Failed > 0 {
		s.l.Info("failed log requests the operator did not finish in time", zap.Int("requests", result.Failed))
	}
	return result, nil
}

func (s *Service) announce(ctx context.Context, before, after *supportlog.LogRequest) {
	tenantInfo := pagination.TenantInfo{OrgID: after.OrganizationID, BuID: after.BusinessUnitID}
	scoped := dbscope.WithTenant(context.WithoutCancel(ctx), tenantInfo.DBTenant())

	s.publish(scoped, after, "updated")
	if !after.Status.Finished() || before.Status.Finished() {
		return
	}

	action, comment := ActionSent, "The instance's logs from "+periodPhrase(after.Period)+
		" were sent to Trenova support, as "+after.RequesterName+" asked"
	if after.Status == supportlog.StatusFailed {
		action, comment = ActionFailed, "The instance's logs from "+periodPhrase(after.Period)+
			" could not be sent to Trenova support: "+after.FailureMessage
	}
	s.record(scoped, &entry{
		tenantInfo: tenantInfo,
		actor:      permissionports.SystemAuditActor(),
		request:    after,
		operation:  permission.OpUpdate,
		action:     action,
		before:     view(before),
		comment:    comment,
	})

	noticeCtx, cancel := context.WithTimeout(scoped, noticeTimeout)
	defer cancel()
	if err := s.notifier.LogRequestFinished(noticeCtx, &supportlogports.FinishedNotice{Request: after}); err != nil {
		s.l.Error(
			"could not tell the person who asked that their log request finished",
			zap.String("requestID", after.ID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) publish(ctx context.Context, req *supportlog.LogRequest, action string) {
	err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: req.OrganizationID,
		BusinessUnitID: req.BusinessUnitID,
		Resource:       RealtimeResource,
		Action:         action,
		RecordID:       req.ID,
	})
	if err != nil {
		s.l.Warn(
			"could not announce a log request change",
			zap.String("requestID", req.ID.String()),
			zap.Error(err),
		)
	}
}

type entry struct {
	tenantInfo pagination.TenantInfo
	actor      kernelports.AuditActor
	request    *supportlog.LogRequest
	operation  permission.Operation
	action     string
	before     any
	comment    string
}

func (s *Service) record(ctx context.Context, e *entry) {
	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceSupportAccess,
		ResourceID:     e.request.ID.String(),
		Operation:      e.operation,
		Action:         e.action,
		Actor:          e.actor,
		OrganizationID: e.tenantInfo.OrgID,
		BusinessUnitID: e.tenantInfo.BuID,
		Before:         e.before,
		After:          view(e.request),
		Comment:        e.comment,
		Metadata: map[string]any{
			"supportLogs":    e.action,
			"logRequestId":   e.request.ID.String(),
			"requesterEmail": e.request.RequesterEmail,
		},
	})
}

func view(r *supportlog.LogRequest) map[string]any {
	out := map[string]any{
		"period":         r.Period.String(),
		"since":          r.Since,
		"until":          r.Until,
		"status":         r.Status.String(),
		"requesterName":  r.RequesterName,
		"requesterEmail": r.RequesterEmail,
	}
	if r.FailureMessage != "" {
		out["failureMessage"] = r.FailureMessage
	}
	if r.FinishedAt != nil {
		out["finishedAt"] = *r.FinishedAt
	}
	return out
}

func periodPhrase(p supportlog.Period) string {
	switch p {
	case supportlog.PeriodHour:
		return "the last hour"
	case supportlog.PeriodDay:
		return "the last 24 hours"
	case supportlog.PeriodWeek:
		return "the last 7 days"
	default:
		return "a period"
	}
}
