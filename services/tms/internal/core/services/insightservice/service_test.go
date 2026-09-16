package insightservice

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/insightservice/detector"
	"github.com/emoss08/trenova/internal/core/services/insightservice/narrator"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type stubDetector struct {
	key      string
	category insight.Category
	resource permission.Resource
	surfaces []insight.Surface
	findings []detector.Finding
	err      error
	lastRun  detector.Params
	runs     int
}

func (d *stubDetector) Key() string                     { return d.key }
func (d *stubDetector) Category() insight.Category      { return d.category }
func (d *stubDetector) Operation() permission.Operation { return permission.OpRead }
func (d *stubDetector) Surfaces() []insight.Surface     { return d.surfaces }

func (d *stubDetector) Explain() detector.Explanation {
	return detector.Explanation{
		Measures:  "what " + d.key + " computes",
		Threshold: "when it speaks",
		Excludes:  "what it passes over",
	}
}

func (d *stubDetector) Resource() permission.Resource {
	if d.resource == "" {
		return permission.ResourceShipment
	}

	return d.resource
}

func (d *stubDetector) Detect(
	_ context.Context,
	params detector.Params,
) ([]detector.Finding, error) {
	d.runs++
	d.lastRun = params

	return d.findings, d.err
}

type stubRepo struct {
	replaced    []repositories.ReplaceDetectorFindingsRequest
	active      []*insight.Insight
	byID        *insight.Insight
	history     []*insight.Insight
	historyErr  error
	err         error
	lastList    repositories.ListActiveInsightsRequest
	lastBrowse  repositories.ListInsightsRequest
	lastHistory repositories.ListInsightHistoryRequest
}

func (r *stubRepo) ListHistory(
	_ context.Context,
	req repositories.ListInsightHistoryRequest,
) ([]*insight.Insight, error) {
	r.lastHistory = req

	return r.history, r.historyErr
}

func (r *stubRepo) ReplaceDetectorFindings(
	_ context.Context,
	req repositories.ReplaceDetectorFindingsRequest,
) (repositories.ReplaceDetectorFindingsResult, error) {
	if r.err != nil {
		return repositories.ReplaceDetectorFindingsResult{}, r.err
	}

	r.replaced = append(r.replaced, req)

	return repositories.ReplaceDetectorFindingsResult{Created: len(req.Insights)}, nil
}

func (r *stubRepo) ListActive(
	_ context.Context,
	req repositories.ListActiveInsightsRequest,
) ([]*insight.Insight, error) {
	r.lastList = req

	return r.active, r.err
}

func (r *stubRepo) List(
	_ context.Context,
	req repositories.ListInsightsRequest,
) (*pagination.ListResult[*insight.Insight], error) {
	r.lastBrowse = req

	return &pagination.ListResult[*insight.Insight]{Items: r.active, Total: len(r.active)}, r.err
}

func (r *stubRepo) Restore(
	_ context.Context,
	req repositories.RestoreInsightRequest,
) (*insight.Insight, error) {
	return &insight.Insight{ID: req.ID, Status: insight.StatusActive}, r.err
}

func (r *stubRepo) GetByID(
	context.Context,
	repositories.GetInsightByIDRequest,
) (*insight.Insight, error) {
	return r.byID, r.err
}

func (r *stubRepo) Dismiss(
	_ context.Context,
	req repositories.DismissInsightRequest,
) (*insight.Insight, error) {
	return &insight.Insight{ID: req.ID, Status: insight.StatusDismissed}, r.err
}

type stubPermissions struct {
	allowedResources map[permission.Resource]bool
	err              error
	checks           int
}

func (p *stubPermissions) Check(
	_ context.Context,
	req *services.PermissionCheckRequest,
) (*services.PermissionCheckResult, error) {
	p.checks++
	if p.err != nil {
		return nil, p.err
	}

	return &services.PermissionCheckResult{
		Allowed: p.allowedResources[permission.Resource(req.Resource)],
	}, nil
}

// A narrator with no completion service behind it falls back to detector
// wording on every call, which is exactly the shape these tests want: the
// narration path is covered in its own package.
func silentNarrator() *narrator.Service {
	return narrator.New(narrator.Params{
		Logger:     zap.NewNop(),
		Completion: failingCompletion{},
	})
}

type failingCompletion struct{}

func (failingCompletion) CompleteStructured(
	context.Context,
	*services.StructuredCompletionRequest,
) (*services.StructuredCompletionResult, error) {
	return nil, services.ErrNoProviderConfigured
}

func (failingCompletion) Diagnose(
	context.Context,
	*services.DiagnoseRequest,
) (*services.DiagnoseResult, error) {
	return nil, services.ErrNoProviderConfigured
}

func (failingCompletion) CompleteChat(
	context.Context,
	*services.ChatCompletionRequest,
) (*services.ChatCompletionResult, error) {
	return nil, services.ErrNoProviderConfigured
}

func (failingCompletion) StreamChat(
	context.Context,
	*services.ChatCompletionRequest,
	services.ChatStreamSink,
) (*services.ChatCompletionResult, error) {
	return nil, services.ErrNoProviderConfigured
}

func newService(repo *stubRepo, perms *stubPermissions, ds ...detector.Detector) *Service {
	return &Service{
		l:           zap.NewNop(),
		repo:        repo,
		detectors:   detector.NewRegistry(ds...),
		narrator:    silentNarrator(),
		permissions: perms,
	}
}

func testFinding(key string) detector.Finding {
	return detector.Finding{
		DedupeKey: key,
		Subject:   "Acme Foods",
		Headline:  "On-time delivery for Acme Foods is 82.4%",
		Severity:  insight.SeverityWarning,
		Metrics: []insight.Metric{
			detector.Percent(
				"onTimePercent",
				"On-time delivery",
				decimal.NewFromFloat(82.4),
				insight.DirectionLowerIsWorse,
			),
		},
	}
}

func tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func TestRefresh_StoresWhatTheDetectorsFound(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	svc := newService(repo, &stubPermissions{}, &stubDetector{
		key:      "ontime-decline",
		category: insight.CategoryServiceQuality,
		findings: []detector.Finding{testFinding("ontime-decline:cus_1")},
	})

	result, err := svc.Refresh(t.Context(), RefreshRequest{TenantInfo: tenant(), Now: 1_800_000_000})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Created)
	require.Len(t, repo.replaced, 1)
	require.Len(t, repo.replaced[0].Insights, 1)

	stored := repo.replaced[0].Insights[0]
	assert.Equal(t, insight.CategoryServiceQuality, stored.Category)
	assert.Equal(t, insight.StatusActive, stored.Status)
	assert.Equal(t, "ontime-decline", stored.DetectorKey)
}

// An insight with no model behind it is still complete: the detector's plainer
// wording, and narrated recorded as false so nobody mistakes it for prose.
func TestRefresh_StoresTheDetectorWordingWhenNothingNarrates(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	finding := testFinding("ontime-decline:cus_1")
	svc := newService(repo, &stubPermissions{}, &stubDetector{
		key:      "ontime-decline",
		category: insight.CategoryServiceQuality,
		findings: []detector.Finding{finding},
	})

	_, err := svc.Refresh(t.Context(), RefreshRequest{TenantInfo: tenant(), Now: 1_800_000_000})
	require.NoError(t, err)

	stored := repo.replaced[0].Insights[0]
	assert.False(t, stored.Narrated)
	assert.Equal(t, finding.Headline, stored.Headline)
	assert.Empty(t, stored.ModelIdentifier)
}

// A home screen that goes blank because one aggregate broke is worse than one
// missing a section.
func TestRefresh_KeepsGoingWhenOneDetectorFails(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	broken := &stubDetector{key: "broken", err: errors.New("query exploded")}
	working := &stubDetector{
		key:      "working",
		category: insight.CategoryCashFlow,
		findings: []detector.Finding{testFinding("working:cus_1")},
	}

	result, err := svc(repo, broken, working).Refresh(
		t.Context(),
		RefreshRequest{TenantInfo: tenant(), Now: 1_800_000_000},
	)
	require.NoError(t, err)

	assert.Equal(t, []string{"broken"}, result.Failed)
	assert.Equal(t, 1, result.Created)
	assert.Equal(t, 1, working.runs)
}

func TestRefresh_ReportsAStorageFailureWithoutStoppingOtherDetectors(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{err: errors.New("database down")}
	first := &stubDetector{
		key:      "first",
		category: insight.CategoryCashFlow,
		findings: []detector.Finding{testFinding("first:cus_1")},
	}
	second := &stubDetector{
		key:      "second",
		category: insight.CategoryCostLeakage,
		findings: []detector.Finding{testFinding("second:cus_1")},
	}

	result, err := svc(repo, first, second).Refresh(
		t.Context(),
		RefreshRequest{TenantInfo: tenant(), Now: 1_800_000_000},
	)
	require.NoError(t, err)

	assert.ElementsMatch(t, []string{"first", "second"}, result.Failed)
	assert.Equal(t, 1, second.runs)
}

// A detector returning something unshowable is a bug in that detector, and the
// bad finding is dropped rather than stored or allowed to fail the run.
func TestRefresh_DropsAMalformedFindingAndKeepsTheRest(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	malformed := testFinding("bad")
	malformed.Metrics = nil

	svc := newService(repo, &stubPermissions{}, &stubDetector{
		key:      "mixed",
		category: insight.CategoryCashFlow,
		findings: []detector.Finding{malformed, testFinding("good:cus_1")},
	})

	result, err := svc.Refresh(t.Context(), RefreshRequest{TenantInfo: tenant(), Now: 1_800_000_000})
	require.NoError(t, err)

	assert.Equal(t, 1, result.Created)
	require.Len(t, repo.replaced[0].Insights, 1)
	assert.Equal(t, "good:cus_1", repo.replaced[0].Insights[0].DedupeKey)
}

func TestRefresh_AnchorsTheWindowToTheGivenInstant(t *testing.T) {
	t.Parallel()

	now := int64(1_800_000_000)
	stub := &stubDetector{key: "ontime", category: insight.CategoryServiceQuality}

	_, err := svc(&stubRepo{}, stub).Refresh(
		t.Context(),
		RefreshRequest{TenantInfo: tenant(), Now: now},
	)
	require.NoError(t, err)

	assert.Equal(t, now, stub.lastRun.WindowEnd)
	assert.Equal(t, 30, stub.lastRun.WindowDays())
}

// A dismissal has to reach storage as a cutoff, or the next refresh puts the
// card a person just waved away straight back.
func TestRefresh_PassesTheDismissalSuppressionCutoff(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	now := int64(1_800_000_000)

	_, err := svc(repo, &stubDetector{
		key:      "ontime",
		category: insight.CategoryServiceQuality,
		findings: []detector.Finding{testFinding("ontime:cus_1")},
	}).Refresh(t.Context(), RefreshRequest{TenantInfo: tenant(), Now: now})
	require.NoError(t, err)

	require.Len(t, repo.replaced, 1)
	assert.Equal(
		t,
		now-int64(insight.DismissalSuppression.Seconds()),
		repo.replaced[0].SuppressedBefore,
	)
}

func TestRefresh_MarksNumbersStaleAfterTheRefreshHorizon(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	now := int64(1_800_000_000)

	_, err := svc(repo, &stubDetector{
		key:      "ontime",
		category: insight.CategoryServiceQuality,
		findings: []detector.Finding{testFinding("ontime:cus_1")},
	}).Refresh(t.Context(), RefreshRequest{TenantInfo: tenant(), Now: now})
	require.NoError(t, err)

	stored := repo.replaced[0].Insights[0]
	assert.Equal(t, now, stored.DetectedAt)
	assert.Greater(t, stored.StaleAt, now)
	assert.False(t, stored.IsStale(now))
}

// An insight headline names a customer and a figure. A reader who cannot read
// customers must not receive it in a payload at all, so the restriction goes
// into the query rather than being applied to what comes back.
func TestListActive_AsksOnlyForDetectorsTheReaderMaySee(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	perms := &stubPermissions{allowedResources: map[permission.Resource]bool{
		permission.ResourceShipment: true,
	}}

	svc := newService(repo, perms,
		&stubDetector{key: "shipments", resource: permission.ResourceShipment},
		&stubDetector{key: "workers", resource: permission.ResourceWorker},
	)

	_, err := svc.ListActive(t.Context(), services.ListInsightsRequest{
		TenantInfo: tenant(),
		UserID:     pulid.MustNew("usr_"),
		Limit:      10,
	})
	require.NoError(t, err)

	assert.Equal(
		t,
		repositories.AllowedDetectorKeys{"shipments"},
		repo.lastList.AllowedDetectorKeys,
	)
}

// A page asking for its own slice narrows the reader's set; it never widens it.
// The accounting dashboard must not be a way to see findings a reader's
// permissions keep off their home screen.
func TestListActive_RestrictsASurfaceToWhatTheReaderMaySee(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	perms := &stubPermissions{allowedResources: map[permission.Resource]bool{
		permission.ResourceShipment: true,
	}}

	svc := newService(repo, perms,
		&stubDetector{
			key:      "unbilled-aging",
			resource: permission.ResourceShipment,
			surfaces: []insight.Surface{insight.SurfaceAccounting},
		},
		&stubDetector{
			key:      "empty-miles",
			resource: permission.ResourceShipment,
			surfaces: []insight.Surface{insight.SurfaceDispatch},
		},
		&stubDetector{
			key:      "credential-expiry",
			resource: permission.ResourceWorker,
			surfaces: []insight.Surface{insight.SurfaceAccounting, insight.SurfaceDispatch},
		},
	)

	_, err := svc.ListActive(t.Context(), services.ListInsightsRequest{
		TenantInfo: tenant(),
		UserID:     pulid.MustNew("usr_"),
		Surface:    insight.SurfaceAccounting,
		Limit:      10,
	})
	require.NoError(t, err)

	assert.Equal(
		t,
		repositories.AllowedDetectorKeys{"unbilled-aging"},
		repo.lastList.AllowedDetectorKeys,
	)
}

// A surface no detector claims is an empty page, not the whole home screen.
func TestListActive_AsksForNothingOnASurfaceNoDetectorClaims(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	perms := &stubPermissions{allowedResources: map[permission.Resource]bool{
		permission.ResourceShipment: true,
	}}

	svc := newService(repo, perms,
		&stubDetector{key: "shipments", resource: permission.ResourceShipment},
	)

	_, err := svc.ListActive(t.Context(), services.ListInsightsRequest{
		TenantInfo: tenant(),
		UserID:     pulid.MustNew("usr_"),
		Surface:    insight.SurfaceFleet,
		Limit:      10,
	})
	require.NoError(t, err)

	assert.Empty(t, repo.lastList.AllowedDetectorKeys)
}

// The one shape that could be read as "no restriction" is the one that would
// hand a reader every finding in the organization, so a reader who may see
// nothing must produce an empty set rather than an absent filter.
func TestListActive_AsksForNothingWhenTheReaderMaySeeNothing(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	svc := newService(repo, &stubPermissions{},
		&stubDetector{key: "shipments", resource: permission.ResourceShipment},
	)

	_, err := svc.ListActive(t.Context(), services.ListInsightsRequest{
		TenantInfo: tenant(),
		UserID:     pulid.MustNew("usr_"),
		Limit:      10,
	})
	require.NoError(t, err)

	assert.Empty(t, repo.lastList.AllowedDetectorKeys)
}

// A detector turned off in a release leaves its findings behind. Nothing is left
// to say what permission they needed, so its key never joins the allowed set.
func TestListActive_NeverAllowsADetectorThatNoLongerExists(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	perms := &stubPermissions{allowedResources: map[permission.Resource]bool{
		permission.ResourceShipment: true,
	}}

	svc := newService(repo, perms,
		&stubDetector{key: "shipments", resource: permission.ResourceShipment},
	)

	_, err := svc.ListActive(t.Context(), services.ListInsightsRequest{
		TenantInfo: tenant(),
		UserID:     pulid.MustNew("usr_"),
		Limit:      10,
	})
	require.NoError(t, err)

	assert.NotContains(t, repo.lastList.AllowedDetectorKeys, "retired-detector")
}

// A failed permission check must not be read as permission granted.
func TestListActive_AllowsNothingWhenThePermissionCheckFails(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	perms := &stubPermissions{err: errors.New("permission service down")}

	svc := newService(repo, perms,
		&stubDetector{key: "shipments", resource: permission.ResourceShipment},
	)

	_, err := svc.ListActive(t.Context(), services.ListInsightsRequest{
		TenantInfo: tenant(),
		UserID:     pulid.MustNew("usr_"),
		Limit:      10,
	})
	require.NoError(t, err)

	assert.Empty(t, repo.lastList.AllowedDetectorKeys)
}

// Detectors repeat across rows, so the question is asked once per detector
// rather than once per card.
func TestListActive_ResolvesEachDetectorsPermissionOnce(t *testing.T) {
	t.Parallel()

	active := make([]*insight.Insight, 0, 6)
	for range 6 {
		active = append(active, &insight.Insight{
			ID:          pulid.MustNew("inst_"),
			DetectorKey: "shipments",
		})
	}

	perms := &stubPermissions{allowedResources: map[permission.Resource]bool{
		permission.ResourceShipment: true,
	}}

	_, err := newService(&stubRepo{active: active}, perms,
		&stubDetector{key: "shipments", resource: permission.ResourceShipment},
	).ListActive(
		t.Context(),
		services.ListInsightsRequest{TenantInfo: tenant(), UserID: pulid.MustNew("usr_"), Limit: 10},
	)
	require.NoError(t, err)

	assert.Equal(t, 1, perms.checks)
}

// Browsing the history carries the same restriction as the home screen: the
// page must not be able to widen what a reader can see.
func TestList_CarriesTheSameDetectorRestrictionAsTheWidget(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	perms := &stubPermissions{allowedResources: map[permission.Resource]bool{
		permission.ResourceShipment: true,
	}}

	svc := newService(repo, perms,
		&stubDetector{key: "shipments", resource: permission.ResourceShipment},
		&stubDetector{key: "workers", resource: permission.ResourceWorker},
	)

	_, err := svc.List(t.Context(), services.BrowseInsightsRequest{
		TenantInfo: tenant(),
		UserID:     pulid.MustNew("usr_"),
		Statuses:   []insight.Status{insight.StatusDismissed},
		Limit:      25,
	})
	require.NoError(t, err)

	assert.Equal(
		t,
		repositories.AllowedDetectorKeys{"shipments"},
		repo.lastBrowse.AllowedDetectorKeys,
	)
}

func TestList_PassesTheRequestedFiltersThrough(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	perms := &stubPermissions{allowedResources: map[permission.Resource]bool{
		permission.ResourceShipment: true,
	}}

	svc := newService(repo, perms,
		&stubDetector{key: "shipments", resource: permission.ResourceShipment},
	)

	_, err := svc.List(t.Context(), services.BrowseInsightsRequest{
		TenantInfo: tenant(),
		UserID:     pulid.MustNew("usr_"),
		Categories: []insight.Category{insight.CategoryCashFlow},
		Severities: []insight.Severity{insight.SeverityCritical},
		Statuses:   []insight.Status{insight.StatusResolved},
		Limit:      50,
		Offset:     25,
	})
	require.NoError(t, err)

	assert.Equal(t, []insight.Category{insight.CategoryCashFlow}, repo.lastBrowse.Categories)
	assert.Equal(t, []insight.Severity{insight.SeverityCritical}, repo.lastBrowse.Severities)
	assert.Equal(t, []insight.Status{insight.StatusResolved}, repo.lastBrowse.Statuses)
	assert.Equal(t, 50, repo.lastBrowse.Limit)
	assert.Equal(t, 25, repo.lastBrowse.Offset)
}

func TestGetDetail_ReturnsTheFindingItsTrendAndTheRuleBehindIt(t *testing.T) {
	t.Parallel()

	found := &insight.Insight{
		ID:          pulid.MustNew("inst_"),
		DetectorKey: "shipments",
		DedupeKey:   "shipments:cus_1",
	}
	older := &insight.Insight{ID: pulid.MustNew("inst_"), DedupeKey: "shipments:cus_1"}

	repo := &stubRepo{byID: found, history: []*insight.Insight{older}}
	perms := &stubPermissions{allowedResources: map[permission.Resource]bool{
		permission.ResourceShipment: true,
	}}

	detail, err := newService(repo, perms,
		&stubDetector{key: "shipments", resource: permission.ResourceShipment},
	).GetDetail(t.Context(), services.GetInsightDetailRequest{
		ID:         found.ID,
		UserID:     pulid.MustNew("usr_"),
		TenantInfo: tenant(),
	})
	require.NoError(t, err)

	assert.Equal(t, found.ID, detail.Insight.ID)
	assert.Equal(t, []*insight.Insight{older}, detail.History)
	assert.Equal(t, "what shipments computes", detail.Explanation.Measures)
	assert.Equal(t, "what it passes over", detail.Explanation.Excludes)
}

// The finding being read must not appear inside its own history, or the trend
// starts with a duplicate of the number already on screen.
func TestGetDetail_KeepsTheFindingOutOfItsOwnHistory(t *testing.T) {
	t.Parallel()

	found := &insight.Insight{
		ID:          pulid.MustNew("inst_"),
		DetectorKey: "shipments",
		DedupeKey:   "shipments:cus_1",
	}
	repo := &stubRepo{byID: found}
	perms := &stubPermissions{allowedResources: map[permission.Resource]bool{
		permission.ResourceShipment: true,
	}}

	_, err := newService(repo, perms,
		&stubDetector{key: "shipments", resource: permission.ResourceShipment},
	).GetDetail(t.Context(), services.GetInsightDetailRequest{
		ID:         found.ID,
		UserID:     pulid.MustNew("usr_"),
		TenantInfo: tenant(),
	})
	require.NoError(t, err)

	assert.Equal(t, found.ID, repo.lastHistory.ExcludeID)
	assert.Equal(t, found.DedupeKey, repo.lastHistory.DedupeKey)
}

// Reading by id has no filter to fold the permission into, so a reader who
// guesses an id must still be refused.
func TestGetDetail_RefusesAFindingTheReaderMayNotSee(t *testing.T) {
	t.Parallel()

	found := &insight.Insight{
		ID:          pulid.MustNew("inst_"),
		DetectorKey: "workers",
		DedupeKey:   "workers:wct_1",
	}
	repo := &stubRepo{byID: found}
	perms := &stubPermissions{allowedResources: map[permission.Resource]bool{
		permission.ResourceShipment: true,
	}}

	_, err := newService(repo, perms,
		&stubDetector{key: "workers", resource: permission.ResourceWorker},
	).GetDetail(t.Context(), services.GetInsightDetailRequest{
		ID:         found.ID,
		UserID:     pulid.MustNew("usr_"),
		TenantInfo: tenant(),
	})

	require.ErrorIs(t, err, ErrInsightNotVisible)
}

// A retired detector leaves findings with nothing left to say what permission
// they needed, so reading one directly is refused rather than allowed.
func TestGetDetail_RefusesAFindingFromADetectorThatNoLongerExists(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{byID: &insight.Insight{
		ID:          pulid.MustNew("inst_"),
		DetectorKey: "retired-detector",
	}}
	perms := &stubPermissions{allowedResources: map[permission.Resource]bool{
		permission.ResourceShipment: true,
	}}

	_, err := newService(repo, perms).GetDetail(
		t.Context(),
		services.GetInsightDetailRequest{
			ID:         pulid.MustNew("inst_"),
			UserID:     pulid.MustNew("usr_"),
			TenantInfo: tenant(),
		},
	)

	require.ErrorIs(t, err, ErrInsightNotVisible)
}

// A finding without its trend is still worth reading, so a failed history read
// degrades the view rather than failing it.
func TestGetDetail_StillReturnsTheFindingWhenItsHistoryCannotBeRead(t *testing.T) {
	t.Parallel()

	found := &insight.Insight{
		ID:          pulid.MustNew("inst_"),
		DetectorKey: "shipments",
		DedupeKey:   "shipments:cus_1",
	}
	repo := &stubRepo{byID: found, historyErr: errors.New("history query failed")}
	perms := &stubPermissions{allowedResources: map[permission.Resource]bool{
		permission.ResourceShipment: true,
	}}

	detail, err := newService(repo, perms,
		&stubDetector{key: "shipments", resource: permission.ResourceShipment},
	).GetDetail(t.Context(), services.GetInsightDetailRequest{
		ID:         found.ID,
		UserID:     pulid.MustNew("usr_"),
		TenantInfo: tenant(),
	})
	require.NoError(t, err)

	assert.Equal(t, found.ID, detail.Insight.ID)
	assert.Empty(t, detail.History)
}

func TestRestore_ReturnsADismissedFindingToActive(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("inst_")

	restored, err := newService(&stubRepo{}, &stubPermissions{}).Restore(
		t.Context(),
		services.RestoreInsightRequest{ID: id, TenantInfo: tenant()},
	)
	require.NoError(t, err)

	assert.Equal(t, id, restored.ID)
	assert.Equal(t, insight.StatusActive, restored.Status)
}

// The widget's default applies when a caller does not say how many it wants.
func TestListActive_AppliesADefaultLimit(t *testing.T) {
	t.Parallel()

	repo := &stubRepo{}
	perms := &stubPermissions{allowedResources: map[permission.Resource]bool{
		permission.ResourceShipment: true,
	}}

	_, err := newService(repo, perms,
		&stubDetector{key: "shipments", resource: permission.ResourceShipment},
	).ListActive(
		t.Context(),
		services.ListInsightsRequest{TenantInfo: tenant(), UserID: pulid.MustNew("usr_")},
	)
	require.NoError(t, err)

	assert.Positive(t, repo.lastList.Limit)
}

func TestDismiss_RecordsTheReaderAndTheirReason(t *testing.T) {
	t.Parallel()

	id := pulid.MustNew("inst_")

	dismissed, err := newService(&stubRepo{}, &stubPermissions{}).Dismiss(
		t.Context(),
		services.DismissInsightRequest{
			ID:         id,
			UserID:     pulid.MustNew("usr_"),
			Reason:     "Known seasonal pattern",
			TenantInfo: tenant(),
		},
	)
	require.NoError(t, err)

	assert.Equal(t, id, dismissed.ID)
	assert.Equal(t, insight.StatusDismissed, dismissed.Status)
}

// svc builds a service whose permission stub allows everything, for the refresh
// tests where visibility is not what is under examination.
func svc(repo *stubRepo, ds ...detector.Detector) *Service {
	return newService(repo, &stubPermissions{
		allowedResources: map[permission.Resource]bool{permission.ResourceShipment: true},
	}, ds...)
}
