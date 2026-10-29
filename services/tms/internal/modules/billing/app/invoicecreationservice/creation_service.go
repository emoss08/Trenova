package invoicecreationservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingcontrolpolicyservice"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/chargeallocationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/accessorialchargeports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/seqgen"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger               *zap.Logger
	AccessorialRepo      accessorialchargeports.AccessorialChargeRepository
	AdjustmentRepo       invoiceadjustmentports.InvoiceAdjustmentRepository
	BillingPolicy        *billingcontrolpolicyservice.Service
	BillingQueueRepo     billingqueueports.BillingQueueRepository
	BillingRepo          tenantports.BillingControlRepository
	ChargeAllocationRepo chargeallocationports.ChargeAllocationRepository
	CustomerRepo         customerports.CustomerRepository
	DB                   kernelports.Transactor
	DetentionBilling     detentionports.DetentionBillingService
	OrderRepo            orderports.OrderRepository
	Repo                 invoiceports.InvoiceRepository
	SequenceGenerator    seqgen.Generator
	SequenceProvider     seqgen.FormatProvider
	ShipmentRepo         shipmentports.ShipmentRepository
	Validator            *invoicevalidation.Validator
	AuditService         auditports.AuditService
	Realtime             realtimeports.RealtimeService
}

// Service creates invoices from approved billing queue items, shipments, orders and
// invoice runs.
type Service struct {
	accessorialRepo      accessorialchargeports.AccessorialChargeRepository
	adjustmentRepo       invoiceadjustmentports.InvoiceAdjustmentRepository
	billingPolicy        *billingcontrolpolicyservice.Service
	billingQueueRepo     billingqueueports.BillingQueueRepository
	billingRepo          tenantports.BillingControlRepository
	chargeAllocationRepo chargeallocationports.ChargeAllocationRepository
	customerRepo         customerports.CustomerRepository
	db                   kernelports.Transactor
	detentionBilling     detentionports.DetentionBillingService
	l                    *zap.Logger
	orderRepo            orderports.OrderRepository
	rec                  *invoiceaction.Recorder
	repo                 invoiceports.InvoiceRepository
	sequenceGenerator    seqgen.Generator
	sequenceProvider     seqgen.FormatProvider
	shipmentRepo         shipmentports.ShipmentRepository
	validator            *invoicevalidation.Validator
}

var _ invoiceports.InvoiceCreationService = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	l := p.Logger.Named("service.invoice")
	return &Service{
		accessorialRepo:      p.AccessorialRepo,
		adjustmentRepo:       p.AdjustmentRepo,
		billingPolicy:        p.BillingPolicy,
		billingQueueRepo:     p.BillingQueueRepo,
		billingRepo:          p.BillingRepo,
		chargeAllocationRepo: p.ChargeAllocationRepo,
		customerRepo:         p.CustomerRepo,
		db:                   p.DB,
		detentionBilling:     p.DetentionBilling,
		l:                    l,
		orderRepo:            p.OrderRepo,
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{
				Logger:       l,
				AuditService: p.AuditService,
				Realtime:     p.Realtime,
			},
		),
		repo:              p.Repo,
		sequenceGenerator: p.SequenceGenerator,
		sequenceProvider:  p.SequenceProvider,
		shipmentRepo:      p.ShipmentRepo,
		validator:         p.Validator,
	}
}
