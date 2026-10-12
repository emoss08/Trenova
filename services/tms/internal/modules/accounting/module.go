package accounting

import (
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/accountingsyncrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/accounttyperepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalperiodrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/fiscalyearrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/glaccountrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/glbalancerepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalentryrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalpostingrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalreversalrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/journalsourcerepository"
	"github.com/emoss08/trenova/internal/modules/accounting/adapters/postgres/manualjournalrepository"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingconnectionservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountinginboundservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingmappingservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accountingsyncservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/accounttypeservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalcloseservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalperiodservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/fiscalyearservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/glaccountservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/glbalanceservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalentryservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/journalreversalservice"
	"github.com/emoss08/trenova/internal/modules/accounting/app/manualjournalservice"
	"github.com/emoss08/trenova/internal/modules/accounting/jobs/accountingsyncjobs/accountingsynccontract"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingsyncports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			accountingconnectionservice.New,
			func(s *accountingconnectionservice.Service) accountingsyncports.AccountingConnectionService { return s },
			accountingmappingservice.New,
			func(s *accountingmappingservice.Service) accountingsyncports.AccountingMappingService { return s },
			accountinginboundservice.New,
			func(s *accountinginboundservice.Service) accountingsyncports.AccountingInboundService { return s },
			glbalancerepository.New,
			accountingsyncrepository.NewConnectionRepository,
			accountingsyncrepository.NewAppCredentialRepository,
			accountingsyncrepository.NewSubscriptionRepository,
			accountingsyncrepository.NewReferenceRepository,
			accountingsyncrepository.NewMappingRepository,
			accountingsyncrepository.NewSyncRecordRepository,
			accountingsyncrepository.NewInboundChangeRepository,
			accountingsyncrepository.NewBackfillRepository,
			accountingsyncrepository.NewPayablesRepository,
			accountingsyncrepository.NewDriftFindingRepository,
			accountingsyncrepository.NewDriftSource,
			accountingsyncrepository.NewLedgerSource,
			accounttyperepository.New,
			glaccountrepository.New,
			fiscalyearrepository.New,
			fiscalperiodrepository.New,
			journalentryrepository.New,
			journalpostingrepository.New,
			journalpostingrepository.NewReview,
			journalreversalrepository.New,
			journalsourcerepository.New,
			manualjournalrepository.New,
			accounttypeservice.NewValidator,
			glaccountservice.NewValidator,
			fiscalyearservice.NewValidator,
			fiscalperiodservice.NewValidator,
			journalreversalservice.NewValidator,
			manualjournalservice.NewValidator,
			accounttypeservice.New,
			glaccountservice.New,
			glbalanceservice.New,
			fiscalcloseservice.New,
			fiscalyearservice.New,
			fiscalperiodservice.New,
			accountingsyncservice.NewEnqueuer,
			func(e *accountingsyncservice.Enqueuer) accountingsyncports.AccountingSyncEnqueuer { return e },
			func(e *accountingsyncservice.Enqueuer) accountingsyncports.AccountingLedgerEnqueuer { return e },
			accountingsyncservice.New,
			func(s *accountingsyncservice.Service) accountingsyncports.AccountingSyncService { return s },
			journalentryservice.New,
			journalreversalservice.New,
			manualjournalservice.New,
		),
		fx.Provide(fx.Annotate(
			accountingsynccontract.NewSyncDispatcher,
			fx.As(new(accountingsyncports.AccountingSyncDispatcher)),
		)),
		fx.Provide(fx.Annotate(
			accountingsynccontract.NewDriftChecker,
			fx.As(new(accountingsyncports.AccountingDriftChecker)),
		)),
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
