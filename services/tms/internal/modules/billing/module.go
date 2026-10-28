package billing

import (
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/accountsreceivablerepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/bankreceiptbatchrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/bankreceiptrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/bankreceiptworkitemrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/billingqueuefilterpresetrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/billingqueuerepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/billingqueuereviewrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/billingtransferrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/chargeallocationrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/collectionrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/costingrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerledgerrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerpaymentrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/customerstatementrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/factoringrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoiceadjustmentrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicedisputerepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicerunrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/invoicesharerepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/latechargerepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/paymentreminderrepository"
	"github.com/emoss08/trenova/internal/modules/billing/adapters/postgres/recordversionrepository"
	"github.com/emoss08/trenova/internal/modules/billing/app/accountingdriftservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/accountsreceivableservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptbatchservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/bankreceiptworkitemservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueapprovalservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingtransferservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/collectionservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/costingservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/creditreviewservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/customerpaymentservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/customerstatementservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/detentionbillingservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/detentioncreditservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/factoringtransport"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceadjustmentservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicedisputeservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicerunservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceshareservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/latechargeservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/paymentreminderservice"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/factoringjobs/factoringcontract"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptbatchports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerledgerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerstatementports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringsubmissionports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoicerunports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/paymentreminderports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documenttemplateports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"go.uber.org/fx"
)

func SharedOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			accountsreceivablerepository.New,
			accountsreceivablerepository.NewAnalytics,
			accountsreceivablerepository.NewOpenBalanceReader,
			costingrepository.New,
			costingrepository.NewActuals,
			costingservice.NewValidator,
			accountsreceivableservice.New,
			func(s *accountsreceivableservice.Service) customerledgerports.ARSnapshotReader {
				return s
			},
			detentionbillingservice.New,
			invoiceshareservice.New,
			func(s *invoiceshareservice.Service) invoiceports.InvoiceShareService { return s },
			costingservice.New,
			billingcontrolpolicyservice.New,
			billingtransferrepository.New,
			billingqueuefilterpresetrepository.New,
			recordversionrepository.New,
			bankreceiptrepository.New,
			bankreceiptbatchrepository.New,
			bankreceiptworkitemrepository.New,
			chargeallocationrepository.New,
			customerpaymentrepository.New,
			fx.Annotate(
				customerledgerrepository.New,
				fx.ResultTags(creditreviewservice.LedgerStoreName),
			),
			collectionrepository.New,
			paymentreminderrepository.New,
			paymentreminderservice.New,
			func(s *paymentreminderservice.Service) paymentreminderports.Service { return s },
			customerstatementrepository.New,
			customerstatementservice.New,
			func(s *customerstatementservice.Service) customerstatementports.Service { return s },
			collectionservice.New,
			func(s *collectionservice.Service) collectionports.Service { return s },
			creditreviewservice.New,
			func(s *creditreviewservice.Service) customerports.CreditReviewer { return s },
			creditreviewservice.NewLedger,
			invoiceadjustmentrepository.New,
			invoicerepository.New,
			invoicedisputerepository.New,
			latechargerepository.New,
			invoicerunrepository.New,
			invoicesharerepository.New,
			billingqueuerepository.New,
			billingqueuereviewrepository.New,
			customerpaymentservice.NewValidator,
			invoiceadjustmentservice.NewValidator,
			invoiceservice.NewValidator,
			invoicerunservice.NewValidator,
			billingqueueservice.NewValidator,
			billingtransferservice.New,
			fx.Annotate(
				bankreceiptbatchservice.New,
				fx.As(new(bankreceiptbatchports.BankReceiptBatchService)),
			),
			bankreceiptservice.New,
			func(s *bankreceiptservice.Service) bankreceiptports.BankReceiptService { return s },
			bankreceiptworkitemservice.New,
			fx.Annotate(
				customerpaymentservice.New,
				fx.As(new(customerpaymentports.CustomerPaymentService)),
			),
			accountingdriftservice.New,
			func(s *accountingdriftservice.Service) accountingdriftports.AccountingDriftReconciler { return s },
			func(s *accountingdriftservice.Service) accountingdriftports.AccountingDriftRechecker { return s },
			func(s *accountingdriftservice.Service) accountingdriftports.AccountingDriftService { return s },
			invoiceadjustmentservice.New,
			detentioncreditservice.New,
			func(s *detentioncreditservice.Service) detentionports.BilledDetentionCreditor { return s },
			invoiceservice.NewService,
			func(s *invoiceservice.Service) invoiceports.InvoiceService { return s },
			invoicerunservice.New,
			invoicedisputeservice.New,
			latechargeservice.New,
			func(s *invoicerunservice.Service) invoicerunports.InvoiceRunSweeper { return s },
			billingqueueservice.New,
			billingqueueservice.NewReview,
			billingqueueapprovalservice.New,
			invoiceservice.NewContextBuilder,
			fx.Annotate(
				func(b *invoiceservice.ContextBuilder) documenttemplateports.ContextBuilder { return b },
				fx.ResultTags(documenttemplateports.ContextBuilderGroup),
			),
		),
		factoringOptions(),
	)
}

func factoringOptions() fx.Option {
	return fx.Options(
		fx.Provide(
			factoringrepository.NewAgreementRepository,
			factoringrepository.NewNoaRepository,
			factoringrepository.NewInvoiceStateRepository,
			factoringrepository.NewSubmissionRepository,
			factoringrepository.NewReserveEntryRepository,
			factoringrepository.NewRemittanceRepository,
			factoringrepository.NewTransportRepository,
			factoringrepository.NewWatchtowerReader,
			factoringcontract.NewRequester,
			factoringservice.NewAgreementService,
			factoringservice.NewNoaService,
			factoringservice.NewReadinessService,
			factoringservice.NewSubmissionService,
			factoringservice.NewLedgerService,
			factoringservice.NewRemittanceService,
			factoringservice.NewRemittanceIntake,
			factoringservice.NewTransportService,
			factoringcontract.NewSendStarter,
			factoringcontract.NewRemittanceStageStarter,
			fx.Annotate(
				factoringtransport.NewEmail,
				fx.ResultTags(factoringsubmissionports.TransportGroup),
			),
			fx.Annotate(
				factoringtransport.NewSftp,
				fx.ResultTags(factoringsubmissionports.TransportGroup),
			),
			fx.Annotate(
				factoringservice.NewDocumentObserver,
				fx.ResultTags(documentports.DocumentChangeObserverGroup),
			),
			fx.Annotate(
				billingqueueservice.NewDocumentObserver,
				fx.ResultTags(documentports.DocumentChangeObserverGroup),
			),
		),
		fx.Decorate(factoringservice.WatchInvoices),
	)
}

func APIOptions() fx.Option {
	return fx.Options()
}
