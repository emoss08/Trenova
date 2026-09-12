package fuelpurchaseservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	// feedOverlap re-reads a day either side of the watermark, because a network
	// can post a transaction after the day it happened. Reading a purchase twice
	// costs nothing: the transaction reference unique index refuses the second.
	feedOverlap = 24 * 60 * 60

	// feedFirstRunWindow is how far back a feed reaches the first time it runs.
	// A quarter covers the current IFTA period, which is the point of the feed.
	feedFirstRunWindow = 92 * 24 * 60 * 60

	maxFeedReferenceLength = 255
)

type SyncFeedRequest struct {
	TenantInfo pagination.TenantInfo
	Provider   fuelpurchase.CardProvider
	UserID     pulid.ID
}

type SyncFeedResult struct {
	Provider        fuelpurchase.CardProvider `json:"provider"`
	BatchID         pulid.ID                  `json:"batchId"`
	Fetched         int                       `json:"fetched"`
	Committed       int                       `json:"committed"`
	Queued          int                       `json:"queued"`
	AlreadyImported int                       `json:"alreadyImported"`
	CardsDiscovered int                       `json:"cardsDiscovered"`
}

// SyncFeed reads one organization's card feed and posts what it can.
//
// The rows a connector returns are staged in exactly the shape a spreadsheet
// upload produces, so they go through the same resolution and duplicate checks a
// person's import does. Rows that resolve are posted; rows that do not stay in
// the batch as a review queue, because a feed guessing at a tractor would put
// fuel on the wrong truck's IFTA return.
func (s *Service) SyncFeed(
	ctx context.Context,
	req *SyncFeedRequest,
) (*SyncFeedResult, error) {
	log := s.l.With(
		zap.String("operation", "SyncFeed"),
		zap.String("provider", string(req.Provider)),
	)

	if s.feeds == nil {
		return nil, errortypes.NewBusinessError("Fuel card feeds are not configured")
	}

	feed, err := s.feeds.ResolveFuelFeed(ctx, req.TenantInfo, req.Provider)
	if err != nil {
		return nil, err
	}
	if feed == nil {
		return nil, errortypes.NewBusinessError(
			"{0} is not connected for this organization", req.Provider.Label(),
		)
	}

	return s.runFeed(ctx, log, req, feed)
}

// SyncAllFeeds reads every feed an organization has connected. One provider
// failing does not stop the others, because a broken WEX connection is no reason
// to stop reading Comdata.
func (s *Service) SyncAllFeeds(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*SyncFeedResult, error) {
	if s.feeds == nil {
		return nil, errortypes.NewBusinessError("Fuel card feeds are not configured")
	}

	feeds, err := s.feeds.ListFuelFeeds(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	results := make([]*SyncFeedResult, 0, len(feeds))
	for _, feed := range feeds {
		log := s.l.With(
			zap.String("operation", "SyncAllFeeds"),
			zap.String("provider", string(feed.Provider)),
		)

		result, runErr := s.runFeed(ctx, log, &SyncFeedRequest{
			TenantInfo: tenantInfo,
			Provider:   feed.Provider,
		}, feed)
		if runErr != nil {
			log.Error("fuel card feed failed", zap.Error(runErr))
			continue
		}

		results = append(results, result)
	}

	return results, nil
}

func (s *Service) runFeed(
	ctx context.Context,
	log *zap.Logger,
	req *SyncFeedRequest,
	feed *services.ResolvedFuelFeed,
) (*SyncFeedResult, error) {
	state, err := s.repo.GetFeedState(ctx, &repositories.GetFuelFeedStateRequest{
		TenantInfo: req.TenantInfo,
		Provider:   feed.Provider,
		FeedType:   fuelpurchase.FeedTypeTransactions,
	})
	if err != nil {
		return nil, err
	}

	now := s.now()
	fetched, err := feed.Connector.FetchTransactions(
		ctx,
		&services.FetchFuelTransactionsRequest{
			TenantInfo: req.TenantInfo,
			Provider:   feed.Provider,
			Config:     feed.Config,
			Since:      state.Since(feedOverlap, now-feedFirstRunWindow),
			Until:      now,
			Cursor:     state.Cursor,
		},
	)
	if err != nil {
		// The watermark is deliberately left where it was, so the window this run
		// failed on is read again next time.
		state.RecordFailure(now, err)
		if saveErr := s.repo.SaveFeedState(ctx, state); saveErr != nil {
			log.Error("failed to record fuel feed failure", zap.Error(saveErr))
		}

		return nil, err
	}

	result, err := s.postFeedRows(ctx, req, feed, fetched)
	if err != nil {
		state.RecordFailure(now, err)
		if saveErr := s.repo.SaveFeedState(ctx, state); saveErr != nil {
			log.Error("failed to record fuel feed failure", zap.Error(saveErr))
		}

		return nil, err
	}

	state.RecordSuccess(now, fetched.Cursor)
	if err = s.repo.SaveFeedState(ctx, state); err != nil {
		return nil, err
	}

	return result, nil
}

func (s *Service) postFeedRows(
	ctx context.Context,
	req *SyncFeedRequest,
	feed *services.ResolvedFuelFeed,
	fetched *services.FetchFuelTransactionsResult,
) (*SyncFeedResult, error) {
	result := &SyncFeedResult{Provider: feed.Provider}
	if fetched.Staged == nil || len(fetched.Staged.Rows) == 0 {
		return result, nil
	}
	result.Fetched = len(fetched.Staged.Rows)

	if feed.DiscoverCards {
		discovered, err := s.discoverCards(ctx, req.TenantInfo, feed.Provider, fetched.Cards)
		if err != nil {
			return nil, err
		}
		result.CardsDiscovered = discovered
	}

	batch, err := s.openFeedBatch(ctx, req, feed, fetched)
	if err != nil {
		return nil, err
	}
	result.BatchID = batch.ID

	rows, summary, err := s.resolveRows(ctx, batch, fetched.Staged)
	if err != nil {
		return nil, err
	}
	if err = s.repo.ReplaceImportRows(ctx, batch, rows); err != nil {
		return nil, err
	}

	stagedAt := s.now()
	batch.Status = fuelpurchase.ImportStatusParsed
	batch.Summary = summary
	batch.RowCount = summary.RowCount
	batch.ErrorCount = summary.ErrorCount
	batch.StagedAt = &stagedAt
	if err = validateEntity(batch); err != nil {
		return nil, err
	}
	if batch, err = s.repo.UpdateImportBatch(ctx, batch); err != nil {
		return nil, err
	}

	result.Queued = queuedRowCount(rows)

	commit, err := s.commitFeedRows(ctx, req, batch, rows)
	if err != nil {
		return nil, err
	}
	if commit != nil {
		result.Committed = commit.Committed
		result.AlreadyImported = commit.AlreadyImported
	}

	s.auditFeedRun(req, batch, result)
	s.publish(ctx, req.TenantInfo, realtimeImport, permission.OpImport, batch.ID, req.UserID)
	if result.Committed > 0 {
		s.publish(ctx, req.TenantInfo, realtimePurchase, permission.OpImport, batch.ID, req.UserID)
	}

	return result, nil
}

// openFeedBatch records the run. A batch per sync is what the schema requires for
// imported purchases, and it is also what makes a run auditable: the rows are
// kept beside the purchases they became.
func (s *Service) openFeedBatch(
	ctx context.Context,
	req *SyncFeedRequest,
	feed *services.ResolvedFuelFeed,
	fetched *services.FetchFuelTransactionsResult,
) (*fuelpurchase.ImportBatch, error) {
	batch := &fuelpurchase.ImportBatch{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Provider:       feed.Provider,
		Origin:         fuelpurchase.ImportOriginFeed,
		FeedReference:  truncateReference(fetched.Reference),
		SourceFormat:   fetched.Format,
		Status:         fuelpurchase.ImportStatusPending,
	}
	if len(fetched.Staged.Mapping) > 0 {
		batch.Mapping = fetched.Staged.Mapping.Strings()
	}
	batch.UnmappedHeaders = fetched.Staged.Unmapped
	batch.Normalize()

	if err := validateEntity(batch); err != nil {
		return nil, err
	}

	return s.repo.CreateImportBatch(ctx, batch)
}

// commitFeedRows posts the rows that resolved. The batch is kept open whenever
// anything is left for review, so somebody can fix the cause and have the rest
// posted without the file being fetched again.
func (s *Service) commitFeedRows(
	ctx context.Context,
	req *SyncFeedRequest,
	batch *fuelpurchase.ImportBatch,
	rows []*fuelpurchase.ImportRow,
) (*repositories.CommitImportResult, error) {
	purchases := make([]*fuelpurchase.FuelPurchase, 0, len(rows))
	rowIDByReference := make(map[string]pulid.ID, len(rows))

	for _, row := range rows {
		if !row.WillCommit() || row.TransactionReference == "" {
			continue
		}

		purchase := *row.Parsed
		purchase.ID = ""
		purchase.Version = 0
		purchase.OrganizationID = batch.OrganizationID
		purchase.BusinessUnitID = batch.BusinessUnitID
		purchase.Source = fuelpurchase.PurchaseSourceCardImport
		purchase.ImportBatchID = &batch.ID
		purchase.CreatedByID = req.UserID
		if purchase.TransactionReference == "" {
			purchase.TransactionReference = row.TransactionReference
		}
		purchase.Normalize()

		// A row that cannot pass validation is left in the queue rather than
		// failing the whole run; the rest of the file is still worth posting.
		multiErr := errortypes.NewMultiError()
		purchase.Validate(multiErr)
		if multiErr.HasErrors() {
			row.Status = fuelpurchase.ImportRowStatusError
			row.Error = validationSummary(multiErr)
			continue
		}

		purchases = append(purchases, &purchase)
		rowIDByReference[purchase.TransactionReference] = row.ID
	}

	if len(purchases) == 0 {
		return nil, nil
	}

	return s.repo.CommitImport(ctx, &repositories.CommitImportRequest{
		Batch:            batch,
		Purchases:        purchases,
		RowIDByReference: rowIDByReference,
		CommittedByID:    req.UserID,
		CommittedAt:      s.now(),
		KeepOpen:         len(purchases) < len(rows),
	})
}

// discoverCards registers the cards a run saw that nobody had entered. They are
// created unassigned and suspended, so they are visible and assignable without
// silently becoming usable for matching before a person has looked at them.
func (s *Service) discoverCards(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	provider fuelpurchase.CardProvider,
	cards []services.ProviderCard,
) (int, error) {
	if len(cards) == 0 {
		return 0, nil
	}

	lastFours := make([]string, 0, len(cards))
	for _, card := range cards {
		lastFours = append(lastFours, card.LastFour)
	}

	existing, err := s.repo.FindCardsByLastFour(
		ctx,
		&repositories.FindFuelCardsByLastFourRequest{
			TenantInfo: tenantInfo,
			Provider:   provider,
			LastFours:  lastFours,
		},
	)
	if err != nil {
		return 0, err
	}

	now := s.now()
	created := 0

	for _, card := range cards {
		if _, known := existing[card.LastFour]; known {
			continue
		}

		discoveredAt := now
		entity := &fuelpurchase.FuelCard{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			Provider:       provider,
			LastFour:       card.LastFour,
			Label:          discoveredCardLabel(card, provider),
			ExternalCardID: card.ExternalCardID,
			Status:         fuelpurchase.CardStatusSuspended,
			DiscoveredAt:   &discoveredAt,
		}
		entity.Normalize()

		if err = validateEntity(entity); err != nil {
			return created, err
		}
		if _, err = s.repo.CreateCard(ctx, entity); err != nil {
			return created, err
		}

		created++
	}

	return created, nil
}

func discoveredCardLabel(card services.ProviderCard, provider fuelpurchase.CardProvider) string {
	if label := strings.TrimSpace(card.Label); label != "" {
		return label
	}

	return provider.Label() + " " + card.LastFour
}

func queuedRowCount(rows []*fuelpurchase.ImportRow) int {
	queued := 0
	for _, row := range rows {
		if !row.WillCommit() {
			queued++
		}
	}

	return queued
}

func truncateReference(reference string) string {
	reference = strings.TrimSpace(reference)
	if len(reference) <= maxFeedReferenceLength {
		return reference
	}

	return reference[:maxFeedReferenceLength]
}

func (s *Service) auditFeedRun(
	req *SyncFeedRequest,
	batch *fuelpurchase.ImportBatch,
	result *SyncFeedResult,
) {
	s.audit(&auditParams{
		resource:   permission.ResourceFuelPurchaseImport,
		resourceID: batch.ID.String(),
		operation:  permission.OpImport,
		userID:     req.UserID,
		tenant:     req.TenantInfo,
		current:    batch,
		comment: "Synced " + req.Provider.Label() + ": " +
			strconv.Itoa(result.Committed) + " posted, " +
			strconv.Itoa(result.Queued) + " held for review, " +
			strconv.Itoa(result.CardsDiscovered) + " cards discovered",
	})
}
