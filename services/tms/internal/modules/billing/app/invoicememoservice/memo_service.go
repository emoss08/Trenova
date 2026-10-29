//nolint:gocognit // existing legacy workflow/API shape is intentionally kept stable
package invoicememoservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicepostingservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/seqgen"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	AccessorialRepo   accessorialchargeports.AccessorialChargeRepository
	BillingQueueRepo  billingqueueports.BillingQueueRepository
	BillingRepo       tenantports.BillingControlRepository
	CustomerRepo      customerports.CustomerRepository
	DB                kernelports.Transactor
	Posting           *invoicepostingservice.Service
	Repo              invoiceports.InvoiceRepository
	SequenceGenerator seqgen.Generator
	Validator         *invoicevalidation.Validator
	AuditService      auditports.AuditService
	Realtime          realtimeports.RealtimeService
}

// Service raises credit and debit memos against posted invoices.
type Service struct {
	accessorialRepo   accessorialchargeports.AccessorialChargeRepository
	billingQueueRepo  billingqueueports.BillingQueueRepository
	billingRepo       tenantports.BillingControlRepository
	customerRepo      customerports.CustomerRepository
	db                kernelports.Transactor
	l                 *zap.Logger
	posting           *invoicepostingservice.Service
	rec               *invoiceaction.Recorder
	repo              invoiceports.InvoiceRepository
	sequenceGenerator seqgen.Generator
	validator         *invoicevalidation.Validator
}

var _ invoiceports.InvoiceMemoService = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	l := p.Logger.Named("service.invoice")
	return &Service{
		accessorialRepo:  p.AccessorialRepo,
		billingQueueRepo: p.BillingQueueRepo,
		billingRepo:      p.BillingRepo,
		customerRepo:     p.CustomerRepo,
		db:               p.DB,
		l:                l,
		posting:          p.Posting,
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{
				Logger:       l,
				AuditService: p.AuditService,
				Realtime:     p.Realtime,
			},
		),
		repo:              p.Repo,
		sequenceGenerator: p.SequenceGenerator,
		validator:         p.Validator,
	}
}
