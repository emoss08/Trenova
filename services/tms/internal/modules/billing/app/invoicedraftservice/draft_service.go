package invoicedraftservice

import (
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/billing/app/invoiceaction"
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         invoiceports.InvoiceRepository
	AuditService auditports.AuditService
	Realtime     realtimeports.RealtimeService
}

// Service edits a draft invoice's delivery details.
type Service struct {
	rec  *invoiceaction.Recorder
	repo invoiceports.InvoiceRepository
}

var _ invoiceports.InvoiceDraftService = (*Service)(nil)

func New(p Params) *Service {
	l := p.Logger.Named("service.invoice")
	return &Service{
		rec: invoiceaction.NewRecorder(
			invoiceaction.RecorderParams{
				Logger:       l,
				AuditService: p.AuditService,
				Realtime:     p.Realtime,
			},
		),
		repo: p.Repo,
	}
}
