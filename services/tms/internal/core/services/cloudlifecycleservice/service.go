package cloudlifecycleservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/subscription"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultSweepLimit = 500
	expiredPageSize   = 200
	maxExpiredPerRun  = 2_000
)

var (
	ErrNotExpired         = errors.New("the organization's subscription has not expired")
	ErrStorageUnsupported = errors.New("the storage backend cannot delete by prefix")
)

type Params struct {
	fx.In

	Subscriptions repositories.SubscriptionRepository
	Purge         repositories.TenantPurgeRepository
	Plans         services.PlanService
	Sessions      repositories.SessionRepository `optional:"true"`
	Storage       storage.Client                 `optional:"true"`
	Emails        services.PlatformEmailService  `optional:"true"`
	Logger        *zap.Logger
}

type Service struct {
	subscriptions repositories.SubscriptionRepository
	purge         repositories.TenantPurgeRepository
	plans         services.PlanService
	sessions      repositories.SessionRepository
	storage       storage.PrefixDeleter
	emails        services.PlatformEmailService
	l             *zap.Logger
	now           func() int64
}

func New(p Params) *Service {
	logger := p.Logger.Named("service.cloud-lifecycle")

	var deleter storage.PrefixDeleter
	if candidate, ok := p.Storage.(storage.PrefixDeleter); ok {
		deleter = candidate
	}

	return &Service{
		subscriptions: p.Subscriptions,
		purge:         p.Purge,
		plans:         p.Plans,
		sessions:      p.Sessions,
		storage:       deleter,
		emails:        p.Emails,
		l:             logger,
		now:           func() int64 { return time.Now().Unix() },
	}
}

type TenantRef struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
}

func (r TenantRef) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: r.OrganizationID, BuID: r.BusinessUnitID}
}

type SweepResult struct {
	Examined     int         `json:"examined"`
	ReadOnly     int         `json:"readOnly"`
	Expired      int         `json:"expired"`
	Failed       int         `json:"failed"`
	Failures     []string    `json:"failures"`
	PurgeTargets []TenantRef `json:"purgeTargets"`
}

func (s *Service) Sweep(ctx context.Context, now int64) (*SweepResult, error) {
	if now <= 0 {
		now = s.now()
	}

	due, err := s.subscriptions.ListDue(ctx, &repositories.ListDueSubscriptionsRequest{
		Now:   now,
		Limit: defaultSweepLimit,
	})
	if err != nil {
		return nil, fmt.Errorf("list due subscriptions: %w", err)
	}

	result := &SweepResult{
		Failures:     make([]string, 0),
		PurgeTargets: make([]TenantRef, 0),
	}
	queued := make(map[pulid.ID]struct{})

	for _, sub := range due {
		result.Examined++
		expired, transitionErr := s.advance(ctx, sub, now, result)
		if transitionErr != nil {
			result.Failed++
			result.Failures = append(result.Failures, sub.OrganizationID.String())
			s.l.Error("failed to advance a cloud subscription",
				zap.String("organizationId", sub.OrganizationID.String()),
				zap.Error(transitionErr),
			)
			continue
		}
		if expired {
			queued[sub.OrganizationID] = struct{}{}
			result.PurgeTargets = append(result.PurgeTargets, TenantRef{
				OrganizationID: sub.OrganizationID,
				BusinessUnitID: sub.BusinessUnitID,
			})
		}
	}

	if err = s.collectAwaitingPurge(ctx, queued, result); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) advance(
	ctx context.Context,
	sub *subscription.Subscription,
	now int64,
	result *SweepResult,
) (bool, error) {
	tenantInfo := pagination.TenantInfo{OrgID: sub.OrganizationID, BuID: sub.BusinessUnitID}
	ctx = dbscope.WithTenant(ctx, tenantInfo.DBTenant())
	target := sub.EffectiveStatus(now)

	current := sub
	if current.Status == subscription.StatusTrialing &&
		(target == subscription.StatusReadOnly || target == subscription.StatusExpired) {
		updated, err := s.transition(ctx, current, subscription.StatusReadOnly)
		if err != nil {
			return false, err
		}
		current = updated
		result.ReadOnly++
		s.notify(ctx, current, noticeTrialEnded)
	}

	if current.Status != subscription.StatusReadOnly || target != subscription.StatusExpired {
		return false, nil
	}

	updated, err := s.transition(ctx, current, subscription.StatusExpired)
	if err != nil {
		return false, err
	}
	result.Expired++
	s.notify(ctx, updated, noticeExpired)
	s.revokeSessions(ctx, tenantInfo)

	return true, nil
}

func (s *Service) transition(
	ctx context.Context,
	sub *subscription.Subscription,
	status subscription.Status,
) (*subscription.Subscription, error) {
	updated, err := s.subscriptions.UpdateStatus(ctx, &repositories.UpdateSubscriptionStatusRequest{
		TenantInfo: pagination.TenantInfo{OrgID: sub.OrganizationID, BuID: sub.BusinessUnitID},
		ID:         sub.ID,
		Version:    sub.Version,
		Status:     status,
	})
	if err != nil {
		return nil, fmt.Errorf("move subscription to %s: %w", status, err)
	}
	s.plans.Invalidate(sub.OrganizationID)

	return updated, nil
}

func (s *Service) collectAwaitingPurge(
	ctx context.Context,
	queued map[pulid.ID]struct{},
	result *SweepResult,
) error {
	after := pulid.Nil
	collected := 0
	for collected < maxExpiredPerRun {
		page, err := s.subscriptions.ListExpired(ctx, &repositories.ListExpiredSubscriptionsRequest{
			AfterID: after,
			Limit:   expiredPageSize,
		})
		if err != nil {
			return fmt.Errorf("list expired subscriptions: %w", err)
		}

		for _, sub := range page {
			collected++
			if _, seen := queued[sub.OrganizationID]; seen {
				continue
			}
			queued[sub.OrganizationID] = struct{}{}
			result.PurgeTargets = append(result.PurgeTargets, TenantRef{
				OrganizationID: sub.OrganizationID,
				BusinessUnitID: sub.BusinessUnitID,
			})
		}

		if len(page) < expiredPageSize {
			return nil
		}
		after = page[len(page)-1].ID
	}

	return nil
}

type noticeKind int

const (
	noticeTrialEnded noticeKind = iota + 1
	noticeExpired
)

func (s *Service) notify(
	ctx context.Context,
	sub *subscription.Subscription,
	kind noticeKind,
) {
	tenantInfo := pagination.TenantInfo{OrgID: sub.OrganizationID, BuID: sub.BusinessUnitID}
	log := s.l.With(zap.String("organizationId", sub.OrganizationID.String()))
	if s.emails == nil {
		log.Warn("no platform email service is configured; lifecycle email not sent")
		return
	}

	profile, err := s.purge.OrganizationProfile(ctx, tenantInfo)
	if err != nil {
		log.Warn("failed to read the organization for its lifecycle email", zap.Error(err))
		profile = &repositories.TenantProfile{}
	}

	members, err := s.purge.ListMembers(ctx, tenantInfo)
	if err != nil {
		log.Error("failed to list the organization's members for its lifecycle email", zap.Error(err))
		return
	}

	for _, member := range members {
		if member.Username == tenant.SystemUsername || member.EmailAddress == "" {
			continue
		}

		switch kind {
		case noticeTrialEnded:
			err = s.emails.SendTrialEnded(ctx, &services.TrialEndedEmail{
				To:            member.EmailAddress,
				Name:          member.Name,
				CompanyName:   profile.Name,
				ReadOnlyUntil: sub.ReadOnlyUntil,
				Timezone:      profile.Timezone,
			})
		case noticeExpired:
			err = s.emails.SendAccountPurged(ctx, &services.AccountPurgedEmail{
				To:          member.EmailAddress,
				Name:        member.Name,
				CompanyName: profile.Name,
			})
		}
		if err != nil {
			log.Error("failed to send a cloud subscription email",
				zap.String("userId", member.UserID.String()),
				zap.Error(err),
			)
		}
	}
}

func (s *Service) revokeSessions(ctx context.Context, tenantInfo pagination.TenantInfo) {
	if s.sessions == nil {
		return
	}

	members, err := s.purge.ListMembers(ctx, tenantInfo)
	if err != nil {
		s.l.Error("failed to list members whose sessions should end",
			zap.String("organizationId", tenantInfo.OrgID.String()),
			zap.Error(err),
		)
		return
	}

	for _, member := range members {
		if member.OtherMemberships > 0 {
			continue
		}
		s.revokeUserSessions(ctx, member.UserID)
	}
}

func (s *Service) revokeUserSessions(ctx context.Context, userID pulid.ID) {
	if s.sessions == nil {
		return
	}

	if err := s.sessions.DeleteAllForUser(ctx, userID); err != nil {
		s.l.Error("failed to end a user's sessions",
			zap.String("userId", userID.String()),
			zap.Error(err),
		)
	}
}

func (s *Service) RequireExpired(ctx context.Context, ref TenantRef) error {
	tenantInfo := ref.TenantInfo()
	ctx = dbscope.WithTenant(ctx, tenantInfo.DBTenant())

	sub, err := s.subscriptions.GetByOrganization(ctx, repositories.GetSubscriptionRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return ErrNotExpired
		}
		return fmt.Errorf("read subscription: %w", err)
	}
	if sub.Status != subscription.StatusExpired {
		return ErrNotExpired
	}

	return nil
}

func (s *Service) ListMembers(
	ctx context.Context,
	ref TenantRef,
) ([]*repositories.TenantMember, error) {
	return s.purge.ListMembers(ctx, ref.TenantInfo())
}

func (s *Service) PurgeRows(
	ctx context.Context,
	ref TenantRef,
	batchSize, maxBatches int,
) (*repositories.PurgeTenantRowsResult, error) {
	return s.purge.PurgeRows(ctx, &repositories.PurgeTenantRowsRequest{
		TenantInfo: ref.TenantInfo(),
		BatchSize:  batchSize,
		MaxBatches: maxBatches,
	})
}

func (s *Service) PurgeStorage(ctx context.Context, ref TenantRef) (int64, error) {
	if s.storage == nil {
		return 0, ErrStorageUnsupported
	}

	return s.storage.DeletePrefix(ctx, ref.OrganizationID.String()+"/")
}

type PurgeUsersResult struct {
	Deleted     int `json:"deleted"`
	Deactivated int `json:"deactivated"`
	Reassigned  int `json:"reassigned"`
}

func (s *Service) PurgeUsers(
	ctx context.Context,
	ref TenantRef,
	userIDs []pulid.ID,
) (*PurgeUsersResult, error) {
	result := new(PurgeUsersResult)
	for _, userID := range userIDs {
		outcome, err := s.purge.PurgeUser(ctx, &repositories.PurgeTenantUserRequest{
			TenantInfo: ref.TenantInfo(),
			UserID:     userID,
		})
		if err != nil {
			return result, fmt.Errorf("purge user %s: %w", userID, err)
		}

		switch {
		case outcome.Deleted:
			result.Deleted++
			s.revokeUserSessions(ctx, userID)
		case outcome.Deactivated:
			result.Deactivated++
			s.revokeUserSessions(ctx, userID)
		case outcome.Reassigned:
			result.Reassigned++
		}
	}

	return result, nil
}

func (s *Service) Finalize(
	ctx context.Context,
	ref TenantRef,
) (*repositories.DeleteTenantResult, error) {
	result, err := s.purge.DeleteTenant(ctx, ref.TenantInfo())
	if err != nil {
		return nil, err
	}
	s.plans.Invalidate(ref.OrganizationID)

	return result, nil
}
