package extractionshadowservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/extractionshadow"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

const (
	testOrg  = pulid.ID("org_test")
	testBU   = pulid.ID("bu_test")
	testUser = pulid.ID("usr_test")
	testNow  = int64(1_790_000_000)
)

var errUpstream = errors.New("upstream timeout")

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: testOrg, BuID: testBU, UserID: testUser}
}

func testActor() *services.RequestActor {
	return &services.RequestActor{PrincipalType: services.PrincipalTypeUser, PrincipalID: testUser, UserID: testUser}
}

type settingsStore struct {
	settings *extractionshadow.ShadowSettings
	saves    int
}

func (f *settingsStore) Get(
	context.Context,
	pagination.TenantInfo,
) (*extractionshadow.ShadowSettings, error) {
	if f.settings == nil {
		return nil, errortypes.NewNotFoundError("settings not found")
	}
	copied := *f.settings
	return &copied, nil
}

func (f *settingsStore) Save(
	_ context.Context,
	entity *extractionshadow.ShadowSettings,
) (*extractionshadow.ShadowSettings, error) {
	f.saves++
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("exss_")
	} else {
		entity.Version++
	}
	copied := *entity
	f.settings = &copied
	return entity, nil
}

type resultStore struct {
	items       []*extractionshadow.ShadowResult
	conflicts   int
	createdHook func()
}

func (f *resultStore) find(id pulid.ID) *extractionshadow.ShadowResult {
	for _, item := range f.items {
		if item.ID == id {
			return item
		}
	}
	return nil
}

func (f *resultStore) Create(
	_ context.Context,
	entity *extractionshadow.ShadowResult,
) (*extractionshadow.ShadowResult, bool, error) {
	for _, item := range f.items {
		if item.DocumentID == entity.DocumentID && item.ExtractedAt == entity.ExtractedAt {
			copied := *item
			return &copied, false, nil
		}
	}
	entity.ID = pulid.MustNew("exsr_")
	entity.CreatedAt = testNow
	copied := *entity
	f.items = append(f.items, &copied)
	if f.createdHook != nil {
		f.createdHook()
	}
	return entity, true, nil
}

func (f *resultStore) GetByID(
	_ context.Context,
	req repositories.GetExtractionShadowResultRequest,
) (*extractionshadow.ShadowResult, error) {
	if item := f.find(req.ID); item != nil {
		copied := *item
		return &copied, nil
	}
	return nil, errortypes.NewNotFoundError("result not found")
}

func (f *resultStore) GetByExtraction(
	_ context.Context,
	req repositories.GetExtractionShadowResultByExtractionRequest,
) (*extractionshadow.ShadowResult, error) {
	for _, item := range f.items {
		if item.DocumentID == req.DocumentID && item.ExtractedAt == req.ExtractedAt {
			copied := *item
			return &copied, nil
		}
	}
	return nil, errortypes.NewNotFoundError("result not found")
}

func (f *resultStore) Save(
	_ context.Context,
	entity *extractionshadow.ShadowResult,
) (*extractionshadow.ShadowResult, error) {
	stored := f.find(entity.ID)
	if stored == nil {
		return nil, errortypes.NewNotFoundError("result not found")
	}
	if f.conflicts > 0 {
		f.conflicts--
		stored.Version++
		return nil, dberror.CreateVersionMismatchError("ExtractionShadowResult", entity.ID.String())
	}
	if stored.Version != entity.Version {
		return nil, dberror.CreateVersionMismatchError("ExtractionShadowResult", entity.ID.String())
	}
	entity.Version++
	*stored = *entity
	return entity, nil
}

func (f *resultStore) CountCreatedSince(
	_ context.Context,
	_ pagination.TenantInfo,
	since int64,
) (int, error) {
	count := 0
	for _, item := range f.items {
		if item.CreatedAt >= since {
			count++
		}
	}
	return count, nil
}

func (f *resultStore) ListScored(
	_ context.Context,
	req *repositories.ListScoredExtractionShadowResultsRequest,
) ([]*extractionshadow.ShadowResult, error) {
	out := make([]*extractionshadow.ShadowResult, 0, len(f.items))
	for _, item := range f.items {
		if item.IsScored() && item.ProviderID == req.ProviderID && *item.ScoredAt >= req.Since {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *resultStore) TotalsByStatus(
	_ context.Context,
	req repositories.TotalExtractionShadowResultsRequest,
) ([]repositories.ExtractionShadowStatusTotal, error) {
	byStatus := map[extractionshadow.ResultStatus]*repositories.ExtractionShadowStatusTotal{}
	order := make([]extractionshadow.ResultStatus, 0, len(extractionshadow.AllResultStatuses()))
	for _, item := range f.items {
		if item.ProviderID != req.ProviderID || item.CreatedAt < req.Since {
			continue
		}
		total, ok := byStatus[item.Status]
		if !ok {
			total = &repositories.ExtractionShadowStatusTotal{Status: item.Status, CostUSD: decimal.Zero}
			byStatus[item.Status] = total
			order = append(order, item.Status)
		}
		total.Count++
		total.CostUSD = total.CostUSD.Add(item.CostUSD)
		if item.LatencyMs > 0 {
			total.LatencyMsSum += item.LatencyMs
			total.Timed++
		}
	}
	out := make([]repositories.ExtractionShadowStatusTotal, 0, len(order))
	for _, status := range order {
		out = append(out, *byStatus[status])
	}
	return out, nil
}

func (f *resultStore) ListConnection(
	context.Context,
	*repositories.ListExtractionShadowResultConnectionRequest,
) (*pagination.CursorListResult[*extractionshadow.ShadowResult], error) {
	return &pagination.CursorListResult[*extractionshadow.ShadowResult]{Items: f.items}, nil
}

func (f *resultStore) PurgeBefore(
	_ context.Context,
	req repositories.PurgeExtractionShadowResultsRequest,
) (int64, error) {
	kept := f.items[:0]
	var purged int64
	for _, item := range f.items {
		if item.CreatedAt < req.Before && purged < int64(req.Limit) {
			purged++
			continue
		}
		kept = append(kept, item)
	}
	f.items = kept
	return purged, nil
}

type correctionStore struct {
	repositories.AICorrectionRepository
	items []*aicorrection.Correction
}

func (f *correctionStore) GetLatestByDocument(
	_ context.Context,
	req *repositories.GetLatestAICorrectionByDocumentRequest,
) (*aicorrection.Correction, error) {
	var latest *aicorrection.Correction
	for _, item := range f.items {
		if item.DocumentID == nil || *item.DocumentID != req.DocumentID || item.Task != req.Task {
			continue
		}
		if latest == nil || item.CapturedAt > latest.CapturedAt {
			latest = item
		}
	}
	if latest == nil {
		return nil, errortypes.NewNotFoundError("correction not found")
	}
	return latest, nil
}

type contentStore struct {
	extractedAt map[pulid.ID]int64
}

func (f *contentStore) GetByDocumentID(
	_ context.Context,
	documentID pulid.ID,
	_ pagination.TenantInfo,
) (*documentcontent.Content, error) {
	at, ok := f.extractedAt[documentID]
	if !ok {
		return nil, errortypes.NewNotFoundError("content not found")
	}
	return &documentcontent.Content{DocumentID: documentID, LastExtractedAt: &at}, nil
}

type providerStore struct {
	repositories.AIProviderRepository
	providers map[pulid.ID]*aiprovider.Provider
	err       error
}

func (f *providerStore) GetByID(
	_ context.Context,
	req repositories.GetAIProviderByIDRequest,
) (*aiprovider.Provider, error) {
	if f.err != nil {
		return nil, f.err
	}
	provider, ok := f.providers[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("provider not found")
	}
	return provider, nil
}

type retentionStore struct {
	repositories.DataRetentionRepository
	settings *tenant.DataRetention
}

func (f *retentionStore) Get(
	context.Context,
	repositories.GetDataRetentionRequest,
) (*tenant.DataRetention, error) {
	if f.settings == nil {
		return nil, errortypes.NewNotFoundError("data retention not found")
	}
	return f.settings, nil
}

type fakeBudget struct {
	decision *agentquality.BudgetDecision
}

func (f *fakeBudget) CheckEvaluationBudget(
	context.Context,
	pagination.TenantInfo,
) (*agentquality.BudgetDecision, error) {
	return f.decision, nil
}

type fakePredictor struct {
	prediction *services.ShadowDraftPrediction
	err        error
	requests   []*services.PredictShadowDraftRequest
}

func (f *fakePredictor) PredictShadowDraft(
	_ context.Context,
	req *services.PredictShadowDraftRequest,
) (*services.ShadowDraftPrediction, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	return f.prediction, nil
}

type fakeStarter struct {
	started []*services.ExtractionShadowStart
	err     error
}

func (f *fakeStarter) StartExtractionShadow(
	_ context.Context,
	start *services.ExtractionShadowStart,
) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.started = append(f.started, start)
	return extractionshadow.WorkflowID(start.ResultID), nil
}

type world struct {
	svc         *Service
	runner      *Runner
	scorer      *Scorer
	settings    *settingsStore
	results     *resultStore
	corrections *correctionStore
	contents    *contentStore
	providers   *providerStore
	retention   *retentionStore
	budget      *fakeBudget
	predictor   *fakePredictor
	starter     *fakeStarter
	candidate   *aiprovider.Provider
	production  *aiprovider.Provider
}

func newWorld() *world {
	candidate := &aiprovider.Provider{
		ID:      pulid.MustNew("aip_"),
		Name:    "Fine-tuned Qwen",
		Model:   "trenova-extract",
		Enabled: true,
		Tasks:   []aiprovider.Task{aiprovider.TaskDocumentExtraction},
	}
	production := &aiprovider.Provider{
		ID:      pulid.MustNew("aip_"),
		Name:    "Frontier",
		Model:   "frontier-large",
		Enabled: true,
		Tasks:   []aiprovider.Task{aiprovider.TaskDocumentExtraction},
	}
	w := &world{
		settings:    &settingsStore{},
		results:     &resultStore{},
		corrections: &correctionStore{},
		contents:    &contentStore{extractedAt: map[pulid.ID]int64{}},
		providers: &providerStore{providers: map[pulid.ID]*aiprovider.Provider{
			candidate.ID:  candidate,
			production.ID: production,
		}},
		retention:  &retentionStore{},
		budget:     &fakeBudget{decision: &agentquality.BudgetDecision{}},
		predictor:  &fakePredictor{},
		starter:    &fakeStarter{},
		candidate:  candidate,
		production: production,
	}
	w.scorer = &Scorer{
		results:     w.results,
		corrections: w.corrections,
		contents:    w.contents,
		now:         func() int64 { return testNow },
	}
	w.svc = &Service{
		l:         zap.NewNop(),
		settings:  w.settings,
		results:   w.results,
		scorer:    w.scorer,
		contents:  w.contents,
		providers: w.providers,
		retention: w.retention,
		audit:     &mocks.NoopAuditService{},
		predictor: w.predictor,
		starter:   w.starter,
		now:       func() int64 { return testNow },
	}
	w.runner = &Runner{
		l:         zap.NewNop(),
		results:   w.results,
		scorer:    w.scorer,
		budget:    w.budget,
		predictor: w.predictor,
		now:       func() int64 { return testNow },
	}
	return w
}

func (w *world) enable(percent int) {
	providerID := w.candidate.ID
	w.settings.settings = &extractionshadow.ShadowSettings{
		ID:             pulid.MustNew("exss_"),
		OrganizationID: testOrg,
		BusinessUnitID: testBU,
		Enabled:        true,
		ProviderID:     &providerID,
		SamplePercent:  percent,
		DailyLimit:     extractionshadow.DefaultDailyLimit,
	}
}

func (w *world) document(extractedAt int64) pulid.ID {
	documentID := pulid.MustNew("doc_")
	w.contents.extractedAt[documentID] = extractedAt
	return documentID
}
