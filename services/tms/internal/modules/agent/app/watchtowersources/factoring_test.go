package watchtowersources

import (
	"context"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const factoringNow = int64(1_790_000_000)

func factoringTenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
}

func readyState(tenant pagination.TenantInfo, readyFor int64) *factoring.InvoiceState {
	since := factoringNow - readyFor
	return &factoring.InvoiceState{
		InvoiceID:        pulid.MustNew("inv_"),
		OrganizationID:   tenant.OrgID,
		BusinessUnitID:   tenant.BuID,
		CustomerID:       pulid.MustNew("cus_"),
		Verdict:          factoring.VerdictReady,
		OpenBalanceMinor: 125_050,
		ReadySince:       &since,
	}
}

type fakeFactoringStates struct {
	factoringports.InvoiceStateRepository

	ready []*factoring.InvoiceState
	err   error
	req   factoringports.ListReadyUnsubmittedRequest
}

func (f *fakeFactoringStates) ListReadyUnsubmitted(
	_ context.Context,
	req factoringports.ListReadyUnsubmittedRequest,
) ([]*factoring.InvoiceState, error) {
	f.req = req
	return f.ready, f.err
}

type fakeFactoringReader struct {
	labels      map[pulid.ID]*factoringports.InvoiceLabel
	missing     []*factoringports.CustomerMissingNoa
	expiring    []*factoringports.ExpiringNoa
	labelErr    error
	missingErr  error
	expiringErr error
	labelReq    *factoringports.ListInvoiceLabelsRequest
	missingReq  *factoringports.ListCustomersMissingNoaRequest
	expiringReq *factoringports.ListNoasExpiringSoonRequest
}

func (f *fakeFactoringReader) ListInvoiceLabels(
	_ context.Context,
	req *factoringports.ListInvoiceLabelsRequest,
) (map[pulid.ID]*factoringports.InvoiceLabel, error) {
	f.labelReq = req
	return f.labels, f.labelErr
}

func (f *fakeFactoringReader) ListCustomersMissingNoa(
	_ context.Context,
	req *factoringports.ListCustomersMissingNoaRequest,
) ([]*factoringports.CustomerMissingNoa, error) {
	f.missingReq = req
	return f.missing, f.missingErr
}

func (f *fakeFactoringReader) ListNoasExpiringSoon(
	_ context.Context,
	req *factoringports.ListNoasExpiringSoonRequest,
) ([]*factoringports.ExpiringNoa, error) {
	f.expiringReq = req
	return f.expiring, f.expiringErr
}

func newTestFactoringSource(
	states *fakeFactoringStates,
	reader *fakeFactoringReader,
) *FactoringSource {
	source := NewFactoringSource(states, reader).(*FactoringSource)
	source.now = func() int64 { return factoringNow }
	return source
}

func assertStorable(t *testing.T, input watchtowerports.WatchtowerItemInput) {
	t.Helper()
	item := &watchtower.Item{
		OrganizationID: input.TenantInfo.OrgID,
		BusinessUnitID: input.TenantInfo.BuID,
		SourceKind:     input.SourceKind,
		SourceID:       input.SourceID,
		Severity:       input.Severity,
		Title:          input.Title,
		Summary:        input.Summary,
		SubjectType:    input.SubjectType,
		SubjectID:      input.SubjectID,
		EventKind:      input.EventKind,
		Path:           input.Path,
		OccurredAt:     input.OccurredAt,
	}
	multiErr := errortypes.NewMultiError()
	item.Validate(multiErr)
	assert.False(t, multiErr.HasErrors(), "%v", multiErr)
}

func TestFactoringKindReadsAsInvoices(t *testing.T) {
	t.Parallel()

	assert.True(t, watchtower.SourceFactoring.IsValid())
	assert.Equal(t, permission.ResourceInvoice, watchtower.SourceFactoring.ReadResource())
	assert.Equal(t, "Factoring", watchtower.SourceFactoring.Label())
	assert.False(t, watchtower.SourceFactoring.OverseesAgents())
	assert.Contains(t, watchtower.AllSourceKinds(), watchtower.SourceFactoring)
}

func TestDescribeFactoringReadyUnsubmitted(t *testing.T) {
	t.Parallel()

	tenant := factoringTenant()
	state := readyState(tenant, factoring.ReadyUnsubmittedSeconds+3600)
	label := &factoringports.InvoiceLabel{
		InvoiceID:    state.InvoiceID,
		Number:       "INV-1042",
		CurrencyCode: "usd",
		CustomerName: "Acme Foods",
		FactorName:   "Summit Capital",
	}

	item, open := DescribeFactoringReadyUnsubmitted(tenant, state, label, factoringNow)
	require.True(t, open)
	assert.Equal(t, watchtower.SourceFactoring, item.SourceKind)
	assert.Equal(t, "ready:"+state.InvoiceID.String(), item.SourceID)
	assert.Equal(t, FactoringReadySourceID(state.InvoiceID), item.SourceID)
	assert.Equal(t, watchtower.SeverityWarning, item.Severity)
	assert.Equal(t, "Invoice INV-1042 has been ready to factor for over a day", item.Title)
	assert.Contains(t, item.Summary, "1250.50 USD")
	assert.Contains(t, item.Summary, "Acme Foods")
	assert.Contains(t, item.Summary, "Submit it to Summit Capital")
	assert.Equal(t, agent.SubjectInvoice, item.SubjectType)
	assert.Equal(t, state.InvoiceID, item.SubjectID)
	assert.Equal(t, agent.EventFactoringInvoiceReady, item.EventKind)
	assert.Equal(t, *state.ReadySince+factoring.ReadyUnsubmittedSeconds, item.OccurredAt)
	assert.Equal(t, tenant, item.TenantInfo)
	assert.Contains(t, item.Path, "/billing/invoices")
	assert.Contains(t, item.Path, state.InvoiceID.String())
	assertStorable(t, item)
}

func TestDescribeFactoringReadyUnsubmittedGoesCriticalAfterThreeDays(t *testing.T) {
	t.Parallel()

	tenant := factoringTenant()
	state := readyState(tenant, 4*timeutils.SecondsPerDay)

	item, open := DescribeFactoringReadyUnsubmitted(tenant, state, nil, factoringNow)
	require.True(t, open)
	assert.Equal(t, watchtower.SeverityCritical, item.Severity)
	assert.Equal(t, "An invoice has been ready to factor for 4 days", item.Title)
	assert.Contains(t, item.Summary, "Submit it to the factor")
	assert.Contains(t, item.Summary, "1250.50 USD")
}

func TestDescribeFactoringReadyUnsubmittedClearsWhenNotLate(t *testing.T) {
	t.Parallel()

	tenant := factoringTenant()

	fresh := readyState(tenant, factoring.ReadyUnsubmittedSeconds-60)
	_, open := DescribeFactoringReadyUnsubmitted(tenant, fresh, nil, factoringNow)
	assert.False(t, open, "a day has not passed yet")

	notReady := readyState(tenant, 2*factoring.ReadyUnsubmittedSeconds)
	notReady.Verdict = factoring.VerdictNotReady
	_, open = DescribeFactoringReadyUnsubmitted(tenant, notReady, nil, factoringNow)
	assert.False(t, open, "an invoice that stopped being ready clears")

	submitted := readyState(tenant, 2*factoring.ReadyUnsubmittedSeconds)
	submitted.Verdict = factoring.VerdictExcluded
	_, open = DescribeFactoringReadyUnsubmitted(tenant, submitted, nil, factoringNow)
	assert.False(t, open, "a factored invoice is excluded and clears")

	undated := readyState(tenant, 2*factoring.ReadyUnsubmittedSeconds)
	undated.ReadySince = nil
	_, open = DescribeFactoringReadyUnsubmitted(tenant, undated, nil, factoringNow)
	assert.False(t, open)

	_, open = DescribeFactoringReadyUnsubmitted(tenant, nil, nil, factoringNow)
	assert.False(t, open)
}

func missingNoa() *factoringports.CustomerMissingNoa {
	return &factoringports.CustomerMissingNoa{
		CustomerID:         pulid.MustNew("cus_"),
		CustomerCode:       "ACME",
		CustomerName:       "Acme Foods",
		FactoringCompanyID: pulid.MustNew("fac_"),
		FactorName:         "Summit Capital",
		ProfileUpdatedAt:   factoringNow - 3600,
	}
}

func TestDescribeFactoringMissingNoaWithNoNotice(t *testing.T) {
	t.Parallel()

	tenant := factoringTenant()
	row := missingNoa()

	item, open := DescribeFactoringMissingNoa(tenant, row, factoringNow)
	require.True(t, open)
	assert.Equal(t, watchtower.SourceFactoring, item.SourceKind)
	assert.Equal(
		t,
		"noa-missing:"+row.CustomerID.String()+":"+row.FactoringCompanyID.String(),
		item.SourceID,
	)
	assert.Equal(t, watchtower.SeverityWarning, item.Severity)
	assert.Equal(t, "Acme Foods has no signed notice of assignment for Summit Capital", item.Title)
	assert.Contains(t, item.Summary, "no notice has been drafted")
	assert.Empty(t, item.SubjectType, "there is no notice for an agent to work on yet")
	assert.True(t, item.SubjectID.IsNil())
	assert.Empty(t, item.EventKind)
	assert.Equal(t, row.ProfileUpdatedAt, item.OccurredAt)
	assert.Contains(t, item.Path, "/billing/configuration-files/customers")
	assert.Contains(t, item.Path, row.CustomerID.String())
	assertStorable(t, item)
}

func TestDescribeFactoringMissingNoaFollowsTheOpenNotice(t *testing.T) {
	t.Parallel()

	tenant := factoringTenant()
	sentAt := time.Date(2026, time.September, 3, 15, 0, 0, 0, time.UTC).Unix()
	expiredAt := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC).Unix()
	createdAt := factoringNow - 7200

	cases := []struct {
		name       string
		status     factoring.NoaStatus
		sentAt     *int64
		expiresAt  *int64
		summary    string
		title      string
		occurredAt int64
	}{
		{
			name:       "drafted",
			status:     factoring.NoaStatusDraft,
			summary:    "drafted but has not been sent",
			title:      "has no signed notice",
			occurredAt: createdAt,
		},
		{
			name:       "sent",
			status:     factoring.NoaStatusSent,
			sentAt:     &sentAt,
			summary:    "it was sent on September 3, 2026",
			title:      "has no signed notice",
			occurredAt: sentAt,
		},
		{
			name:       "expired",
			status:     factoring.NoaStatusAcknowledged,
			expiresAt:  &expiredAt,
			summary:    "The notice expired on August 1, 2026",
			title:      "has expired",
			occurredAt: expiredAt,
		},
	}
	for _, tc := range cases {
		row := missingNoa()
		row.NoaID = pulid.MustNew("facnoa_")
		row.NoaStatus = tc.status
		row.NoaSentAt = tc.sentAt
		row.NoaExpiresAt = tc.expiresAt
		row.NoaCreatedAt = &createdAt

		item, open := DescribeFactoringMissingNoa(tenant, row, factoringNow)
		require.True(t, open, tc.name)
		assert.Contains(t, item.Summary, tc.summary, tc.name)
		assert.Contains(t, item.Title, tc.title, tc.name)
		assert.Equal(t, agent.SubjectFactoringNoa, item.SubjectType, tc.name)
		assert.Equal(t, row.NoaID, item.SubjectID, tc.name)
		assert.Equal(t, tc.occurredAt, item.OccurredAt, tc.name)
		assert.Equal(
			t,
			FactoringMissingNoaSourceID(row.CustomerID, row.FactoringCompanyID),
			item.SourceID,
			tc.name,
		)
		assertStorable(t, item)
	}
}

func TestDescribeFactoringMissingNoaClearsOnceAcknowledged(t *testing.T) {
	t.Parallel()

	tenant := factoringTenant()

	open := missingNoa()
	open.NoaID = pulid.MustNew("facnoa_")
	open.NoaStatus = factoring.NoaStatusAcknowledged
	_, raised := DescribeFactoringMissingNoa(tenant, open, factoringNow)
	assert.False(t, raised, "a signed notice with no expiry covers the customer")

	future := factoringNow + timeutils.SecondsPerDay
	open.NoaExpiresAt = &future
	_, raised = DescribeFactoringMissingNoa(tenant, open, factoringNow)
	assert.False(t, raised, "a signed notice that has not expired covers the customer")

	_, raised = DescribeFactoringMissingNoa(tenant, nil, factoringNow)
	assert.False(t, raised)

	noFactor := missingNoa()
	noFactor.FactoringCompanyID = pulid.Nil
	_, raised = DescribeFactoringMissingNoa(tenant, noFactor, factoringNow)
	assert.False(t, raised)
}

func expiringNoa(expiresIn int64) *factoringports.ExpiringNoa {
	acknowledged := factoringNow - 300*timeutils.SecondsPerDay
	return &factoringports.ExpiringNoa{
		NoaID:              pulid.MustNew("facnoa_"),
		CustomerID:         pulid.MustNew("cus_"),
		CustomerName:       "Acme Foods",
		FactoringCompanyID: pulid.MustNew("fac_"),
		FactorName:         "Summit Capital",
		AcknowledgedAt:     &acknowledged,
		ExpiresAt:          factoringNow + expiresIn,
	}
}

func TestDescribeFactoringNoaExpiring(t *testing.T) {
	t.Parallel()

	tenant := factoringTenant()
	row := expiringNoa(20 * timeutils.SecondsPerDay)

	item, open := DescribeFactoringNoaExpiring(tenant, row, factoringNow)
	require.True(t, open)
	assert.Equal(t, watchtower.SourceFactoring, item.SourceKind)
	assert.Equal(t, "noa-expiring:"+row.NoaID.String(), item.SourceID)
	assert.Equal(t, watchtower.SeverityWarning, item.Severity)
	assert.Equal(
		t,
		"Acme Foods's notice of assignment for Summit Capital expires in 20 days",
		item.Title,
	)
	assert.Contains(t, item.Summary, "Get a renewed notice signed")
	assert.Equal(t, agent.SubjectFactoringNoa, item.SubjectType)
	assert.Equal(t, row.NoaID, item.SubjectID)
	assert.Equal(t, agent.EventFactoringNoaExpiring, item.EventKind)
	assert.Equal(t, row.ExpiresAt-FactoringNoaWarningSeconds(), item.OccurredAt)
	assert.Contains(t, item.Path, row.CustomerID.String())
	assertStorable(t, item)
}

func TestDescribeFactoringNoaExpiringSeverityAndWindow(t *testing.T) {
	t.Parallel()

	tenant := factoringTenant()

	soon, open := DescribeFactoringNoaExpiring(
		tenant,
		expiringNoa(5*timeutils.SecondsPerDay),
		factoringNow,
	)
	require.True(t, open)
	assert.Equal(t, watchtower.SeverityCritical, soon.Severity)

	today, open := DescribeFactoringNoaExpiring(tenant, expiringNoa(3600), factoringNow)
	require.True(t, open)
	assert.Contains(t, today.Title, "expires within a day")

	_, open = DescribeFactoringNoaExpiring(
		tenant,
		expiringNoa(FactoringNoaWarningSeconds()+60),
		factoringNow,
	)
	assert.False(t, open, "outside the warning window")

	_, open = DescribeFactoringNoaExpiring(tenant, expiringNoa(0), factoringNow)
	assert.False(t, open, "an expired notice is a missing one, not an expiring one")

	_, open = DescribeFactoringNoaExpiring(tenant, nil, factoringNow)
	assert.False(t, open)
}

func TestFactoringSourceSnapshot(t *testing.T) {
	t.Parallel()

	tenant := factoringTenant()
	late := readyState(tenant, 2*factoring.ReadyUnsubmittedSeconds)
	states := &fakeFactoringStates{ready: []*factoring.InvoiceState{late, nil}}
	reader := &fakeFactoringReader{
		labels: map[pulid.ID]*factoringports.InvoiceLabel{
			late.InvoiceID: {InvoiceID: late.InvoiceID, Number: "INV-7"},
		},
		missing:  []*factoringports.CustomerMissingNoa{missingNoa(), nil},
		expiring: []*factoringports.ExpiringNoa{expiringNoa(10 * timeutils.SecondsPerDay), nil},
	}
	source := newTestFactoringSource(states, reader)

	assert.Equal(t, watchtower.SourceFactoring, source.Kind())

	items, err := source.Snapshot(t.Context(), tenant)
	require.NoError(t, err)
	require.Len(t, items, 3)

	assert.Equal(t, tenant, states.req.TenantInfo)
	assert.Equal(t, factoringNow-factoring.ReadyUnsubmittedSeconds, states.req.ReadyBefore)
	assert.Equal(t, snapshotLimit, states.req.Limit)
	assert.Equal(t, []pulid.ID{late.InvoiceID}, reader.labelReq.InvoiceIDs)
	assert.Equal(t, factoringNow, reader.missingReq.At)
	assert.Equal(t, factoringNow, reader.expiringReq.At)
	assert.Equal(t, factoringNow+FactoringNoaWarningSeconds(), reader.expiringReq.Before)

	ids := make(map[string]struct{}, len(items))
	for _, item := range items {
		assert.Equal(t, watchtower.SourceFactoring, item.SourceKind)
		assert.Equal(t, tenant, item.TenantInfo)
		ids[item.SourceID] = struct{}{}
	}
	assert.Len(t, ids, 3, "each item has its own source id")
	assert.Contains(t, ids, FactoringReadySourceID(late.InvoiceID))
	assert.Equal(t, "Invoice INV-7 has been ready to factor for 2 days", items[0].Title)
}

func TestFactoringSourceSnapshotFailsWhenAReadFails(t *testing.T) {
	t.Parallel()

	tenant := factoringTenant()

	_, err := newTestFactoringSource(
		&fakeFactoringStates{err: assert.AnError},
		&fakeFactoringReader{},
	).Snapshot(t.Context(), tenant)
	require.ErrorIs(t, err, assert.AnError)

	_, err = newTestFactoringSource(
		&fakeFactoringStates{},
		&fakeFactoringReader{labelErr: assert.AnError},
	).Snapshot(t.Context(), tenant)
	require.ErrorIs(t, err, assert.AnError)

	_, err = newTestFactoringSource(
		&fakeFactoringStates{},
		&fakeFactoringReader{missingErr: assert.AnError},
	).Snapshot(t.Context(), tenant)
	require.ErrorIs(t, err, assert.AnError)

	_, err = newTestFactoringSource(
		&fakeFactoringStates{},
		&fakeFactoringReader{expiringErr: assert.AnError},
	).Snapshot(t.Context(), tenant)
	require.ErrorIs(t, err, assert.AnError, "a partial snapshot would resolve live items")
}
