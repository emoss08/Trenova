package fuelpurchaseservice_test

import (
	"context"
	"errors"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/fuelpurchaseservice"
	"github.com/emoss08/trenova/pkg/fuelimport"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/rateimport"
	"github.com/stretchr/testify/require"
)

type fakeConnector struct {
	capability services.FuelFeedCapability
	result     *services.FetchFuelTransactionsResult
	err        error
	calls      int
	lastReq    *services.FetchFuelTransactionsRequest
}

func (c *fakeConnector) IntegrationType() integration.Type {
	return integration.TypeComdataFuel
}

func (c *fakeConnector) Capability() services.FuelFeedCapability {
	if c.capability == "" {
		return services.FuelFeedCapabilityIFTAGrade
	}
	return c.capability
}

func (c *fakeConnector) FetchTransactions(
	_ context.Context,
	req *services.FetchFuelTransactionsRequest,
) (*services.FetchFuelTransactionsResult, error) {
	c.calls++
	c.lastReq = req
	if c.err != nil {
		return nil, c.err
	}

	return c.result, nil
}

func (c *fakeConnector) TestConnection(context.Context, map[string]string) error { return nil }

type fakeFeedResolver struct {
	feeds map[fuelpurchase.CardProvider]*services.ResolvedFuelFeed
}

func newFakeFeedResolver() *fakeFeedResolver {
	return &fakeFeedResolver{feeds: map[fuelpurchase.CardProvider]*services.ResolvedFuelFeed{}}
}

func (f *fakeFeedResolver) install(
	provider fuelpurchase.CardProvider,
	connector services.FuelCardProvider,
	discoverCards bool,
) {
	f.feeds[provider] = &services.ResolvedFuelFeed{
		Connector:       connector,
		IntegrationType: integration.TypeComdataFuel,
		Provider:        provider,
		Config:          map[string]string{},
		DiscoverCards:   discoverCards,
	}
}

func (f *fakeFeedResolver) ResolveFuelFeed(
	_ context.Context,
	_ pagination.TenantInfo,
	provider fuelpurchase.CardProvider,
) (*services.ResolvedFuelFeed, error) {
	return f.feeds[provider], nil
}

func (f *fakeFeedResolver) ListFuelFeeds(
	_ context.Context,
	_ pagination.TenantInfo,
) ([]*services.ResolvedFuelFeed, error) {
	feeds := make([]*services.ResolvedFuelFeed, 0, len(f.feeds))
	for _, feed := range f.feeds {
		feeds = append(feeds, feed)
	}

	return feeds, nil
}

// stagedFrom runs rows through the real staging pipeline, so a test exercises the
// same parsing a connector's output goes through rather than a hand-built shape
// that can drift from it.
func stagedFrom(headers []string, rows ...[]string) *fuelimport.StageResult {
	return fuelimport.Stage(&rateimport.Sheet{
		Headers:      headers,
		Rows:         rows,
		FirstDataRow: 2,
	}, fuelimport.StageOptions{
		Provider:        fuelpurchase.CardProviderComdata,
		DefaultCurrency: "USD",
	})
}

var feedHeaders = []string{
	"Trans Date", "Card Number", "Unit", "State",
	"Product", "Gallons", "Amount", "Trans ID",
}

func feedRow(reference, unit, lastFour string) []string {
	return []string{
		"01/05/2026", lastFour, unit, "TX",
		"Diesel", "125.400", "501.60", reference,
	}
}

func installFeed(
	h *harness,
	result *services.FetchFuelTransactionsResult,
	discoverCards bool,
) *fakeConnector {
	connector := &fakeConnector{result: result}
	h.feeds.install(fuelpurchase.CardProviderComdata, connector, discoverCards)

	return connector
}

func syncFeed(t *testing.T, h *harness) *fuelpurchaseservice.SyncFeedResult {
	t.Helper()

	result, err := h.svc.SyncFeed(t.Context(), &fuelpurchaseservice.SyncFeedRequest{
		TenantInfo: h.tenant,
		Provider:   fuelpurchase.CardProviderComdata,
		UserID:     h.userID,
	})
	require.NoError(t, err)

	return result
}

func TestSyncFeedPostsRowsThatResolve(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	installFeed(h, &services.FetchFuelTransactionsResult{
		Staged:    stagedFrom(feedHeaders, feedRow("C-1001", "TRC-001", "4411")),
		Reference: "/outbound/AC00029",
		Format:    fuelpurchase.SourceFormatCSV,
	}, false)

	result := syncFeed(t, h)

	require.Equal(t, 1, result.Fetched)
	require.Equal(t, 1, result.Committed)
	require.Zero(t, result.Queued)

	purchases := h.repo.allPurchases()
	require.Len(t, purchases, 1)
	require.Equal(t, "C-1001", purchases[0].TransactionReference)
	require.Equal(t, h.tractorA.ID, purchases[0].TractorID)
	require.Equal(t, fuelpurchase.PurchaseSourceCardImport, purchases[0].Source)
}

// A feed must never guess at a tractor: putting fuel on the wrong truck moves
// gallons between IFTA returns. The row waits for a person instead.
func TestSyncFeedHoldsARowWhoseTractorIsUnknown(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: stagedFrom(feedHeaders, feedRow("C-1001", "TRC-NOPE", "4411")),
	}, false)

	result := syncFeed(t, h)

	require.Equal(t, 1, result.Queued)
	require.Zero(t, result.Committed)
	require.Empty(t, h.repo.allPurchases())

	batch := h.repo.batch(result.BatchID)
	require.Equal(t, fuelpurchase.ImportStatusParsed, batch.Status)
	require.True(t, batch.IsFeed())
}

// A batch that still has rows waiting stays open, so assigning the missing card
// and posting the rest does not need the file fetched again.
func TestSyncFeedKeepsABatchOpenWhenAnythingIsQueued(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: stagedFrom(
			feedHeaders,
			feedRow("C-1001", "TRC-001", "4411"),
			feedRow("C-1002", "TRC-NOPE", "4411"),
		),
	}, false)

	result := syncFeed(t, h)

	require.Equal(t, 1, result.Committed)
	require.Equal(t, 1, result.Queued)

	batch := h.repo.batch(result.BatchID)
	require.Equal(t, fuelpurchase.ImportStatusParsed, batch.Status)
	require.Nil(t, batch.CommittedAt)
	require.True(t, batch.CanCommit())
}

func TestSyncFeedClosesABatchWhenEverythingPosted(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: stagedFrom(feedHeaders, feedRow("C-1001", "TRC-001", "4411")),
	}, false)

	batch := h.repo.batch(syncFeed(t, h).BatchID)
	require.Equal(t, fuelpurchase.ImportStatusCommitted, batch.Status)
	require.NotNil(t, batch.CommittedAt)
}

func TestSyncFeedDiscoversAnUnknownCardUnassignedAndSuspended(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: stagedFrom(feedHeaders, feedRow("C-1001", "TRC-001", "4411")),
		Cards: []services.ProviderCard{
			{LastFour: "4411", Provider: fuelpurchase.CardProviderComdata},
		},
	}, true)

	result := syncFeed(t, h)
	require.Equal(t, 1, result.CardsDiscovered)

	cards := h.repo.allCards()
	require.Len(t, cards, 1)
	require.Equal(t, "4411", cards[0].LastFour)
	require.Equal(t, fuelpurchase.CardStatusSuspended, cards[0].Status)
	require.True(t, cards[0].IsUnassigned())
	require.True(t, cards[0].WasDiscovered())
}

func TestSyncFeedDoesNotDiscoverCardsWhenTheConnectionSaysNotTo(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: stagedFrom(feedHeaders, feedRow("C-1001", "TRC-001", "4411")),
		Cards: []services.ProviderCard{
			{LastFour: "4411", Provider: fuelpurchase.CardProviderComdata},
		},
	}, false)

	require.Zero(t, syncFeed(t, h).CardsDiscovered)
	require.Empty(t, h.repo.allCards())
}

func TestSyncFeedLeavesAnAlreadyRegisteredCardAlone(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	existing := h.newCard("4411")
	_, err := h.repo.CreateCard(t.Context(), existing)
	require.NoError(t, err)

	installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: stagedFrom(feedHeaders, feedRow("C-1001", "TRC-001", "4411")),
		Cards: []services.ProviderCard{
			{LastFour: "4411", Provider: fuelpurchase.CardProviderComdata},
		},
	}, true)

	require.Zero(t, syncFeed(t, h).CardsDiscovered)
	require.Len(t, h.repo.allCards(), 1)
}

// The window is deliberately re-read with an overlap, because a network can post
// a transaction after the day it happened. Reading one twice must cost nothing.
func TestSyncFeedPostsNothingTwiceAcrossOverlappingRuns(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: stagedFrom(feedHeaders, feedRow("C-1001", "TRC-001", "4411")),
	}, false)

	require.Equal(t, 1, syncFeed(t, h).Committed)

	second := syncFeed(t, h)
	require.Zero(t, second.Committed)
	require.Len(t, h.repo.allPurchases(), 1)
}

func TestSyncFeedAdvancesTheWatermarkOnlyOnSuccess(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	connector := installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: stagedFrom(feedHeaders, feedRow("C-1001", "TRC-001", "4411")),
		Cursor: "cursor-1",
	}, false)

	syncFeed(t, h)

	state := h.repo.feedState(fuelpurchase.CardProviderComdata)
	require.Positive(t, state.LastSuccessAt)
	require.Equal(t, "cursor-1", state.Cursor)
	require.Zero(t, state.FailureCount)

	succeededAt := state.LastSuccessAt
	connector.err = errors.New("the server refused the connection")

	_, err := h.svc.SyncFeed(t.Context(), &fuelpurchaseservice.SyncFeedRequest{
		TenantInfo: h.tenant,
		Provider:   fuelpurchase.CardProviderComdata,
		UserID:     h.userID,
	})
	require.Error(t, err)

	after := h.repo.feedState(fuelpurchase.CardProviderComdata)
	require.Equal(t, succeededAt, after.LastSuccessAt)
	require.Equal(t, 1, after.FailureCount)
	require.Contains(t, after.LastError, "refused the connection")
}

// The second run asks for a window starting before the last success, so a
// transaction posted late is still picked up.
func TestSyncFeedReReadsAnOverlappingWindow(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	connector := installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: stagedFrom(feedHeaders, feedRow("C-1001", "TRC-001", "4411")),
	}, false)

	syncFeed(t, h)
	firstUntil := connector.lastReq.Until

	syncFeed(t, h)
	require.Less(t, connector.lastReq.Since, firstUntil)
}

// A row without gallons cannot become a tax record. This is what a spend-only
// card produces, and it has to wait for somebody to complete it.
func TestSyncFeedHoldsARowWithNoGallons(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: stagedFrom(feedHeaders, []string{
			"01/05/2026", "4411", "TRC-001", "TX",
			"Diesel", "", "501.60", "C-1001",
		}),
	}, false)

	result := syncFeed(t, h)

	require.Equal(t, 1, result.Queued)
	require.Zero(t, result.Committed)
	require.Empty(t, h.repo.allPurchases())
}

func TestSyncFeedRecordsNothingWhenTheFeedIsQuiet(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: &fuelimport.StageResult{},
	}, true)

	result := syncFeed(t, h)

	require.Zero(t, result.Fetched)
	require.True(t, result.BatchID.IsNil())
	require.Empty(t, h.repo.allBatches())
}

func TestSyncFeedRefusesAProviderThatIsNotConnected(t *testing.T) {
	t.Parallel()

	h := newHarness(t)

	_, err := h.svc.SyncFeed(t.Context(), &fuelpurchaseservice.SyncFeedRequest{
		TenantInfo: h.tenant,
		Provider:   fuelpurchase.CardProviderWEX,
		UserID:     h.userID,
	})
	require.ErrorContains(t, err, "is not connected")
}

func TestSyncAllFeedsKeepsGoingWhenOneProviderFails(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: stagedFrom(feedHeaders, feedRow("C-1001", "TRC-001", "4411")),
	}, false)
	h.feeds.install(
		fuelpurchase.CardProviderWEX,
		&fakeConnector{err: errors.New("the server refused the connection")},
		false,
	)

	results, err := h.svc.SyncAllFeeds(t.Context(), h.tenant)
	require.NoError(t, err)
	require.Len(t, results, 1)
	require.Equal(t, fuelpurchase.CardProviderComdata, results[0].Provider)
	require.Equal(t, 1, results[0].Committed)
}

func TestSyncFeedScopesEverythingItWritesToTheTenant(t *testing.T) {
	t.Parallel()

	h := newHarness(t)
	installFeed(h, &services.FetchFuelTransactionsResult{
		Staged: stagedFrom(feedHeaders, feedRow("C-1001", "TRC-001", "4411")),
		Cards: []services.ProviderCard{
			{LastFour: "4411", Provider: fuelpurchase.CardProviderComdata},
		},
	}, true)

	result := syncFeed(t, h)

	batch := h.repo.batch(result.BatchID)
	require.Equal(t, h.tenant.OrgID, batch.OrganizationID)
	require.Equal(t, h.tenant.BuID, batch.BusinessUnitID)

	for _, card := range h.repo.allCards() {
		require.Equal(t, h.tenant.OrgID, card.OrganizationID)
		require.Equal(t, h.tenant.BuID, card.BusinessUnitID)
	}
	for _, purchase := range h.repo.allPurchases() {
		require.Equal(t, h.tenant.OrgID, purchase.OrganizationID)
		require.Equal(t, h.tenant.BuID, purchase.BusinessUnitID)
	}
}

var _ repositories.FuelPurchaseRepository = (*fakeRepo)(nil)
