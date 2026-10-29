package captureprocessing

import (
	"github.com/emoss08/trenova/internal/modules/documents/app/capturesupport"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/reference/ports/documenttypeports"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	// Analyzer suggests what a document is and which shipment it belongs to.
	// Without it every item waits for a person to say.
	Analyzer      captureports.CaptureAnalyzer `optional:"true"`
	Batches       captureports.CaptureBatchRepository
	CoverSheets   captureports.CaptureCoverSheetRepository
	DocumentTypes documenttypeports.DocumentTypeRepository
	Imaging       kernelports.Imaging
	Items         captureports.CaptureItemRepository
	Logger        *zap.Logger
	Pages         captureports.CapturePageRepository
	ProfileRepo   captureports.CaptureProfileRepository
	// Shipments resolves a reference number the analyzer found to a shipment.
	Shipments inboundmessageports.InboundShipmentFinder `optional:"true"`
	Support   *capturesupport.Service
}

type Service struct {
	analyzer      captureports.CaptureAnalyzer
	batches       captureports.CaptureBatchRepository
	coverSheets   captureports.CaptureCoverSheetRepository
	documentTypes documenttypeports.DocumentTypeRepository
	imaging       kernelports.Imaging
	items         captureports.CaptureItemRepository
	l             *zap.Logger
	pages         captureports.CapturePageRepository
	profileRepo   captureports.CaptureProfileRepository
	shipments     inboundmessageports.InboundShipmentFinder
	support       *capturesupport.Service
}

func New(p Params) *Service {
	return &Service{
		analyzer:      p.Analyzer,
		batches:       p.Batches,
		coverSheets:   p.CoverSheets,
		documentTypes: p.DocumentTypes,
		imaging:       p.Imaging,
		items:         p.Items,
		l:             p.Logger.Named("service.capture"),
		pages:         p.Pages,
		profileRepo:   p.ProfileRepo,
		shipments:     p.Shipments,
		support:       p.Support,
	}
}
