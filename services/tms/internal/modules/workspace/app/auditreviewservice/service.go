package auditreviewservice

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	RealtimeResource = "audit-reviews"
	DefaultLimit     = 20
	MaxLimit         = 100
	reviewedWindow   = 7 * 24 * time.Hour
)

type Params struct {
	fx.In

	Repo      auditports.ReviewRepository
	Timezones auditports.OverviewRepository
	Auditor   permissionports.SecurityAuditor
	Realtime  realtimeports.RealtimeService
	Now       timeutils.Clock
	Logger    *zap.Logger
}

type Service struct {
	repo      auditports.ReviewRepository
	timezones auditports.OverviewRepository
	auditor   permissionports.SecurityAuditor
	realtime  realtimeports.RealtimeService
	now       timeutils.Clock
	l         *zap.Logger
}

//nolint:gocritic // fx.In parameter structs are passed by value
func New(p Params) *Service {
	return &Service{
		repo:      p.Repo,
		timezones: p.Timezones,
		auditor:   p.Auditor,
		realtime:  p.Realtime,
		now:       p.Now,
		l:         p.Logger.Named("service.audit-reviews"),
	}
}

type View struct {
	*audit.Review
	Title    string
	OffHours bool
}

type List struct {
	Open          []View
	OpenCount     int
	Reviewed      []View
	ReviewedCount int
}

type ListRequest struct {
	TenantInfo pagination.TenantInfo
	Limit      int
}

type SetReviewedRequest struct {
	TenantInfo pagination.TenantInfo
	Actor      kernelports.AuditActor
	ID         pulid.ID
	Reviewed   bool
}

func (s *Service) Raise(ctx context.Context, payload *auditports.AlertPayload) error {
	review := &audit.Review{
		Kind:          payload.Kind,
		DedupeKey:     payload.DedupeKey,
		EntryID:       payload.EntryID,
		Action:        payload.Action,
		Resource:      payload.Resource,
		ResourceID:    payload.ResourceID,
		EntityRef:     payload.EntityRef,
		PrincipalID:   payload.PrincipalID,
		PrincipalType: payload.PrincipalType,
		ActorName:     payload.ActorName,
		Count:         max(payload.Count, 1),
		Threshold:     max(payload.Threshold, 1),
		WindowMinutes: max(payload.WindowMinutes, 1),
		OccurredAt:    payload.OccurredAt,
	}

	stored, created, err := s.repo.Raise(ctx, review)
	if err != nil {
		return err
	}
	if created {
		s.publish(ctx, stored, "created", kernelports.AuditActor{})
	}

	return nil
}

func (s *Service) List(ctx context.Context, req *ListRequest) (*List, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	limit = min(limit, MaxLimit)

	since := time.Unix(s.now(), 0).Add(-reviewedWindow).Unix()
	found, err := s.repo.List(ctx, &auditports.ListReviewsRequest{
		TenantInfo:    req.TenantInfo,
		Limit:         limit,
		ReviewedSince: since,
	})
	if err != nil {
		return nil, err
	}

	loc := s.zone(ctx, req.TenantInfo)
	locale := i18n.FromContext(ctx)

	return &List{
		Open:          Present(locale, loc, found.Open),
		OpenCount:     found.OpenCount,
		Reviewed:      Present(locale, loc, found.Reviewed),
		ReviewedCount: found.ReviewedCount,
	}, nil
}

func (s *Service) ForEntry(
	ctx context.Context,
	tenant pagination.TenantInfo,
	entryID pulid.ID,
) (View, bool, error) {
	reviews, err := s.repo.GetByEntryIDs(ctx, tenant, []pulid.ID{entryID})
	if err != nil || len(reviews) == 0 {
		return View{}, false, err
	}

	views := Present(i18n.FromContext(ctx), s.zone(ctx, tenant), reviews[:1])
	return views[0], true, nil
}

func (s *Service) SetReviewed(ctx context.Context, req *SetReviewedRequest) (*View, error) {
	before, err := s.repo.GetByID(ctx, req.TenantInfo, req.ID)
	if err != nil {
		return nil, err
	}
	if before.Open() != req.Reviewed {
		return nil, unchanged(req.Reviewed)
	}

	updated, err := s.repo.SetReviewed(ctx, &auditports.SetReviewedRequest{
		TenantInfo: req.TenantInfo,
		ID:         req.ID,
		ReviewerID: req.Actor.UserID,
		At:         s.now(),
		Reviewed:   req.Reviewed,
	})
	if errors.Is(err, auditports.ErrReviewUnchanged) {
		return nil, unchanged(req.Reviewed)
	}
	if err != nil {
		return nil, err
	}

	action, comment := audit.ActionReviewReviewed, "Security alert marked reviewed"
	if !req.Reviewed {
		action, comment = audit.ActionReviewReopened, "Security alert reopened"
	}
	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceSecurityAuditLog,
		ResourceID:     updated.ID.String(),
		Operation:      permission.OpUpdate,
		Action:         action,
		Actor:          req.Actor,
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Before:         before,
		After:          updated,
		Comment:        comment,
		Metadata: map[string]any{
			"kind":    string(updated.Kind),
			"entryId": updated.EntryID.String(),
		},
	})
	s.publish(ctx, updated, "updated", req.Actor)

	views := Present(i18n.FromContext(ctx), s.zone(ctx, req.TenantInfo), []*audit.Review{updated})
	return &views[0], nil
}

func unchanged(reviewed bool) error {
	if reviewed {
		return errortypes.NewConflictError("Someone already marked this alert reviewed")
	}
	return errortypes.NewConflictError("This alert is already open")
}

func (s *Service) publish(
	ctx context.Context,
	review *audit.Review,
	action string,
	actor kernelports.AuditActor,
) {
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: review.OrganizationID,
		BusinessUnitID: review.BusinessUnitID,
		ActorUserID:    actor.UserID,
		ActorType:      actor.PrincipalType,
		ActorID:        actor.PrincipalID,
		ActorAPIKeyID:  actor.APIKeyID,
		Resource:       RealtimeResource,
		Action:         action,
		RecordID:       review.ID,
	}); err != nil {
		s.l.Warn("failed to publish audit review invalidation", zap.Error(err))
	}
}

func (s *Service) zone(ctx context.Context, tenant pagination.TenantInfo) *time.Location {
	zone, err := s.timezones.OrganizationTimezone(ctx, tenant)
	if err != nil {
		s.l.Warn("organization time zone unavailable; using UTC", zap.Error(err))
		return time.UTC
	}

	return Location(zone)
}

func Location(zone string) *time.Location {
	if zone == "" || zone == "Local" {
		return time.UTC
	}
	loc, err := time.LoadLocation(zone)
	if err != nil {
		return time.UTC
	}

	return loc
}

func Present(locale i18n.Locale, loc *time.Location, reviews []*audit.Review) []View {
	out := make([]View, 0, len(reviews))
	for _, review := range reviews {
		out = append(out, View{
			Review:   review,
			Title:    Title(locale, review),
			OffHours: audit.IsOffHours(time.Unix(review.OccurredAt, 0).In(loc).Hour()),
		})
	}

	return out
}
