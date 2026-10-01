package jurisdictionruleservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/referencedataguard"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type repoStub struct {
	repositories.JurisdictionRuleRepository

	stored  *jurisdictionrule.JurisdictionRule
	created *jurisdictionrule.JurisdictionRule
	updated *jurisdictionrule.JurisdictionRule
	verify  *repositories.VerifyJurisdictionRuleRequest
}

func (s *repoStub) GetByID(
	_ context.Context, _ *repositories.GetJurisdictionRuleByIDRequest,
) (*jurisdictionrule.JurisdictionRule, error) {
	return s.stored, nil
}

func (s *repoStub) Create(
	_ context.Context, e *jurisdictionrule.JurisdictionRule,
) (*jurisdictionrule.JurisdictionRule, error) {
	s.created = e
	return e, nil
}

func (s *repoStub) Update(
	_ context.Context, e *jurisdictionrule.JurisdictionRule,
) (*jurisdictionrule.JurisdictionRule, error) {
	s.updated = e
	return e, nil
}

func (s *repoStub) Verify(
	_ context.Context, req *repositories.VerifyJurisdictionRuleRequest,
) (*jurisdictionrule.JurisdictionRule, error) {
	s.verify = req
	return s.stored, nil
}

func validRule() *jurisdictionrule.JurisdictionRule {
	return &jurisdictionrule.JurisdictionRule{
		ID:                 pulid.MustNew("jrl_"),
		StateID:            pulid.MustNew("us_"),
		Status:             jurisdictionrule.StatusActive,
		MaxWidthFeet:       jurisdictionrule.FederalMaxWidthFeet,
		MaxHeightFeet:      jurisdictionrule.FederalMaxHeightFeet,
		MaxLengthFeet:      jurisdictionrule.FederalMaxLengthFeet,
		MaxWeightPounds:    jurisdictionrule.FederalMaxWeightPounds,
		PermitLeadTimeDays: 1,
		PermitValidityDays: 5,
		// Matches the schema default, so the holiday case below is a real change
		// rather than a no-op against a zero value.
		HolidayRestricted: true,
		VerificationState: jurisdictionrule.VerificationVerified,
	}
}

func newService(t *testing.T, repo repositories.JurisdictionRuleRepository) *service {
	t.Helper()

	auditService := mocks.NewMockAuditService(t)
	auditService.EXPECT().
		LogAction(mock.Anything, mock.Anything).
		Return(nil).
		Maybe()

	return &service{
		repo:          repo,
		auditService:  auditService,
		referenceData: selfHostedGuard(t),
		l:             zap.NewNop(),
	}
}

func selfHostedGuard(t *testing.T) *referencedataguard.Guard {
	t.Helper()

	guard, err := referencedataguard.FromPlatform(&config.PlatformConfig{
		Mode: config.PlatformModeSelfHosted,
	})
	require.NoError(t, err)

	return guard
}

func operatorActor() *services.RequestActor {
	return &services.RequestActor{
		PrincipalType:  services.PrincipalTypeUser,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	}
}

// Verification is earned, never asserted by the caller. A create that could
// claim Verified would let a row arrive already carrying the badge an operator
// relies on when deciding not to pull a permit.
func TestCreate_AlwaysLandsUnverified(t *testing.T) {
	repo := &repoStub{}
	svc := newService(t, repo)

	entity := validRule()
	verifiedAt := int64(1700000000)
	entity.VerifiedAt = &verifiedAt

	created, err := svc.Create(t.Context(), entity, operatorActor())
	require.NoError(t, err)

	assert.Equal(t, jurisdictionrule.VerificationUnverified, created.VerificationState)
	assert.Nil(t, created.VerifiedAt)
}

// Changing what a row claims is legal invalidates any prior confirmation.
// Otherwise someone verifies a conservative row and then raises the limits
// under the badge that verification earned.
func TestUpdate_ChangingALimitResetsVerification(t *testing.T) {
	stored := validRule()
	repo := &repoStub{stored: stored}
	svc := newService(t, repo)

	edited := validRule()
	edited.ID = stored.ID
	edited.MaxWidthFeet = 12.0

	updated, err := svc.Update(t.Context(), edited, operatorActor())
	require.NoError(t, err)

	assert.Equal(t, jurisdictionrule.VerificationUnverified, updated.VerificationState)
	assert.Nil(t, updated.VerifiedAt)
}

// Correcting a citation is not a change to the numbers, so it must not throw
// away a verification someone did the work for.
func TestUpdate_EditingOnlyTheSourceKeepsVerification(t *testing.T) {
	verifiedAt := int64(1700000000)
	stored := validRule()
	stored.VerifiedAt = &verifiedAt
	repo := &repoStub{stored: stored}
	svc := newService(t, repo)

	edited := validRule()
	edited.ID = stored.ID
	edited.SourceNote = "Corrected the citation to the current statute revision"
	edited.SourceURL = "https://example.gov/statute"

	updated, err := svc.Update(t.Context(), edited, operatorActor())
	require.NoError(t, err)

	assert.Equal(t, jurisdictionrule.VerificationVerified, updated.VerificationState)
	require.NotNil(t, updated.VerifiedAt)
	assert.Equal(t, verifiedAt, *updated.VerifiedAt)
}

func TestUpdate_EveryGovernedFieldResetsVerification(t *testing.T) {
	cases := map[string]func(*jurisdictionrule.JurisdictionRule){
		"height":        func(r *jurisdictionrule.JurisdictionRule) { r.MaxHeightFeet = 14 },
		"length":        func(r *jurisdictionrule.JurisdictionRule) { r.MaxLengthFeet = 60 },
		"weight":        func(r *jurisdictionrule.JurisdictionRule) { r.MaxWeightPounds = 90000 },
		"leadTime":      func(r *jurisdictionrule.JurisdictionRule) { r.PermitLeadTimeDays = 9 },
		"validity":      func(r *jurisdictionrule.JurisdictionRule) { r.PermitValidityDays = 9 },
		"daylightOnly":  func(r *jurisdictionrule.JurisdictionRule) { r.DaylightOnly = true },
		"rushHour":      func(r *jurisdictionrule.JurisdictionRule) { r.RushHourRestricted = true },
		"weekend":       func(r *jurisdictionrule.JurisdictionRule) { r.WeekendRestricted = true },
		"holiday":       func(r *jurisdictionrule.JurisdictionRule) { r.HolidayRestricted = false },
		"superloadWide": func(r *jurisdictionrule.JurisdictionRule) { w := 16.0; r.SuperloadWidthFeet = &w },
	}

	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			stored := validRule()
			svc := newService(t, &repoStub{stored: stored})

			edited := validRule()
			edited.ID = stored.ID
			mutate(edited)

			updated, err := svc.Update(t.Context(), edited, operatorActor())
			require.NoError(t, err)
			assert.Equal(t, jurisdictionrule.VerificationUnverified, updated.VerificationState)
		})
	}
}

func TestVerify_RequiresATraceableNote(t *testing.T) {
	svc := newService(t, &repoStub{stored: validRule()})

	_, err := svc.Verify(t.Context(), &services.VerifyJurisdictionRuleRequest{
		RuleID:     pulid.MustNew("jrl_"),
		Actor:      operatorActor(),
		State:      jurisdictionrule.VerificationVerified,
		SourceNote: "checked",
	})

	require.Error(t, err)
}

// Unverified is where a row starts and where an edit returns it. Letting it be
// set directly would be a way to drop a verification without changing anything,
// leaving no diff to explain it.
func TestVerify_RejectsMarkingARowUnverified(t *testing.T) {
	svc := newService(t, &repoStub{stored: validRule()})

	_, err := svc.Verify(t.Context(), &services.VerifyJurisdictionRuleRequest{
		RuleID:     pulid.MustNew("jrl_"),
		Actor:      operatorActor(),
		State:      jurisdictionrule.VerificationUnverified,
		SourceNote: "Checked against the state permit office handbook",
	})

	require.Error(t, err)
}

func TestVerify_AcceptsDisputed(t *testing.T) {
	repo := &repoStub{stored: validRule()}
	svc := newService(t, repo)

	_, err := svc.Verify(t.Context(), &services.VerifyJurisdictionRuleRequest{
		RuleID:     pulid.MustNew("jrl_"),
		Actor:      operatorActor(),
		State:      jurisdictionrule.VerificationDisputed,
		SourceNote: "State permit office contradicts this width; escalated",
	})

	require.NoError(t, err)
	require.NotNil(t, repo.verify)
	assert.Equal(t, jurisdictionrule.VerificationDisputed, repo.verify.State)
	assert.Positive(t, repo.verify.VerifiedAt)
}

// The audit entry for global reference data must not claim a tenant scope the
// table does not have.
func TestLogAction_LeavesTenantUnsetForGlobalData(t *testing.T) {
	auditService := mocks.NewMockAuditService(t)

	var captured *services.LogActionParams
	auditService.EXPECT().
		LogAction(mock.Anything, mock.Anything).
		RunAndReturn(func(p *services.LogActionParams, _ ...services.LogOption) error {
			captured = p
			return nil
		})

	svc := &service{
		repo:          &repoStub{},
		auditService:  auditService,
		referenceData: selfHostedGuard(t),
		l:             zap.NewNop(),
	}

	_, err := svc.Create(t.Context(), validRule(), operatorActor())
	require.NoError(t, err)

	require.NotNil(t, captured)
	assert.Equal(t, permission.ResourceJurisdictionRule, captured.Resource)
	assert.True(t, captured.Critical, "a global limit change must outlive ordinary retention")
	assert.True(t, captured.OrganizationID.IsNil())
	assert.True(t, captured.BusinessUnitID.IsNil())
}

func TestWrites_AreRefusedForATenantThatIsNotAStewardInCloudMode(t *testing.T) {
	steward := pulid.MustNew("org_")
	guard, err := referencedataguard.FromPlatform(&config.PlatformConfig{
		Mode:                  config.PlatformModeCloud,
		ReferenceDataStewards: []string{steward.String()},
	})
	require.NoError(t, err)

	repo := &repoStub{stored: validRule()}
	svc := newService(t, repo)
	svc.referenceData = guard

	tenant := operatorActor()

	_, err = svc.Create(t.Context(), validRule(), tenant)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))

	_, err = svc.Update(t.Context(), validRule(), tenant)
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))

	_, err = svc.Verify(t.Context(), &services.VerifyJurisdictionRuleRequest{
		RuleID:     pulid.MustNew("jrl_"),
		Actor:      tenant,
		State:      jurisdictionrule.VerificationVerified,
		SourceNote: "Checked against the state permit office handbook",
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsAuthorizationError(err))

	assert.Nil(t, repo.created)
	assert.Nil(t, repo.updated)
	assert.Nil(t, repo.verify)

	stewardActor := operatorActor()
	stewardActor.OrganizationID = steward
	_, err = svc.Create(t.Context(), validRule(), stewardActor)
	require.NoError(t, err)
}
