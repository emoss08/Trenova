package factoringservice

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringreadiness"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoringsubmission"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/documenttemplate"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testNow = int64(1_800_000_000)

type submissionHarness struct {
	svc        *SubmissionService
	tenant     pagination.TenantInfo
	factor     *factoringcompany.FactoringCompany
	agreement  *factoring.Agreement
	repo       *fakeSubmissions
	transports *fakeTransports
	states     *fakeStates
	invoices   *fakeInvoices
	templates  *fakeTemplates
	store      *fakeDocumentStore
	sender     *fakeSender
	starter    *fakeStarter
	requester  *fakeRequester
	events     *fakeEvents
	customerID pulid.ID
	actor      *permissionports.RequestActor
}

func newSubmissionHarness(t *testing.T) *submissionHarness {
	t.Helper()
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	factor := &factoringcompany.FactoringCompany{
		ID:     pulid.MustNew("fac_"),
		Name:   "Lone Star Funding",
		Status: domaintypes.StatusActive,
	}
	to := 30
	agreement := &factoring.Agreement{
		ID:                 pulid.MustNew("facagr_"),
		FactoringCompanyID: factor.ID,
		Status:             factoring.AgreementStatusActive,
		AdvanceRateBP:      9000,
		ReserveRateBP:      1000,
		Recourse:           factoring.RecourseFull,
		FeeTiers: []*factoring.FeeTier{
			{DaysFrom: 0, DaysTo: &to, RateBP: 300},
			{DaysFrom: 30, RateBP: 500},
		},
	}
	customerID := pulid.MustNew("cus_")
	store := &fakeDocumentStore{stored: map[pulid.ID]*documentports.StoreDocumentRequest{}}
	h := &submissionHarness{
		tenant:    tenant,
		factor:    factor,
		agreement: agreement,
		repo:      newFakeSubmissions(),
		transports: &fakeTransports{entity: &factoringsubmission.Transport{
			ID:                 pulid.MustNew("factrn_"),
			FactoringCompanyID: factor.ID,
			Method:             factoringsubmission.TransportMethodEmail,
			EmailRecipients:    []string{"schedules@lonestar.test"},
			AttachPackets:      true,
		}},
		states:     &fakeStates{states: map[pulid.ID]*factoringreadiness.InvoiceState{}},
		invoices:   &fakeInvoices{byID: map[pulid.ID]*invoice.Invoice{}},
		templates:  &fakeTemplates{},
		store:      store,
		sender:     &fakeSender{method: factoringsubmission.TransportMethodEmail},
		starter:    &fakeStarter{},
		requester:  &fakeRequester{},
		events:     &fakeEvents{},
		customerID: customerID,
		actor: &permissionports.RequestActor{
			PrincipalType: permissionports.PrincipalTypeUser,
			PrincipalID:   pulid.MustNew("usr_"),
			UserID:        pulid.MustNew("usr_"),
		},
	}
	h.svc = NewSubmissionService(SubmissionParams{
		Repo:             h.repo,
		Transports:       h.transports,
		States:           h.states,
		Agreements:       &fakeAgreements{agreement: agreement},
		Invoices:         h.invoices,
		InvoiceDocuments: &fakeDocumentReader{store: store},
		Documents:        fakeDocuments{},
		Store:            store,
		Imaging:          fakeImaging(),
		Factors:          &fakeFactors{factor: factor},
		Customers: &fakeCustomers{customers: []*customer.Customer{
			{ID: customerID, Name: "Halstead Grocery", Code: "=HALSTEAD"},
		}},
		Organizations: fakeOrganizations{},
		Templates:     h.templates,
		Sequences:     &fakeSequences{},
		Cipher:        fakeCipher{},
		Senders:       []factoringsubmissionports.FactoringTransport{h.sender},
		Starter:       h.starter,
		Requester:     h.requester,
		Events:        h.events,
		Logger:        zap.NewNop(),
	}).(*SubmissionService)
	h.svc.now = func() int64 { return testNow }
	return h
}

func (h *submissionHarness) readyInvoice(number string, totalMinor int64) *invoice.Invoice {
	due := testNow + 10*86400
	inv := &invoice.Invoice{
		ID:                 pulid.MustNew("inv_"),
		OrganizationID:     h.tenant.OrgID,
		BusinessUnitID:     h.tenant.BuID,
		CustomerID:         h.customerID,
		Number:             number,
		Status:             invoice.StatusPosted,
		BillType:           billingqueue.BillTypeInvoice,
		CurrencyCode:       "USD",
		InvoiceDate:        testNow - 86400,
		DueDate:            &due,
		TotalAmountMinor:   totalMinor,
		FactoringCompanyID: &h.factor.ID,
		BillToName:         "Halstead Grocery",
		ShipmentProNumber:  "5501829",
		PDFDocumentID:      pulid.MustNew("doc_"),
	}
	h.invoices.byID[inv.ID] = inv
	since := testNow - 3600
	h.states.states[inv.ID] = &factoringreadiness.InvoiceState{
		InvoiceID:          inv.ID,
		FactoringCompanyID: &h.factor.ID,
		Verdict:            factoringreadiness.VerdictReady,
		ReadySince:         &since,
	}
	return inv
}

func (h *submissionHarness) request(
	ids ...pulid.ID,
) *factoringsubmissionports.PlanSubmissionRequest {
	return &factoringsubmissionports.PlanSubmissionRequest{
		TenantInfo:         h.tenant,
		FactoringCompanyID: h.factor.ID,
		InvoiceIDs:         ids,
	}
}

func TestPlanSubmissionComputesExpectedFunding(t *testing.T) {
	t.Parallel()
	h := newSubmissionHarness(t)
	first := h.readyInvoice("INV-1", 100000)
	second := h.readyInvoice("INV-2", 50000)

	plan, err := h.svc.Plan(t.Context(), h.request(first.ID, second.ID))
	require.NoError(t, err)
	assert.Equal(t, 2, plan.InvoiceCount)
	assert.Equal(t, int64(150000), plan.FaceMinor)
	assert.Equal(t, int64(135000), plan.ExpectedAdvanceMinor)
	assert.Equal(t, int64(4500), plan.ExpectedFeeMinor)
	assert.Equal(t, int64(15000), plan.ExpectedReserveMinor)
	assert.Equal(t, "schedules@lonestar.test", plan.TransportTarget)
	assert.Equal(t, "Halstead Grocery", plan.Items[0].CustomerName)
	assert.Equal(t, 10, plan.Items[0].Funding.Days)

	reversed, err := h.svc.Plan(t.Context(), h.request(second.ID, first.ID, first.ID))
	require.NoError(t, err)
	assert.Equal(t, plan.IdempotencyKey, reversed.IdempotencyKey)
}

func TestPlanSubmissionRefusesInvoicesThatAreNotReady(t *testing.T) {
	t.Parallel()
	h := newSubmissionHarness(t)
	ready := h.readyInvoice("INV-1", 100000)
	notReady := h.readyInvoice("INV-2", 1000)
	h.states.states[notReady.ID].Verdict = factoringreadiness.VerdictNotReady
	h.states.states[notReady.ID].ReadySince = nil
	submitted := h.readyInvoice("INV-3", 1000)
	h.states.states[submitted.ID].Verdict = factoringreadiness.VerdictSubmitted
	other := h.readyInvoice("INV-4", 1000)
	otherFactor := pulid.MustNew("fac_")
	other.FactoringCompanyID = &otherFactor

	_, err := h.svc.Plan(t.Context(), h.request(ready.ID, notReady.ID, submitted.ID, other.ID))
	require.Error(t, err)
	assert.True(t, errortypes.IsBusinessError(err))
	for _, want := range []string{"INV-2 is not ready", "INV-3 is already on a schedule", "INV-4 is not assigned"} {
		assert.Contains(t, err.Error(), want)
	}
	assert.NotContains(t, err.Error(), "INV-1")

	_, err = h.svc.Plan(t.Context(), h.request())
	require.Error(t, err)
}

func TestPlanSubmissionNeedsTransportAndAgreement(t *testing.T) {
	t.Parallel()
	h := newSubmissionHarness(t)
	inv := h.readyInvoice("INV-1", 1000)
	h.transports.entity = nil
	_, err := h.svc.Plan(t.Context(), h.request(inv.ID))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "Set up how schedules reach")

	h = newSubmissionHarness(t)
	inv = h.readyInvoice("INV-1", 1000)
	h.svc.agreements = &fakeAgreements{}
	_, err = h.svc.Plan(t.Context(), h.request(inv.ID))
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no agreement in force")
}

func TestSubmitStoresAndStartsSendingOnce(t *testing.T) {
	t.Parallel()
	h := newSubmissionHarness(t)
	inv := h.readyInvoice("INV-1", 100000)

	created, err := h.svc.Submit(t.Context(), h.request(inv.ID), h.actor)
	require.NoError(t, err)
	assert.Equal(t, factoringsubmission.SubmissionStatusSending, created.Status)
	assert.Equal(t, "FS1", created.Number)
	assert.Equal(t, h.actor.UserID, created.SubmittedByID)
	require.Len(t, h.starter.started, 1)
	require.Len(t, h.requester.targets, 1)
	assert.Equal(t, []pulid.ID{inv.ID}, h.requester.targets[0].InvoiceIDs)

	again, err := h.svc.Submit(t.Context(), h.request(inv.ID), h.actor)
	require.NoError(t, err)
	assert.Equal(t, created.ID, again.ID)
	assert.Equal(t, 1, h.repo.created)
	assert.Len(t, h.starter.started, 1)

	_, err = h.svc.Submit(t.Context(), h.request(inv.ID), nil)
	require.Error(t, err)
}

func TestPrepareAndDeliverSendsTheScheduleAndPackets(t *testing.T) {
	t.Parallel()
	h := newSubmissionHarness(t)
	inv := h.readyInvoice("INV-1", 100000)
	created, err := h.svc.Submit(t.Context(), h.request(inv.ID), h.actor)
	require.NoError(t, err)
	req := &factoringsubmissionports.DeliverSubmissionRequest{
		TenantInfo:   h.tenant,
		SubmissionID: created.ID,
		Group:        1,
		Attempt:      2,
	}

	require.NoError(t, h.svc.PrepareDocuments(t.Context(), req))
	require.NotNil(t, h.repo.docs)
	assert.Len(t, h.store.stored, 3)
	csvDoc := h.store.stored[h.repo.docs.ScheduleCSVDocumentID]
	require.NotNil(t, csvDoc)
	assert.Contains(t, string(csvDoc.Content), "'=HALSTEAD")
	assert.Equal(t, factoringsubmission.SubmissionDocumentResourceType, csvDoc.ResourceType)
	assert.Equal(t, permissionports.PrincipalTypeSystem, csvDoc.Actor.PrincipalType)

	require.NoError(t, h.svc.PrepareDocuments(t.Context(), req))
	assert.Len(t, h.store.stored, 3)

	result, err := h.svc.Deliver(t.Context(), req)
	require.NoError(t, err)
	assert.Equal(t, factoringsubmission.SubmissionStatusSent, result.Status)
	require.Len(t, h.sender.requests, 1)
	sent := h.sender.requests[0]
	assert.Equal(t, SendKey(created), sent.IdempotencyKey)
	names := make([]string, 0, len(sent.Files))
	for _, file := range sent.Files {
		names = append(names, file.Name)
	}
	assert.Equal(t, []string{"FS1.pdf", "FS1.csv", "INV-1.pdf"}, names)
	require.Len(t, h.templates.messages, 1)
	message := h.templates.messages[0].(*documenttemplate.FactoringScheduleEmailContext)
	assert.Equal(t, "Lone Star Funding", message.FactorName)

	stored, err := h.svc.Get(
		t.Context(),
		factoringsubmissionports.GetSubmissionRequest{ID: created.ID, TenantInfo: h.tenant},
	)
	require.NoError(t, err)
	assert.Equal(t, factoringsubmission.SubmissionStatusSent, stored.Status)
	assert.Equal(t, 2, stored.Attempts)
	assert.Equal(t, "ref-1", stored.TransportReference)
	assert.Equal(t, []string{string(agenttrigger.EventFactoringSubmissionSent)}, h.events.kinds())

	skipped, err := h.svc.Deliver(t.Context(), req)
	require.NoError(t, err)
	assert.True(t, skipped.Skipped)
	assert.Len(t, h.sender.requests, 1)
}

func TestDeliverWithoutPacketsSendsOnlyTheSchedule(t *testing.T) {
	t.Parallel()
	h := newSubmissionHarness(t)
	h.transports.entity.AttachPackets = false
	inv := h.readyInvoice("INV-1", 100000)
	created, err := h.svc.Submit(t.Context(), h.request(inv.ID), h.actor)
	require.NoError(t, err)
	req := &factoringsubmissionports.DeliverSubmissionRequest{
		TenantInfo:   h.tenant,
		SubmissionID: created.ID,
		Group:        1,
	}
	require.NoError(t, h.svc.PrepareDocuments(t.Context(), req))
	_, err = h.svc.Deliver(t.Context(), req)
	require.NoError(t, err)
	assert.Len(t, h.sender.requests[0].Files, 2)
}

func TestFailedScheduleCanBeRetriedOrCancelled(t *testing.T) {
	t.Parallel()
	h := newSubmissionHarness(t)
	inv := h.readyInvoice("INV-1", 100000)
	created, err := h.svc.Submit(t.Context(), h.request(inv.ID), h.actor)
	require.NoError(t, err)

	require.NoError(
		t,
		h.svc.MarkFailed(t.Context(), &factoringsubmissionports.FailSubmissionRequest{
			TenantInfo: h.tenant, SubmissionID: created.ID, Group: 2, Cause: "stale", Attempts: 6,
		}),
	)
	current, err := h.svc.Get(
		t.Context(),
		factoringsubmissionports.GetSubmissionRequest{ID: created.ID, TenantInfo: h.tenant},
	)
	require.NoError(t, err)
	assert.Equal(t, factoringsubmission.SubmissionStatusSending, current.Status)

	require.NoError(
		t,
		h.svc.MarkFailed(t.Context(), &factoringsubmissionports.FailSubmissionRequest{
			TenantInfo: h.tenant, SubmissionID: created.ID, Group: 1, Cause: "connection refused", Attempts: 6,
		}),
	)
	current, err = h.svc.Get(
		t.Context(),
		factoringsubmissionports.GetSubmissionRequest{ID: created.ID, TenantInfo: h.tenant},
	)
	require.NoError(t, err)
	assert.Equal(t, factoringsubmission.SubmissionStatusFailed, current.Status)
	assert.Equal(t, "connection refused", current.LastError)
	assert.Equal(t, []string{string(agenttrigger.EventFactoringSubmissionFailed)}, h.events.kinds())

	_, err = h.svc.Retry(t.Context(), &factoringsubmissionports.SubmissionActionRequest{
		TenantInfo: h.tenant, SubmissionID: created.ID, Version: current.Version - 1,
	}, h.actor)
	require.Error(t, err)
	assert.True(t, errortypes.IsConflictError(err))

	h.transports.entity.EmailRecipients = []string{"new@lonestar.test"}
	retried, err := h.svc.Retry(t.Context(), &factoringsubmissionports.SubmissionActionRequest{
		TenantInfo: h.tenant, SubmissionID: created.ID, Version: current.Version,
	}, h.actor)
	require.NoError(t, err)
	assert.Equal(t, factoringsubmission.SubmissionStatusSending, retried.Status)
	assert.Equal(t, 2, retried.SendAttemptGroup)
	assert.Equal(t, "new@lonestar.test", retried.TransportTarget)
	require.Len(t, h.starter.started, 2)
	assert.Equal(t, 2, h.starter.started[1].SendAttemptGroup)

	require.NoError(
		t,
		h.svc.MarkFailed(t.Context(), &factoringsubmissionports.FailSubmissionRequest{
			TenantInfo: h.tenant, SubmissionID: created.ID, Group: 2, Cause: "again", Attempts: 6,
		}),
	)
	current, err = h.svc.Get(
		t.Context(),
		factoringsubmissionports.GetSubmissionRequest{ID: created.ID, TenantInfo: h.tenant},
	)
	require.NoError(t, err)

	_, err = h.svc.Cancel(t.Context(), &factoringsubmissionports.SubmissionActionRequest{
		TenantInfo: h.tenant, SubmissionID: created.ID, Version: current.Version,
	}, h.actor)
	require.Error(t, err)

	cancelled, err := h.svc.Cancel(t.Context(), &factoringsubmissionports.SubmissionActionRequest{
		TenantInfo: h.tenant, SubmissionID: created.ID, Version: current.Version, Reason: "Wrong factor",
	}, h.actor)
	require.NoError(t, err)
	assert.Equal(t, factoringsubmission.SubmissionStatusCancelled, cancelled.Status)
	assert.Equal(t, factoringsubmission.SubmissionItemStatusReleased, cancelled.Items[0].Status)
	assert.Len(t, h.requester.targets, 2)
}

func TestScheduleCSVKeepsCellsAsText(t *testing.T) {
	t.Parallel()
	sub := &factoringsubmission.Submission{
		Number:       "FS1",
		CurrencyCode: "USD",
		Items: []*factoringsubmission.SubmissionItem{{
			InvoiceID:     pulid.MustNew("inv_"),
			InvoiceNumber: "+INV-1",
			FaceMinor:     123456,
		}},
	}
	content, err := scheduleCSV(sub, &scheduleSources{
		org:       fakeOrganizationValue(),
		invoices:  nil,
		customers: nil,
	})
	require.NoError(t, err)
	lines := strings.Split(strings.TrimSpace(string(content)), "\n")
	require.Len(t, lines, 2)
	assert.True(t, strings.HasPrefix(lines[0], "Schedule,Invoice"))
	assert.Contains(t, lines[1], "'+INV-1")
	assert.Contains(t, lines[1], "1234.56")
}
