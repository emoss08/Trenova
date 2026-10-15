package factoringservice

import (
	"bytes"
	"context"
	"io"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documentcontent"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/internal/modules/parties/ports/factoringcompanyports"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeRemittances struct {
	factoringports.RemittanceRepository
	mu         sync.Mutex
	byID       map[pulid.ID]*factoring.Remittance
	candidates []*factoring.RemittanceCandidate
	profile    *factoring.RemittanceProfile
	saved      []*factoring.RemittanceProfile
}

func newFakeRemittances() *fakeRemittances {
	return &fakeRemittances{byID: map[pulid.ID]*factoring.Remittance{}}
}

func copyRemittance(rem *factoring.Remittance) *factoring.Remittance {
	out := *rem
	out.Lines = make([]*factoring.RemittanceLine, 0, len(rem.Lines))
	for _, line := range rem.Lines {
		copied := *line
		out.Lines = append(out.Lines, &copied)
	}
	return &out
}

func (f *fakeRemittances) Create(
	_ context.Context,
	rem *factoring.Remittance,
) (*factoring.Remittance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	rem.ID = pulid.MustNew("facrem_")
	rem.CreatedAt = testNow
	f.byID[rem.ID] = copyRemittance(rem)
	return rem, nil
}

func (f *fakeRemittances) Update(
	_ context.Context,
	rem *factoring.Remittance,
) (*factoring.Remittance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored, ok := f.byID[rem.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("remittance not found")
	}
	if stored.Version != rem.Version {
		return nil, errortypes.NewConflictError("stale")
	}
	rem.Version++
	lines := stored.Lines
	f.byID[rem.ID] = copyRemittance(rem)
	f.byID[rem.ID].Lines = lines
	out := copyRemittance(rem)
	out.Lines = nil
	return out, nil
}

func (f *fakeRemittances) Get(
	_ context.Context,
	req factoringports.GetRemittanceRequest,
) (*factoring.Remittance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored, ok := f.byID[req.ID]
	if !ok || stored.OrganizationID != req.TenantInfo.OrgID {
		return nil, errortypes.NewNotFoundError("remittance not found")
	}
	out := copyRemittance(stored)
	if !req.IncludeLines {
		out.Lines = nil
	}
	return out, nil
}

func (f *fakeRemittances) FindByInboundDocument(
	_ context.Context,
	req *factoringports.FindRemittanceByDocumentRequest,
) (*factoring.Remittance, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, rem := range f.byID {
		if rem.InboundMessageID == req.InboundMessageID && rem.DocumentID == req.DocumentID {
			return copyRemittance(rem), nil
		}
	}
	return nil, errortypes.NewNotFoundError("remittance not found")
}

func (f *fakeRemittances) ReplaceLines(_ context.Context, rem *factoring.Remittance) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	stored := f.byID[rem.ID]
	stored.Lines = make([]*factoring.RemittanceLine, 0, len(rem.Lines))
	for _, line := range rem.Lines {
		if line.ID.IsNil() {
			line.ID = pulid.MustNew("facrml_")
		}
		copied := *line
		stored.Lines = append(stored.Lines, &copied)
	}
	return nil
}

func (f *fakeRemittances) UpdateLines(_ context.Context, lines []*factoring.RemittanceLine) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, line := range lines {
		stored := f.byID[line.RemittanceID]
		for idx, existing := range stored.Lines {
			if existing.ID == line.ID {
				copied := *line
				stored.Lines[idx] = &copied
			}
		}
	}
	return nil
}

func (f *fakeRemittances) ListCandidates(
	_ context.Context,
	req *factoringports.ListRemittanceCandidatesRequest,
) ([]*factoring.RemittanceCandidate, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(req.ItemIDs) == 0 {
		return f.candidates, nil
	}
	out := make([]*factoring.RemittanceCandidate, 0, len(req.ItemIDs))
	for _, candidate := range f.candidates {
		for _, id := range req.ItemIDs {
			if candidate.Item.ID == id {
				out = append(out, candidate)
			}
		}
	}
	return out, nil
}

func (f *fakeRemittances) GetProfile(
	context.Context,
	factoringports.GetRemittanceProfileRequest,
) (*factoring.RemittanceProfile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.profile == nil {
		return nil, errortypes.NewNotFoundError("profile not found")
	}
	return f.profile, nil
}

func (f *fakeRemittances) SaveProfile(_ context.Context, profile *factoring.RemittanceProfile) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.saved = append(f.saved, profile)
	f.profile = profile
	return nil
}

type fakeLedger struct {
	factoringports.LedgerService
	mu        sync.Mutex
	fundings  []*factoringports.RecordFundingRequest
	releases  []*factoringports.ReserveMovementRequest
	charges   []*factoringports.ReserveMovementRequest
	versions  map[pulid.ID]int64
	refuseFor pulid.ID
}

func (f *fakeLedger) bump(submissionID pulid.ID) *factoring.Submission {
	f.versions[submissionID]++
	return &factoring.Submission{ID: submissionID, Version: f.versions[submissionID]}
}

func (f *fakeLedger) PlanFunding(
	_ context.Context,
	req *factoringports.RecordFundingRequest,
	_ *permissionports.RequestActor,
) (*factoringports.FundingPlan, error) {
	plan := &factoringports.FundingPlan{}
	for _, item := range req.Items {
		if item.Outcome == factoring.FundingOutcomeRejected {
			plan.RejectCount++
			continue
		}
		plan.FundedCount++
		plan.AdvanceMinor += *item.AdvanceMinor
		plan.Movements = append(plan.Movements, &factoringports.PlannedMovement{ItemID: item.ItemID})
	}
	return plan, nil
}

func (f *fakeLedger) RecordFunding(
	_ context.Context,
	req *factoringports.RecordFundingRequest,
	_ *permissionports.RequestActor,
) (*factoringports.FundingResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if req.Version != f.versions[req.SubmissionID] {
		return nil, errortypes.NewConflictError("stale schedule")
	}
	if req.SubmissionID == f.refuseFor {
		return nil, errortypes.NewBusinessError("The period is closed")
	}
	f.fundings = append(f.fundings, req)
	entries := make([]*factoring.ReserveEntry, 0, len(req.Items))
	for _, item := range req.Items {
		if item.Outcome == factoring.FundingOutcomeFunded {
			entries = append(entries, &factoring.ReserveEntry{
				ID:               pulid.MustNew("facres_"),
				SubmissionItemID: item.ItemID,
				RemittanceLineID: item.RemittanceLineID,
			})
		}
	}
	return &factoringports.FundingResult{Submission: f.bump(req.SubmissionID), Entries: entries}, nil
}

func (f *fakeLedger) ReleaseReserve(
	_ context.Context,
	req *factoringports.ReserveMovementRequest,
	_ *permissionports.RequestActor,
) (*factoringports.ReserveMovementResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if req.Version != f.versions[req.SubmissionID] {
		return nil, errortypes.NewConflictError("stale schedule")
	}
	f.releases = append(f.releases, req)
	return &factoringports.ReserveMovementResult{
		Submission: f.bump(req.SubmissionID),
		Entry:      &factoring.ReserveEntry{ID: pulid.MustNew("facres_")},
	}, nil
}

func (f *fakeLedger) PlanRelease(
	_ context.Context,
	req *factoringports.ReserveMovementRequest,
	_ *permissionports.RequestActor,
) (*factoringports.ReserveMovementPlan, error) {
	return &factoringports.ReserveMovementPlan{Movement: &factoringports.PlannedMovement{
		ItemID: req.ItemID,
		Lines: []factoringports.PlannedLine{
			{Role: factoring.AccountRoleCash, Debit: *req.CashMinor},
		},
	}}, nil
}

func (f *fakeLedger) RecordChargeback(
	_ context.Context,
	req *factoringports.ReserveMovementRequest,
	_ *permissionports.RequestActor,
) (*factoringports.ReserveMovementResult, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.charges = append(f.charges, req)
	return &factoringports.ReserveMovementResult{
		Submission: f.bump(req.SubmissionID),
		Entry:      &factoring.ReserveEntry{ID: pulid.MustNew("facres_")},
	}, nil
}

type fakeRemitDocuments struct {
	documentports.InvoiceDocumentService
	docs     map[pulid.ID]*document.Document
	contents map[pulid.ID][]byte
}

func (f *fakeRemitDocuments) Get(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*document.Document, error) {
	doc, ok := f.docs[req.ID]
	if !ok {
		return nil, errortypes.NewNotFoundError("document not found")
	}
	return doc, nil
}

func (f *fakeRemitDocuments) GetDownloadContent(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*documentports.DocumentContent, error) {
	return &documentports.DocumentContent{
		Body: io.NopCloser(bytes.NewReader(f.contents[req.ID])),
	}, nil
}

type fakeRemitFactors struct {
	factoringcompanyports.FactoringCompanyRepository
	factor *factoringcompany.FactoringCompany
}

func (f *fakeRemitFactors) GetByID(
	context.Context,
	factoringcompanyports.GetFactoringCompanyByIDRequest,
) (*factoringcompany.FactoringCompany, error) {
	return f.factor, nil
}

func (f *fakeRemitFactors) FindByEmail(
	_ context.Context,
	req factoringcompanyports.FindFactoringCompaniesByEmailRequest,
) (*factoringcompanyports.FactoringCompaniesByEmail, error) {
	found := &factoringcompanyports.FactoringCompaniesByEmail{}
	if req.Address == f.factor.Email {
		found.Exact = append(found.Exact, f.factor)
	}
	return found, nil
}

type fakeRemitSubmissions struct {
	factoringports.SubmissionRepository
	ledger *fakeLedger
}

func (f *fakeRemitSubmissions) GetByIDs(
	_ context.Context,
	req factoringports.GetSubmissionsByIDsRequest,
) ([]*factoring.Submission, error) {
	out := make([]*factoring.Submission, 0, len(req.IDs))
	for _, id := range req.IDs {
		out = append(out, &factoring.Submission{ID: id, Version: f.ledger.versions[id]})
	}
	return out, nil
}

type remittanceHarness struct {
	svc         *RemittanceService
	remittances *fakeRemittances
	ledger      *fakeLedger
	documents   *fakeRemitDocuments
	events      *fakeEvents
	factor      *factoringcompany.FactoringCompany
	tenantInfo  pagination.TenantInfo
	actor       *permissionports.RequestActor
	subA, subB  pulid.ID
	pending     []*factoring.SubmissionItem
	funded      *factoring.SubmissionItem
}

func newRemittanceHarness(t *testing.T) *remittanceHarness {
	t.Helper()
	orgID, buID := pulid.MustNew("org_"), pulid.MustNew("bu_")
	h := &remittanceHarness{
		remittances: newFakeRemittances(),
		ledger:      &fakeLedger{versions: map[pulid.ID]int64{}},
		documents: &fakeRemitDocuments{
			docs:     map[pulid.ID]*document.Document{},
			contents: map[pulid.ID][]byte{},
		},
		events:     &fakeEvents{},
		tenantInfo: pagination.TenantInfo{OrgID: orgID, BuID: buID},
		actor: &permissionports.RequestActor{
			PrincipalType: permissionports.PrincipalTypeUser,
			PrincipalID:   pulid.MustNew("usr_"),
			UserID:        pulid.MustNew("usr_"),
		},
		subA: pulid.MustNew("facsub_"),
		subB: pulid.MustNew("facsub_"),
	}
	h.factor = &factoringcompany.FactoringCompany{
		ID:             pulid.MustNew("fac_"),
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Name:           "RTS Financial",
		Email:          "remit@rts.test",
		Status:         domaintypes.StatusActive,
	}
	h.ledger.versions[h.subA] = 3
	h.ledger.versions[h.subB] = 7
	sent := testNow - 86_400
	funded := testNow - 30*86_400
	for idx, number := range []string{"INV-100", "INV-200", "INV-300"} {
		sub := h.subA
		if idx == 2 {
			sub = h.subB
		}
		item := &factoring.SubmissionItem{
			ID:                   pulid.MustNew("facsit_"),
			SubmissionID:         sub,
			InvoiceNumber:        number,
			Status:               factoring.SubmissionItemStatusPending,
			FaceMinor:            int64(100_000 * (idx + 1)),
			ExpectedAdvanceMinor: int64(90_000 * (idx + 1)),
			ExpectedFeeMinor:     3_000,
			ExpectedReserveMinor: int64(7_000 * (idx + 1)),
		}
		h.pending = append(h.pending, item)
		h.remittances.candidates = append(h.remittances.candidates, &factoring.RemittanceCandidate{
			Item:     item,
			SentAt:   &sent,
			Recourse: true,
		})
	}
	h.funded = &factoring.SubmissionItem{
		ID:                  pulid.MustNew("facsit_"),
		SubmissionID:        h.subB,
		InvoiceNumber:       "INV-050",
		Status:              factoring.SubmissionItemStatusFunded,
		FundedOn:            &funded,
		AdvanceMinor:         new(int64(45_000)),
		FeeMinor:             new(int64(1_500)),
		ReserveMinor:         new(int64(3_500)),
		FaceMinor:            50_000,
		ExpectedReserveMinor: 3_500,
	}
	h.remittances.candidates = append(h.remittances.candidates, &factoring.RemittanceCandidate{
		Item:     h.funded,
		Recourse: true,
	})
	h.svc = NewRemittanceService(RemittanceParams{
		DB:          fakeTx{},
		Remittances: h.remittances,
		Submissions: &fakeRemitSubmissions{ledger: h.ledger},
		Ledger:      h.ledger,
		Factors:     &fakeRemitFactors{factor: h.factor},
		Documents:   h.documents,
		Events:      h.events,
		Logger:      zap.NewNop(),
	}).(*RemittanceService)
	h.svc.now = func() int64 { return testNow }
	return h
}

func (h *remittanceHarness) create(t *testing.T) *factoring.Remittance {
	t.Helper()
	rem, err := h.svc.Create(t.Context(), &factoringports.CreateRemittanceRequest{
		TenantInfo:         h.tenantInfo,
		FactoringCompanyID: h.factor.ID,
	}, h.actor)
	require.NoError(t, err)
	return rem
}

func (h *remittanceHarness) upload(rem *factoring.Remittance, name, content string) pulid.ID {
	id := pulid.MustNew("doc_")
	h.documents.docs[id] = &document.Document{
		ID:           id,
		OriginalName: name,
		FileSize:     int64(len(content)),
		ResourceType: factoring.RemittanceDocumentResourceType,
		ResourceID:   rem.ID.String(),
	}
	h.documents.contents[id] = []byte(content)
	return id
}

const standardRemittance = "Invoice Number,Invoice Amount,Advance Amount,Fee,Amount Paid,Funding Date,Type\n" +
	"INV-100,1000.00,900.00,30.00,,2027-01-15,Purchase\n" +
	"INV-200,2000.00,1800.00,,,2027-01-15,Purchase\n" +
	"INV-300,3000.00,,,,2027-01-15,Declined\n" +
	"MYSTERY,,123.45,,,2027-01-15,Purchase\n" +
	"INV-050,,,,35.00,2027-01-15,Reserve release\n" +
	"Total,6000.00,2823.45,,35.00,,\n"

func (h *remittanceHarness) staged(t *testing.T) *factoring.Remittance {
	t.Helper()
	rem := h.create(t)
	doc := h.upload(rem, "rts-2027-01-15.csv", standardRemittance)
	staged, err := h.svc.Stage(t.Context(), &factoringports.StageRemittanceRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
		Version:    rem.Version,
		DocumentID: doc,
	}, h.actor)
	require.NoError(t, err)
	return staged
}

func lineByInvoice(rem *factoring.Remittance, invoice string) *factoring.RemittanceLine {
	for _, line := range rem.Lines {
		if line.InvoiceNumber == invoice {
			return line
		}
	}
	return nil
}

func TestRemittanceStageReadsMatchesAndAnnounces(t *testing.T) {
	t.Parallel()
	h := newRemittanceHarness(t)

	rem := h.staged(t)
	assert.Equal(t, factoring.RemittanceStatusParsed, rem.Status)
	assert.Equal(t, factoring.RemittanceFormatCsv, rem.SourceFormat)
	require.Len(t, rem.Lines, 5, "the total row is not a line")
	assert.Equal(t, 4, rem.MatchedCount, "exact invoices and exact amounts match on their own")
	assert.Equal(t, 1, rem.ExceptionCount)

	assert.Equal(t, h.pending[0].ID, lineByInvoice(rem, "INV-100").SubmissionItemID)
	assert.Equal(t, factoring.RemittanceLineRejection, lineByInvoice(rem, "INV-300").Kind)
	assert.Equal(t, factoring.RemittanceLineReserveRelease, lineByInvoice(rem, "INV-050").Kind)
	assert.Equal(t, factoring.RemittanceLineMatched, lineByInvoice(rem, "INV-050").MatchStatus)
	assert.True(t, lineByInvoice(rem, "MYSTERY").MatchStatus.IsException())

	assert.Equal(t, []string{string(agent.EventFactoringRemittanceStaged)}, h.events.kinds())
	assert.Empty(t, h.remittances.saved, "a guessed layout is not saved as the factor's")
}

func TestRemittanceStageRefusesAFileUploadedForAnotherRemittance(t *testing.T) {
	t.Parallel()
	h := newRemittanceHarness(t)
	first := h.create(t)
	other := h.create(t)
	doc := h.upload(other, "rts.csv", standardRemittance)

	_, err := h.svc.Stage(t.Context(), &factoringports.StageRemittanceRequest{
		TenantInfo: h.tenantInfo,
		ID:         first.ID,
		Version:    first.Version,
		DocumentID: doc,
	}, h.actor)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not uploaded for this remittance")
}

func TestRemittanceNeedsMappingThenSavesTheFactorsLayout(t *testing.T) {
	t.Parallel()
	h := newRemittanceHarness(t)
	rem := h.create(t)
	odd := "Ref,Paid Out,Wire Dt\nINV-100,900.00,2027-01-15\n"
	doc := h.upload(rem, "odd.csv", odd)

	staged, err := h.svc.Stage(t.Context(), &factoringports.StageRemittanceRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
		Version:    rem.Version,
		DocumentID: doc,
	}, h.actor)
	require.NoError(t, err)
	assert.Equal(t, factoring.RemittanceStatusNeedsMapping, staged.Status)
	assert.Equal(t, []string{"Ref", "Paid Out", "Wire Dt"}, staged.Headers)

	_, err = h.svc.Stage(t.Context(), &factoringports.StageRemittanceRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
		Version:    staged.Version,
		DocumentID: doc,
		Mapping:    map[string]int{"invoiceNumber": 0},
	}, h.actor)
	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr, "a mapping without an amount is refused field by field")

	mapped, err := h.svc.Stage(t.Context(), &factoringports.StageRemittanceRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
		Version:    staged.Version,
		DocumentID: doc,
		Mapping:    map[string]int{"invoiceNumber": 0, "advance": 1, "date": 2},
	}, h.actor)
	require.NoError(t, err)
	assert.Equal(t, factoring.RemittanceStatusParsed, mapped.Status)
	assert.Equal(t, 1, mapped.MatchedCount)
	require.Len(t, h.remittances.saved, 1)
	assert.Equal(t, map[string]int{"invoiceNumber": 0, "advance": 1, "date": 2},
		h.remittances.saved[0].Mapping)

	next := h.create(t)
	nextDoc := h.upload(next, "odd-2.csv", odd)
	again, err := h.svc.Stage(t.Context(), &factoringports.StageRemittanceRequest{
		TenantInfo: h.tenantInfo,
		ID:         next.ID,
		Version:    next.Version,
		DocumentID: nextDoc,
	}, h.actor)
	require.NoError(t, err)
	assert.Equal(t, factoring.RemittanceStatusParsed, again.Status,
		"the factor's next file stages on the saved layout")
}

func TestRemittancePdfWithoutAModelFails(t *testing.T) {
	t.Parallel()
	h := newRemittanceHarness(t)
	rem := h.create(t)
	doc := h.upload(rem, "rts.pdf", "%PDF-1.7")

	staged, err := h.svc.Stage(t.Context(), &factoringports.StageRemittanceRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
		Version:    rem.Version,
		DocumentID: doc,
	}, h.actor)
	require.NoError(t, err)
	assert.Equal(t, factoring.RemittanceStatusFailed, staged.Status)
	assert.Equal(t, problemNoPdfModel, staged.Error)
}

func TestRemittanceResolveAndApplyBooksThroughTheLedger(t *testing.T) {
	t.Parallel()
	h := newRemittanceHarness(t)
	rem := h.staged(t)
	mystery := lineByInvoice(rem, "MYSTERY")

	_, err := h.svc.UpdateLine(t.Context(), &factoringports.UpdateRemittanceLineRequest{
		TenantInfo:   h.tenantInfo,
		RemittanceID: rem.ID,
		Version:      rem.Version,
		LineID:       mystery.ID,
		Action:       factoringports.RemittanceLineActionMatch,
		ItemID:       h.pending[0].ID,
	}, h.actor)
	require.Error(t, err, "an item another line matched cannot be matched twice")

	plan, err := h.svc.PlanApply(t.Context(), &factoringports.RemittanceVersionRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
		Version:    rem.Version,
	}, h.actor)
	require.NoError(t, err)
	assert.Equal(t, 2, plan.FundedCount)
	assert.Equal(t, 1, plan.RejectedCount)
	assert.Equal(t, 1, plan.ReleasedCount)
	assert.Equal(t, int64(3_500), plan.ReleasedMinor)
	assert.Equal(t, 1, plan.LeftOpenCount)
	assert.Empty(t, h.ledger.fundings, "a plan books nothing")

	applied, err := h.svc.Apply(t.Context(), &factoringports.RemittanceVersionRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
		Version:    rem.Version,
	}, h.actor)
	require.NoError(t, err)
	assert.Equal(t, factoring.RemittanceStatusPartiallyApplied, applied.Remittance.Status)
	assert.Equal(t, 4, applied.Remittance.AppliedCount)

	require.Len(t, h.ledger.fundings, 2, "one funding per schedule")
	first := h.ledger.fundings[0]
	assert.Equal(t, h.subA, first.SubmissionID)
	assert.Equal(t, int64(3), first.Version)
	require.Len(t, first.Items, 2)
	assert.Equal(t, int64(90_000), *first.Items[0].AdvanceMinor)
	assert.Equal(t, int64(3_000), *first.Items[0].FeeMinor)
	assert.Equal(t, lineByInvoice(rem, "INV-100").ID, first.Items[0].RemittanceLineID)
	second := h.ledger.fundings[1]
	assert.Equal(t, factoring.FundingOutcomeRejected, second.Items[0].Outcome)

	require.Len(t, h.ledger.releases, 1)
	release := h.ledger.releases[0]
	assert.Equal(t, int64(3_500), *release.CashMinor)
	assert.Equal(t, int64(8), release.Version,
		"the release reads the schedule's version after its funding was booked")
	assert.Equal(t, lineByInvoice(rem, "INV-050").ID, release.RemittanceLineID)

	stored, err := h.svc.Get(t.Context(), factoringports.GetRemittanceRequest{
		TenantInfo:   h.tenantInfo,
		ID:           rem.ID,
		IncludeLines: true,
	})
	require.NoError(t, err)
	assert.NotEmpty(t, lineByInvoice(stored, "INV-100").ReserveEntryID)
	assert.Equal(t, factoring.RemittanceLineApplied, lineByInvoice(stored, "INV-300").MatchStatus)

	open := lineByInvoice(stored, "MYSTERY")
	settled, err := h.svc.UpdateLine(t.Context(), &factoringports.UpdateRemittanceLineRequest{
		TenantInfo:   h.tenantInfo,
		RemittanceID: rem.ID,
		Version:      stored.Version,
		LineID:       open.ID,
		Action:       factoringports.RemittanceLineActionIgnore,
		Reason:       "Wire fee booked by hand",
	}, h.actor)
	require.NoError(t, err)
	assert.Equal(t, 0, settled.ExceptionCount)

	_, err = h.svc.Apply(t.Context(), &factoringports.RemittanceVersionRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
		Version:    settled.Version,
	}, h.actor)
	require.Error(t, err, "nothing is left to book")
}

func TestRemittanceApplyNeedsAPersonAndRollsBackOnALedgerRefusal(t *testing.T) {
	t.Parallel()
	h := newRemittanceHarness(t)
	rem := h.staged(t)

	_, err := h.svc.Apply(t.Context(), &factoringports.RemittanceVersionRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
		Version:    rem.Version,
	}, &permissionports.RequestActor{
		PrincipalType: permissionports.PrincipalTypeAPIKey,
		PrincipalID:   pulid.MustNew("ak_"),
	})
	require.Error(t, err)

	h.ledger.refuseFor = h.subB
	_, err = h.svc.Apply(t.Context(), &factoringports.RemittanceVersionRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
		Version:    rem.Version,
	}, h.actor)
	require.Error(t, err)
	stored, getErr := h.svc.Get(t.Context(), factoringports.GetRemittanceRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
	})
	require.NoError(t, getErr)
	assert.Equal(t, factoring.RemittanceStatusParsed, stored.Status)
}

func TestRemittanceIngestEmailFindsTheFactorAndIsIdempotent(t *testing.T) {
	t.Parallel()
	h := newRemittanceHarness(t)
	messageID := pulid.MustNew("inmsg_")
	docID := pulid.MustNew("doc_")
	h.documents.docs[docID] = &document.Document{
		ID:           docID,
		OriginalName: "remit.csv",
		FileSize:     int64(len(standardRemittance)),
		ResourceType: inboundmessage.AttachmentResourceType,
		ResourceID:   messageID.String(),
	}
	h.documents.contents[docID] = []byte(standardRemittance)
	logo := pulid.MustNew("doc_")
	req := &factoringports.IngestRemittanceEmailRequest{
		TenantInfo:       h.tenantInfo,
		InboundMessageID: messageID,
		FromAddress:      "someone@else.test",
		ReceivedAt:       testNow,
		Attachments: []factoringports.RemittanceEmailAttachment{
			{DocumentID: logo, FileName: "logo.png", ContentType: "image/png"},
			{DocumentID: docID, FileName: "remit.csv", ContentType: "text/csv"},
		},
	}

	unknown, err := h.svc.IngestEmail(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, intakeNoFactor, unknown.Problem)
	assert.Empty(t, unknown.RemittanceIDs)

	req.FromAddress = h.factor.Email
	ingested, err := h.svc.IngestEmail(t.Context(), req)
	require.NoError(t, err)
	assert.Empty(t, ingested.Problem)
	require.Len(t, ingested.RemittanceIDs, 1, "the logo is not a remittance")
	rem, err := h.svc.Get(t.Context(), factoringports.GetRemittanceRequest{
		TenantInfo: h.tenantInfo,
		ID:         ingested.RemittanceIDs[0],
	})
	require.NoError(t, err)
	assert.Equal(t, factoring.RemittanceSourceEmail, rem.Source)
	assert.Equal(t, factoring.RemittanceStatusParsed, rem.Status)
	assert.Equal(t, messageID, rem.InboundMessageID)

	again, err := h.svc.IngestEmail(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, ingested.RemittanceIDs, again.RemittanceIDs, "a retried message opens nothing twice")
}

type fakeRemitContent struct {
	documentports.DocumentContentService
	mu      sync.Mutex
	content *documentcontent.Content
}

func (f *fakeRemitContent) GetContent(
	context.Context,
	pulid.ID,
	pagination.TenantInfo,
) (*documentcontent.Content, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.content == nil {
		return nil, errortypes.NewNotFoundError("content not found")
	}
	return f.content, nil
}

type fakeRemitCompletion struct {
	agentports.CompletionService
	reply string
	req   *agentports.StructuredCompletionRequest
}

func (f *fakeRemitCompletion) CompleteStructured(
	_ context.Context,
	req *agentports.StructuredCompletionRequest,
) (*agentports.StructuredCompletionResult, error) {
	f.req = req
	return &agentports.StructuredCompletionResult{Text: f.reply}, nil
}

type fakeStageStarter struct {
	started []pulid.ID
}

func (f *fakeStageStarter) StartStage(
	_ context.Context,
	_ pagination.TenantInfo,
	rem *factoring.Remittance,
) {
	f.started = append(f.started, rem.ID)
}

func TestRemittancePdfWaitsForItsTextThenReadsItWithTheModel(t *testing.T) {
	t.Parallel()
	h := newRemittanceHarness(t)
	content := &fakeRemitContent{}
	completion := &fakeRemitCompletion{reply: `{"lines":[` +
		`{"invoiceNumber":"INV-100","factorReference":"","debtor":"Acme","kind":"Funding",` +
		`"face":"1,000.00","advance":"900.00","fee":"30.00","reserve":"","cash":"","date":"01/15/2027"},` +
		`{"invoiceNumber":"INV-050","factorReference":"","debtor":"","kind":"ReserveRelease",` +
		`"face":"","advance":"","fee":"","reserve":"","cash":"35.00","date":"2027-01-15"}]}`}
	starter := &fakeStageStarter{}
	h.svc.content = content
	h.svc.completion = completion
	h.svc.starter = starter

	rem := h.create(t)
	doc := h.upload(rem, "rts.pdf", "%PDF-1.7")
	waiting, err := h.svc.Stage(t.Context(), &factoringports.StageRemittanceRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
		Version:    rem.Version,
		DocumentID: doc,
	}, h.actor)
	require.NoError(t, err)
	assert.Equal(t, factoring.RemittanceStatusPending, waiting.Status)
	assert.True(t, waiting.AwaitsReading())
	assert.Equal(t, []pulid.ID{rem.ID}, starter.started)

	still, err := h.svc.StagePending(t.Context(), &factoringports.RemittanceVersionRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
	}, false)
	require.NoError(t, err)
	assert.True(t, still.Waiting)

	content.content = &documentcontent.Content{
		Status:      documentcontent.StatusIndexed,
		ContentText: "RTS Financial purchase report ...",
	}
	read, err := h.svc.StagePending(t.Context(), &factoringports.RemittanceVersionRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
	}, false)
	require.NoError(t, err)
	assert.False(t, read.Waiting)
	assert.Equal(t, factoring.RemittanceStatusParsed, read.Remittance.Status)
	assert.Equal(t, 2, read.Remittance.MatchedCount)
	require.NotNil(t, completion.req)
	assert.Equal(t, aiprovider.TaskRemittanceExtraction, completion.req.Task)
	assert.Equal(t, aiusage.FeatureFactoringRemittanceExtract, completion.req.Attribution.Feature)
	assert.False(t, completion.req.Context.Sections[0].Trusted, "the report is outside content")
}

func TestRemittancePdfStillUnreadWhenTheWaitEndsFails(t *testing.T) {
	t.Parallel()
	h := newRemittanceHarness(t)
	h.svc.content = &fakeRemitContent{content: &documentcontent.Content{
		Status: documentcontent.StatusExtracting,
	}}
	h.svc.completion = &fakeRemitCompletion{}
	h.svc.starter = &fakeStageStarter{}
	rem := h.create(t)
	doc := h.upload(rem, "rts.pdf", "%PDF-1.7")
	_, err := h.svc.Stage(t.Context(), &factoringports.StageRemittanceRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
		Version:    rem.Version,
		DocumentID: doc,
	}, h.actor)
	require.NoError(t, err)

	gaveUp, err := h.svc.StagePending(t.Context(), &factoringports.RemittanceVersionRequest{
		TenantInfo: h.tenantInfo,
		ID:         rem.ID,
	}, true)
	require.NoError(t, err)
	assert.Equal(t, factoring.RemittanceStatusFailed, gaveUp.Remittance.Status)
	assert.Equal(t, problemPdfSlow, gaveUp.Remittance.Error)
}
