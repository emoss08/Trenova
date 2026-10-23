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

	"github.com/emoss08/trenova/internal/modules/agent/app/briefingservice/briefingfacts"
	"github.com/emoss08/trenova/internal/modules/agent/app/briefingservice/briefingwriter"
	"github.com/emoss08/trenova/internal/modules/agent/domain/briefing"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/briefingports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/app/planservice"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
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
	Repo         briefingports.BriefingRepository
	Organization tenantports.OrganizationRepository
	Writer       *briefingwriter.Service
	Permissions  permissionports.PermissionEngine
	Realtime     agentports.RealtimeService `optional:"true"`
	// Days writes a page a person asked for again, on a worker.
	Days    briefingports.BriefingDayWriter
	Sources briefingfacts.Sources
	Plans   platformplanports.PlanService `optional:"true"`
}

type Service struct {
	l            *zap.Logger
	repo         briefingports.BriefingRepository
	organization tenantports.OrganizationRepository
	facts        *briefingfacts.Builder
	writer       *briefingwriter.Service
	permissions  permissionports.PermissionEngine
	realtime     agentports.RealtimeService
	days         briefingports.BriefingDayWriter
	plans        platformplanports.PlanService
	now          func() int64
}

var _ briefingports.BriefingService = (*Service)(nil)

func New(p Params) *Service {
	return &Service{
		l:            p.Logger.Named("service.briefing"),
		repo:         p.Repo,
		organization: p.Organization,
		facts:        briefingfacts.NewBuilder(p.Sources),
		writer:       p.Writer,
		permissions:  p.Permissions,
		realtime:     p.Realtime,
		days:         p.Days,
		plans:        p.Plans,
		now:          timeutils.NowUnix,
	}
}

func AsService(s *Service) briefingports.BriefingService { return s }

// WriteForDay gathers the day once and writes every role's page from it.
//
// The gathering is shared deliberately: five roles asking the same eight
// repositories the same eight questions would be forty queries for one
// organization's morning, and the roles would not even agree with each
// other if a number moved between them.
func (s *Service) WriteForDay(
	ctx context.Context,
	req briefingports.WriteBriefingRequest,
) (*briefingports.WriteBriefingResult, error) {
	organization, err := s.organization.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, fmt.Errorf("load organization: %w", err)
	}

	timezone := timeutils.LoadLocation(organization.Timezone).String()
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

	result := &briefingports.WriteBriefingResult{}
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
		if written.Narrated {
			entity.ModelIdentifier = written.ModelIdentifier
			entity.ProviderID = written.ProviderID
		}
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
// current day, cut to the sections they may read. A missing briefing is not
// an error: the morning job may simply not have run yet, and the caller
// says so.
func (s *Service) Today(
	ctx context.Context,
	req briefingports.GetBriefingRequest,
	actor *permissionports.RequestActor,
) (*briefing.Briefing, error) {
	if err := s.assertMayRead(ctx, actor); err != nil {
		return nil, err
	}

	day := req.BriefingDate
	if day == "" {
		location, err := orgzone.Location(ctx, s.organization, req.TenantInfo)
		if err != nil {
			return nil, err
		}
		day = timeutils.CurrentDateInTimezone(location.String())
	}

	page, err := s.repo.GetForDay(ctx, briefingports.GetBriefingForDayRequest{
		TenantInfo:   req.TenantInfo,
		RoleKey:      roleOrDefault(req.RoleKey),
		BriefingDate: day,
	})
	if err != nil {
		return nil, err
	}

	return s.readable(ctx, actor, page)
}

func (s *Service) Get(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	actor *permissionports.RequestActor,
) (*briefing.Briefing, error) {
	if err := s.assertMayRead(ctx, actor); err != nil {
		return nil, err
	}

	page, err := s.repo.GetByID(ctx, briefingports.GetBriefingByIDRequest{
		ID:         id,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}

	return s.readable(ctx, actor, page)
}

func (s *Service) List(
	ctx context.Context,
	req briefingports.ListBriefingsRequest,
	actor *permissionports.RequestActor,
) ([]*briefing.Briefing, error) {
	if err := s.assertMayRead(ctx, actor); err != nil {
		return nil, err
	}

	pages, err := s.repo.List(ctx, briefingports.ListBriefingsQuery{
		TenantInfo: req.TenantInfo,
		RoleKey:    req.RoleKey,
		Limit:      req.Limit,
	})
	if err != nil {
		return nil, err
	}
	access, err := s.readAccess(ctx, actor)
	if err != nil {
		return nil, err
	}
	access.withholdAll(pages)

	return pages, nil
}

func (s *Service) MarkRead(
	ctx context.Context,
	tenant pagination.TenantInfo,
	id pulid.ID,
	actor *permissionports.RequestActor,
) (*briefing.Briefing, error) {
	if err := s.assertMayRead(ctx, actor); err != nil {
		return nil, err
	}

	page, err := s.repo.MarkRead(
		ctx,
		briefingports.GetBriefingByIDRequest{ID: id, TenantInfo: tenant},
		s.now(),
	)
	if err != nil {
		return nil, err
	}

	return s.readable(ctx, actor, page)
}

// Regenerate writes today's briefing again from current figures. It is the
// create permission rather than the read one, because it spends a model
// call and replaces what a colleague may already have read.
func (s *Service) Regenerate(
	ctx context.Context,
	req briefingports.GetBriefingRequest,
	actor *permissionports.RequestActor,
) (*briefing.Briefing, error) {
	if err := s.assertMay(ctx, actor, permission.OpCreate); err != nil {
		return nil, err
	}
	if err := planservice.RequireCapability(
		ctx,
		s.plans,
		req.TenantInfo,
		platformplan.CapabilityAgentAutomation,
	); err != nil {
		return nil, err
	}

	role := roleOrDefault(req.RoleKey)
	result, err := s.days.WriteForDay(ctx, briefingports.WriteBriefingRequest{
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

	return s.readable(ctx, actor, result.Briefings[0])
}

// readable is a page as its reader may see it, or nil when there is none.
func (s *Service) readable(
	ctx context.Context,
	actor *permissionports.RequestActor,
	page *briefing.Briefing,
) (*briefing.Briefing, error) {
	if page == nil {
		return nil, nil //nolint:nilnil // the morning job has not written the page yet
	}
	access, err := s.readAccess(ctx, actor)
	if err != nil {
		return nil, err
	}
	access.withhold(page)

	return page, nil
}

func (s *Service) assertMayRead(ctx context.Context, actor *permissionports.RequestActor) error {
	return s.assertMay(ctx, actor, permission.OpRead)
}

func (s *Service) assertMay(
	ctx context.Context,
	actor *permissionports.RequestActor,
	operation permission.Operation,
) error {
	if s.permissions == nil || actor == nil {
		return nil
	}

	result, err := s.permissions.Check(ctx, &permissionports.PermissionCheckRequest{
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
