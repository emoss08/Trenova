package agentprofileservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/billingcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/platformplan"
	"github.com/emoss08/trenova/internal/modules/identity/domain/shipmentcontrol"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/platformplanports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchcontrolports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	secondsPerDay = 24 * 60 * 60
	// A scheduled pass writes an organization's profile again at most once a
	// day and a person's at most once a week; a person asking, or guidance
	// changing, writes it at once.
	organizationInterval = 20 * 60 * 60
	userInterval         = 6 * secondsPerDay
)

var errProfileRefInvalid = errors.New(
	"a profile names an organization, and a person only for a person's profile",
)

type Params struct {
	fx.In

	Logger           *zap.Logger
	Profiles         agentprofileports.ProfileRepository
	Guidance         agentprofileports.GuidanceRepository
	Facts            agentprofileports.FactsReader
	Organizations    tenantports.OrganizationRepository
	Controls         tenantports.AgentControlRepository
	DispatchControls dispatchcontrolports.DispatchControlRepository `optional:"true"`
	BillingControls  tenantports.BillingControlRepository           `optional:"true"`
	ShipmentControls tenantports.ShipmentControlRepository          `optional:"true"`
	Plans            platformplanports.PlanService                  `optional:"true"`
	Scheduler        agentprofileports.Scheduler                    `optional:"true"`
	Audit            auditports.AuditService                        `optional:"true"`
	Realtime         realtimeports.RealtimeService                  `optional:"true"`
	// Memories says who manages agent memory, and so keeps guidance about
	// the organization at once. Without it everyone offers it.
	Memories agentmemoryports.AgentMemoryService `optional:"true"`
}

type Service struct {
	l                *zap.Logger
	profiles         agentprofileports.ProfileRepository
	guidance         agentprofileports.GuidanceRepository
	facts            agentprofileports.FactsReader
	organizations    tenantports.OrganizationRepository
	controls         tenantports.AgentControlRepository
	dispatchControls dispatchcontrolports.DispatchControlRepository
	billingControls  tenantports.BillingControlRepository
	shipmentControls tenantports.ShipmentControlRepository
	plans            platformplanports.PlanService
	scheduler        agentprofileports.Scheduler
	audit            auditports.AuditService
	realtime         realtimeports.RealtimeService
	memories         agentmemoryports.AgentMemoryService
	now              func() int64
}

var (
	_ agentprofileports.Service   = (*Service)(nil)
	_ agentprofileports.Generator = (*Service)(nil)
	_ agentprofileports.Reader    = (*Service)(nil)
)

func New(p Params) *Service { //nolint:gocritic // fx param structs are passed by value
	return &Service{
		l:                p.Logger.Named("service.agentprofile"),
		profiles:         p.Profiles,
		guidance:         p.Guidance,
		facts:            p.Facts,
		organizations:    p.Organizations,
		controls:         p.Controls,
		dispatchControls: p.DispatchControls,
		billingControls:  p.BillingControls,
		shipmentControls: p.ShipmentControls,
		plans:            p.Plans,
		scheduler:        p.Scheduler,
		audit:            p.Audit,
		realtime:         p.Realtime,
		memories:         p.Memories,
		now:              timeutils.NowUnix,
	}
}

// manages reports whether the actor is a person holding agent-memory manage:
// the one who keeps guidance about the organization at once and decides what
// others offered. The memory service owns that question.
func (s *Service) manages(ctx context.Context, actor *permissionports.RequestActor) bool {
	if s.memories == nil || actor == nil || !actor.IsUser() || actor.UserID.IsNil() {
		return false
	}

	return s.memories.MayManageShared(
		ctx,
		pagination.TenantInfo{OrgID: actor.OrganizationID, BuID: actor.BusinessUnitID},
		actor.UserID,
	)
}

func AsService(s *Service) agentprofileports.Service { return s }

func AsGenerator(s *Service) agentprofileports.Generator { return s }

func AsReader(s *Service) agentprofileports.Reader { return s }

func validRef(ref agentprofileports.SubjectRef) error {
	if !ref.Valid() {
		return errortypes.NewValidationError(
			"subject",
			errortypes.ErrInvalid,
			errProfileRefInvalid.Error(),
		)
	}

	return nil
}

// Prepare works out the facts and decides whether a new version is due. When
// it is, the plan carries the request that writes it.
func (s *Service) Prepare(
	ctx context.Context,
	req *agentprofileports.GenerateRequest,
) (*agentprofileports.GenerationPlan, error) {
	if req == nil {
		return nil, errProfileRefInvalid
	}
	if err := validRef(req.Ref); err != nil {
		return nil, err
	}
	trigger := req.Trigger
	if !trigger.IsValid() {
		trigger = agentprofile.TriggerScheduled
	}

	plan := &agentprofileports.GenerationPlan{
		Ref:           req.Ref,
		Trigger:       trigger,
		RequestedByID: req.RequestedByID,
		Facts:         agentprofile.Facts{},
		GuidanceIDs:   []pulid.ID{},
	}

	skip, err := s.gate(ctx, req.Ref.Tenant())
	if err != nil {
		return nil, err
	}
	if skip != "" {
		plan.Skip = skip
		return plan, nil
	}

	current, err := s.profiles.GetCurrent(ctx, req.Ref)
	if err != nil {
		return nil, err
	}
	if current != nil {
		plan.CurrentID = current.ID
		if trigger == agentprofile.TriggerScheduled && s.tooSoon(req.Ref.Subject, current) {
			plan.Skip = agentprofileports.SkipTooSoon
			return plan, nil
		}
	}

	facts, organizationName, err := s.workOutFacts(ctx, req.Ref)
	if err != nil {
		return nil, err
	}
	if req.Ref.Subject == agentprofile.SubjectUser && !HasActivity(facts) {
		plan.Skip = agentprofileports.SkipNoActivity
		return plan, nil
	}

	guidance, err := s.guidance.ListActive(ctx, req.Ref)
	if err != nil {
		return nil, err
	}

	plan.Facts = facts
	plan.Digest = Digest(facts, guidance)
	plan.VersionID = pulid.MustNew("aprf_")
	plan.OrganizationName = organizationName
	for _, line := range guidance {
		plan.GuidanceIDs = append(plan.GuidanceIDs, line.ID)
	}

	if current != nil && current.Digest == plan.Digest &&
		trigger != agentprofile.TriggerRequested {
		if err = s.profiles.MarkChecked(ctx, req.Ref, current.ID, s.now()); err != nil {
			return nil, err
		}
		plan.Skip = agentprofileports.SkipUnchanged

		return plan, nil
	}

	plan.Request = &llmports.StructuredCompletionRequest{
		TenantInfo: req.Ref.Tenant(),
		Task:       aiprovider.TaskAssistantChat,
		System:     systemPrompt(req.Ref.Subject),
		Context: contextSections(&writerInput{
			Subject:      req.Ref.Subject,
			Organization: organizationName,
			Facts:        facts,
			Guidance:     guidance,
			Current:      current,
		}),
		OutputSchema: outputSchema(),
		SchemaName:   schemaName,
		MaxTokens:    maxOutputTokens,
		Attribution:  attribution(req.Ref, req.RequestedByID),
	}

	return plan, nil
}

func attribution(
	ref agentprofileports.SubjectRef,
	requestedBy pulid.ID,
) aiusageports.AIUsageAttribution {
	user := requestedBy
	if ref.Subject == agentprofile.SubjectUser {
		user = ref.PersonID
	}

	return aiusageports.AIUsageAttribution{
		UserID:  user,
		Feature: aiusage.FeatureAgentProfile,
	}
}

// gate says why an organization gets no profile: its plan lacks agent
// automation, or it turned learning off or put agents in shadow mode.
func (s *Service) gate(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (agentprofileports.SkipReason, error) {
	if s.plans != nil {
		if err := s.plans.RequireCapability(
			ctx,
			tenantInfo,
			platformplan.CapabilityAgentAutomation,
		); err != nil {
			if errortypes.IsPlanRestrictionError(err) {
				return agentprofileports.SkipNotAllowed, nil
			}

			return "", fmt.Errorf("read the organization's plan: %w", err)
		}
	}

	control, err := s.controls.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return "", fmt.Errorf("read the organization's agent control: %w", err)
	}
	if control.LearningOff || control.ShadowMode {
		return agentprofileports.SkipLearningOff, nil
	}

	return "", nil
}

func (s *Service) tooSoon(subject agentprofile.Subject, current *agentprofile.Profile) bool {
	interval := int64(organizationInterval)
	if subject == agentprofile.SubjectUser {
		interval = userInterval
	}

	return s.now()-max(current.CreatedAt, current.CheckedAt) < interval
}

func (s *Service) workOutFacts(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
) (agentprofile.Facts, string, error) {
	now := s.now()
	tenantInfo := ref.Tenant()

	org, err := s.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
		TenantInfo:   tenantInfo,
		IncludeState: true,
	})
	if err != nil {
		return nil, "", fmt.Errorf("read the organization: %w", err)
	}

	if ref.Subject == agentprofile.SubjectUser {
		activity, uErr := s.facts.UserActivity(ctx, &agentprofileports.UserFactsRequest{
			Window: agentprofileports.FactsWindow{
				TenantInfo: tenantInfo,
				Start:      now - agentprofileports.UserWindowDays*secondsPerDay,
				End:        now,
			},
			UserID: ref.PersonID,
		})
		if uErr != nil {
			return nil, "", uErr
		}

		return UserFacts(activity, agentprofileports.UserWindowDays), org.Name, nil
	}

	activity, err := s.facts.OrganizationActivity(ctx, agentprofileports.FactsWindow{
		TenantInfo: tenantInfo,
		Start:      now - agentprofileports.OrganizationWindowDays*secondsPerDay,
		End:        now,
	})
	if err != nil {
		return nil, "", err
	}

	return OrganizationFacts(&OrganizationInputs{
		Organization:    org,
		Activity:        activity,
		DispatchControl: s.dispatchControl(ctx, tenantInfo),
		BillingControl:  s.billingControl(ctx, tenantInfo),
		ShipmentControl: s.shipmentControl(ctx, tenantInfo),
		WindowDays:      agentprofileports.OrganizationWindowDays,
	}), org.Name, nil
}

// The settings are optional facts: a setting that cannot be read is left out
// of the profile rather than failing it.
func (s *Service) dispatchControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) *dispatchcontrol.DispatchControl {
	if s.dispatchControls == nil {
		return nil
	}
	dc, err := s.dispatchControls.GetByOrgID(ctx, dispatchcontrolports.GetDispatchControlRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		s.l.Warn("agent profile: dispatch settings could not be read", zap.Error(err))
		return nil
	}

	return dc
}

func (s *Service) billingControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) *billingcontrol.BillingControl {
	if s.billingControls == nil {
		return nil
	}
	bc, err := s.billingControls.GetByOrgID(ctx, tenantInfo.OrgID)
	if err != nil {
		s.l.Warn("agent profile: billing settings could not be read", zap.Error(err))
		return nil
	}

	return bc
}

func (s *Service) shipmentControl(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) *shipmentcontrol.ShipmentControl {
	if s.shipmentControls == nil {
		return nil
	}
	sc, err := s.shipmentControls.Get(ctx, tenantports.GetShipmentControlRequest{
		TenantInfo: tenantInfo,
	})
	if err != nil {
		s.l.Warn("agent profile: shipment settings could not be read", zap.Error(err))
		return nil
	}

	return sc
}

// Finish keeps what the model wrote as the current version, or records why it
// could not be kept; a failed version never replaces the current one.
func (s *Service) Finish(
	ctx context.Context,
	req *agentprofileports.FinishGenerationRequest,
) (*agentprofileports.GenerationOutcome, error) {
	if req == nil || !req.Plan.Ready() {
		return &agentprofileports.GenerationOutcome{}, nil
	}
	plan := req.Plan

	entity := &agentprofile.Profile{
		ID:             plan.VersionID,
		OrganizationID: plan.Ref.OrganizationID,
		BusinessUnitID: plan.Ref.BusinessUnitID,
		Subject:        plan.Ref.Subject,
		UserID:         plan.Ref.PersonPtr(),
		Trigger:        plan.Trigger,
		Facts:          plan.Facts,
		GuidanceIDs:    plan.GuidanceIDs,
		Digest:         plan.Digest,
		CheckedAt:      s.now(),
	}
	if plan.RequestedByID.IsNotNil() {
		requestedBy := plan.RequestedByID
		entity.RequestedByID = &requestedBy
	}

	if req.Result != nil {
		entity.Model = req.Result.ModelIdentifier
		entity.ProviderID = req.Result.ProviderID
		entity.InputTokens = max(req.Result.InputTokens, 0)
		entity.OutputTokens = max(req.Result.OutputTokens, 0)
	}

	content, failure := s.written(ctx, req)
	if failure != "" {
		entity.Fail(failure)
		if err := s.validate(entity); err != nil {
			return nil, err
		}
		saved, err := s.profiles.RecordFailure(ctx, entity)
		if err != nil {
			return nil, err
		}
		s.announce(ctx, plan.Ref, saved.ID, "failed")

		return &agentprofileports.GenerationOutcome{ProfileID: saved.ID, Status: saved.Status}, nil
	}

	entity.Status = agentprofile.StatusCurrent
	entity.Content = content
	if err := s.validate(entity); err != nil {
		return nil, err
	}
	saved, err := s.profiles.Publish(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.announce(ctx, plan.Ref, saved.ID, "updated")

	return &agentprofileports.GenerationOutcome{ProfileID: saved.ID, Status: saved.Status}, nil
}

func (s *Service) written(
	ctx context.Context,
	req *agentprofileports.FinishGenerationRequest,
) (content, failure string) {
	if message := strings.TrimSpace(req.Error); message != "" {
		return "", message
	}
	if req.Result == nil {
		return "", "The model returned no profile."
	}

	guidance, err := s.guidanceTexts(ctx, req.Plan)
	if err != nil {
		return "", err.Error()
	}
	if name := strings.TrimSpace(req.Plan.OrganizationName); name != "" {
		guidance = append(guidance, name)
	}
	content, err = readReply(req.Result.Text, req.Plan.Facts, guidance)
	if err != nil {
		s.l.Warn("agent profile: the model's profile was not kept",
			zap.String("organization", req.Plan.Ref.OrganizationID.String()),
			zap.String("subject", string(req.Plan.Ref.Subject)),
			zap.Error(err))

		return "", err.Error()
	}

	return content, ""
}

// guidanceTexts reads back the guidance the plan was written from, so a figure
// a person gave ("about 200 loads a week in peak") is one the profile may cite.
func (s *Service) guidanceTexts(
	ctx context.Context,
	plan *agentprofileports.GenerationPlan,
) ([]string, error) {
	if len(plan.GuidanceIDs) == 0 {
		return nil, nil
	}
	lines, err := s.guidance.List(ctx, &agentprofileports.ListGuidanceRequest{
		Ref:            plan.Ref,
		IncludeRetired: true,
		Limit:          agentprofile.MaxActiveGuidance * 2,
	})
	if err != nil {
		return nil, fmt.Errorf("read the guidance the profile was written from: %w", err)
	}

	wanted := make(map[pulid.ID]struct{}, len(plan.GuidanceIDs))
	for _, id := range plan.GuidanceIDs {
		wanted[id] = struct{}{}
	}
	texts := make([]string, 0, len(plan.GuidanceIDs))
	for _, line := range lines {
		if _, ok := wanted[line.ID]; ok {
			texts = append(texts, line.Content)
		}
	}

	return texts, nil
}

func (s *Service) validate(entity interface {
	Validate(*errortypes.MultiError)
},
) error {
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) Prune(ctx context.Context, ref agentprofileports.SubjectRef) error {
	if err := validRef(ref); err != nil {
		return err
	}
	if _, err := s.profiles.Prune(ctx, agentprofileports.PruneProfilesRequest{
		Ref:  ref,
		Keep: agentprofileports.KeptVersions,
	}); err != nil {
		return err
	}

	return nil
}

func (s *Service) View(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
) (*agentprofileports.ProfileView, error) {
	if err := validRef(ref); err != nil {
		return nil, err
	}

	current, err := s.profiles.GetCurrent(ctx, ref)
	if err != nil {
		return nil, err
	}
	latest, err := s.profiles.ListHistory(ctx, agentprofileports.ListProfileHistoryRequest{
		Ref:   ref,
		Limit: 1,
	})
	if err != nil {
		return nil, err
	}
	guidance, err := s.guidance.List(ctx, &agentprofileports.ListGuidanceRequest{Ref: ref})
	if err != nil {
		return nil, err
	}

	view := &agentprofileports.ProfileView{Ref: ref, Current: current, Guidance: guidance}
	if len(latest) == 1 && latest[0].Status == agentprofile.StatusFailed {
		view.LatestFailure = latest[0]
	}

	return view, nil
}

func (s *Service) History(
	ctx context.Context,
	req agentprofileports.ListProfileHistoryRequest,
) ([]*agentprofile.Profile, error) {
	if err := validRef(req.Ref); err != nil {
		return nil, err
	}

	return s.profiles.ListHistory(ctx, req)
}

func (s *Service) ListGuidance(
	ctx context.Context,
	req *agentprofileports.ListGuidanceRequest,
) ([]*agentprofile.Guidance, error) {
	if err := validRef(req.Ref); err != nil {
		return nil, err
	}

	return s.guidance.List(ctx, req)
}

// AddGuidance keeps a line about how to read a profile and asks for the
// profile to be written again with it. A line about the organization shapes
// what every agent is told, so only a person who manages agent memory keeps
// one at once; from anyone else it is offered, kept as Suggested until an
// administrator approves or dismisses it. A person's own lines are theirs.
func (s *Service) AddGuidance(
	ctx context.Context,
	req *agentprofileports.AddGuidanceRequest,
) (*agentprofile.Guidance, error) {
	entity, err := s.PlanGuidance(ctx, req)
	if err != nil {
		return nil, err
	}

	created, err := s.guidance.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	if created.IsSuggested() {
		s.logGuidance(
			created,
			nil,
			req.Actor,
			permission.OpCreate,
			guidanceComment(created, "Offered"),
		)
		s.announce(ctx, req.Ref, created.ID, "offered")

		return created, nil
	}

	s.logGuidance(created, nil, req.Actor, permission.OpCreate, guidanceComment(created, "Added"))
	s.announce(ctx, req.Ref, created.ID, "guided")
	s.schedule(ctx, req.Ref, agentprofile.TriggerGuidance, req.Actor.UserID)

	return created, nil
}

func (s *Service) PlanGuidance(
	ctx context.Context,
	req *agentprofileports.AddGuidanceRequest,
) (*agentprofile.Guidance, error) {
	if req == nil || req.Actor == nil || req.Actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError("Guidance is given by a person")
	}
	if err := validRef(req.Ref); err != nil {
		return nil, err
	}

	count, err := s.guidance.CountOpen(ctx, req.Ref)
	if err != nil {
		return nil, err
	}
	if count >= agentprofile.MaxActiveGuidance {
		return nil, errortypes.NewValidationError(
			"content",
			errortypes.ErrInvalid,
			"This profile already has 50 lines of guidance, kept or offered. Retire or dismiss one that no longer holds first.",
		)
	}

	source := req.Source
	if !source.IsValid() {
		source = agentprofile.GuidanceSourcePerson
	}
	status := agentprofile.GuidanceStatusActive
	if req.Ref.Subject == agentprofile.SubjectOrganization && !s.manages(ctx, req.Actor) {
		status = agentprofile.GuidanceStatusSuggested
	}
	entity := &agentprofile.Guidance{
		OrganizationID: req.Ref.OrganizationID,
		BusinessUnitID: req.Ref.BusinessUnitID,
		Subject:        req.Ref.Subject,
		UserID:         req.Ref.PersonPtr(),
		Content:        strings.TrimSpace(req.Content),
		Status:         status,
		Source:         source,
		CreatedByID:    req.Actor.UserID,
	}
	if req.ThreadID.IsNotNil() {
		thread := req.ThreadID
		entity.ThreadID = &thread
	}
	if err = s.validate(entity); err != nil {
		return nil, err
	}

	return entity, nil
}

// ApproveGuidance keeps a line someone offered about the organization and
// asks for the profile to be written again with it. The caller has checked
// the actor manages agent memory; a line that is not waiting is left as it is.
func (s *Service) ApproveGuidance(
	ctx context.Context,
	req *agentprofileports.DecideGuidanceRequest,
) (*agentprofile.Guidance, error) {
	entity, err := s.offered(ctx, req)
	if err != nil {
		return nil, err
	}
	if !entity.IsSuggested() {
		return entity, nil
	}

	previous := jsonutils.MustToJSON(entity)
	entity.Approve()
	if err = s.validate(entity); err != nil {
		return nil, err
	}
	updated, err := s.guidance.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logGuidance(updated, previous, req.Actor, permission.OpManage,
		guidanceComment(updated, "Approved offered"))
	s.announce(ctx, req.Ref, updated.ID, "guided")
	s.schedule(ctx, req.Ref, agentprofile.TriggerGuidance, req.Actor.UserID)

	return updated, nil
}

// DismissGuidance turns an offered line down; it never shapes the profile, so
// nothing is written again.
func (s *Service) DismissGuidance(
	ctx context.Context,
	req *agentprofileports.DecideGuidanceRequest,
) (*agentprofile.Guidance, error) {
	entity, err := s.offered(ctx, req)
	if err != nil {
		return nil, err
	}
	if !entity.IsSuggested() {
		return entity, nil
	}

	previous := jsonutils.MustToJSON(entity)
	entity.Dismiss(req.Actor.UserID, s.now())
	if err = s.validate(entity); err != nil {
		return nil, err
	}
	updated, err := s.guidance.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logGuidance(updated, previous, req.Actor, permission.OpManage,
		guidanceComment(updated, "Dismissed offered"))
	s.announce(ctx, req.Ref, updated.ID, "guided")

	return updated, nil
}

// offered reads a line a decision names. Lines are offered about the
// organization alone; one for another subject reads as not found.
func (s *Service) offered(
	ctx context.Context,
	req *agentprofileports.DecideGuidanceRequest,
) (*agentprofile.Guidance, error) {
	if req == nil || req.Actor == nil || req.Actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError("Offered guidance is decided by a person")
	}
	if err := validRef(req.Ref); err != nil {
		return nil, err
	}
	if req.Ref.Subject != agentprofile.SubjectOrganization {
		return nil, errortypes.NewNotFoundError("Guidance not found")
	}

	entity, err := s.guidance.GetByID(ctx, agentprofileports.GetGuidanceRequest{
		ID:         req.ID,
		TenantInfo: req.Ref.Tenant(),
	})
	if err != nil {
		return nil, err
	}
	if !sameSubject(entity, req.Ref) {
		return nil, errortypes.NewNotFoundError("Guidance not found")
	}

	return entity, nil
}

// RetireGuidance stops a line from shaping the profile and asks for the
// profile to be written again without it. Guidance for another subject reads
// as not found, so a person can never retire another person's lines.
func (s *Service) RetireGuidance(
	ctx context.Context,
	req *agentprofileports.RetireGuidanceRequest,
) (*agentprofile.Guidance, error) {
	if req == nil || req.Actor == nil || req.Actor.UserID.IsNil() {
		return nil, errortypes.NewAuthorizationError("Guidance is retired by a person")
	}
	if err := validRef(req.Ref); err != nil {
		return nil, err
	}

	entity, err := s.guidance.GetByID(ctx, agentprofileports.GetGuidanceRequest{
		ID:         req.ID,
		TenantInfo: req.Ref.Tenant(),
	})
	if err != nil {
		return nil, err
	}
	if !sameSubject(entity, req.Ref) {
		return nil, errortypes.NewNotFoundError("Guidance not found")
	}
	if !entity.IsActive() {
		return entity, nil
	}

	previous := jsonutils.MustToJSON(entity)
	entity.Retire(req.Actor.UserID, s.now())
	if err = s.validate(entity); err != nil {
		return nil, err
	}
	updated, err := s.guidance.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logGuidance(
		updated,
		previous,
		req.Actor,
		permission.OpUpdate,
		guidanceComment(updated, "Retired"),
	)
	s.announce(ctx, req.Ref, updated.ID, "guided")
	s.schedule(ctx, req.Ref, agentprofile.TriggerGuidance, req.Actor.UserID)

	return updated, nil
}

func sameSubject(entity *agentprofile.Guidance, ref agentprofileports.SubjectRef) bool {
	if entity.Subject != ref.Subject {
		return false
	}
	if ref.Subject != agentprofile.SubjectUser {
		return entity.UserID == nil || entity.UserID.IsNil()
	}

	return entity.UserID != nil && *entity.UserID == ref.PersonID
}

// Regenerate asks a worker to write the profile now, whatever the schedule.
// It refuses where a scheduled pass would skip, so the person is told why
// nothing will happen.
func (s *Service) Regenerate(ctx context.Context, req *agentprofileports.RegenerateRequest) error {
	if req == nil || req.Actor == nil || req.Actor.UserID.IsNil() {
		return errortypes.NewAuthorizationError("A profile is written again when a person asks")
	}
	if err := validRef(req.Ref); err != nil {
		return err
	}

	skip, err := s.gate(ctx, req.Ref.Tenant())
	if err != nil {
		return err
	}
	switch skip {
	case agentprofileports.SkipNotAllowed:
		return errortypes.NewBusinessError(
			"Your plan does not include agent automation, so profiles are not written.")
	case agentprofileports.SkipLearningOff:
		return errortypes.NewBusinessError(
			"Agents are not learning from their work in this organization. Turn learning on in AI Control first.",
		)
	case agentprofileports.SkipUnchanged,
		agentprofileports.SkipTooSoon,
		agentprofileports.SkipNoActivity:
	}
	if s.scheduler == nil {
		return errortypes.NewBusinessError(
			"Profiles cannot be written right now. Try again shortly.",
		)
	}
	if req.Ref.Subject == agentprofile.SubjectUser {
		facts, _, factsErr := s.workOutFacts(ctx, req.Ref)
		if factsErr != nil {
			return factsErr
		}
		if !HasActivity(facts) {
			return errortypes.NewBusinessError(
				"There isn't enough of your work on record yet to write about. Work with agents for a while and try again.",
			)
		}
	}

	return s.scheduler.Request(ctx, &agentprofileports.GenerateRequest{
		Ref:           req.Ref,
		Trigger:       agentprofile.TriggerRequested,
		RequestedByID: req.Actor.UserID,
	})
}

func (s *Service) schedule(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
	trigger agentprofile.Trigger,
	by pulid.ID,
) {
	if s.scheduler == nil {
		return
	}
	if err := s.scheduler.Request(ctx, &agentprofileports.GenerateRequest{
		Ref:           ref,
		Trigger:       trigger,
		RequestedByID: by,
	}); err != nil {
		s.l.Warn(
			"agent profile: the profile could not be queued to be written again; the nightly pass will",
			zap.String("organization", ref.OrganizationID.String()),
			zap.Error(err),
		)
	}
}

// ForPrompt reads what a turn is told. A profile that cannot be read leaves
// the turn without it rather than failing the turn.
func (s *Service) ForPrompt(
	ctx context.Context,
	req agentprofileports.PromptProfilesRequest,
) (agentprofileports.PromptProfiles, error) {
	var out agentprofileports.PromptProfiles
	tenantInfo := req.TenantInfo
	if tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return out, nil
	}

	var errs []error
	if req.Organization {
		org, err := s.profiles.GetCurrent(ctx, agentprofileports.OrganizationSubject(tenantInfo))
		switch {
		case err != nil:
			errs = append(errs, err)
		case org != nil:
			out.Organization = org.Content
			out.OrganizationCheckedAt = org.CheckedAt
		}
	}
	if req.UserID.IsNotNil() {
		user, err := s.profiles.GetCurrent(
			ctx,
			agentprofileports.UserSubject(tenantInfo, req.UserID),
		)
		switch {
		case err != nil:
			errs = append(errs, err)
		case user != nil:
			out.User = user.Content
			out.UserCheckedAt = user.CheckedAt
		}
	}

	return out, errors.Join(errs...)
}

func (s *Service) announce(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
	recordID pulid.ID,
	action string,
) {
	if s.realtime == nil {
		return
	}

	req := &realtimeports.PublishResourceInvalidationRequest{
		OrganizationID: ref.OrganizationID,
		BusinessUnitID: ref.BusinessUnitID,
		Resource:       agentprofileports.RealtimeResource,
		Action:         action,
		RecordID:       recordID,
		ActorType:      permissionports.PrincipalTypeSystem,
		Entity: map[string]any{
			"id":      recordID,
			"subject": ref.Subject,
		},
	}
	if ref.Subject == agentprofile.SubjectUser {
		req.AudienceUserID = ref.PersonID
	}
	if err := s.realtime.PublishResourceInvalidation(ctx, req); err != nil {
		s.l.Warn("agent profile: the change could not be announced", zap.Error(err))
	}
}

func guidanceComment(entity *agentprofile.Guidance, verb string) string {
	about := "the organization's profile"
	if entity.Subject == agentprofile.SubjectUser {
		about = "their own profile"
	}

	return verb + " guidance for " + about
}

func (s *Service) logGuidance(
	entity *agentprofile.Guidance,
	previous map[string]any,
	actor *permissionports.RequestActor,
	op permission.Operation,
	comment string,
) {
	if s.audit == nil {
		return
	}

	auditActor := actor.AuditActorOrSystem()
	if err := s.audit.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceAgentMemory,
		ResourceID:     entity.ID.String(),
		Operation:      op,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(entity),
		PreviousState:  previous,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}, withComment(comment)); err != nil {
		s.l.Error("failed to log agent profile guidance audit", zap.Error(err))
	}
}

func withComment(comment string) auditports.LogOption {
	return func(entry *audit.Entry) error {
		entry.Comment = comment
		return nil
	}
}
