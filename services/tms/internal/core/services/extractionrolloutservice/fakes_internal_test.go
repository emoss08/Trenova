package extractionrolloutservice

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/internal/core/domain/aiprovider"
	"github.com/emoss08/trenova/internal/core/domain/extractionrollout"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/notificationservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/dberror"
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

var errUnavailable = errors.New("database unavailable")

func testTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: testOrg, BuID: testBU, UserID: testUser}
}

func testActor() *services.RequestActor {
	return &services.RequestActor{
		PrincipalType: services.PrincipalTypeUser,
		PrincipalID:   testUser,
		UserID:        testUser,
	}
}

type rolloutStore struct {
	rollout     *extractionrollout.ExtractionRollout
	saves       int
	staleSaves  int
	onStaleSave func(*rolloutStore)
}

func (f *rolloutStore) Get(
	context.Context,
	pagination.TenantInfo,
) (*extractionrollout.ExtractionRollout, error) {
	if f.rollout == nil {
		return nil, errortypes.NewNotFoundError("rollout not found")
	}
	copied := *f.rollout
	return &copied, nil
}

func (f *rolloutStore) Save(
	_ context.Context,
	entity *extractionrollout.ExtractionRollout,
) (*extractionrollout.ExtractionRollout, error) {
	if f.staleSaves > 0 {
		f.staleSaves--
		if f.onStaleSave != nil {
			f.onStaleSave(f)
		}
		return nil, dberror.CreateVersionMismatchError("ExtractionRollout", entity.ID.String())
	}
	f.saves++
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("exro_")
	} else {
		entity.Version++
	}
	copied := *entity
	f.rollout = &copied
	return entity, nil
}

type assignmentStore struct {
	items     []*extractionrollout.RolloutAssignment
	createErr error
}

func (f *assignmentStore) find(
	documentID pulid.ID,
	extractedAt int64,
) *extractionrollout.RolloutAssignment {
	for _, item := range f.items {
		if item.DocumentID == documentID && item.ExtractedAt == extractedAt {
			return item
		}
	}
	return nil
}

func (f *assignmentStore) Create(
	_ context.Context,
	entity *extractionrollout.RolloutAssignment,
) (*extractionrollout.RolloutAssignment, bool, error) {
	if f.createErr != nil {
		return nil, false, f.createErr
	}
	if existing := f.find(entity.DocumentID, entity.ExtractedAt); existing != nil {
		copied := *existing
		return &copied, false, nil
	}
	entity.ID = pulid.MustNew("exra_")
	entity.CreatedAt = testNow
	copied := *entity
	f.items = append(f.items, &copied)
	return entity, true, nil
}

func (f *assignmentStore) GetByExtraction(
	_ context.Context,
	req repositories.GetRolloutAssignmentRequest,
) (*extractionrollout.RolloutAssignment, error) {
	existing := f.find(req.DocumentID, req.ExtractedAt)
	if existing == nil {
		return nil, errortypes.NewNotFoundError("assignment not found")
	}
	copied := *existing
	return &copied, nil
}

func (f *assignmentStore) Save(
	_ context.Context,
	entity *extractionrollout.RolloutAssignment,
) (*extractionrollout.RolloutAssignment, error) {
	existing := f.find(entity.DocumentID, entity.ExtractedAt)
	if existing == nil {
		return nil, errortypes.NewNotFoundError("assignment not found")
	}
	entity.Version++
	*existing = *entity
	return entity, nil
}

func (f *assignmentStore) Totals(
	_ context.Context,
	req repositories.TotalRolloutAssignmentsRequest,
) ([]repositories.RolloutAssignmentTotal, error) {
	type key struct {
		arm      extractionrollout.Arm
		servedBy extractionrollout.ServedBy
		outcome  extractionrollout.Outcome
	}
	counts := map[key]int{}
	for _, item := range f.items {
		if item.CandidateProviderID != req.CandidateProviderID || item.CreatedAt < req.Since {
			continue
		}
		servedBy := extractionrollout.ServedByNone
		switch {
		case item.ServedProviderID == nil:
		case *item.ServedProviderID == item.CandidateProviderID:
			servedBy = extractionrollout.ServedByCandidate
		default:
			servedBy = extractionrollout.ServedByOther
		}
		counts[key{item.Arm, servedBy, item.Outcome}]++
	}
	totals := make([]repositories.RolloutAssignmentTotal, 0, len(counts))
	for k, count := range counts {
		totals = append(totals, repositories.RolloutAssignmentTotal{
			Arm:      k.arm,
			ServedBy: k.servedBy,
			Outcome:  k.outcome,
			Count:    count,
		})
	}
	return totals, nil
}

func (f *assignmentStore) PurgeBefore(
	_ context.Context,
	req repositories.PurgeRolloutAssignmentsRequest,
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

func (f *correctionStore) TotalsByProvider(
	_ context.Context,
	req *repositories.TotalAICorrectionsByProviderRequest,
) ([]repositories.AICorrectionProviderTotal, error) {
	var candidate, production repositories.AICorrectionProviderTotal
	candidate.Candidate = true
	for _, item := range f.items {
		if item.Task != req.Task || item.CapturedAt < req.Since || item.ExtractionProviderID == nil {
			continue
		}
		side := &production
		if *item.ExtractionProviderID == req.ProviderID {
			side = &candidate
		}
		side.Scored += item.ScoredCount
		side.Correct += item.CorrectCount
	}
	return []repositories.AICorrectionProviderTotal{candidate, production}, nil
}

func (f *correctionStore) ListForAccuracy(
	_ context.Context,
	req repositories.ListAICorrectionsForAccuracyRequest,
) ([]*aicorrection.Correction, error) {
	out := make([]*aicorrection.Correction, 0, len(f.items))
	for _, item := range f.items {
		if item.Task == req.Task && item.CapturedAt >= req.Since {
			out = append(out, item)
		}
	}
	return out, nil
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
}

func (retentionStore) Get(
	context.Context,
	repositories.GetDataRetentionRequest,
) (*tenant.DataRetention, error) {
	return nil, errortypes.NewNotFoundError("retention not found")
}

type fakeNotifier struct {
	sent []notificationservice.NotifyPermittedRequest
}

func (f *fakeNotifier) NotifyPermitted(
	_ context.Context,
	req notificationservice.NotifyPermittedRequest,
) (int, error) {
	f.sent = append(f.sent, req)
	return 1, nil
}

type world struct {
	svc         *Service
	rollouts    *rolloutStore
	assignments *assignmentStore
	corrections *correctionStore
	providers   *providerStore
	notifier    *fakeNotifier
	candidate   *aiprovider.Provider
	production  *aiprovider.Provider
}

func newWorld() *world {
	candidate := &aiprovider.Provider{
		ID:      pulid.MustNew("aip_"),
		Name:    "Fine-tuned Qwen",
		Enabled: true,
		Tasks:   []aiprovider.Task{aiprovider.TaskDocumentExtraction},
	}
	production := &aiprovider.Provider{
		ID:      pulid.MustNew("aip_"),
		Name:    "Frontier",
		Enabled: true,
		Tasks:   []aiprovider.Task{aiprovider.TaskDocumentExtraction},
	}
	w := &world{
		rollouts:    &rolloutStore{},
		assignments: &assignmentStore{},
		corrections: &correctionStore{},
		providers: &providerStore{providers: map[pulid.ID]*aiprovider.Provider{
			candidate.ID:  candidate,
			production.ID: production,
		}},
		notifier:   &fakeNotifier{},
		candidate:  candidate,
		production: production,
	}
	w.svc = &Service{
		l:           zap.NewNop(),
		rollouts:    w.rollouts,
		assignments: w.assignments,
		corrections: w.corrections,
		providers:   w.providers,
		retention:   retentionStore{},
		audit:       &mocks.NoopAuditService{},
		notifier:    w.notifier,
		now:         func() int64 { return testNow },
	}
	return w
}

func (w *world) start(percent int) {
	rollout := extractionrollout.Default(testOrg, testBU)
	rollout.ID = pulid.MustNew("exro_")
	rollout.Apply(&extractionrollout.Change{
		Enabled:                    true,
		ProviderID:                 w.candidate.ID,
		Percent:                    percent,
		MaxAccuracyDropPoints:      extractionrollout.DefaultMaxAccuracyDropPoints,
		MaxRejectionIncreasePoints: extractionrollout.DefaultMaxRejectionIncreasePoints,
	}, testNow-100)
	w.rollouts.rollout = rollout
}

func (w *world) settled(
	arm extractionrollout.Arm,
	served pulid.ID,
	outcome extractionrollout.Outcome,
	count int,
) {
	for range count {
		assignment := &extractionrollout.RolloutAssignment{
			ID:                  pulid.MustNew("exra_"),
			OrganizationID:      testOrg,
			BusinessUnitID:      testBU,
			DocumentID:          pulid.MustNew("doc_"),
			ExtractedAt:         testNow,
			Arm:                 arm,
			CandidateProviderID: w.candidate.ID,
			Outcome:             outcome,
			CreatedAt:           testNow,
		}
		if served.IsNotNil() {
			assignment.ServedProviderID = &served
		}
		w.assignments.items = append(w.assignments.items, assignment)
	}
}

func (w *world) correction(provider pulid.ID, scored, correct int) *aicorrection.Correction {
	fields := make([]aicorrection.FieldResult, 0, scored)
	for i := range scored {
		outcome := aicorrection.OutcomeCorrected
		if i < correct {
			outcome = aicorrection.OutcomeCorrect
		}
		fields = append(fields, aicorrection.FieldResult{Key: "rate", Outcome: outcome})
	}
	correction := &aicorrection.Correction{
		ID:                   pulid.MustNew("aicr_"),
		OrganizationID:       testOrg,
		BusinessUnitID:       testBU,
		Task:                 aicorrection.TaskShipmentDraftExtraction,
		ExtractionProviderID: &provider,
		FieldResults:         fields,
		ScoredCount:          scored,
		CorrectCount:         correct,
		CorrectedCount:       scored - correct,
		CapturedAt:           testNow,
	}
	w.corrections.items = append(w.corrections.items, correction)
	return correction
}
