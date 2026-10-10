package ledgersync

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/accounting/domain/journalentry"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/journalentryports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
)

var Module = fx.Decorate(DecoratePosting, DecorateReview)

type postingRepository struct {
	next   journalentryports.JournalPostingRepository
	ledger accountingsyncports.AccountingLedgerEnqueuer
}

type reviewRepository struct {
	journalentryports.JournalReviewRepository
	entries journalentryports.JournalEntryRepository
	ledger  accountingsyncports.AccountingLedgerEnqueuer
}

func DecoratePosting(
	next journalentryports.JournalPostingRepository,
	ledger accountingsyncports.AccountingLedgerEnqueuer,
) journalentryports.JournalPostingRepository {
	return &postingRepository{next: next, ledger: ledger}
}

func DecorateReview(
	next journalentryports.JournalReviewRepository,
	entries journalentryports.JournalEntryRepository,
	ledger accountingsyncports.AccountingLedgerEnqueuer,
) journalentryports.JournalReviewRepository {
	return &reviewRepository{JournalReviewRepository: next, entries: entries, ledger: ledger}
}

//nolint:gocritic // the repository contract takes the params by value
func (r *postingRepository) CreatePosting(
	ctx context.Context,
	params journalentryports.CreateJournalPostingParams,
) error {
	if err := r.next.CreatePosting(ctx, params); err != nil {
		return err
	}
	if !params.IsPosted {
		return nil
	}
	return r.ledger.EnqueueJournal(ctx, &accountingsyncports.AccountingJournalPosted{
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
		EntryID:        params.EntryID,
		EntryNumber:    params.EntryNumber,
		EntryType:      journalentry.EntryType(params.EntryType),
		ReversalOfID:   reversalOf(params.IsReversal, params.ReversalOfID),
		AccountingDate: params.AccountingDate,
	})
}

func (r *reviewRepository) PostEntry(
	ctx context.Context,
	params *journalentryports.PostJournalEntryParams,
) error {
	if err := r.JournalReviewRepository.PostEntry(ctx, params); err != nil {
		return err
	}
	entry, err := r.entries.GetByID(ctx, journalentryports.GetJournalEntryByIDRequest{
		ID:         params.EntryID,
		TenantInfo: params.TenantInfo,
	})
	if err != nil {
		return err
	}
	return r.ledger.EnqueueJournal(ctx, &accountingsyncports.AccountingJournalPosted{
		TenantInfo:     params.TenantInfo,
		EntryID:        entry.ID,
		EntryNumber:    entry.EntryNumber,
		EntryType:      entry.EntryType,
		ReversalOfID:   reversalOf(entry.IsReversal, entry.ReversalOfID),
		AccountingDate: entry.AccountingDate,
	})
}

func reversalOf(isReversal bool, id pulid.ID) pulid.ID {
	if !isReversal {
		return pulid.Nil
	}
	return id
}
