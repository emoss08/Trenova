package extractionevalservice

import (
	"context"
	"errors"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentquality"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aicorrection"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aicorrectionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiproviderports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/extractioneval"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentcontentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/extractionevalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	testOrg  = pulid.ID("org_test")
	testBU   = pulid.ID("bu_test")
	testUser = pulid.ID("usr_test")
	testNow  = int64(1_790_000_000)
)

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: testOrg, BuID: testBU, UserID: testUser}
}

func testActor() *permissionports.RequestActor {
	return &permissionports.RequestActor{
		PrincipalType: permissionports.PrincipalTypeUser,
		PrincipalID:   testUser,
		UserID:        testUser,
	}
}

type caseStore struct {
	extractionevalports.ExtractionEvalCaseRepository
	items []*extractioneval.ExtractionCase
}

func (f *caseStore) Create(
	_ context.Context,
	entity *extractioneval.ExtractionCase,
) (*extractioneval.ExtractionCase, error) {
	for _, item := range f.items {
		if item.InputHash == entity.InputHash {
			return nil, errortypes.NewBusinessError(
				"This document is already in the evaluation set",
			)
		}
	}
	entity.ID = pulid.MustNew("eec_")
	f.items = append(f.items, entity)
	return entity, nil
}

func (f *caseStore) GetByID(
	_ context.Context,
	req extractionevalports.GetExtractionEvalCaseRequest,
) (*extractioneval.ExtractionCase, error) {
	for _, item := range f.items {
		if item.ID == req.ID {
			copied := *item
			return &copied, nil
		}
	}
	return nil, errortypes.NewNotFoundError("case not found")
}

func (f *caseStore) GetBySourceCorrection(
	_ context.Context,
	_ pagination.TenantInfo,
	correctionID pulid.ID,
) (*extractioneval.ExtractionCase, error) {
	for _, item := range f.items {
		if item.SourceCorrectionID != nil && *item.SourceCorrectionID == correctionID {
			return item, nil
		}
	}
	return nil, errortypes.NewNotFoundError("case not found")
}

func (f *caseStore) Update(
	_ context.Context,
	entity *extractioneval.ExtractionCase,
) (*extractioneval.ExtractionCase, error) {
	for i, item := range f.items {
		if item.ID == entity.ID {
			entity.Version++
			f.items[i] = entity
			return entity, nil
		}
	}
	return nil, errortypes.NewNotFoundError("case not found")
}

func (f *caseStore) Delete(
	_ context.Context,
	req extractionevalports.GetExtractionEvalCaseRequest,
) error {
	f.items = slices.DeleteFunc(f.items, func(item *extractioneval.ExtractionCase) bool {
		return item.ID == req.ID
	})
	return nil
}

func (f *caseStore) ListActive(
	_ context.Context,
	req extractionevalports.ListActiveExtractionEvalCasesRequest,
) ([]*extractioneval.ExtractionCase, error) {
	out := make([]*extractioneval.ExtractionCase, 0, len(f.items))
	for _, item := range f.items {
		if item.Status == extractioneval.CaseStatusActive && len(out) < req.Limit {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *caseStore) Counts(
	context.Context,
	extractionevalports.CountExtractionEvalCasesRequest,
) (*extractionevalports.ExtractionEvalCaseCounts, error) {
	counts := &extractionevalports.ExtractionEvalCaseCounts{}
	for _, item := range f.items {
		switch item.Status {
		case extractioneval.CaseStatusCandidate:
			counts.Candidate++
		case extractioneval.CaseStatusActive:
			counts.Active++
		case extractioneval.CaseStatusRetired:
			counts.Retired++
		}
	}
	return counts, nil
}

type runStore struct {
	extractionevalports.ExtractionEvalRunRepository
	runs      map[pulid.ID]*extractioneval.ExtractionRun
	results   *resultStore
	createErr error
	purged    []extractionevalports.PurgeExtractionEvalRunsRequest
}

func (f *runStore) Create(
	_ context.Context,
	run *extractioneval.ExtractionRun,
	results []*extractioneval.ExtractionResult,
) (*extractioneval.ExtractionRun, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}
	run.ID = pulid.MustNew("eer_")
	f.runs[run.ID] = run
	for _, result := range results {
		result.ID = pulid.MustNew("eeres_")
		result.RunID = run.ID
		result.OrganizationID = run.OrganizationID
		result.BusinessUnitID = run.BusinessUnitID
		f.results.items = append(f.results.items, result)
	}
	return run, nil
}

func (f *runStore) GetByID(
	_ context.Context,
	req extractionevalports.GetExtractionEvalRunRequest,
) (*extractioneval.ExtractionRun, error) {
	run, ok := f.runs[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("run not found")
	}
	copied := *run
	return &copied, nil
}

func (f *runStore) Update(
	_ context.Context,
	entity *extractioneval.ExtractionRun,
) (*extractioneval.ExtractionRun, error) {
	entity.Version++
	stored := *entity
	f.runs[entity.ID] = &stored
	return entity, nil
}

func (f *runStore) ListRecentFinished(
	context.Context,
	pagination.TenantInfo,
	aicorrection.Task,
	int,
) ([]*extractioneval.ExtractionRun, error) {
	return nil, nil
}

func (f *runStore) PurgeBefore(
	_ context.Context,
	req extractionevalports.PurgeExtractionEvalRunsRequest,
) (int64, error) {
	f.purged = append(f.purged, req)
	return 3, nil
}

type resultStore struct {
	extractionevalports.ExtractionEvalResultRepository
	items []*extractioneval.ExtractionResult
}

func (f *resultStore) GetByID(
	_ context.Context,
	req extractionevalports.GetExtractionEvalResultRequest,
) (*extractioneval.ExtractionResult, error) {
	for _, item := range f.items {
		if item.ID == req.ID {
			copied := *item
			return &copied, nil
		}
	}
	return nil, errortypes.NewNotFoundError("result not found")
}

func (f *resultStore) Save(
	_ context.Context,
	entity *extractioneval.ExtractionResult,
) (*extractioneval.ExtractionResult, error) {
	for i, item := range f.items {
		if item.ID == entity.ID {
			stored := *entity
			f.items[i] = &stored
			return entity, nil
		}
	}
	return nil, errortypes.NewNotFoundError("result not found")
}

func (f *resultStore) ListPending(
	_ context.Context,
	req extractionevalports.ListPendingExtractionEvalResultsRequest,
) ([]*extractioneval.ExtractionResult, error) {
	out := make([]*extractioneval.ExtractionResult, 0, len(f.items))
	for _, item := range f.items {
		if item.RunID == req.RunID && item.Status == extractioneval.ResultStatusPending &&
			item.Ordinal > req.AfterOrdinal {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *resultStore) ListByRun(
	_ context.Context,
	_ pagination.TenantInfo,
	runID pulid.ID,
) ([]*extractioneval.ExtractionResult, error) {
	out := make([]*extractioneval.ExtractionResult, 0, len(f.items))
	for _, item := range f.items {
		if item.RunID == runID {
			out = append(out, item)
		}
	}
	return out, nil
}

func (f *resultStore) SkipPending(
	_ context.Context,
	_ pagination.TenantInfo,
	runID pulid.ID,
) (int64, error) {
	var skipped int64
	for _, item := range f.items {
		if item.RunID == runID && item.Status == extractioneval.ResultStatusPending {
			item.Status = extractioneval.ResultStatusSkipped
			skipped++
		}
	}
	return skipped, nil
}

type correctionStore struct {
	aicorrectionports.AICorrectionRepository
	items       map[pulid.ID]*aicorrection.Correction
	since       int64
	rows        []*aicorrection.Correction
	weekly      []aicorrection.WeekTotal
	weeklySince int64
	weeklyErr   error
}

func (f *correctionStore) WeeklyTotalsByProvider(
	_ context.Context,
	req *aicorrectionports.WeeklyAICorrectionTotalsRequest,
) ([]aicorrection.WeekTotal, error) {
	f.weeklySince = req.Since
	return f.weekly, f.weeklyErr
}

func (f *correctionStore) GetByID(
	_ context.Context,
	req aicorrectionports.GetAICorrectionRequest,
) (*aicorrection.Correction, error) {
	item, ok := f.items[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("correction not found")
	}
	return item, nil
}

func (f *correctionStore) ListForAccuracy(
	_ context.Context,
	req aicorrectionports.ListAICorrectionsForAccuracyRequest,
) ([]*aicorrection.Correction, error) {
	f.since = req.Since
	return f.rows, nil
}

type documentStore struct {
	documentports.DocumentRepository
	docs map[pulid.ID]*document.Document
}

func (f *documentStore) GetByID(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	doc, ok := f.docs[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("document not found")
	}
	return doc, nil
}

type contentStore struct {
	documentcontentports.DocumentContentRepository
	pages []*documentcontent.Page
}

func (f *contentStore) ListPagesByDocumentID(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) ([]*documentcontent.Page, error) {
	return f.pages, nil
}

func (f *contentStore) GetByDocumentID(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentcontent.Content, error) {
	return &documentcontent.Content{ConverterVersion: "1.36.0"}, nil
}

type casePagesStore struct {
	items map[pulid.ID]*extractioneval.CasePages
}

func (f *casePagesStore) Upsert(
	_ context.Context,
	entity *extractioneval.CasePages,
) (*extractioneval.CasePages, error) {
	copied := *entity
	f.items[entity.CaseID] = &copied
	return entity, nil
}

func (f *casePagesStore) Get(
	_ context.Context,
	req *extractionevalports.GetExtractionEvalCasePagesRequest,
) (*extractioneval.CasePages, error) {
	item, ok := f.items[req.CaseID]
	if !ok {
		return nil, errortypes.NewNotFoundError("case pages not found")
	}
	copied := *item
	return &copied, nil
}

type providerStore struct {
	aiproviderports.AIProviderRepository
	providers map[pulid.ID]*aiprovider.Provider
	err       error
}

func (f *providerStore) GetByID(
	_ context.Context,
	req aiproviderports.GetAIProviderByIDRequest,
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
	tenantports.DataRetentionRepository
	settings *tenant.DataRetention
}

func (f *retentionStore) Get(
	context.Context,
	tenantports.GetDataRetentionRequest,
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
	prediction *extractionevalports.ExtractionPrediction
	err        error
	requests   []*extractionevalports.PredictExtractionRequest
}

func (f *fakePredictor) Predict(
	_ context.Context,
	req *extractionevalports.PredictExtractionRequest,
) (*extractionevalports.ExtractionPrediction, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	return f.prediction, nil
}

type fakeStarter struct {
	started []*extractionevalports.ExtractionEvalRunStart
	err     error
}

func (f *fakeStarter) StartExtractionEvalRun(
	_ context.Context,
	start *extractionevalports.ExtractionEvalRunStart,
) (string, error) {
	if f.err != nil {
		return "", f.err
	}
	f.started = append(f.started, start)
	return "extraction-eval-run:" + start.RunID.String(), nil
}

var errUpstream = errors.New("upstream timeout")

type world struct {
	svc         *Service
	cases       *caseStore
	runs        *runStore
	results     *resultStore
	corrections *correctionStore
	documents   *documentStore
	contents    *contentStore
	casePages   *casePagesStore
	providers   *providerStore
	retention   *retentionStore
	budget      *fakeBudget
	predictor   *fakePredictor
	starter     *fakeStarter
}

func newWorld() *world {
	results := &resultStore{}
	w := &world{
		cases:   &caseStore{},
		results: results,
		runs: &runStore{
			runs:    map[pulid.ID]*extractioneval.ExtractionRun{},
			results: results,
		},
		corrections: &correctionStore{items: map[pulid.ID]*aicorrection.Correction{}},
		documents:   &documentStore{docs: map[pulid.ID]*document.Document{}},
		contents:    &contentStore{},
		casePages:   &casePagesStore{items: map[pulid.ID]*extractioneval.CasePages{}},
		providers:   &providerStore{providers: map[pulid.ID]*aiprovider.Provider{}},
		retention:   &retentionStore{},
		budget:      &fakeBudget{decision: &agentquality.BudgetDecision{}},
		predictor:   &fakePredictor{},
		starter:     &fakeStarter{},
	}
	w.svc = &Service{
		l:           zap.NewNop(),
		cases:       w.cases,
		runs:        w.runs,
		results:     w.results,
		corrections: w.corrections,
		documents:   w.documents,
		contents:    w.contents,
		casePages:   w.casePages,
		providers:   w.providers,
		audit:       &mocks.NoopAuditService{},
		retention:   w.retention,
		budget:      w.budget,
		predictor:   w.predictor,
		starter:     w.starter,
		now:         func() int64 { return testNow },
	}
	return w
}
