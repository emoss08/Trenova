package dispatchlearningservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatchlearningservice"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchlearning"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchlearningports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/internal/testutil/mocks/auditportsmock"
	"github.com/emoss08/trenova/internal/testutil/mocks/dispatchcontrolportsmock"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

var tenant = pagination.TenantInfo{
	OrgID:  pulid.ID("org_1"),
	BuID:   pulid.ID("bu_1"),
	UserID: pulid.ID("usr_1"),
}

type fakeRepo struct {
	style  *dispatchlearning.DispatchStyle
	pages  []*dispatchlearningports.DecisionPage
	asked  []dispatchlearningports.DecisionCursor
	saves  int
	saved  []dispatchlearning.DispatchStyle
	listed *dispatchlearningports.ListTenantsRequest
}

func (f *fakeRepo) Get(
	context.Context,
	pagination.TenantInfo,
) (*dispatchlearning.DispatchStyle, error) {
	return f.style, nil
}

func (f *fakeRepo) GetOrCreate(
	_ context.Context,
	style *dispatchlearning.DispatchStyle,
) (*dispatchlearning.DispatchStyle, error) {
	if f.style == nil {
		f.style = style
	}
	return f.style, nil
}

func (f *fakeRepo) Save(
	_ context.Context,
	style *dispatchlearning.DispatchStyle,
) (*dispatchlearning.DispatchStyle, error) {
	f.saves++
	style.Version++
	f.saved = append(f.saved, *style)
	f.style = style
	return style, nil
}

func (f *fakeRepo) ListDecisions(
	_ context.Context,
	req *dispatchlearningports.ListDecisionsRequest,
) (*dispatchlearningports.DecisionPage, error) {
	f.asked = append(f.asked, req.After)
	if len(f.pages) == 0 {
		return &dispatchlearningports.DecisionPage{Last: req.After}, nil
	}
	page := f.pages[0]
	f.pages = f.pages[1:]
	return page, nil
}

func (f *fakeRepo) ListLearningTenants(
	_ context.Context,
	req *dispatchlearningports.ListTenantsRequest,
) ([]pagination.TenantInfo, error) {
	f.listed = req
	return []pagination.TenantInfo{tenant}, nil
}

func overrideDecision(eventID pulid.ID) *dispatchlearning.Decision {
	return &dispatchlearning.Decision{
		EventID:        eventID,
		ChosenWorkerID: pulid.ID("wrk_b"),
		Planned: dispatchlearning.Pairing{
			WorkerID:  pulid.ID("wrk_a"),
			CostCents: 10_000,
			Lines: []dispatcheconomics.Line{
				{Component: dispatcheconomics.ComponentHomeTime, Cents: 8_000},
			},
		},
		Alternatives: []dispatchlearning.Pairing{{
			WorkerID:   pulid.ID("wrk_b"),
			CostCents:  11_000,
			ExtraCents: 1_000,
			Lines: []dispatcheconomics.Line{
				{Component: dispatcheconomics.ComponentEmptyMiles, Cents: 11_000},
			},
		}},
	}
}

func newService(
	t *testing.T,
	repo *fakeRepo,
	control *dispatchcontrol.DispatchControl,
) (*dispatchlearningservice.Service, *auditportsmock.MockAuditService) {
	t.Helper()

	controls := dispatchcontrolportsmock.NewMockDispatchControlRepository(t)
	controls.EXPECT().
		GetOrCreate(mock.Anything, tenant.OrgID, tenant.BuID).
		Return(control, nil).
		Maybe()
	audit := auditportsmock.NewMockAuditService(t)

	return dispatchlearningservice.New(dispatchlearningservice.Params{
		Logger:       zap.NewNop(),
		Repo:         repo,
		Controls:     controls,
		AuditService: audit,
		Clock:        func() int64 { return 50_000 },
	}), audit
}

func TestLearn_DoesNothingWhileLearningIsOff(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc, _ := newService(t, repo, &dispatchcontrol.DispatchControl{LearnDispatchStyle: false})

	result, err := svc.Learn(t.Context(), &dispatchlearningports.LearnRequest{TenantInfo: tenant})

	require.NoError(t, err)
	assert.Zero(t, result.Learned)
	assert.Empty(t, repo.asked)
	assert.Nil(t, repo.style, "no style row is created for an organization that does not learn")
}

func TestLearn_LearnsEachPageAndSavesTheCursorWithTheWeights(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{pages: []*dispatchlearningports.DecisionPage{
		{
			Decisions: []*dispatchlearning.Decision{overrideDecision(pulid.ID("se_1"))},
			Scanned:   200,
			Last:      dispatchlearningports.DecisionCursor{At: 61_000, ID: pulid.ID("se_200")},
		},
		{
			Decisions: []*dispatchlearning.Decision{overrideDecision(pulid.ID("se_201"))},
			Scanned:   3,
			Last:      dispatchlearningports.DecisionCursor{At: 62_000, ID: pulid.ID("se_203")},
		},
	}}
	svc, _ := newService(t, repo, &dispatchcontrol.DispatchControl{LearnDispatchStyle: true})

	result, err := svc.Learn(t.Context(), &dispatchlearningports.LearnRequest{TenantInfo: tenant})

	require.NoError(t, err)
	assert.Equal(t, 2, result.Learned)
	assert.Equal(t, 2, result.Overrides)
	assert.Equal(t, 203, result.Scanned)
	require.Len(t, repo.asked, 2, "a short page ends the run")
	assert.Equal(t, dispatchlearningports.DecisionCursor{At: 50_000}, repo.asked[0],
		"a new style reads only decisions made after learning started")
	assert.Equal(
		t,
		dispatchlearningports.DecisionCursor{At: 61_000, ID: pulid.ID("se_200")},
		repo.asked[1],
	)

	require.Len(t, repo.saved, 2, "each page saves its weights with its cursor")
	assert.Equal(t, int64(61_000), repo.saved[0].ObservedThroughAt)
	assert.Equal(t, int64(1), repo.saved[0].Decisions)
	assert.Equal(t, int64(62_000), repo.saved[1].ObservedThroughAt)
	assert.Equal(t, int64(2), repo.saved[1].Decisions)
	assert.Greater(t, repo.style.Weights[dispatcheconomics.ComponentHomeTime].Multiplier, 1.0)
	assert.Equal(t, int64(50_000), repo.style.LearningSince)
}

func TestLearn_SavesNothingWhenThereIsNothingNew(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc, _ := newService(t, repo, &dispatchcontrol.DispatchControl{LearnDispatchStyle: true})

	result, err := svc.Learn(t.Context(), &dispatchlearningports.LearnRequest{TenantInfo: tenant})

	require.NoError(t, err)
	assert.Zero(t, result.Scanned)
	assert.Zero(t, repo.saves)
}

func TestMultipliers_AreEmptyBeforeAnythingIsLearned(t *testing.T) {
	t.Parallel()

	svc, _ := newService(t, &fakeRepo{}, &dispatchcontrol.DispatchControl{})

	multipliers, err := svc.Multipliers(t.Context(), tenant)

	require.NoError(t, err)
	assert.Empty(t, multipliers)
}

func TestGet_ShowsAnEmptyStyleBeforeLearningRan(t *testing.T) {
	t.Parallel()

	svc, _ := newService(t, &fakeRepo{}, &dispatchcontrol.DispatchControl{LearnDispatchStyle: true})

	style, err := svc.Get(t.Context(), &dispatchlearningports.GetStyleRequest{TenantInfo: tenant})

	require.NoError(t, err)
	assert.True(t, style.Enabled)
	assert.Zero(t, style.Decisions)
	assert.Empty(t, style.Weights)
	assert.NotNil(t, style.Weights)
	assert.Nil(t, style.LearningSince)
	assert.Nil(t, style.UpdatedAt)
}

func TestReset_ForgetsTheWeightsKeepsTheCursorAndAudits(t *testing.T) {
	t.Parallel()

	stored := dispatchlearning.New(tenant.OrgID, tenant.BuID, 1_000)
	stored.ID = pulid.ID("dsty_1")
	stored.Learn(overrideDecision(pulid.ID("se_1")))
	stored.Advance(4_000, pulid.ID("se_1"))
	repo := &fakeRepo{style: stored}
	svc, audit := newService(t, repo, &dispatchcontrol.DispatchControl{LearnDispatchStyle: true})

	var logged *auditports.LogActionParams
	audit.EXPECT().
		LogAction(mock.Anything, mock.Anything, mock.Anything).
		Run(func(_ context.Context, params *auditports.LogActionParams, _ ...auditports.LogOption) {
			logged = params
		}).
		Return(nil).
		Once()

	style, err := svc.Reset(t.Context(), &dispatchlearningports.ResetStyleRequest{
		TenantInfo: tenant,
		Actor: &permissionports.RequestActor{
			PrincipalType: permissionports.PrincipalTypeUser,
			PrincipalID:   tenant.UserID,
			UserID:        tenant.UserID,
		},
	})

	require.NoError(t, err)
	assert.Zero(t, style.Decisions)
	assert.Empty(t, style.Weights)
	require.NotNil(t, style.LearningSince)
	assert.Equal(t, int64(50_000), *style.LearningSince)
	assert.Equal(
		t,
		int64(4_000),
		repo.style.ObservedThroughAt,
		"reset never relearns old decisions",
	)

	require.NotNil(t, logged)
	assert.Equal(t, permission.ResourceDispatchControl, logged.Resource)
	assert.Equal(t, permission.OpUpdate, logged.Operation)
	assert.Equal(t, tenant.UserID, logged.UserID)
	assert.Equal(t, "dsty_1", logged.ResourceID)
}

func TestReset_WithNothingLearnedWritesNothing(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc, _ := newService(t, repo, &dispatchcontrol.DispatchControl{})

	style, err := svc.Reset(t.Context(), &dispatchlearningports.ResetStyleRequest{
		TenantInfo: tenant,
		Actor:      &permissionports.RequestActor{UserID: tenant.UserID},
	})

	require.NoError(t, err)
	assert.False(t, style.Enabled)
	assert.Zero(t, repo.saves)
}

func TestStartLearning_LearnsOnlyFromDecisionsMadeFromNowOn(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{style: &dispatchlearning.DispatchStyle{
		OrganizationID:    tenant.OrgID,
		BusinessUnitID:    tenant.BuID,
		ObservedThroughAt: 10_000,
		ObservedThroughID: pulid.ID("sev_old"),
		Weights: map[dispatcheconomics.Component]dispatchlearning.Weight{
			dispatcheconomics.ComponentHomeTime: {Multiplier: 1.6, Samples: 12},
		},
	}}
	svc, _ := newService(t, repo, &dispatchcontrol.DispatchControl{LearnDispatchStyle: true})

	require.NoError(t, svc.StartLearning(t.Context(), tenant))

	require.Equal(t, 1, repo.saves)
	assert.Equal(t, int64(50_000), repo.style.ObservedThroughAt,
		"decisions made while learning was off are skipped")
	assert.True(t, repo.style.ObservedThroughID.IsNil())
	assert.InDelta(t, 1.6, repo.style.Weights[dispatcheconomics.ComponentHomeTime].Multiplier, 0.001,
		"what was already learned is kept")
}

func TestStartLearning_ANewStyleStartsAtNow(t *testing.T) {
	t.Parallel()

	repo := &fakeRepo{}
	svc, _ := newService(t, repo, &dispatchcontrol.DispatchControl{LearnDispatchStyle: true})

	require.NoError(t, svc.StartLearning(t.Context(), tenant))

	require.NotNil(t, repo.style)
	assert.Equal(t, int64(50_000), repo.style.ObservedThroughAt,
		"history from before learning was turned on is never swept")
	assert.Zero(t, repo.saves, "a style created at now needs no second write")
}
