package agentprofileservice

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/testutil/mocks/platformplanportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/tenantportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testNow = int64(1_790_000_000)

type fakeProfiles struct {
	mu        sync.Mutex
	current   map[agentprofileports.SubjectRef]*agentprofile.Profile
	published []*agentprofile.Profile
	failed    []*agentprofile.Profile
	checked   []pulid.ID
}

func newFakeProfiles() *fakeProfiles {
	return &fakeProfiles{current: map[agentprofileports.SubjectRef]*agentprofile.Profile{}}
}

func (f *fakeProfiles) GetCurrent(
	_ context.Context,
	ref agentprofileports.SubjectRef,
) (*agentprofile.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.current[ref], nil
}

func (f *fakeProfiles) GetByID(
	context.Context,
	agentprofileports.GetProfileRequest,
) (*agentprofile.Profile, error) {
	return nil, errortypes.NewNotFoundError("Profile version not found")
}

func (f *fakeProfiles) ListHistory(
	_ context.Context,
	req agentprofileports.ListProfileHistoryRequest,
) ([]*agentprofile.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	if n := len(f.failed); n > 0 {
		return []*agentprofile.Profile{f.failed[n-1]}, nil
	}
	if current := f.current[req.Ref]; current != nil {
		return []*agentprofile.Profile{current}, nil
	}

	return []*agentprofile.Profile{}, nil
}

func (f *fakeProfiles) Publish(
	_ context.Context,
	entity *agentprofile.Profile,
) (*agentprofile.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, kept := range f.published {
		if entity.ID.IsNotNil() && kept.ID == entity.ID {
			return kept, nil
		}
	}
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("aprf_")
	}
	entity.Status = agentprofile.StatusCurrent
	f.published = append(f.published, entity)
	f.current[refFor(entity)] = entity

	return entity, nil
}

func (f *fakeProfiles) RecordFailure(
	_ context.Context,
	entity *agentprofile.Profile,
) (*agentprofile.Profile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entity.ID = pulid.MustNew("aprf_")
	f.failed = append(f.failed, entity)

	return entity, nil
}

func (f *fakeProfiles) MarkChecked(
	_ context.Context,
	_ agentprofileports.SubjectRef,
	id pulid.ID,
	_ int64,
) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.checked = append(f.checked, id)

	return nil
}

func (f *fakeProfiles) Prune(context.Context, agentprofileports.PruneProfilesRequest) (int, error) {
	return 0, nil
}

func refFor(entity *agentprofile.Profile) agentprofileports.SubjectRef {
	ref := agentprofileports.SubjectRef{
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
		Subject:        entity.Subject,
	}
	if entity.UserID != nil {
		ref.PersonID = *entity.UserID
	}

	return ref
}

type fakeGuidance struct {
	mu    sync.Mutex
	lines []*agentprofile.Guidance
}

func (f *fakeGuidance) matching(ref agentprofileports.SubjectRef, active bool) []*agentprofile.Guidance {
	out := make([]*agentprofile.Guidance, 0, len(f.lines))
	for _, line := range f.lines {
		if sameSubject(line, ref) && (!active || line.IsActive()) {
			out = append(out, line)
		}
	}

	return out
}

func (f *fakeGuidance) ListActive(
	_ context.Context,
	ref agentprofileports.SubjectRef,
) ([]*agentprofile.Guidance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.matching(ref, true), nil
}

func (f *fakeGuidance) List(
	_ context.Context,
	req *agentprofileports.ListGuidanceRequest,
) ([]*agentprofile.Guidance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.matching(req.Ref, !req.IncludeRetired), nil
}

func (f *fakeGuidance) CountActive(_ context.Context, ref agentprofileports.SubjectRef) (int, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	return len(f.matching(ref, true)), nil
}

func (f *fakeGuidance) GetByID(
	_ context.Context,
	req agentprofileports.GetGuidanceRequest,
) (*agentprofile.Guidance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	for _, line := range f.lines {
		if line.ID == req.ID {
			return line, nil
		}
	}

	return nil, errortypes.NewNotFoundError("Guidance not found")
}

func (f *fakeGuidance) Create(
	_ context.Context,
	entity *agentprofile.Guidance,
) (*agentprofile.Guidance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	entity.ID = pulid.MustNew("aprg_")
	f.lines = append(f.lines, entity)

	return entity, nil
}

func (f *fakeGuidance) Update(
	_ context.Context,
	entity *agentprofile.Guidance,
) (*agentprofile.Guidance, error) {
	return entity, nil
}

type fakeFacts struct {
	organization *agentprofileports.OrganizationActivity
	user         *agentprofileports.UserActivity
}

func (f *fakeFacts) OrganizationActivity(
	context.Context,
	agentprofileports.FactsWindow,
) (*agentprofileports.OrganizationActivity, error) {
	return f.organization, nil
}

func (f *fakeFacts) UserActivity(
	context.Context,
	*agentprofileports.UserFactsRequest,
) (*agentprofileports.UserActivity, error) {
	return f.user, nil
}

func (f *fakeFacts) ActiveUsers(
	context.Context,
	agentprofileports.ActiveUsersRequest,
) ([]pulid.ID, error) {
	return nil, nil
}

type fakeControls struct {
	control tenant.AgentControl
}

func (f *fakeControls) GetOrCreate(context.Context, pagination.TenantInfo) (*tenant.AgentControl, error) {
	control := f.control

	return &control, nil
}

func (f *fakeControls) Update(_ context.Context, entity *tenant.AgentControl) (*tenant.AgentControl, error) {
	return entity, nil
}

type fakeScheduler struct {
	requests []*agentprofileports.GenerateRequest
}

func (f *fakeScheduler) Request(_ context.Context, req *agentprofileports.GenerateRequest) error {
	f.requests = append(f.requests, req)

	return nil
}

type harness struct {
	svc       *Service
	profiles  *fakeProfiles
	guidance  *fakeGuidance
	facts     *fakeFacts
	controls  *fakeControls
	scheduler *fakeScheduler
	plans     *platformplanportsmock.MockPlanService
	tenant    pagination.TenantInfo
	userID    pulid.ID
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	h := &harness{
		profiles:  newFakeProfiles(),
		guidance:  &fakeGuidance{},
		facts:     &fakeFacts{organization: sampleActivity()},
		controls:  &fakeControls{},
		scheduler: &fakeScheduler{},
		plans:     platformplanportsmock.NewMockPlanService(t),
		tenant:    pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		userID:    pulid.MustNew("usr_"),
	}

	orgs := tenantportsmock.NewMockOrganizationRepository(t)
	orgs.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&tenant.Organization{
		Name:                   "Acme Freight",
		AssetOperationsEnabled: true,
	}, nil).Maybe()
	h.plans.EXPECT().RequireCapability(mock.Anything, mock.Anything, mock.Anything).
		Return(nil).Maybe()

	h.svc = New(Params{
		Logger:        zap.NewNop(),
		Profiles:      h.profiles,
		Guidance:      h.guidance,
		Facts:         h.facts,
		Organizations: orgs,
		Controls:      h.controls,
		Plans:         h.plans,
		Scheduler:     h.scheduler,
	})
	h.svc.now = func() int64 { return testNow }

	return h
}

func (h *harness) person() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    h.userID,
		UserID:         h.userID,
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
	}
}

func (h *harness) org() agentprofileports.SubjectRef {
	return agentprofileports.OrganizationSubject(h.tenant)
}

func reply(profile string) *agentports.StructuredCompletionResult {
	return &agentports.StructuredCompletionResult{
		Text:            `{"profile":` + quote(profile) + `,"notes":""}`,
		ModelIdentifier: "test-model",
	}
}

func quote(text string) string {
	return `"` + strings.ReplaceAll(text, `"`, `\"`) + `"`
}

func TestPrepare_WritesAPlanAttributedToTheProfileFeature(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	plan, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{Ref: h.org()})
	require.NoError(t, err)

	require.True(t, plan.Ready())
	assert.Equal(t, agentprofile.TriggerScheduled, plan.Trigger)
	assert.Equal(t, aiusage.FeatureAgentProfile, plan.Request.Attribution.Feature)
	assert.NotEmpty(t, plan.Facts)
	assert.Len(t, plan.Digest, agentprofile.DigestChars)
	assert.Contains(t, plan.Request.System, "Write only from the facts and guidance")
}

func TestPrepare_SkipsWhenLearningIsOffOrInShadowMode(t *testing.T) {
	t.Parallel()

	for _, control := range []tenant.AgentControl{{LearningOff: true}, {ShadowMode: true}} {
		h := newHarness(t)
		h.controls.control = control

		plan, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{Ref: h.org()})
		require.NoError(t, err)
		assert.Equal(t, agentprofileports.SkipLearningOff, plan.Skip)
		assert.False(t, plan.Ready())
	}
}

func TestPrepare_ASchedulePassBeforeTheProfileIsDueWritesNothing(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.profiles.current[h.org()] = &agentprofile.Profile{
		ID: pulid.MustNew("aprf_"), Status: agentprofile.StatusCurrent, CreatedAt: testNow - 3600,
	}

	plan, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{Ref: h.org()})
	require.NoError(t, err)
	assert.Equal(t, agentprofileports.SkipTooSoon, plan.Skip)

	asked, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{
		Ref:     h.org(),
		Trigger: agentprofile.TriggerRequested,
	})
	require.NoError(t, err)
	assert.True(t, asked.Ready(), "a person asking writes it whenever they ask")
}

func TestPrepare_UnchangedFactsOnlyMarkTheVersionChecked(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	first, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{Ref: h.org()})
	require.NoError(t, err)

	currentID := pulid.MustNew("aprf_")
	h.profiles.current[h.org()] = &agentprofile.Profile{
		ID:        currentID,
		Status:    agentprofile.StatusCurrent,
		Digest:    first.Digest,
		CreatedAt: testNow - 2*organizationInterval,
	}

	again, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{Ref: h.org()})
	require.NoError(t, err)
	assert.Equal(t, agentprofileports.SkipUnchanged, again.Skip)
	assert.Equal(t, []pulid.ID{currentID}, h.profiles.checked)
}

func TestPrepare_APersonWithNoWorkOnRecordGetsNoProfile(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.facts.user = &agentprofileports.UserActivity{JobTitle: "Dispatcher"}

	plan, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{
		Ref: agentprofileports.UserSubject(h.tenant, h.userID),
	})
	require.NoError(t, err)
	assert.Equal(t, agentprofileports.SkipNoActivity, plan.Skip)
}

func TestFinish_KeepsAProfileThatCitesOnlyTheFacts(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	plan, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{Ref: h.org()})
	require.NoError(t, err)

	outcome, err := h.svc.Finish(t.Context(), &agentprofileports.FinishGenerationRequest{
		Plan:   plan,
		Result: reply("Acme Freight runs its own fleet, mostly sleepers, and brokers about 40% of its moves."),
	})
	require.NoError(t, err)
	assert.Equal(t, agentprofile.StatusCurrent, outcome.Status)
	require.Len(t, h.profiles.published, 1)
	assert.Equal(t, "test-model", h.profiles.published[0].Model)
}

func TestFinish_AFigureTheFactsDoNotHoldFailsTheVersion(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	plan, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{Ref: h.org()})
	require.NoError(t, err)

	outcome, err := h.svc.Finish(t.Context(), &agentprofileports.FinishGenerationRequest{
		Plan:   plan,
		Result: reply("Acme Freight runs 4,000 trucks."),
	})
	require.NoError(t, err)
	assert.Equal(t, agentprofile.StatusFailed, outcome.Status)
	assert.Empty(t, h.profiles.published, "a failed version never replaces the current one")
	require.Len(t, h.profiles.failed, 1)
	assert.Contains(t, h.profiles.failed[0].ErrorMessage, "4,000")
}

func TestFinish_AFigureAPersonGaveMayBeCited(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	_, err := h.svc.AddGuidance(t.Context(), &agentprofileports.AddGuidanceRequest{
		Ref:     h.org(),
		Content: "In peak season we move about 350 loads a week.",
		Actor:   h.person(),
	})
	require.NoError(t, err)

	plan, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{
		Ref:     h.org(),
		Trigger: agentprofile.TriggerGuidance,
	})
	require.NoError(t, err)
	require.Len(t, plan.GuidanceIDs, 1)

	outcome, err := h.svc.Finish(t.Context(), &agentprofileports.FinishGenerationRequest{
		Plan:   plan,
		Result: reply("In peak season it moves about 350 loads a week."),
	})
	require.NoError(t, err)
	assert.Equal(t, agentprofile.StatusCurrent, outcome.Status)
}

func TestFinish_AModelErrorIsRecordedAsAFailedVersion(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	plan, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{Ref: h.org()})
	require.NoError(t, err)

	outcome, err := h.svc.Finish(t.Context(), &agentprofileports.FinishGenerationRequest{
		Plan:  plan,
		Error: "The model could not write the profile: rate limited",
	})
	require.NoError(t, err)
	assert.Equal(t, agentprofile.StatusFailed, outcome.Status)
}

func TestAddGuidance_IsAPersonsAndAsksForTheProfileAgain(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	_, err := h.svc.AddGuidance(t.Context(), &agentprofileports.AddGuidanceRequest{
		Ref:     h.org(),
		Content: "The Laredo lane is seasonal.",
	})
	require.Error(t, err, "guidance with nobody behind it is refused")

	created, err := h.svc.AddGuidance(t.Context(), &agentprofileports.AddGuidanceRequest{
		Ref:     h.org(),
		Content: "  The Laredo lane is seasonal.  ",
		Source:  agentprofile.GuidanceSourceAgent,
		Actor:   h.person(),
	})
	require.NoError(t, err)
	assert.Equal(t, "The Laredo lane is seasonal.", created.Content)
	assert.Equal(t, agentprofile.GuidanceSourceAgent, created.Source)
	assert.Equal(t, h.userID, created.CreatedByID)

	require.Len(t, h.scheduler.requests, 1)
	assert.Equal(t, agentprofile.TriggerGuidance, h.scheduler.requests[0].Trigger)
}

func TestAddGuidance_RefusesPastTheCap(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	for range agentprofile.MaxActiveGuidance {
		_, err := h.svc.AddGuidance(t.Context(), &agentprofileports.AddGuidanceRequest{
			Ref: h.org(), Content: "A line.", Actor: h.person(),
		})
		require.NoError(t, err)
	}

	_, err := h.svc.AddGuidance(t.Context(), &agentprofileports.AddGuidanceRequest{
		Ref: h.org(), Content: "One too many.", Actor: h.person(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "50 lines")
}

func TestRetireGuidance_AnotherPersonsLineReadsAsNotFound(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	theirs := pulid.MustNew("usr_")
	line, err := h.svc.AddGuidance(t.Context(), &agentprofileports.AddGuidanceRequest{
		Ref:     agentprofileports.UserSubject(h.tenant, theirs),
		Content: "I only handle the Midwest accounts.",
		Actor:   h.person(),
	})
	require.NoError(t, err)

	_, err = h.svc.RetireGuidance(t.Context(), &agentprofileports.RetireGuidanceRequest{
		Ref:   agentprofileports.UserSubject(h.tenant, h.userID),
		ID:    line.ID,
		Actor: h.person(),
	})
	assert.True(t, errortypes.IsNotFoundError(err))

	retired, err := h.svc.RetireGuidance(t.Context(), &agentprofileports.RetireGuidanceRequest{
		Ref:   agentprofileports.UserSubject(h.tenant, theirs),
		ID:    line.ID,
		Actor: h.person(),
	})
	require.NoError(t, err)
	assert.Equal(t, agentprofile.GuidanceStatusRetired, retired.Status)
}

func TestRegenerate_SaysWhyWhenLearningIsOff(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.controls.control.LearningOff = true

	err := h.svc.Regenerate(t.Context(), &agentprofileports.RegenerateRequest{
		Ref:   h.org(),
		Actor: h.person(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Turn learning on")
	assert.Empty(t, h.scheduler.requests)
}

func TestForPrompt_ReadsTheCurrentVersions(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.profiles.current[h.org()] = &agentprofile.Profile{
		Status: agentprofile.StatusCurrent, Content: "Runs a regional fleet.", CheckedAt: testNow,
	}
	h.profiles.current[agentprofileports.UserSubject(h.tenant, h.userID)] = &agentprofile.Profile{
		Status: agentprofile.StatusCurrent, Content: "Dispatches regional loads.",
	}

	both, err := h.svc.ForPrompt(t.Context(), agentprofileports.PromptProfilesRequest{
		TenantInfo:   h.tenant,
		UserID:       h.userID,
		Organization: true,
	})
	require.NoError(t, err)
	assert.Equal(t, "Runs a regional fleet.", both.Organization)
	assert.Equal(t, "Dispatches regional loads.", both.User)

	orgOnly, err := h.svc.ForPrompt(t.Context(), agentprofileports.PromptProfilesRequest{
		TenantInfo:   h.tenant,
		Organization: true,
	})
	require.NoError(t, err)
	assert.Empty(t, orgOnly.User)
}

func TestPrepare_AnUnchangedProfileIsNotWorkedOutAgainUntilItIsDue(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	person := agentprofileports.UserSubject(h.tenant, h.userID)
	h.profiles.current[person] = &agentprofile.Profile{
		ID:        pulid.MustNew("aprf_"),
		Status:    agentprofile.StatusCurrent,
		CreatedAt: testNow - 30*secondsPerDay,
		CheckedAt: testNow - secondsPerDay,
	}

	plan, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{Ref: person})
	require.NoError(t, err)
	assert.Equal(t, agentprofileports.SkipTooSoon, plan.Skip,
		"a person's facts checked yesterday are not worked out again for a week")
}

func TestFinish_RunAgainAfterACommitKeepsOneVersion(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	plan, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{Ref: h.org()})
	require.NoError(t, err)
	require.True(t, plan.VersionID.IsNotNil(), "the plan names the version before it is written")

	finish := &agentprofileports.FinishGenerationRequest{
		Plan:   plan,
		Result: reply("Acme Freight runs its own fleet."),
	}
	first, err := h.svc.Finish(t.Context(), finish)
	require.NoError(t, err)
	second, err := h.svc.Finish(t.Context(), finish)
	require.NoError(t, err)

	assert.Equal(t, first.ProfileID, second.ProfileID)
	assert.Equal(t, plan.VersionID, first.ProfileID)
	assert.Len(t, h.profiles.published, 1, "a retried activity does not stack a second version")
}

func TestFinish_TheOrganizationsOwnNameIsNotAFigure(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.svc.organizations = tenantportsmock.NewMockOrganizationRepository(t)
	h.svc.organizations.(*tenantportsmock.MockOrganizationRepository).EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&tenant.Organization{Name: "Route 66 Logistics", AssetOperationsEnabled: true}, nil)

	plan, err := h.svc.Prepare(t.Context(), &agentprofileports.GenerateRequest{Ref: h.org()})
	require.NoError(t, err)
	assert.Equal(t, "Route 66 Logistics", plan.OrganizationName)

	outcome, err := h.svc.Finish(t.Context(), &agentprofileports.FinishGenerationRequest{
		Plan:   plan,
		Result: reply("Route 66 Logistics runs its own fleet, mostly sleepers."),
	})
	require.NoError(t, err)
	assert.Equal(t, agentprofile.StatusCurrent, outcome.Status)
}

func TestRegenerate_SaysWhenAPersonHasNoWorkToWriteAbout(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	h.facts.user = &agentprofileports.UserActivity{JobTitle: "Dispatcher"}

	err := h.svc.Regenerate(t.Context(), &agentprofileports.RegenerateRequest{
		Ref:   agentprofileports.UserSubject(h.tenant, h.userID),
		Actor: h.person(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "enough of your work")
	assert.Empty(t, h.scheduler.requests, "nothing is queued that would write nothing")
}
