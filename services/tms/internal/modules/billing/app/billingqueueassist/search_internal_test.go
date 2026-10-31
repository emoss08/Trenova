package billingqueueassist

import (
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestSearchFailure_TellsThePersonWhatTheyCanDoAboutIt(t *testing.T) {
	t.Parallel()

	for name, tc := range map[string]struct {
		err  error
		says string
	}{
		"a sentence Nova could not read": {
			err:  fmt.Errorf("%w: bad json", llmports.ErrOutsideSchema),
			says: "try different words",
		},
		"no provider": {
			err:  fmt.Errorf("nova QueryCompose: %w", llmports.ErrNoProviderConfigured),
			says: "AI Control",
		},
		"every provider down": {
			err:  &llmports.ChatProvidersFailedError{Err: errors.New("overloaded")},
			says: "couldn't be reached",
		},
		"a refusal already written for the person": {
			err:  errortypes.NewBusinessError("AI is turned off for this organization"),
			says: "AI is turned off for this organization",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			got := searchFailure(tc.err)

			assert.True(t, errortypes.IsBusinessError(got))
			assert.Contains(t, got.Error(), tc.says)
		})
	}
}

func TestResolveDates_KeepsTheWindowsASentenceCanNameAndHandsBackTheRest(t *testing.T) {
	t.Parallel()

	today := time.Date(2026, time.October, 31, 0, 0, 0, 0, time.UTC)
	day := func(s string) *string { return &s }
	days := func(n int) *int { return &n }

	tests := []struct {
		name       string
		answer     searchDatesAnswer
		want       *billingqueueports.SearchDates
		unresolved string
	}{
		{name: "nothing named", answer: searchDatesAnswer{}},
		{
			name:   "the last N days",
			answer: searchDatesAnswer{LastDays: days(7)},
			want:   &billingqueueports.SearchDates{LastDays: 7},
		},
		{name: "more than a year back", answer: searchDatesAnswer{LastDays: days(400)}},
		{
			name:   "a range of days",
			answer: searchDatesAnswer{From: day("2026-10-19"), To: day("2026-10-25")},
			want:   &billingqueueports.SearchDates{From: "2026-10-19", To: "2026-10-25"},
		},
		{
			name:   "an open start reads to today",
			answer: searchDatesAnswer{From: day("2026-10-01")},
			want:   &billingqueueports.SearchDates{From: "2026-10-01", To: "2026-10-31"},
		},
		{
			name:       "an end with no start",
			answer:     searchDatesAnswer{To: day("2026-10-01")},
			unresolved: "2026-10-01",
		},
		{
			name:       "backwards",
			answer:     searchDatesAnswer{From: day("2026-10-25"), To: day("2026-10-19")},
			unresolved: "2026-10-25 2026-10-19",
		},
		{
			name:       "not a day",
			answer:     searchDatesAnswer{From: day("last Tuesday")},
			unresolved: "last Tuesday",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, unresolved := resolveDates(tt.answer, today)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.unresolved, unresolved)
		})
	}
}

func TestSearchToday_CountsFromThePersonsOwnDate(t *testing.T) {
	t.Parallel()

	assert.Equal(t, time.Date(2026, time.October, 31, 0, 0, 0, 0, time.UTC), searchToday("2026-10-31"))
	fallback := searchToday("31/10/2026")
	assert.Equal(t, 0, fallback.Hour(), "a date it cannot read falls back to the start of the server's day")
}

func TestResolveIntent_FindsThePayerAndTheShipmentsCustomerApart(t *testing.T) {
	t.Parallel()

	customers := customerportsmock.NewMockCustomerRepository(t)
	acmeID, globexID := pulid.MustNew("cus_"), pulid.MustNew("cus_")
	matching := func(query string) any {
		return mock.MatchedBy(func(req *customerports.CustomerSelectOptionsRequest) bool {
			return req.Query == query
		})
	}
	customers.EXPECT().SelectOptions(mock.Anything, matching("Acme")).Return(
		&pagination.ListResult[*customer.Customer]{Items: []*customer.Customer{{ID: acmeID, Name: "Acme Foods"}}},
		nil,
	)
	customers.EXPECT().SelectOptions(mock.Anything, matching("Globex")).Return(
		&pagination.ListResult[*customer.Customer]{Items: []*customer.Customer{{ID: globexID, Name: "Globex"}}},
		nil,
	)
	customers.EXPECT().SelectOptions(mock.Anything, matching("Nobody")).Return(
		&pagination.ListResult[*customer.Customer]{}, nil,
	)
	s := &Service{customers: customers}
	days := 7
	today := time.Date(2026, time.October, 31, 0, 0, 0, 0, time.UTC)

	intent, err := s.resolveIntent(t.Context(), pagination.TenantInfo{}, &searchAnswer{
		Payer:     "Acme",
		Customer:  "Globex",
		Delivered: searchDatesAnswer{LastDays: &days},
	}, today)
	require.NoError(t, err)
	assert.Equal(t, acmeID, intent.PayerID)
	assert.Equal(t, "Acme Foods", intent.PayerName)
	assert.Equal(t, globexID, intent.ShipmentCustomerID)
	assert.Equal(t, "Globex", intent.ShipmentCustomerName)
	assert.Equal(t, &billingqueueports.SearchDates{LastDays: 7}, intent.Delivered)
	assert.Nil(t, intent.PickedUp)
	assert.Empty(t, intent.Unresolved)

	missing, err := s.resolveIntent(t.Context(), pagination.TenantInfo{}, &searchAnswer{Customer: "Nobody"}, today)
	require.NoError(t, err)
	assert.True(t, missing.ShipmentCustomerID.IsNil())
	assert.Equal(t, "Nobody", missing.Unresolved, "a customer it cannot find is what it could not read")
}
