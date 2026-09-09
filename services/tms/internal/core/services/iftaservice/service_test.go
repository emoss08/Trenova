package iftaservice_test

import (
	"context"
	"errors"
	"sort"
	"testing"
	"time"
	_ "time/tzdata"

	"github.com/emoss08/trenova/internal/core/domain/ifta"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/iftaservice"
	"github.com/emoss08/trenova/internal/testutil/mocks"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

type fakeRepo struct {
	jurisdictions map[pulid.ID]*ifta.Jurisdiction
	rates         map[pulid.ID]*ifta.TaxRate
	entries       map[pulid.ID]*ifta.JurisdictionMileageEntry
	returns       map[pulid.ID]*ifta.Return
	lines         map[pulid.ID][]*ifta.ReturnLine
	miles         *repositories.MileAccumulation
	accumulateReq *repositories.AccumulateMilesRequest
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		jurisdictions: map[pulid.ID]*ifta.Jurisdiction{},
		rates:         map[pulid.ID]*ifta.TaxRate{},
		entries:       map[pulid.ID]*ifta.JurisdictionMileageEntry{},
		returns:       map[pulid.ID]*ifta.Return{},
		lines:         map[pulid.ID][]*ifta.ReturnLine{},
		miles:         &repositories.MileAccumulation{},
	}
}

func (f *fakeRepo) ListJurisdictions(
	_ context.Context,
	req *repositories.ListJurisdictionsRequest,
) ([]*ifta.Jurisdiction, error) {
	out := make([]*ifta.Jurisdiction, 0, len(f.jurisdictions))
	for _, j := range f.jurisdictions {
		if req.MembersOnly && !j.IsIftaMember {
			continue
		}
		out = append(out, j)
	}
	sort.Slice(out, func(i, k int) bool { return out[i].Code < out[k].Code })
	return out, nil
}

func (f *fakeRepo) GetJurisdictionByID(_ context.Context, id pulid.ID) (*ifta.Jurisdiction, error) {
	j, ok := f.jurisdictions[id]
	if !ok {
		return nil, errortypes.NewNotFoundError("jurisdiction not found")
	}
	return j, nil
}

func (f *fakeRepo) GetJurisdictionsByIDs(
	_ context.Context,
	ids []pulid.ID,
) ([]*ifta.Jurisdiction, error) {
	out := make([]*ifta.Jurisdiction, 0, len(ids))
	for _, id := range ids {
		if j, ok := f.jurisdictions[id]; ok {
			out = append(out, j)
		}
	}
	return out, nil
}

func (f *fakeRepo) GetJurisdictionByCode(
	_ context.Context,
	countryCode, code string,
) (*ifta.Jurisdiction, error) {
	for _, j := range f.jurisdictions {
		if j.CountryCode == countryCode && j.Code == code {
			return j, nil
		}
	}
	return nil, errortypes.NewNotFoundError("jurisdiction not found")
}

func (f *fakeRepo) FindJurisdictionsByCodes(
	_ context.Context,
	keys []string,
) (map[string]*ifta.Jurisdiction, error) {
	out := make(map[string]*ifta.Jurisdiction, len(keys))
	for _, key := range keys {
		for _, j := range f.jurisdictions {
			if j.Key() == key {
				out[key] = j
			}
		}
	}
	return out, nil
}

func (f *fakeRepo) ListTaxRates(
	_ context.Context,
	_ *repositories.ListTaxRatesRequest,
) (*pagination.CursorListResult[*ifta.TaxRate], error) {
	items := make([]*ifta.TaxRate, 0, len(f.rates))
	for _, r := range f.rates {
		items = append(items, r)
	}
	return pagination.NewCursorListResult(items, len(items)+1), nil
}

func (f *fakeRepo) GetTaxRateByID(_ context.Context, id pulid.ID) (*ifta.TaxRate, error) {
	r, ok := f.rates[id]
	if !ok {
		return nil, errortypes.NewNotFoundError("rate not found")
	}
	return r, nil
}

func (f *fakeRepo) UpsertTaxRates(
	_ context.Context,
	rates []*ifta.TaxRate,
) ([]*ifta.TaxRate, error) {
	for _, rate := range rates {
		replaced := false
		for _, existing := range f.rates {
			if existing.Key() == rate.Key() && existing.Year == rate.Year &&
				existing.Quarter == rate.Quarter {
				existing.RatePerGallon = rate.RatePerGallon
				existing.SurchargeRatePerGallon = rate.SurchargeRatePerGallon
				existing.Version++
				*rate = *existing
				replaced = true
				break
			}
		}
		if !replaced {
			if rate.ID.IsNil() {
				rate.ID = pulid.MustNew("iftr_")
			}
			f.rates[rate.ID] = rate
		}
	}
	return rates, nil
}

func (f *fakeRepo) DeleteTaxRate(_ context.Context, id pulid.ID, _ int64) error {
	if _, ok := f.rates[id]; !ok {
		return errortypes.NewNotFoundError("rate not found")
	}
	delete(f.rates, id)
	return nil
}

func (f *fakeRepo) ResolveRates(
	_ context.Context,
	req *repositories.ResolveRatesRequest,
) (map[ifta.RateKey]*ifta.TaxRate, error) {
	out := make(map[ifta.RateKey]*ifta.TaxRate, len(f.rates))
	for _, r := range f.rates {
		if r.Year == req.Year && r.Quarter == req.Quarter {
			out[r.Key()] = r
		}
	}
	return out, nil
}

func (f *fakeRepo) ListMileageEntries(
	_ context.Context,
	_ *repositories.ListMileageEntriesRequest,
) (*pagination.CursorListResult[*ifta.JurisdictionMileageEntry], error) {
	items := make([]*ifta.JurisdictionMileageEntry, 0, len(f.entries))
	for _, e := range f.entries {
		items = append(items, e)
	}
	return pagination.NewCursorListResult(items, len(items)+1), nil
}

func (f *fakeRepo) GetMileageEntryByID(
	_ context.Context,
	req *repositories.GetMileageEntryByIDRequest,
) (*ifta.JurisdictionMileageEntry, error) {
	e, ok := f.entries[req.ID]
	if !ok || e.OrganizationID != req.TenantInfo.OrgID {
		return nil, errortypes.NewNotFoundError("entry not found")
	}
	return e, nil
}

func (f *fakeRepo) CreateMileageEntry(
	_ context.Context,
	entity *ifta.JurisdictionMileageEntry,
) (*ifta.JurisdictionMileageEntry, error) {
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("ifme_")
	}
	f.entries[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) UpdateMileageEntry(
	_ context.Context,
	entity *ifta.JurisdictionMileageEntry,
) (*ifta.JurisdictionMileageEntry, error) {
	entity.Version++
	f.entries[entity.ID] = entity
	return entity, nil
}

func (f *fakeRepo) DeleteMileageEntry(
	_ context.Context,
	req *repositories.DeleteMileageEntryRequest,
) error {
	delete(f.entries, req.ID)
	return nil
}

func (f *fakeRepo) ListReturns(
	_ context.Context,
	_ *repositories.ListReturnsRequest,
) (*pagination.CursorListResult[*ifta.Return], error) {
	items := make([]*ifta.Return, 0, len(f.returns))
	for _, r := range f.returns {
		items = append(items, r)
	}
	return pagination.NewCursorListResult(items, len(items)+1), nil
}

func (f *fakeRepo) GetReturnByID(
	_ context.Context,
	req *repositories.GetReturnByIDRequest,
) (*ifta.Return, error) {
	r, ok := f.returns[req.ID]
	if !ok || r.OrganizationID != req.TenantInfo.OrgID {
		return nil, errortypes.NewNotFoundError("return not found")
	}
	copied := *r
	if req.IncludeLines {
		copied.Lines = f.lines[r.ID]
	}
	return &copied, nil
}

func (f *fakeRepo) GetReturnsByIDs(
	_ context.Context,
	req *repositories.GetReturnsByIDsRequest,
) ([]*ifta.Return, error) {
	out := make([]*ifta.Return, 0, len(req.IDs))
	for _, id := range req.IDs {
		r, ok := f.returns[id]
		if !ok || r.OrganizationID != req.TenantInfo.OrgID {
			continue
		}
		copied := *r
		out = append(out, &copied)
	}
	return out, nil
}

func (f *fakeRepo) GetOpenReturnForPeriod(
	_ context.Context,
	req *repositories.GetOpenReturnForPeriodRequest,
) (*ifta.Return, error) {
	for _, r := range f.returns {
		if r.OrganizationID == req.TenantInfo.OrgID && r.Year == req.Year &&
			r.Quarter == req.Quarter && r.Status != ifta.ReturnStatusFiled {
			copied := *r
			return &copied, nil
		}
	}
	return nil, nil
}

func (f *fakeRepo) GetLatestReturnForPeriod(
	_ context.Context,
	req *repositories.GetLatestReturnForPeriodRequest,
) (*ifta.Return, error) {
	var latest *ifta.Return
	for _, r := range f.returns {
		if r.OrganizationID != req.TenantInfo.OrgID || r.Year != req.Year ||
			r.Quarter != req.Quarter {
			continue
		}
		if latest == nil || r.AmendmentNumber > latest.AmendmentNumber {
			latest = r
		}
	}
	if latest == nil {
		return nil, nil
	}
	copied := *latest
	return &copied, nil
}

func (f *fakeRepo) CreateReturn(_ context.Context, entity *ifta.Return) (*ifta.Return, error) {
	for _, r := range f.returns {
		if r.OrganizationID == entity.OrganizationID && r.Year == entity.Year &&
			r.Quarter == entity.Quarter && r.Status != ifta.ReturnStatusFiled {
			return nil, errortypes.NewConflictError("open return exists")
		}
	}
	if entity.ID.IsNil() {
		entity.ID = pulid.MustNew("ifr_")
	}
	stored := *entity
	f.returns[entity.ID] = &stored
	return entity, nil
}

func (f *fakeRepo) UpdateReturn(_ context.Context, entity *ifta.Return) (*ifta.Return, error) {
	current, ok := f.returns[entity.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("return not found")
	}
	if current.Version != entity.Version {
		return nil, errors.New("version mismatch in fake repo")
	}
	entity.Version++
	stored := *entity
	f.returns[entity.ID] = &stored
	return entity, nil
}

func (f *fakeRepo) ReplaceReturnLines(
	_ context.Context,
	ret *ifta.Return,
	lines []*ifta.ReturnLine,
) (*ifta.Return, error) {
	current, ok := f.returns[ret.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("return not found")
	}
	if current.Version != ret.Version {
		return nil, errors.New("version mismatch in fake repo")
	}
	ret.Version++
	now := time.Now().Unix()
	ret.ComputedAt = &now
	for i, line := range lines {
		line.ReturnID = ret.ID
		line.SortOrder = i
	}
	f.lines[ret.ID] = lines
	stored := *ret
	f.returns[ret.ID] = &stored
	ret.Lines = lines
	return ret, nil
}

func (f *fakeRepo) DeleteReturn(_ context.Context, req *repositories.DeleteReturnRequest) error {
	r, ok := f.returns[req.ID]
	if !ok || r.OrganizationID != req.TenantInfo.OrgID {
		return errortypes.NewNotFoundError("return not found")
	}
	delete(f.returns, req.ID)
	delete(f.lines, req.ID)
	return nil
}

func (f *fakeRepo) AccumulateMiles(
	_ context.Context,
	req *repositories.AccumulateMilesRequest,
) (*repositories.MileAccumulation, error) {
	f.accumulateReq = req
	return f.miles, nil
}

type fakeFuel struct {
	rows []*repositories.FuelAccumulationRow
	req  *repositories.AccumulateFuelRequest
}

func (f *fakeFuel) AccumulateFuel(
	_ context.Context,
	req *repositories.AccumulateFuelRequest,
) ([]*repositories.FuelAccumulationRow, error) {
	f.req = req
	return f.rows, nil
}

type harness struct {
	svc       *iftaservice.Service
	repo      *fakeRepo
	fuel      *fakeFuel
	tenant    pagination.TenantInfo
	userID    pulid.ID
	tractorID pulid.ID
	tx        *ifta.Jurisdiction
	ok        *ifta.Jurisdiction
	period    ifta.Period
	now       int64
	ops       []permission.Operation
}

func (h *harness) recordAudit(params *services.LogActionParams, _ ...services.LogOption) {
	h.ops = append(h.ops, params.Operation)
}

func newHarness(t *testing.T) *harness {
	t.Helper()

	h := &harness{
		repo:      newFakeRepo(),
		fuel:      &fakeFuel{},
		tenant:    pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		userID:    pulid.MustNew("usr_"),
		tractorID: pulid.MustNew("trac_"),
		period:    ifta.NewPeriod(2026, 1),
		now:       time.Date(2026, time.May, 15, 12, 0, 0, 0, time.UTC).Unix(),
	}
	h.tx = &ifta.Jurisdiction{
		ID: pulid.MustNew("ifj_"), CountryCode: "US", Code: "TX", Name: "Texas",
		IsIftaMember: true, Status: ifta.JurisdictionStatusActive,
	}
	h.ok = &ifta.Jurisdiction{
		ID: pulid.MustNew("ifj_"), CountryCode: "US", Code: "OK", Name: "Oklahoma",
		IsIftaMember: true, Status: ifta.JurisdictionStatusActive,
	}
	h.repo.jurisdictions[h.tx.ID] = h.tx
	h.repo.jurisdictions[h.ok.ID] = h.ok

	h.repo.miles = &repositories.MileAccumulation{
		RouteRows: []*repositories.MileRow{{
			TractorID: h.tractorID, CountryCode: "US", JurisdictionCode: "TX",
			JurisdictionID: h.tx.ID, Miles: dec("1000"), LoadedMiles: dec("1000"), MoveCount: 3,
		}},
	}
	h.fuel.rows = []*repositories.FuelAccumulationRow{{
		TractorID: h.tractorID, JurisdictionID: h.tx.ID, FuelType: diesel,
		Gallons: dec("100"), TaxPaidGallons: dec("100"), PurchaseCount: 2,
	}}
	h.addRate(h.tx, "0.2000")

	tractors := mocks.NewMockTractorRepository(t)
	tractors.EXPECT().
		GetByIDs(mock.Anything, mock.Anything).
		Return([]*tractor.Tractor{{
			ID: h.tractorID, OrganizationID: h.tenant.OrgID, BusinessUnitID: h.tenant.BuID,
			FuelType: diesel, IFTAQualified: true,
		}}, nil).
		Maybe()
	tractors.EXPECT().
		GetByID(mock.Anything, mock.Anything).
		Return(&tractor.Tractor{
			ID: h.tractorID, OrganizationID: h.tenant.OrgID, BusinessUnitID: h.tenant.BuID,
		}, nil).
		Maybe()

	orgs := mocks.NewMockOrganizationCacheRepository(t)
	orgs.EXPECT().
		GetByID(mock.Anything, h.tenant.OrgID).
		Return(&tenant.Organization{ID: h.tenant.OrgID, Timezone: "America/Chicago"}, nil).
		Maybe()

	audit := mocks.NewMockAuditService(t)
	audit.EXPECT().LogAction(mock.Anything, mock.Anything).Run(h.recordAudit).Return(nil).Maybe()
	audit.EXPECT().
		LogAction(mock.Anything, mock.Anything, mock.Anything).
		Run(h.recordAudit).
		Return(nil).
		Maybe()

	h.svc = iftaservice.NewWithDeps(iftaservice.Deps{
		Repo:         h.repo,
		Fuel:         h.fuel,
		TractorRepo:  tractors,
		OrgCacheRepo: orgs,
		AuditService: audit,
		Now:          func() int64 { return h.now },
	})

	return h
}

func (h *harness) addRate(j *ifta.Jurisdiction, rate string) *ifta.TaxRate {
	r := &ifta.TaxRate{
		ID: pulid.MustNew("iftr_"), JurisdictionID: j.ID, Year: h.period.Year,
		Quarter: h.period.Quarter, FuelType: diesel, RatePerGallon: dec(rate),
	}
	h.repo.rates[r.ID] = r
	return r
}

func (h *harness) generate(t *testing.T) *ifta.Return {
	t.Helper()
	ret, err := h.svc.Generate(t.Context(), &iftaservice.GenerateReturnRequest{
		TenantInfo: h.tenant, Period: h.period, UserID: h.userID,
	})
	require.NoError(t, err)
	return ret
}

func (h *harness) finalize(t *testing.T, ret *ifta.Return) *ifta.Return {
	t.Helper()
	finalized, err := h.svc.Finalize(t.Context(), &iftaservice.ReturnActionRequest{
		TenantInfo: h.tenant, ID: ret.ID, Version: ret.Version, UserID: h.userID,
	})
	require.NoError(t, err)
	return finalized
}

func (h *harness) file(t *testing.T, ret *ifta.Return) *ifta.Return {
	t.Helper()
	h.now += 3600
	filed, err := h.svc.MarkFiled(t.Context(), &iftaservice.MarkFiledRequest{
		TenantInfo: h.tenant, ID: ret.ID, Version: ret.Version, UserID: h.userID,
		FiledAt: h.now - 60, FilingReference: "TX-2026-Q1-0001",
	})
	require.NoError(t, err)
	return filed
}

func (h *harness) hasOp(op permission.Operation) bool {
	for _, recorded := range h.ops {
		if recorded == op {
			return true
		}
	}
	return false
}

func assertValidationField(t *testing.T, err error, field string) {
	t.Helper()
	var vErr *errortypes.Error
	require.ErrorAs(t, err, &vErr)
	assert.Equal(t, field, vErr.Field)
}

func TestGenerate_UsesOrganizationTimezoneForBounds(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	ret := h.generate(t)

	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)
	start, end := h.period.Bounds(chicago)
	assert.Equal(t, "America/Chicago", ret.Timezone)
	assert.Equal(t, start, ret.PeriodStart)
	assert.Equal(t, end, ret.PeriodEnd)
	assert.Equal(t, ifta.ReturnStatusDraft, ret.Status)
	require.NotNil(t, h.repo.accumulateReq)
	assert.Equal(t, start, h.repo.accumulateReq.Start)
	assert.Equal(t, end, h.repo.accumulateReq.End)
	require.NotNil(t, h.fuel.req)
	assert.Equal(t, start, h.fuel.req.Start)
	assert.Equal(t, end, h.fuel.req.End)

	require.Len(t, ret.Lines, 1)
	assert.Equal(t, "1000", ret.TotalMiles.String())
	assert.Equal(t, "100", ret.TotalGallons.String())
	assert.Equal(t, int64(0), ret.NetDueMinor)
	assert.NotNil(t, ret.ComputedAt)
	assert.True(t, h.hasOp(permission.OpCreate))
}

func TestGenerate_RefusesWhenAnOpenReturnExists(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.generate(t)

	_, err := h.svc.Generate(t.Context(), &iftaservice.GenerateReturnRequest{
		TenantInfo: h.tenant, Period: h.period, UserID: h.userID,
	})

	var conflict *errortypes.ConflictError
	require.ErrorAs(t, err, &conflict)
}

func TestGenerate_RefusesWhenTheReturnIsFiled(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.file(t, h.finalize(t, h.generate(t)))

	_, err := h.svc.Generate(t.Context(), &iftaservice.GenerateReturnRequest{
		TenantInfo: h.tenant, Period: h.period, UserID: h.userID,
	})

	var conflict *errortypes.ConflictError
	require.ErrorAs(t, err, &conflict)
	assert.Contains(t, err.Error(), "amend")
}

func TestRecompute_DraftOnly(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ret := h.generate(t)

	h.repo.miles.RouteRows[0].Miles = dec("2000")
	h.repo.miles.RouteRows[0].LoadedMiles = dec("2000")
	recomputed, err := h.svc.Recompute(t.Context(), &iftaservice.ReturnActionRequest{
		TenantInfo: h.tenant, ID: ret.ID, Version: ret.Version, UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, "2000", recomputed.TotalMiles.String())
	assert.Equal(t, ret.Version+1, recomputed.Version)
	assert.True(t, h.hasOp(permission.OpUpdate))

	finalized := h.finalize(t, recomputed)
	_, err = h.svc.Recompute(t.Context(), &iftaservice.ReturnActionRequest{
		TenantInfo: h.tenant, ID: finalized.ID, Version: finalized.Version, UserID: h.userID,
	})
	assertValidationField(t, err, "status")
}

func TestRecompute_RefusesAStaleVersion(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ret := h.generate(t)

	_, err := h.svc.Recompute(t.Context(), &iftaservice.ReturnActionRequest{
		TenantInfo: h.tenant, ID: ret.ID, Version: ret.Version + 5, UserID: h.userID,
	})

	assertValidationField(t, err, "version")
}

func TestFinalize_RecomputesBeforeLocking(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ret := h.generate(t)

	h.repo.miles.RouteRows[0].Miles = dec("1500")
	h.repo.miles.RouteRows[0].LoadedMiles = dec("1500")
	finalized := h.finalize(t, ret)

	assert.Equal(t, ifta.ReturnStatusFinalized, finalized.Status)
	assert.Equal(t, "1500", finalized.TotalMiles.String(), "finalize ran the computation")
	require.NotNil(t, finalized.FinalizedAt)
	assert.Equal(t, h.now, *finalized.FinalizedAt)
	assert.Equal(t, h.userID, finalized.FinalizedByID)
	assert.True(t, h.hasOp(permission.OpApprove))
}

func TestFinalize_RefusesWhileARateIsMissing(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	ret := h.generate(t)
	h.repo.miles.RouteRows = append(h.repo.miles.RouteRows, &repositories.MileRow{
		TractorID: h.tractorID, CountryCode: "US", JurisdictionCode: "OK",
		JurisdictionID: h.ok.ID, Miles: dec("200"), LoadedMiles: dec("200"), MoveCount: 1,
	})

	_, err := h.svc.Finalize(t.Context(), &iftaservice.ReturnActionRequest{
		TenantInfo: h.tenant, ID: ret.ID, Version: ret.Version, UserID: h.userID,
	})

	assertValidationField(t, err, "problems")
	stored := h.repo.returns[ret.ID]
	assert.Equal(t, ifta.ReturnStatusDraft, stored.Status)
	assert.True(t, stored.HasProblem(ifta.ProblemMissingRate), "the recompute was kept")
	assert.False(t, h.hasOp(permission.OpApprove))

	h.addRate(h.ok, "0.1900")
	finalized := h.finalize(t, stored)
	assert.Equal(t, ifta.ReturnStatusFinalized, finalized.Status)
}

func TestReopen_RequiresAReasonAndClearsTheStamps(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	finalized := h.finalize(t, h.generate(t))

	_, err := h.svc.Reopen(t.Context(), &iftaservice.ReopenReturnRequest{
		TenantInfo: h.tenant, ID: finalized.ID, Version: finalized.Version,
		UserID: h.userID, Reason: "too short",
	})
	assertValidationField(t, err, "reason")
	assert.False(t, h.hasOp(permission.OpReopen))

	reopened, err := h.svc.Reopen(t.Context(), &iftaservice.ReopenReturnRequest{
		TenantInfo: h.tenant, ID: finalized.ID, Version: finalized.Version,
		UserID: h.userID, Reason: "  A late fuel receipt arrived from the Amarillo yard  ",
	})
	require.NoError(t, err)
	assert.Equal(t, ifta.ReturnStatusDraft, reopened.Status)
	assert.Nil(t, reopened.FinalizedAt)
	assert.True(t, reopened.FinalizedByID.IsNil())
	require.NotNil(t, reopened.ReopenedAt)
	assert.Equal(t, h.userID, reopened.ReopenedByID)
	assert.Equal(t, "A late fuel receipt arrived from the Amarillo yard", reopened.ReopenReason)
	assert.True(t, h.hasOp(permission.OpReopen))
	assert.True(t, reopened.CanRecompute())
}

func TestReopen_FinalizedOnly(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	draft := h.generate(t)

	_, err := h.svc.Reopen(t.Context(), &iftaservice.ReopenReturnRequest{
		TenantInfo: h.tenant, ID: draft.ID, Version: draft.Version,
		UserID: h.userID, Reason: "A perfectly good reason",
	})
	assertValidationField(t, err, "status")

	filed := h.file(t, h.finalize(t, draft))
	_, err = h.svc.Reopen(t.Context(), &iftaservice.ReopenReturnRequest{
		TenantInfo: h.tenant, ID: filed.ID, Version: filed.Version,
		UserID: h.userID, Reason: "A perfectly good reason",
	})
	assertValidationField(t, err, "status")
}

func TestMarkFiled_FinalizedOnlyWithinDateBounds(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	draft := h.generate(t)

	_, err := h.svc.MarkFiled(t.Context(), &iftaservice.MarkFiledRequest{
		TenantInfo: h.tenant, ID: draft.ID, Version: draft.Version, UserID: h.userID,
		FiledAt: h.now,
	})
	assertValidationField(t, err, "status")

	finalized := h.finalize(t, draft)

	_, err = h.svc.MarkFiled(t.Context(), &iftaservice.MarkFiledRequest{
		TenantInfo: h.tenant, ID: finalized.ID, Version: finalized.Version, UserID: h.userID,
		FiledAt: *finalized.FinalizedAt - 1,
	})
	assertValidationField(t, err, "filedAt")

	_, err = h.svc.MarkFiled(t.Context(), &iftaservice.MarkFiledRequest{
		TenantInfo: h.tenant, ID: finalized.ID, Version: finalized.Version, UserID: h.userID,
		FiledAt: h.now + 1,
	})
	assertValidationField(t, err, "filedAt")

	h.now += 3600
	filed, err := h.svc.MarkFiled(t.Context(), &iftaservice.MarkFiledRequest{
		TenantInfo: h.tenant, ID: finalized.ID, Version: finalized.Version, UserID: h.userID,
		FiledAt: h.now - 60, FilingReference: "  TX-0001  ",
	})
	require.NoError(t, err)
	assert.Equal(t, ifta.ReturnStatusFiled, filed.Status)
	require.NotNil(t, filed.FiledAt)
	assert.Equal(t, h.now-60, *filed.FiledAt)
	assert.Equal(t, "TX-0001", filed.FilingReference)
	assert.Equal(t, h.userID, filed.FiledByID)
	assert.True(t, h.hasOp(permission.OpSubmit))
}

func TestAmend_FiledOnlyAndIncrementsTheAmendmentNumber(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	draft := h.generate(t)

	_, err := h.svc.Amend(t.Context(), &iftaservice.AmendReturnRequest{
		TenantInfo: h.tenant, ID: draft.ID, Reason: "Missed a fuel purchase in Oklahoma", UserID: h.userID,
	})
	assertValidationField(t, err, "status")

	filed := h.file(t, h.finalize(t, draft))

	_, err = h.svc.Amend(t.Context(), &iftaservice.AmendReturnRequest{
		TenantInfo: h.tenant, ID: filed.ID, Reason: "short", UserID: h.userID,
	})
	assertValidationField(t, err, "reason")

	amendment, err := h.svc.Amend(t.Context(), &iftaservice.AmendReturnRequest{
		TenantInfo: h.tenant, ID: filed.ID, Reason: "Missed a fuel purchase in Oklahoma", UserID: h.userID,
	})
	require.NoError(t, err)
	assert.Equal(t, 1, amendment.AmendmentNumber)
	require.NotNil(t, amendment.AmendsReturnID)
	assert.Equal(t, filed.ID, *amendment.AmendsReturnID)
	assert.Equal(t, ifta.ReturnStatusDraft, amendment.Status)
	assert.Equal(t, filed.PeriodStart, amendment.PeriodStart)
	assert.Equal(t, filed.Timezone, amendment.Timezone)
	assert.NotEqual(t, filed.ID, amendment.ID)

	latest, err := h.svc.GetForPeriod(t.Context(), h.tenant, h.period)
	require.NoError(t, err)
	assert.Equal(t, amendment.ID, latest.ID)

	_, err = h.svc.Amend(t.Context(), &iftaservice.AmendReturnRequest{
		TenantInfo: h.tenant, ID: filed.ID, Reason: "Missed another fuel purchase", UserID: h.userID,
	})
	var conflict *errortypes.ConflictError
	require.ErrorAs(t, err, &conflict, "the open amendment blocks a second one")
}

func TestDelete_DraftOnly(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	draft := h.generate(t)
	finalized := h.finalize(t, draft)

	err := h.svc.Delete(t.Context(), &iftaservice.ReturnActionRequest{
		TenantInfo: h.tenant, ID: finalized.ID, Version: finalized.Version, UserID: h.userID,
	})
	assertValidationField(t, err, "status")

	reopened, err := h.svc.Reopen(t.Context(), &iftaservice.ReopenReturnRequest{
		TenantInfo: h.tenant, ID: finalized.ID, Version: finalized.Version,
		UserID: h.userID, Reason: "Generated against the wrong quarter",
	})
	require.NoError(t, err)

	require.NoError(t, h.svc.Delete(t.Context(), &iftaservice.ReturnActionRequest{
		TenantInfo: h.tenant, ID: reopened.ID, Version: reopened.Version, UserID: h.userID,
	}))
	_, err = h.svc.Get(t.Context(), h.tenant, reopened.ID)
	var notFound *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFound)
	assert.True(t, h.hasOp(permission.OpDelete))
}

func TestReturns_AreInvisibleToAnotherTenant(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	draft := h.generate(t)

	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	_, err := h.svc.Recompute(t.Context(), &iftaservice.ReturnActionRequest{
		TenantInfo: other, ID: draft.ID, Version: draft.Version, UserID: h.userID,
	})
	var notFound *errortypes.NotFoundError
	require.ErrorAs(t, err, &notFound)
}

func TestCurrentPeriod_IsTheLastCompletedQuarterInTheOrganizationTimezone(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	h.now = time.Date(2026, time.April, 1, 3, 0, 0, 0, time.UTC).Unix()

	current, err := h.svc.CurrentPeriod(t.Context(), h.tenant)
	require.NoError(t, err)
	assert.Equal(t, ifta.NewPeriod(2025, 4), current,
		"03:00 UTC on Apr 1 is still Mar 31 in Chicago, so Q1 is not over yet")

	h.now = time.Date(2026, time.April, 1, 6, 0, 0, 0, time.UTC).Unix()
	current, err = h.svc.CurrentPeriod(t.Context(), h.tenant)
	require.NoError(t, err)
	assert.Equal(t, ifta.NewPeriod(2026, 1), current)

	info, err := h.svc.PeriodInfo(t.Context(), h.tenant, 2026, 1)
	require.NoError(t, err)
	assert.Equal(t, "2026Q1", info.Key)
	assert.Equal(t, "America/Chicago", info.Timezone)
	assert.Greater(t, info.DueDate, info.End)
}

func TestMileageEntry_CreateAssignsThePeriodFromTheOrganizationTimezone(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	chicago, err := time.LoadLocation("America/Chicago")
	require.NoError(t, err)

	entry := &ifta.JurisdictionMileageEntry{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		TractorID:      h.tractorID,
		JurisdictionID: h.tx.ID,
		TraveledAt:     time.Date(2026, time.March, 31, 23, 30, 0, 0, chicago).Unix(),
		Miles:          dec("212.345"),
		Loaded:         false,
	}

	created, err := h.svc.CreateMileageEntry(t.Context(), entry, h.userID)
	require.NoError(t, err)
	assert.Equal(t, ifta.NewPeriod(2026, 1), created.Period(), "still Q1 in Chicago")
	assert.Equal(t, "212.35", created.Miles.StringFixed(2))
	assert.Equal(t, ifta.MileageSourceManual, created.Source)
	assert.Equal(t, h.userID, created.CreatedByID)
	assert.True(t, h.hasOp(permission.OpCreate))
}

func TestMileageEntry_RejectsAnInactiveJurisdiction(t *testing.T) {
	t.Parallel()
	h := newHarness(t)
	inactive := &ifta.Jurisdiction{
		ID: pulid.MustNew("ifj_"), CountryCode: "US", Code: "HI", Name: "Hawaii",
		Status: ifta.JurisdictionStatusInactive,
	}
	h.repo.jurisdictions[inactive.ID] = inactive

	_, err := h.svc.CreateMileageEntry(t.Context(), &ifta.JurisdictionMileageEntry{
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		TractorID:      h.tractorID,
		JurisdictionID: inactive.ID,
		TraveledAt:     h.now,
		Miles:          dec("10"),
	}, h.userID)

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	assert.Contains(t, multiErr.Error(), "inactive")
}

func TestUpsertTaxRates_ValidatesEachRowAndRejectsDuplicates(t *testing.T) {
	t.Parallel()
	h := newHarness(t)

	_, err := h.svc.UpsertTaxRates(t.Context(), &iftaservice.UpsertTaxRatesRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
		Rates: []*ifta.TaxRate{
			{JurisdictionID: h.tx.ID, Year: 2026, Quarter: 2, FuelType: diesel, RatePerGallon: dec("0.21")},
			{JurisdictionID: h.tx.ID, Year: 2026, Quarter: 2, FuelType: diesel, RatePerGallon: dec("0.22")},
			{JurisdictionID: pulid.MustNew("ifj_"), Year: 2026, Quarter: 2, FuelType: diesel},
		},
	})
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, e := range multiErr.Errors {
		fields = append(fields, e.Field)
	}
	assert.Contains(t, fields, "rates[1].fuelType")

	saved, err := h.svc.UpsertTaxRates(t.Context(), &iftaservice.UpsertTaxRatesRequest{
		TenantInfo: h.tenant,
		UserID:     h.userID,
		Rates: []*ifta.TaxRate{
			{JurisdictionID: h.ok.ID, Year: 2026, Quarter: 2, FuelType: diesel, RatePerGallon: dec("0.19")},
		},
	})
	require.NoError(t, err)
	require.Len(t, saved, 1)
	assert.False(t, saved[0].ID.IsNil())
	assert.True(t, h.hasOp(permission.OpManage))

	_, err = h.svc.UpsertTaxRates(t.Context(), &iftaservice.UpsertTaxRatesRequest{
		TenantInfo: h.tenant, UserID: h.userID,
	})
	assertValidationField(t, err, "rates")
}
