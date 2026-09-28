package ledgersync

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

var Module = fx.Decorate(DecoratePosting, DecorateReview)

type postingRepository struct {
	next   repositories.JournalPostingRepository
	ledger services.AccountingLedgerEnqueuer
}

type reviewRepository struct {
	repositories.JournalReviewRepository
	entries repositories.JournalEntryRepository
	ledger  services.AccountingLedgerEnqueuer
}

func DecoratePosting(
	next repositories.JournalPostingRepository,
	ledger services.AccountingLedgerEnqueuer,
) repositories.JournalPostingRepository {
	return &postingRepository{next: next, ledger: ledger}
}

func DecorateReview(
	next repositories.JournalReviewRepository,
	entries repositories.JournalEntryRepository,
	ledger services.AccountingLedgerEnqueuer,
) repositories.JournalReviewRepository {
	return &reviewRepository{JournalReviewRepository: next, entries: entries, ledger: ledger}
}

//nolint:gocritic // the repository contract takes the params by value
func (r *postingRepository) CreatePosting(
	ctx context.Context,
	params repositories.CreateJournalPostingParams,
) error {
	if err := r.next.CreatePosting(ctx, params); err != nil {
		return err
	}
	if !params.IsPosted {
		return nil
	}
	return r.ledger.EnqueueJournal(ctx, &services.AccountingJournalPosted{
		TenantInfo: pagination.TenantInfo{
			OrgID: params.OrganizationID,
			BuID:  params.BusinessUnitID,
		},
		EntryID:        params.EntryID,
		EntryNumber:    params.EntryNumber,
		EntryType:      journalentry.EntryType(params.EntryType),
		AccountingDate: params.AccountingDate,
	})
}

func (r *reviewRepository) PostEntry(
	ctx context.Context,
	params *repositories.PostJournalEntryParams,
) error {
	if err := r.JournalReviewRepository.PostEntry(ctx, params); err != nil {
		return err
	}
	entry, err := r.entries.GetByID(ctx, repositories.GetJournalEntryByIDRequest{
		ID:         params.EntryID,
		TenantInfo: params.TenantInfo,
	})
	if err != nil {
		return err
	}
	return r.ledger.EnqueueJournal(ctx, &services.AccountingJournalPosted{
		TenantInfo:     params.TenantInfo,
		EntryID:        entry.ID,
		EntryNumber:    entry.EntryNumber,
		EntryType:      entry.EntryType,
		AccountingDate: entry.AccountingDate,
	})
}
