// Package briefingservice writes and serves the morning briefing.
//
// The order matters and is the whole design: the figures are gathered
// first and stored, the page is laid out from them, and only then is a
// model asked for wording that is checked back against those same figures.
// A briefing is therefore never blocked on a provider, never delayed by
// one, and never says a number nobody computed.
package briefingservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/briefingservice/briefingfacts"
	"github.com/emoss08/trenova/internal/core/services/briefingservice/briefingwriter"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/realtimeinvalidation"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// RealtimeResource is the resource a written briefing invalidates, so the
// morning widget refreshes itself when the job finishes rather than when
// somebody reloads.
const RealtimeResource = "briefings"

// BriefingPath is where a reader opens the page.
const BriefingPath = "/desk"

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.BriefingRepository
	Organization repositories.OrganizationCacheRepository
	Writer       *briefingwriter.Service
	Permissions  services.PermissionEngine
	Realtime     services.RealtimeService `optional:"true"`
	Sources      briefingfacts.Sources
}

type Service struct {
	l            *zap.Logger
	repo         repositories.BriefingRepository
	organization repositories.OrganizationCacheRepository
	facts        *briefingfacts.Builder
	writer       *briefingwriter.Service
	permissions  services.PermissionEngine
	realtime     services.RealtimeService
	now          func() int64
}

var _ services.BriefingService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.briefing"),
		repo:         p.Repo,
		organization: p.Organization,
		facts:        briefingfacts.NewBuilder(p.Sources),
		writer:       p.Writer,
		permissions:  p.Permissions,
		realtime:     p.Realtime,
		now:          timeutils.NowUnix,
	}
}

func AsService(s *Service) services.BriefingService { return s }

// WriteForDay gathers the day once and writes every role's page from it.
//
// The gathering is shared deliberately: five roles asking the same eight
// repositories the same eight questions would be forty queries for one
// organization's morning, and the roles would not even agree with each
// other if a number moved between them.
func (s *Service) WriteForDay(
	ctx context.Context,
	req services.WriteBriefingRequest,
) (*services.WriteBriefingResult, error) {
	organization, err := s.organization.GetByID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, fmt.Errorf("load organization: %w", err)
	}

	timezone := timeutils.NormalizeTimezone(organization.Timezone)
	now := req.Now
	if now <= 0 {
		now = s.now()
	}
	dayStart, err := timeutils.DayStartUnix(now, timezone)
	if err != nil {
		return nil, fmt.Errorf("day start: %w", err)
	}
	dayEnd, err := timeutils.DayEndUnix(now, timezone)
	if err != nil {
		return nil, fmt.Errorf("day end: %w", err)
	}
	day := timeutils.CurrentDateInTimezone(timezone)
	if req.BriefingDate != "" {
		day = req.BriefingDate
	}

	facts := s.facts.Build(ctx, briefingfacts.Request{
		TenantInfo: req.TenantInfo,
		Now:        now,
		DayStart:   dayStart,
		DayEnd:     dayEnd,
		Timezone:   timezone,
	})

	result := &services.WriteBriefingResult{}
	roles := req.Roles
	if len(roles) == 0 {
		roles = briefing.AllRoleKeys()
	}
	for _, role := range roles {
		written, wErr := s.writeRole(ctx, writeRoleParams{
			tenant:       req.TenantInfo,
			role:         role,
			day:          day,
			facts:        facts,
			organization: organization.Name,
			narrate:      !req.SkipNarration,
		})
		if wErr != nil {
			s.l.Error("failed to write briefing",
				zap.String("role", string(role)),
				zap.String("organization", req.TenantInfo.OrgID.String()),
				zap.Error(wErr),
			)
			result.Failed = append(result.Failed, string(role))

			continue
		}
		result.Written++
		if written.Narrated {
			result.Narrated++
		}
		result.Briefings = append(result.Briefings, written)
	}

	s.publish(ctx, req.TenantInfo)

	return result, nil
}

type writeRoleParams struct {
	tenant       pagination.TenantInfo
	role         briefing.RoleKey
	day          string
	facts        *briefingfacts.Facts
	organization string
	narrate      bool
}

func (s *Service) writeRole(
	ctx context.Context,
	p writeRoleParams,
) (*briefing.Briefing, error) {
	sections := briefingfacts.SectionsFor(p.role, p.facts.Sections)
	entity := &briefing.Briefing{
		OrganizationID: p.tenant.OrgID,
		BusinessUnitID: p.tenant.BuID,
		RoleKey:        p.role,
		BriefingDate:   p.day,
		Status:         briefing.StatusReady,
		Sections:       sections,
		Facts:          p.facts.Map(),
	}

	// A role with nothing computed for it is still written, so the page
	// says the morning is quiet rather than showing yesterday's.
	if len(sections) == 0 {
		entity.Headline = "Nothing needs you this morning."
	} else {
		entity.Headline = sections[0].Summary
	}

	if p.narrate && s.writer != nil && len(sections) > 0 {
		written := s.writer.Write(ctx, &briefingwriter.Request{
			TenantInfo:       p.tenant,
			Role:             p.role,
			OrganizationName: p.organization,
			Sections:         sections,
			Supported:        p.facts.Supported(),
		})
		entity.Headline = written.Headline
		entity.Narrated = written.Narrated
		for index := range entity.Sections {
			if body, ok := written.Bodies[entity.Sections[index].Key]; ok {
				entity.Sections[index].Body = body
			}
		}
	}

	entity.Normalize()
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return s.repo.Upsert(ctx, entity)
}

// Today reads the briefing for the reader's role on the organization's
// current day. A missing briefing is not an error: the morning job may
// simply not have run yet, and the caller says so.
func (s *Service) Today(
	ctx context.Context,
	req services.GetBriefingRequest,
	actor *services.RequestActor,
) (*briefing.Briefing, error) {
	if err := s.assertMayRead(ctx, actor); err != nil {
		return nil, err
	}

	organization, err := s.organization.GetByID(ctx, req.TenantInfo.OrgID)
	if err != nil {
		return nil, fmt.Errorf("load organization: %w", err)
	}

	day := req.BriefingDate
	if day == "" {
		day = timeutils.CurrentDateInTimezone(timeutils.NormalizeTimezone(organization.Timezone))
	}

	return s.repo.GetForDay(ctx, repositories.GetBriefingForDayRequest{
		TenantInfo:   req.TenantInfo,
		RoleKey:      roleOrDefault(req.RoleKey),
		BriefingDate: day,
	})
}

func (s *Service) Get(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	actor *services.RequestActor,
) (*briefing.Briefing, error) {
	if err := s.assertMayRead(ctx, actor); err != nil {
		return nil, err
	}

	return s.repo.GetByID(ctx, repositories.GetBriefingByIDRequest{ID: id, TenantInfo: tenant})
}

func (s *Service) List(
	ctx context.Context,
	req services.ListBriefingsRequest,
	actor *services.RequestActor,
) ([]*briefing.Briefing, error) {
	if err := s.assertMayRead(ctx, actor); err != nil {
		return nil, err
	}

	return s.repo.List(ctx, repositories.ListBriefingsRequest{
		TenantInfo: req.TenantInfo,
		RoleKey:    req.RoleKey,
		Limit:      req.Limit,
	})
}

func (s *Service) MarkRead(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	actor *services.RequestActor,
) (*briefing.Briefing, error) {
	if err := s.assertMayRead(ctx, actor); err != nil {
		return nil, err
	}

	return s.repo.MarkRead(
		ctx,
		repositories.GetBriefingByIDRequest{ID: id, TenantInfo: tenant},
		s.now(),
	)
}

// Regenerate writes today's briefing again from current figures. It is the
// create permission rather than the read one, because it spends a model
// call and replaces what a colleague may already have read.
func (s *Service) Regenerate(
	ctx context.Context,
	req services.GetBriefingRequest,
	actor *services.RequestActor,
) (*briefing.Briefing, error) {
	if err := s.assertMay(ctx, actor, permission.OpCreate); err != nil {
		return nil, err
	}

	role := roleOrDefault(req.RoleKey)
	result, err := s.WriteForDay(ctx, services.WriteBriefingRequest{
		TenantInfo:   req.TenantInfo,
		Roles:        []briefing.RoleKey{role},
		BriefingDate: req.BriefingDate,
	})
	if err != nil {
		return nil, err
	}
	if len(result.Briefings) == 0 {
		return nil, errortypes.NewBusinessError("Today's briefing could not be written")
	}

	return result.Briefings[0], nil
}

func (s *Service) assertMayRead(ctx context.Context, actor *services.RequestActor) error {
	return s.assertMay(ctx, actor, permission.OpRead)
}

func (s *Service) assertMay(
	ctx context.Context,
	actor *services.RequestActor,
	operation permission.Operation,
) error {
	if s.permissions == nil || actor == nil {
		return nil
	}

	result, err := s.permissions.Check(ctx, &services.PermissionCheckRequest{
		PrincipalType:  actor.PrincipalType,
		PrincipalID:    actor.PrincipalID,
		UserID:         actor.UserID,
		APIKeyID:       actor.APIKeyID,
		BusinessUnitID: actor.BusinessUnitID,
		OrganizationID: actor.OrganizationID,
		Resource:       permission.ResourceBriefing.String(),
		Operation:      operation,
	})
	if err != nil {
		return err
	}
	if result == nil || !result.Allowed {
		return errortypes.NewAuthorizationError(
			"You do not have permission to {0} briefings",
			operation,
		)
	}

	return nil
}

func (s *Service) publish(ctx context.Context, tenant pagination.TenantInfo) {
	if s.realtime == nil {
		return
	}

	if err := realtimeinvalidation.Publish(ctx, s.realtime, &realtimeinvalidation.PublishParams{
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		Resource:       RealtimeResource,
		Action:         "updated",
	}); err != nil {
		s.l.Warn("failed to publish briefing invalidation", zap.Error(err))
	}
}

// roleOrDefault falls back to the shared briefing everybody can read, so a
// reader whose role is not set still opens a page.
func roleOrDefault(role briefing.RoleKey) briefing.RoleKey {
	if role.IsValid() {
		return role
	}

	return briefing.RoleGeneral
}

// Headline is the one line a widget shows when it has room for nothing
// else. It reads the model's wording when there is any and the computed
// wording otherwise, so a caller never has to know which it got.
func Headline(entity *briefing.Briefing) string {
	if entity == nil {
		return ""
	}
	if headline := strings.TrimSpace(entity.Headline); headline != "" {
		return headline
	}
	for _, section := range entity.Sections {
		if read := strings.TrimSpace(section.Read()); read != "" {
			return read
		}
	}

	return ""
}
