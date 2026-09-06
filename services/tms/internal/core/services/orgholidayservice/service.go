// Package orgholidayservice maintains the organisation holiday calendar that
// PTO day counting and blackout validation read from.
package orgholidayservice

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const realtimeResource = "org_holiday"

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.OrgHolidayRepository
	AuditService services.AuditService
	Realtime     services.RealtimeService `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         repositories.OrgHolidayRepository
	auditService services.AuditService
	realtime     services.RealtimeService
}

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.org-holiday"),
		repo:         p.Repo,
		auditService: p.AuditService,
		realtime:     p.Realtime,
	}
}

// ListYear returns the calendar for one year: every recurring entry plus the
// one-off entries dated inside it.
func (s *Service) ListYear(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	year int,
) ([]*worker.OrgHoliday, error) {
	if year <= 0 {
		year = time.Now().UTC().Year()
	}
	from := time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC).Unix()
	to := time.Date(year, time.December, 31, 0, 0, 0, 0, time.UTC).Unix()
	return s.repo.List(ctx, &repositories.ListOrgHolidaysRequest{
		TenantInfo: tenantInfo,
		From:       from,
		To:         to,
	})
}

// Calendar loads every holiday and blackout the organisation has so the PTO
// day walk can consult it for any range.
func (s *Service) Calendar(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*worker.HolidayCalendar, error) {
	rows, err := s.repo.List(ctx, &repositories.ListOrgHolidaysRequest{TenantInfo: tenantInfo})
	if err != nil {
		return nil, err
	}
	return worker.NewHolidayCalendar(rows), nil
}

func normalize(entity *worker.OrgHoliday) {
	entity.Name = strings.TrimSpace(entity.Name)
	entity.Description = strings.TrimSpace(entity.Description)
	if entity.HolidayDate > 0 {
		day := time.Unix(entity.HolidayDate, 0).UTC()
		entity.HolidayDate = time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC).Unix()
	}
	if entity.Kind == "" {
		entity.Kind = worker.HolidayKindHoliday
	}
}

func (s *Service) Create(
	ctx context.Context,
	entity *worker.OrgHoliday,
	userID pulid.ID,
) (*worker.OrgHoliday, error) {
	log := s.l.With(zap.String("operation", "Create"))
	normalize(entity)
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		log.Error("failed to create org holiday", zap.Error(err))
		return nil, err
	}

	s.audit(created, nil, permission.OpCreate, userID, "Holiday added", log)
	s.publish(ctx, created, permission.OpCreate, userID)
	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	entity *worker.OrgHoliday,
	userID pulid.ID,
) (*worker.OrgHoliday, error) {
	log := s.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))
	normalize(entity)
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	original, err := s.repo.GetByID(ctx, &repositories.GetOrgHolidayByIDRequest{
		ID:         entity.ID,
		TenantInfo: tenantOf(entity),
	})
	if err != nil {
		return nil, err
	}
	entity.CreatedAt = original.CreatedAt

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update org holiday", zap.Error(err))
		return nil, err
	}

	s.audit(updated, original, permission.OpUpdate, userID, "Holiday updated", log)
	s.publish(ctx, updated, permission.OpUpdate, userID)
	return updated, nil
}

func (s *Service) Delete(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	id pulid.ID,
	userID pulid.ID,
) error {
	log := s.l.With(zap.String("operation", "Delete"), zap.String("id", id.String()))
	original, err := s.repo.GetByID(ctx, &repositories.GetOrgHolidayByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}
	if err = s.repo.Delete(ctx, &repositories.GetOrgHolidayByIDRequest{
		ID:         id,
		TenantInfo: tenantInfo,
	}); err != nil {
		log.Error("failed to delete org holiday", zap.Error(err))
		return err
	}

	s.audit(original, nil, permission.OpDelete, userID, "Holiday removed", log)
	s.publish(ctx, original, permission.OpDelete, userID)
	return nil
}

func (s *Service) audit(
	current, previous *worker.OrgHoliday,
	operation permission.Operation,
	userID pulid.ID,
	comment string,
	log *zap.Logger,
) {
	if s.auditService == nil || userID.IsNil() {
		return
	}
	params := &services.LogActionParams{
		Resource:       permission.ResourceOrgHoliday,
		ResourceID:     current.GetResourceID(),
		Operation:      operation,
		UserID:         userID,
		CurrentState:   jsonutils.MustToJSON(current),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	opts := []services.LogOption{auditservice.WithComment(comment)}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
		opts = append(opts, auditservice.WithDiff(previous, current))
	}
	if err := s.auditService.LogAction(params, opts...); err != nil {
		log.Error("failed to log audit action", zap.Error(err))
	}
}

func (s *Service) publish(
	ctx context.Context,
	entity *worker.OrgHoliday,
	operation permission.Operation,
	userID pulid.ID,
) {
	if s.realtime == nil || entity == nil {
		return
	}
	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		ActorUserID:    userID,
		ActorType:      services.PrincipalTypeUser,
		ActorID:        userID,
		Resource:       realtimeResource,
		Action:         string(operation),
		RecordID:       entity.ID,
	}); err != nil {
		s.l.Warn("failed to publish holiday invalidation", zap.Error(err))
	}
}

func tenantOf(entity *worker.OrgHoliday) pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}
}
