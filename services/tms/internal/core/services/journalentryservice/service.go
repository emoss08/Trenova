package journalentryservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/journalentry"
	"github.com/emoss08/trenova/internal/core/domain/journalsource"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger     *zap.Logger
	EntryRepo  repositories.JournalEntryRepository
	SourceRepo repositories.JournalSourceRepository
}

type Service struct {
	l          *zap.Logger
	entryRepo  repositories.JournalEntryRepository
	sourceRepo repositories.JournalSourceRepository
}

func New(p Params) *Service {
	return &Service{
		l:          p.Logger.Named("service.journal-entry"),
		entryRepo:  p.EntryRepo,
		sourceRepo: p.SourceRepo,
	}
}

func (s *Service) ListEntries(
	ctx context.Context,
	req *repositories.ListJournalEntriesRequest,
) (*pagination.ListResult[*journalentry.Entry], error) {
	return s.entryRepo.List(ctx, req)
}

func (s *Service) GetEntry(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	entryID pulid.ID,
) (*journalentry.Entry, error) {
	return s.entryRepo.GetByID(
		ctx,
		repositories.GetJournalEntryByIDRequest{ID: entryID, TenantInfo: tenantInfo},
	)
}

func (s *Service) GetSourceByObject(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	sourceObjectType, sourceObjectID string,
) (*journalsource.Source, error) {
	return s.sourceRepo.GetByObject(
		ctx,
		repositories.GetJournalSourceByObjectRequest{
			TenantInfo:       tenantInfo,
			SourceObjectType: sourceObjectType,
			SourceObjectID:   sourceObjectID,
		},
	)
}
