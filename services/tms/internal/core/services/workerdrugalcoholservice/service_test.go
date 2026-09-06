package workerdrugalcoholservice_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/workerdrugalcoholservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/internal/testutil/mocks"
)

// fakeRepo is a hand-rolled in-memory stand-in for the testing programme's
// repository. It is written by hand rather than generated because the
// interesting behaviour here is the interplay between tests, violations and
// draws — a fake that actually holds rows exercises that; a mock that returns
// canned values does not.
type fakeRepo struct {
	tests      map[pulid.ID]*worker.WorkerDOTTest
	violations map[pulid.ID]*worker.WorkerDOTViolation
	queries    map[pulid.ID]*worker.WorkerClearinghouseQuery
	entries    map[pulid.ID]*worker.DOTRandomDrawEntry
	draws      map[pulid.ID]*worker.DOTRandomDraw
	pools      map[pulid.ID]*worker.DOTRandomPool
	defaults   *worker.DOTRandomPool
	candidates []pulid.ID
	rollups    []repositories.DrugAlcoholRollup
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		tests:      map[pulid.ID]*worker.WorkerDOTTest{},
		violations: map[pulid.ID]*worker.WorkerDOTViolation{},
		queries:    map[pulid.ID]*worker.WorkerClearinghouseQuery{},
		entries:    map[pulid.ID]*worker.DOTRandomDrawEntry{},
		draws:      map[pulid.ID]*worker.DOTRandomDraw{},
		pools:      map[pulid.ID]*worker.DOTRandomPool{},
	}
}

func (f *fakeRepo) ListTests(
	_ context.Context,
	req *repositories.ListWorkerDOTTestsRequest,
) ([]*worker.WorkerDOTTest, error) {
	out := make([]*worker.WorkerDOTTest, 0, len(f.tests))
	for _, test := range f.tests {
		if !req.WorkerID.IsNil() && test.WorkerID != req.WorkerID {
			continue
		}
		if req.OpenOnly && !test.IsOpen() {
			continue
		}
		out = append(out, test)
	}
	return out, nil
}

func (f *fakeRepo) GetTestByID(
	_ context.Context,
	req *repositories.GetWorkerDOTTestByIDRequest,
) (*worker.WorkerDOTTest, error) {
	test, ok := f.tests[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("test not found")
	}
	return test, nil
}

func (f *fakeRepo) CreateTest(
	_ context.Context,
	entity *worker.WorkerDOTTest,
) (*worker.WorkerDOTTest, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("wdot_")
	}
	f.tests[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateTest(
	_ context.Context,
	entity *worker.WorkerDOTTest,
) (*worker.WorkerDOTTest, error) {
	f.tests[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) ListViolations(
	_ context.Context,
	req *repositories.ListWorkerDOTViolationsRequest,
) ([]*worker.WorkerDOTViolation, error) {
	out := make([]*worker.WorkerDOTViolation, 0, len(f.violations))
	for _, violation := range f.violations {
		if !req.WorkerID.IsNil() && violation.WorkerID != req.WorkerID {
			continue
		}
		out = append(out, violation)
	}
	return out, nil
}

func (f *fakeRepo) GetViolationByID(
	_ context.Context,
	req *repositories.GetWorkerDOTViolationByIDRequest,
) (*worker.WorkerDOTViolation, error) {
	violation, ok := f.violations[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("violation not found")
	}
	return violation, nil
}

func (f *fakeRepo) GetOpenViolation(
	_ context.Context,
	_ pagination.TenantInfo,
	workerID pulid.ID,
) (*worker.WorkerDOTViolation, error) {
	for _, violation := range f.violations {
		if violation.WorkerID == workerID && !violation.IsResolved() {
			return violation, nil
		}
	}
	return nil, nil
}

func (f *fakeRepo) CreateViolation(
	_ context.Context,
	entity *worker.WorkerDOTViolation,
) (*worker.WorkerDOTViolation, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("wdotv_")
	}
	f.violations[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateViolation(
	_ context.Context,
	entity *worker.WorkerDOTViolation,
) (*worker.WorkerDOTViolation, error) {
	f.violations[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) ListQueries(
	_ context.Context,
	req *repositories.ListClearinghouseQueriesRequest,
) ([]*worker.WorkerClearinghouseQuery, error) {
	out := make([]*worker.WorkerClearinghouseQuery, 0, len(f.queries))
	for _, query := range f.queries {
		if !req.WorkerID.IsNil() && query.WorkerID != req.WorkerID {
			continue
		}
		out = append(out, query)
	}
	return out, nil
}

func (f *fakeRepo) GetQueryByID(
	_ context.Context,
	req *repositories.GetClearinghouseQueryByIDRequest,
) (*worker.WorkerClearinghouseQuery, error) {
	query, ok := f.queries[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("query not found")
	}
	return query, nil
}

func (f *fakeRepo) CreateQuery(
	_ context.Context,
	entity *worker.WorkerClearinghouseQuery,
) (*worker.WorkerClearinghouseQuery, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("wchq_")
	}
	f.queries[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateQuery(
	_ context.Context,
	entity *worker.WorkerClearinghouseQuery,
) (*worker.WorkerClearinghouseQuery, error) {
	f.queries[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) ListPools(
	_ context.Context,
	_ *repositories.ListDOTRandomPoolsRequest,
) (*pagination.CursorListResult[*worker.DOTRandomPool], error) {
	return &pagination.CursorListResult[*worker.DOTRandomPool]{}, nil
}

func (f *fakeRepo) GetPoolByID(
	_ context.Context,
	req *repositories.GetDOTRandomPoolByIDRequest,
) (*worker.DOTRandomPool, error) {
	pool, ok := f.pools[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("pool not found")
	}
	return pool, nil
}

func (f *fakeRepo) GetDefaultPool(
	_ context.Context,
	_ pagination.TenantInfo,
) (*worker.DOTRandomPool, error) {
	return f.defaults, nil
}

func (f *fakeRepo) PoolCodeExists(
	_ context.Context,
	_ *repositories.DOTRandomPoolCodeExistsRequest,
) (bool, error) {
	return false, nil
}

func (f *fakeRepo) CreatePool(
	_ context.Context,
	entity *worker.DOTRandomPool,
) (*worker.DOTRandomPool, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("drpool_")
	}
	f.pools[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdatePool(
	_ context.Context,
	entity *worker.DOTRandomPool,
) (*worker.DOTRandomPool, error) {
	f.pools[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) ClearDefaultPool(
	_ context.Context,
	_ pagination.TenantInfo,
	_ pulid.ID,
) error {
	return nil
}

func (f *fakeRepo) ListDraws(
	_ context.Context,
	_ *repositories.ListDOTRandomDrawsRequest,
) ([]*worker.DOTRandomDraw, error) {
	out := make([]*worker.DOTRandomDraw, 0, len(f.draws))
	for _, draw := range f.draws {
		out = append(out, draw)
	}
	return out, nil
}

func (f *fakeRepo) GetDrawByID(
	_ context.Context,
	req *repositories.GetDOTRandomDrawByIDRequest,
) (*worker.DOTRandomDraw, error) {
	draw, ok := f.draws[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("draw not found")
	}
	return draw, nil
}

func (f *fakeRepo) GetDrawByPeriod(
	_ context.Context,
	req *repositories.GetDOTRandomDrawByPeriodRequest,
) (*worker.DOTRandomDraw, error) {
	for _, draw := range f.draws {
		if draw.PoolID == req.PoolID && draw.PeriodKey == req.PeriodKey &&
			draw.Status != worker.RandomDrawStatusCancelled {
			return draw, nil
		}
	}
	return nil, nil
}

func (f *fakeRepo) CreateDrawWithEntries(
	_ context.Context,
	draw *worker.DOTRandomDraw,
	entries []*worker.DOTRandomDrawEntry,
) (*worker.DOTRandomDraw, error) {
	if draw.ID.IsNil() {
		draw.ID = pulid.MustNew("drdraw_")
	}
	for _, entry := range entries {
		if entry.ID.IsNil() {
			entry.ID = pulid.MustNew("drde_")
		}
		entry.DrawID = draw.ID
		f.entries[entry.ID] = entry
	}
	draw.Entries = entries
	f.draws[draw.ID] = draw
	return draw, nil
}

func (f *fakeRepo) UpdateDraw(
	_ context.Context,
	entity *worker.DOTRandomDraw,
) (*worker.DOTRandomDraw, error) {
	f.draws[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) ListPoolCandidates(
	_ context.Context,
	_ *repositories.ListPoolCandidatesRequest,
) ([]pulid.ID, error) {
	return f.candidates, nil
}

func (f *fakeRepo) ListDrawEntries(
	_ context.Context,
	req *repositories.ListDOTRandomDrawEntriesRequest,
) ([]*worker.DOTRandomDrawEntry, error) {
	out := make([]*worker.DOTRandomDrawEntry, 0, len(f.entries))
	for _, entry := range f.entries {
		if !req.DrawID.IsNil() && entry.DrawID != req.DrawID {
			continue
		}
		out = append(out, entry)
	}
	return out, nil
}

func (f *fakeRepo) GetDrawEntryByID(
	_ context.Context,
	req *repositories.GetDOTRandomDrawEntryByIDRequest,
) (*worker.DOTRandomDrawEntry, error) {
	entry, ok := f.entries[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("entry not found")
	}
	return entry, nil
}

func (f *fakeRepo) UpdateDrawEntry(
	_ context.Context,
	entity *worker.DOTRandomDrawEntry,
) (*worker.DOTRandomDrawEntry, error) {
	f.entries[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateProfileRollup(
	_ context.Context,
	req *repositories.UpdateDrugAlcoholRollupRequest,
) error {
	f.rollups = append(f.rollups, req.Rollup)
	return nil
}

func (f *fakeRepo) ListWorkersWithClearinghouseDue(
	_ context.Context,
	_ *repositories.ListWorkersWithClearinghouseDueRequest,
) ([]repositories.WorkerTenantRef, error) {
	return nil, nil
}

type harness struct {
	svc      *workerdrugalcoholservice.Service
	repo     *fakeRepo
	tenant   pagination.TenantInfo
	workerID pulid.ID
	userID   pulid.ID
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	workerID := pulid.MustNew("wrk_")

	workerRepo := mocks.NewMockWorkerRepository(t)
	workerRepo.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&worker.Worker{ID: workerID, OrganizationID: orgID, BusinessUnitID: buID}, nil).
		Maybe()

	repo := newFakeRepo()

	return &harness{
		svc: workerdrugalcoholservice.NewWithDeps(workerdrugalcoholservice.Deps{
			Repo:       repo,
			WorkerRepo: workerRepo,
		}),
		repo:     repo,
		tenant:   pagination.TenantInfo{OrgID: orgID, BuID: buID},
		workerID: workerID,
		userID:   pulid.MustNew("usr_"),
	}
}

func (h *harness) newTest(
	testType worker.DOTTestType,
	substance worker.DOTTestSubstance,
) *worker.WorkerDOTTest {
	collected := int64(1_700_000_000)
	return &worker.WorkerDOTTest{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		TestType:       testType,
		Substance:      substance,
		Status:         worker.DOTTestStatusCollected,
		Result:         worker.DOTResultPending,
		CollectedAt:    &collected,
	}
}

func (h *harness) openViolation(t *testing.T) *worker.WorkerDOTViolation {
	t.Helper()
	violation := &worker.WorkerDOTViolation{
		ID:             pulid.MustNew("wdotv_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		ViolationType:  worker.DOTViolationPositiveTest,
		Status:         worker.DOTViolationStatusOpen,
		OccurredAt:     1_700_000_000,
	}
	h.repo.violations[violation.ID] = violation
	return violation
}

// A violating result is a prohibition, not a note in a file. Leaving the office
// to remember to open one is exactly how a prohibited driver stays on the
// board.
func TestRecordResult_ViolatingResultOpensAViolation(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	created, err := h.svc.RecordTest(
		t.Context(),
		h.newTest(worker.DOTTestRandom, worker.DOTSubstanceDrug),
		h.userID,
	)
	require.NoError(t, err)

	_, err = h.svc.RecordResult(t.Context(), &workerdrugalcoholservice.RecordResultRequest{
		TenantInfo: h.tenant,
		TestID:     created.ID,
		Result:     worker.DOTResultPositive,
		ResultAt:   1_700_100_000,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	violation, err := h.repo.GetOpenViolation(t.Context(), h.tenant, h.workerID)
	require.NoError(t, err)
	require.NotNil(t, violation)
	assert.Equal(t, worker.DOTViolationPositiveTest, violation.ViolationType)
	assert.Equal(t, created.ID, violation.SourceTestID)
	assert.True(t, violation.Prohibits())
}

// A refusal is not a positive, and the file has to say which it was.
func TestRecordResult_RefusalOpensARefusalViolation(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	created, err := h.svc.RecordTest(
		t.Context(),
		h.newTest(worker.DOTTestRandom, worker.DOTSubstanceDrug),
		h.userID,
	)
	require.NoError(t, err)

	_, err = h.svc.RecordResult(t.Context(), &workerdrugalcoholservice.RecordResultRequest{
		TenantInfo: h.tenant,
		TestID:     created.ID,
		Result:     worker.DOTResultAdulterated,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	violation, err := h.repo.GetOpenViolation(t.Context(), h.tenant, h.workerID)
	require.NoError(t, err)
	require.NotNil(t, violation)
	assert.Equal(t, worker.DOTViolationTestRefusal, violation.ViolationType)
}

// The concentration decides the result. A clerk who files a 0.11 as negative
// would otherwise clear a driver the regulation prohibits.
func TestRecordResult_AlcoholConcentrationGradesItself(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	created, err := h.svc.RecordTest(
		t.Context(),
		h.newTest(worker.DOTTestRandom, worker.DOTSubstanceAlcohol),
		h.userID,
	)
	require.NoError(t, err)

	reading := decimal.NewFromFloat(0.11)
	updated, err := h.svc.RecordResult(t.Context(), &workerdrugalcoholservice.RecordResultRequest{
		TenantInfo:           h.tenant,
		TestID:               created.ID,
		Result:               worker.DOTResultNegative,
		AlcoholConcentration: &reading,
		UserID:               h.userID,
	})
	require.NoError(t, err)

	assert.Equal(t, worker.DOTResultPositive, updated.Result)
	violation, err := h.repo.GetOpenViolation(t.Context(), h.tenant, h.workerID)
	require.NoError(t, err)
	require.NotNil(t, violation)
	assert.Equal(t, worker.DOTViolationAlcoholUse, violation.ViolationType)
}

func TestRecordResult_AlcoholNeedsAConcentration(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	created, err := h.svc.RecordTest(
		t.Context(),
		h.newTest(worker.DOTTestRandom, worker.DOTSubstanceAlcohol),
		h.userID,
	)
	require.NoError(t, err)

	_, err = h.svc.RecordResult(t.Context(), &workerdrugalcoholservice.RecordResultRequest{
		TenantInfo: h.tenant,
		TestID:     created.ID,
		Result:     worker.DOTResultNegative,
		UserID:     h.userID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "concentration")
}

// A completed test is the record. A corrected result is a new test, so the
// original stays on file rather than being overwritten.
func TestRecordResult_CompletedTestIsTerminal(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	created, err := h.svc.RecordTest(
		t.Context(),
		h.newTest(worker.DOTTestRandom, worker.DOTSubstanceDrug),
		h.userID,
	)
	require.NoError(t, err)

	req := &workerdrugalcoholservice.RecordResultRequest{
		TenantInfo: h.tenant,
		TestID:     created.ID,
		Result:     worker.DOTResultNegative,
		UserID:     h.userID,
	}
	_, err = h.svc.RecordResult(t.Context(), req)
	require.NoError(t, err)

	_, err = h.svc.RecordResult(t.Context(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already")
}

// The order is the regulation's: a return-to-duty test taken before the SAP has
// evaluated the driver releases nobody.
func TestRecordResult_ReturnToDutyRefusedBeforeSAPEvaluation(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.openViolation(t)

	created, err := h.svc.RecordTest(
		t.Context(),
		h.newTest(worker.DOTTestReturnToDuty, worker.DOTSubstanceDrug),
		h.userID,
	)
	require.NoError(t, err)

	_, err = h.svc.RecordResult(t.Context(), &workerdrugalcoholservice.RecordResultRequest{
		TenantInfo: h.tenant,
		TestID:     created.ID,
		Result:     worker.DOTResultNegative,
		UserID:     h.userID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "SAP evaluation")
}

func TestRecordResult_ReturnToDutyReleasesTheDriver(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	violation := h.openViolation(t)
	referred := int64(1_700_010_000)
	evaluated := int64(1_700_020_000)
	violation.SAPReferredAt = &referred
	violation.SAPEvaluationCompletedAt = &evaluated
	violation.Status = worker.DOTViolationStatusRTDPending

	created, err := h.svc.RecordTest(
		t.Context(),
		h.newTest(worker.DOTTestReturnToDuty, worker.DOTSubstanceDrug),
		h.userID,
	)
	require.NoError(t, err)

	_, err = h.svc.RecordResult(t.Context(), &workerdrugalcoholservice.RecordResultRequest{
		TenantInfo: h.tenant,
		TestID:     created.ID,
		Result:     worker.DOTResultNegative,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	updated, err := h.repo.GetOpenViolation(t.Context(), h.tenant, h.workerID)
	require.NoError(t, err)
	require.NotNil(t, updated)
	assert.Equal(t, worker.DOTViolationStatusFollowUp, updated.Status)
	assert.False(t, updated.Prohibits())
	// The SAP sets the programme; where none has been recorded the regulation's
	// floor applies rather than nothing at all.
	assert.Equal(t, int32(worker.MinimumFollowUpTests), updated.FollowUpTestCount)
}

// The last follow-up test closes the violation on its own. Leaving it open
// until somebody remembers would keep a driver's file showing a live
// prohibition they have already worked through.
func TestRecordResult_FollowUpProgrammeResolvesItself(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	violation := h.openViolation(t)
	referred := int64(1_700_010_000)
	evaluated := int64(1_700_020_000)
	returned := int64(1_700_030_000)
	violation.SAPReferredAt = &referred
	violation.SAPEvaluationCompletedAt = &evaluated
	violation.RTDCompletedAt = &returned
	violation.FollowUpTestCount = 2
	violation.FollowUpTestsCompleted = 1
	violation.Status = worker.DOTViolationStatusFollowUp

	created, err := h.svc.RecordTest(
		t.Context(),
		h.newTest(worker.DOTTestFollowUp, worker.DOTSubstanceDrug),
		h.userID,
	)
	require.NoError(t, err)

	_, err = h.svc.RecordResult(t.Context(), &workerdrugalcoholservice.RecordResultRequest{
		TenantInfo: h.tenant,
		TestID:     created.ID,
		Result:     worker.DOTResultNegative,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	open, err := h.repo.GetOpenViolation(t.Context(), h.tenant, h.workerID)
	require.NoError(t, err)
	assert.Nil(t, open, "the programme is finished, so nothing should still be open")
	assert.Equal(t, worker.DOTViolationStatusResolved, violation.Status)
	require.NotNil(t, violation.ResolvedAt)
}

// A second violating result during the process does not open a second row. The
// return-to-duty process is one process, and the unique index would refuse it
// anyway.
func TestRecordResult_SecondViolationJoinsTheOpenOne(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.openViolation(t)

	created, err := h.svc.RecordTest(
		t.Context(),
		h.newTest(worker.DOTTestRandom, worker.DOTSubstanceDrug),
		h.userID,
	)
	require.NoError(t, err)

	_, err = h.svc.RecordResult(t.Context(), &workerdrugalcoholservice.RecordResultRequest{
		TenantInfo: h.tenant,
		TestID:     created.ID,
		Result:     worker.DOTResultPositive,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	assert.Len(t, h.repo.violations, 1)
}

// Recording the collection is what closes the selection out. Leaving the entry
// open would keep the round showing work the office has already done.
func TestRecordTest_ClosesTheRandomSelection(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	entry := &worker.DOTRandomDrawEntry{
		ID:             pulid.MustNew("drde_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		Substance:      worker.DOTSubstanceDrug,
		Status:         worker.RandomEntryNotified,
	}
	h.repo.entries[entry.ID] = entry

	test := h.newTest(worker.DOTTestRandom, worker.DOTSubstanceDrug)
	test.DrawEntryID = entry.ID
	test.Status = worker.DOTTestStatusCompleted
	test.Result = worker.DOTResultNegative
	resultAt := int64(1_700_100_000)
	test.ResultAt = &resultAt

	created, err := h.svc.RecordTest(t.Context(), test, h.userID)
	require.NoError(t, err)

	assert.Equal(t, worker.RandomEntryCompleted, entry.Status)
	assert.Equal(t, created.ID, entry.TestID)
	require.NotNil(t, entry.CompletedAt)
}

func newPool(tenant pagination.TenantInfo) *worker.DOTRandomPool {
	return &worker.DOTRandomPool{
		ID:                 pulid.MustNew("drpool_"),
		OrganizationID:     tenant.OrgID,
		BusinessUnitID:     tenant.BuID,
		Code:               "DOT",
		Name:               "DOT Random Pool",
		Period:             worker.RandomPeriodQuarterly,
		DrugRatePercent:    worker.DOTMinimumDrugRatePercent,
		AlcoholRatePercent: worker.DOTMinimumAlcoholRatePercent,
		IsDefault:          true,
	}
}

func TestRunDraw_SelectsToTargetAndRecordsItsEvidence(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	pool := newPool(h.tenant)
	h.repo.pools[pool.ID] = pool
	h.repo.defaults = pool
	for range 40 {
		h.repo.candidates = append(h.repo.candidates, pulid.MustNew("wrk_"))
	}

	draw, err := h.svc.RunDraw(t.Context(), &workerdrugalcoholservice.RunDrawRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	// 40 drivers at 50% a year over four rounds is 5 a quarter; at 10% it is 1.
	assert.Equal(t, int32(40), draw.PoolSize)
	assert.Equal(t, int32(5), draw.DrugTarget)
	assert.Equal(t, int32(5), draw.DrugSelected)
	assert.Equal(t, int32(1), draw.AlcoholTarget)
	assert.Equal(t, int32(1), draw.AlcoholSelected)
	assert.Len(t, draw.Entries, 6)

	// Without the seed and the method the selection is unverifiable, which is
	// the whole point of keeping the round.
	assert.NotEmpty(t, draw.Seed)
	assert.Equal(t, worker.RandomSelectionMethod, draw.Method)
	assert.NotEmpty(t, draw.PeriodKey)
	assert.Greater(t, draw.PeriodEnd, draw.PeriodStart)
}

func TestRunDraw_RefusesASecondRoundForThePeriod(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	pool := newPool(h.tenant)
	h.repo.pools[pool.ID] = pool
	h.repo.defaults = pool
	h.repo.candidates = append(h.repo.candidates, pulid.MustNew("wrk_"))

	_, err := h.svc.RunDraw(t.Context(), &workerdrugalcoholservice.RunDrawRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	_, err = h.svc.RunDraw(t.Context(), &workerdrugalcoholservice.RunDrawRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already been drawn")
}

func TestRunDraw_NeedsAPool(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, err := h.svc.RunDraw(t.Context(), &workerdrugalcoholservice.RunDrawRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "default")
}

func TestRunDraw_RefusesAnEmptyPool(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	pool := newPool(h.tenant)
	h.repo.pools[pool.ID] = pool
	h.repo.defaults = pool

	_, err := h.svc.RunDraw(t.Context(), &workerdrugalcoholservice.RunDrawRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "nobody in this pool")
}

// Completion is what recording the collection means. Setting it by hand would
// let an entry claim a test that does not exist.
func TestUpdateDrawEntry_CompletionIsNotSetByHand(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	entry := &worker.DOTRandomDrawEntry{
		ID:             pulid.MustNew("drde_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		DrawID:         pulid.MustNew("drdraw_"),
		WorkerID:       h.workerID,
		Substance:      worker.DOTSubstanceDrug,
		Status:         worker.RandomEntrySelected,
	}
	h.repo.entries[entry.ID] = entry

	_, err := h.svc.UpdateDrawEntry(t.Context(), &workerdrugalcoholservice.UpdateEntryRequest{
		TenantInfo: h.tenant,
		EntryID:    entry.ID,
		Status:     worker.RandomEntryCompleted,
		UserID:     h.userID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "when its test is recorded")
}

func TestCompleteQuery_RefreshesTheStanding(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	query, err := h.svc.RecordQuery(t.Context(), &worker.WorkerClearinghouseQuery{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		QueryType:      worker.ClearinghouseQueryAnnualLimited,
		RequestedAt:    1_700_000_000,
	}, h.userID)
	require.NoError(t, err)

	_, err = h.svc.CompleteQuery(t.Context(), &workerdrugalcoholservice.CompleteQueryRequest{
		TenantInfo:     h.tenant,
		QueryID:        query.ID,
		Result:         worker.ClearinghouseResultViolationsFound,
		CompletedAt:    1_700_100_000,
		ViolationCount: 1,
		UserID:         h.userID,
	})
	require.NoError(t, err)

	require.NotEmpty(t, h.repo.rollups)
	last := h.repo.rollups[len(h.repo.rollups)-1]
	assert.Equal(t, worker.DrugAlcoholProhibited, last.Status)
	require.NotNil(t, last.NextClearinghouseQueryDue)
}

func TestCompleteQuery_AnsweredQueryCannotBeAnsweredAgain(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	query, err := h.svc.RecordQuery(t.Context(), &worker.WorkerClearinghouseQuery{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		QueryType:      worker.ClearinghouseQueryAnnualLimited,
		RequestedAt:    1_700_000_000,
	}, h.userID)
	require.NoError(t, err)

	req := &workerdrugalcoholservice.CompleteQueryRequest{
		TenantInfo:  h.tenant,
		QueryID:     query.ID,
		Result:      worker.ClearinghouseResultNoViolations,
		CompletedAt: 1_700_100_000,
		UserID:      h.userID,
	}
	_, err = h.svc.CompleteQuery(t.Context(), req)
	require.NoError(t, err)

	_, err = h.svc.CompleteQuery(t.Context(), req)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already been answered")
}

func TestRecordViolation_RefusesASecondOpenOne(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.openViolation(t)

	_, err := h.svc.RecordViolation(t.Context(), &worker.WorkerDOTViolation{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		WorkerID:       h.workerID,
		ViolationType:  worker.DOTViolationActualKnowledge,
		OccurredAt:     1_700_200_000,
	}, h.userID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "unresolved violation")
}

func TestStanding_ProhibitionOutranksPassedTests(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.openViolation(t)

	created, err := h.svc.RecordTest(
		t.Context(),
		h.newTest(worker.DOTTestPreEmployment, worker.DOTSubstanceDrug),
		h.userID,
	)
	require.NoError(t, err)
	created.Status = worker.DOTTestStatusCompleted
	created.Result = worker.DOTResultNegative

	standing, err := h.svc.Standing(t.Context(), h.tenant, h.workerID)
	require.NoError(t, err)

	assert.Equal(t, worker.DrugAlcoholProhibited, standing.Status)
	assert.Equal(t, worker.ReturnToDutySAPEvaluation, standing.ReturnToDuty)
	assert.True(t, standing.HasPreEmploymentTest)
}

// The precondition is checked before the write, not after. Checking it
// afterwards would leave a completed test on file beside an error on screen,
// and a retry that fails because the test is already completed.
func TestRecordResult_ReturnToDutyRefusalLeavesNothingBehind(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.openViolation(t)

	created, err := h.svc.RecordTest(
		t.Context(),
		h.newTest(worker.DOTTestReturnToDuty, worker.DOTSubstanceDrug),
		h.userID,
	)
	require.NoError(t, err)

	_, err = h.svc.RecordResult(t.Context(), &workerdrugalcoholservice.RecordResultRequest{
		TenantInfo: h.tenant,
		TestID:     created.ID,
		Result:     worker.DOTResultNegative,
		UserID:     h.userID,
	})
	require.Error(t, err)

	// The test is still open, so the office can record the result properly once
	// the SAP evaluation is on file.
	stored := h.repo.tests[created.ID]
	require.NotNil(t, stored)
	assert.True(t, stored.IsOpen(), "the refused result must not have been written")
	assert.Equal(t, worker.DOTResultPending, stored.Result)
}

// The same guard has to hold for a test recorded already-completed, which is
// how the office enters one that happened last week.
func TestRecordTest_ReturnToDutyRecordedCompleteIsRefusedBeforeInsert(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.openViolation(t)

	test := h.newTest(worker.DOTTestReturnToDuty, worker.DOTSubstanceDrug)
	test.Status = worker.DOTTestStatusCompleted
	test.Result = worker.DOTResultNegative
	resultAt := int64(1_700_100_000)
	test.ResultAt = &resultAt

	_, err := h.svc.RecordTest(t.Context(), test, h.userID)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "SAP evaluation")
	assert.Empty(t, h.repo.tests, "nothing should have been written")
}
