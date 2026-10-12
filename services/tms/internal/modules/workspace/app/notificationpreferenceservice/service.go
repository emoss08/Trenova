package notificationpreferenceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const maxPauseSeconds = 14 * 24 * 60 * 60

type Params struct {
	fx.In

	Logger *zap.Logger
	Repo   notificationports.PreferenceRepository
}

type Service struct {
	l    *zap.Logger
	repo notificationports.PreferenceRepository
}

func New(p Params) *Service {
	return &Service{
		l:    p.Logger.Named("service.notification-preference"),
		repo: p.Repo,
	}
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*notification.Preference, error) {
	stored, found, err := s.repo.Get(ctx, notificationports.GetPreferenceRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if !found {
		return notification.DefaultPreference(tenantInfo.OrgID, tenantInfo.BuID, tenantInfo.UserID), nil
	}

	return stored, nil
}

func (s *Service) Update(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entity *notification.Preference,
) (*notification.Preference, error) {
	entity.OrganizationID = tenantInfo.OrgID
	entity.BusinessUnitID = tenantInfo.BuID
	entity.UserID = tenantInfo.UserID
	if entity.DisabledEventTypes == nil {
		entity.DisabledEventTypes = []string{}
	}
	entity.Normalize()

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	validatePause(entity, timeutils.NowUnix(), multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	if entity.PausedUntil != nil && *entity.PausedUntil <= timeutils.NowUnix() {
		entity.PausedUntil = nil
	}

	saved, err := s.repo.Upsert(ctx, entity)
	if err != nil {
		s.l.Error("failed to save notification preferences",
			zap.String("userID", tenantInfo.UserID.String()),
			zap.Error(err),
		)
		return nil, err
	}

	return saved, nil
}

func validatePause(entity *notification.Preference, now int64, multiErr *errortypes.MultiError) {
	if entity.PausedUntil == nil {
		return
	}
	if *entity.PausedUntil > now+maxPauseSeconds {
		multiErr.Add("pausedUntil", errortypes.ErrInvalid, "Notifications can be paused for at most two weeks")
	}
}
