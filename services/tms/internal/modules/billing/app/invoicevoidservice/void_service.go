package invoicevoidservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoicevalidation"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceadjustmentports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/latechargeports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/chargeallocationports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/orderports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/seqgen"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger               *zap.Logger
	AdjustmentService    invoiceadjustmentports.InvoiceAdjustmentService `optional:"true"`
	BillingQueueRepo     billingqueueports.BillingQueueRepository
	ChargeAllocationRepo chargeallocationports.ChargeAllocationRepository
	CustomerRepo         customerports.CustomerRepository
	DB                   kernelports.Transactor
	DetentionBilling     detentionports.DetentionBillingService
	LateChargeRepo       latechargeports.LateChargeRepository `optional:"true"`
	OrderDerivation      orderports.OrderDerivationService
	OrderRepo            orderports.OrderRepository
	Repo                 invoiceports.InvoiceRepository
	SequenceGenerator    seqgen.Generator
	SequenceProvider     seqgen.FormatProvider
	ShipmentRepo         shipmentports.ShipmentRepository
	Validator            *invoicevalidation.Validator
	AuditService         auditports.AuditService
	Realtime             realtimeports.RealtimeService
}

// Service voids invoices and discards drafts sent back for review.
type Service struct {
	adjustmentService    invoiceadjustmentports.InvoiceAdjustmentService
	billingQueueRepo     billingqueueports.BillingQueueRepository
	chargeAllocationRepo chargeallocationports.ChargeAllocationRepository
	customerRepo         customerports.CustomerRepository
	db                   kernelports.Transactor
	detentionBilling     detentionports.DetentionBillingService
	l                    *zap.Logger
	lateChargeRepo       latechargeports.LateChargeRepository
	orderDerivation      orderports.OrderDerivationService
	orderRepo            orderports.OrderRepository
	rec                  *invoiceaction.Recorder
	repo                 invoiceports.InvoiceRepository
	sequenceGenerator    seqgen.Generator
	sequenceProvider     seqgen.FormatProvider
	shipmentRepo         shipmentports.ShipmentRepository
	validator            *invoicevalidation.Validator
}

var _ invoiceports.InvoiceVoidService = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	l := p.Logger.Named("service.invoice")
	return &Service{
		adjustmentService:    p.AdjustmentService,
		billingQueueRepo:     p.BillingQueueRepo,
		chargeAllocationRepo: p.ChargeAllocationRepo,
		customerRepo:         p.CustomerRepo,
		db:                   p.DB,
		detentionBilling:     p.DetentionBilling,
		l:                    l,
		lateChargeRepo:       p.LateChargeRepo,
		orderDerivation:      p.OrderDerivation,
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
