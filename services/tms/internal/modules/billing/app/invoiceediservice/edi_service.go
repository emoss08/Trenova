//nolint:gocognit // existing legacy workflow/API shape is intentionally kept stable
package invoiceediservice

import (
	"github.com/emoss08/trenova/internal/modules/billing/ports/invoiceports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/edipartnerports"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ladingports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger                      *zap.Logger
	CustomerRepo                customerports.CustomerRepository
	EDICommunicationProfileRepo edipartnerports.EDICommunicationProfileRepository `optional:"true"`
	EDILadingProjects           ladingports.LadingProjectRepository               `optional:"true"`
	EDIPartnerRepo              edipartnerports.EDIPartnerRepository              `optional:"true"`
	Repo                        invoiceports.InvoiceRepository
	WorkflowStarter             kernelports.WorkflowStarter
}

// Service resolves and sends an invoice's outbound 210s.
type Service struct {
	customerRepo                customerports.CustomerRepository
	ediCommunicationProfileRepo edipartnerports.EDICommunicationProfileRepository
	ediLadingProjects           ladingports.LadingProjectRepository
	ediPartnerRepo              edipartnerports.EDIPartnerRepository
	l                           *zap.Logger
	repo                        invoiceports.InvoiceRepository
	workflowStarter             kernelports.WorkflowStarter
}

var _ invoiceports.InvoiceEDIService = (*Service)(nil)

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	l := p.Logger.Named("service.invoice")
	return &Service{
		customerRepo:                p.CustomerRepo,
		ediCommunicationProfileRepo: p.EDICommunicationProfileRepo,
		ediLadingProjects:           p.EDILadingProjects,
		ediPartnerRepo:              p.EDIPartnerRepo,
		l:                           l,
		repo:                        p.Repo,
		workflowStarter:             p.WorkflowStarter,
	}
}
