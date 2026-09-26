package extractionevalservice

import (
	"context"
	"errors"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/documentcontent"
	"github.com/emoss08/trenova/internal/core/domain/extractioneval"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
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

func testActor() *services.RequestActor {
	return &services.RequestActor{PrincipalType: services.PrincipalTypeUser, PrincipalID: testUser, UserID: testUser}
}

type caseStore struct {
	repositories.ExtractionEvalCaseRepository
	items []*extractioneval.ExtractionCase
}

func (f *caseStore) Create(
	_ context.Context,
	entity *extractioneval.ExtractionCase,
) (*extractioneval.ExtractionCase, error) {
	for _, item := range f.items {
		if item.InputHash == entity.InputHash {
			return nil, errortypes.NewBusinessError("This document is already in the evaluation set")
		}
	}
	entity.ID = pulid.MustNew("eec_")
	f.items = append(f.items, entity)
	return entity, nil
}

func (f *caseStore) GetByID(
	_ context.Context,
	req repositories.GetExtractionEvalCaseRequest,
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

func (f *caseStore) Delete(_ context.Context, req repositories.GetExtractionEvalCaseRequest) error {
	f.items = slices.DeleteFunc(f.items, func(item *extractioneval.ExtractionCase) bool {
		return item.ID == req.ID
	})
	return nil
}

func (f *caseStore) ListActive(
	_ context.Context,
	req repositories.ListActiveExtractionEvalCasesRequest,
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
	repositories.CountExtractionEvalCasesRequest,
) (*repositories.ExtractionEvalCaseCounts, error) {
	counts := &repositories.ExtractionEvalCaseCounts{}
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
	repositories.ExtractionEvalRunRepository
	runs      map[pulid.ID]*extractioneval.ExtractionRun
	results   *resultStore
	createErr error
	purged    []repositories.PurgeExtractionEvalRunsRequest
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
	req repositories.GetExtractionEvalRunRequest,
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
	req repositories.PurgeExtractionEvalRunsRequest,
) (int64, error) {
	f.purged = append(f.purged, req)
	return 3, nil
}

type resultStore struct {
	repositories.ExtractionEvalResultRepository
	items []*extractioneval.ExtractionResult
}

func (f *resultStore) GetByID(
	_ context.Context,
	req repositories.GetExtractionEvalResultRequest,
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
	req repositories.ListPendingExtractionEvalResultsRequest,
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
	repositories.AICorrectionRepository
	items map[pulid.ID]*aicorrection.Correction
	since int64
	rows  []*aicorrection.Correction
}

func (f *correctionStore) GetByID(
	_ context.Context,
	req repositories.GetAICorrectionRequest,
) (*aicorrection.Correction, error) {
	item, ok := f.items[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("correction not found")
	}
	return item, nil
}

func (f *correctionStore) ListForAccuracy(
	_ context.Context,
	req repositories.ListAICorrectionsForAccuracyRequest,
) ([]*aicorrection.Correction, error) {
	f.since = req.Since
	return f.rows, nil
}

type documentStore struct {
	repositories.DocumentRepository
	docs map[pulid.ID]*document.Document
}

func (f *documentStore) GetByID(
	_ context.Context,
	req repositories.GetDocumentByIDRequest,
) (*document.Document, error) {
	doc, ok := f.docs[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("document not found")
	}
	return doc, nil
}

type contentStore struct {
	repositories.DocumentContentRepository
	pages []*documentcontent.Page
}

func (f *contentStore) ListPagesByDocumentID(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) ([]*documentcontent.Page, error) {
	return f.pages, nil
}

type providerStore struct {
	repositories.AIProviderRepository
	providers map[pulid.ID]*aiprovider.Provider
}

func (f *providerStore) GetByID(
	_ context.Context,
	req repositories.GetAIProviderByIDRequest,
) (*aiprovider.Provider, error) {
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
	prediction *services.ExtractionPrediction
	err        error
	requests   []*services.PredictExtractionRequest
}

func (f *fakePredictor) Predict(
	_ context.Context,
	req *services.PredictExtractionRequest,
) (*services.ExtractionPrediction, error) {
	f.requests = append(f.requests, req)
	if f.err != nil {
		return nil, f.err
	}
	return f.prediction, nil
}

type fakeStarter struct {
	started []*services.ExtractionEvalRunStart
	err     error
}

func (f *fakeStarter) StartExtractionEvalRun(
	_ context.Context,
	start *services.ExtractionEvalRunStart,
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
	providers   *providerStore
	retention   *retentionStore
	budget      *fakeBudget
	predictor   *fakePredictor
	starter     *fakeStarter
}

func newWorld() *world {
	results := &resultStore{}
	w := &world{
		cases:       &caseStore{},
		results:     results,
		runs:        &runStore{runs: map[pulid.ID]*extractioneval.ExtractionRun{}, results: results},
		corrections: &correctionStore{items: map[pulid.ID]*aicorrection.Correction{}},
		documents:   &documentStore{docs: map[pulid.ID]*document.Document{}},
		contents:    &contentStore{},
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
