package billingqueuereviewservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueactivity"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueuereadiness"
	"github.com/emoss08/trenova/internal/modules/billing/app/billingqueueservice"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/drivernotificationservice"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Queue        *billingqueueservice.Service
	Repo         billingqueueports.BillingQueueRepository
	ShipmentRepo shipmentports.ShipmentRepository
	CustomerRepo customerports.CustomerRepository
	UserRepo     tenantports.UserRepository
	InvoiceRepo  invoiceports.InvoiceRepository
	InvoiceSvc   invoiceports.InvoiceService
	Realtime     realtimeports.RealtimeService
	// The review reads: issues and activity, the shipment's paperwork and
	// detention evidence, the contract's accessorials, and the driver to ask
	// for a missing signature.
	ReviewRepo      billingqueueports.BillingQueueReviewRepository `optional:"true"`
	DocumentRepo    documentports.DocumentRepository               `optional:"true"`
	OccurrenceRepo  detentionports.DetentionOccurrenceRepository   `optional:"true"`
	AgreementRepo   rateagreementports.RateAgreementRepository     `optional:"true"`
	DriverNotifier  *drivernotificationservice.Service             `optional:"true"`
	DocumentContent documentports.DocumentContentService           `optional:"true"`
}

// Service is the review a biller does from one item: settling and
// undoing its issues, holding and releasing it, approving it when it is
// ready and posting its invoice.
type Service struct {
	l               *zap.Logger
	queue           *billingqueueservice.Service
	repo            billingqueueports.BillingQueueRepository
	shipmentRepo    shipmentports.ShipmentRepository
	customerRepo    customerports.CustomerRepository
	invoiceRepo     invoiceports.InvoiceRepository
	invoiceSvc      invoiceports.InvoiceService
	reviewRepo      billingqueueports.BillingQueueReviewRepository
	documentRepo    documentports.DocumentRepository
	drivers         driverNotifier
	documentContent documentports.DocumentContentService
	activity        *billingqueueactivity.Recorder
	readiness       *billingqueuereadiness.Reviewer
}

var _ billingqueueports.BillingQueueReviewService = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) billingqueueports.BillingQueueReviewService {
	l := p.Logger.Named("service.billing-queue")
	activity := billingqueueactivity.New(billingqueueactivity.Params{
		Logger:     l,
		ReviewRepo: p.ReviewRepo,
		UserRepo:   p.UserRepo,
		Realtime:   p.Realtime,
	})
	return &Service{
		l:               l,
		queue:           p.Queue,
		repo:            p.Repo,
		shipmentRepo:    p.ShipmentRepo,
		customerRepo:    p.CustomerRepo,
		invoiceRepo:     p.InvoiceRepo,
		invoiceSvc:      p.InvoiceSvc,
		reviewRepo:      p.ReviewRepo,
		documentRepo:    p.DocumentRepo,
		drivers:         notifierOrNil(p.DriverNotifier),
		documentContent: p.DocumentContent,
		activity:        activity,
		readiness: billingqueuereadiness.New(billingqueuereadiness.Params{
			Logger:         l,
			Activity:       activity,
			ReviewRepo:     p.ReviewRepo,
			DocumentRepo:   p.DocumentRepo,
			OccurrenceRepo: p.OccurrenceRepo,
			AgreementRepo:  p.AgreementRepo,
			InvoiceRepo:    p.InvoiceRepo,
		}),
	}
}

// driverNotifier is the driver portal's notification path, which is how the
// item asks a driver for paperwork.
type driverNotifier interface {
	Notify(ctx context.Context, req *drivernotificationservice.DriverNotification)
}

func notifierOrNil(svc *drivernotificationservice.Service) driverNotifier {
	if svc == nil {
		return nil
	}

	return svc
}
