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

	CoverSheets captureports.CaptureCoverSheetRepository
	Imaging     kernelports.Imaging
	Logger      *zap.Logger
	Pages       captureports.CapturePageRepository
	Support     *capturesupport.Service
}

// Service reads captured pages and assembles an item's pages into one PDF. Every process
// has it: intake reads pages as they arrive and filing assembles what a person files.
type Service struct {
	coverSheets captureports.CaptureCoverSheetRepository
	imaging     kernelports.Imaging
	l           *zap.Logger
	pages       captureports.CapturePageRepository
	support     *capturesupport.Service
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) *Service {
	return &Service{
		coverSheets: p.CoverSheets,
		imaging:     p.Imaging,
		l:           p.Logger.Named("service.capture"),
		pages:       p.Pages,
		support:     p.Support,
	}
}

type BatchProcessorParams struct {
	fx.In

	// Analyzer suggests what a document is and which shipment it belongs to.
	Analyzer      captureports.CaptureAnalyzer
	Batches       captureports.CaptureBatchRepository
	DocumentTypes documenttypeports.DocumentTypeRepository
	Items         captureports.CaptureItemRepository
	Logger        *zap.Logger
	Pages         captureports.CapturePageRepository
	ProfileRepo   captureports.CaptureProfileRepository
	Service       *Service
	// Shipments resolves a reference number the analyzer found to a shipment.
	Shipments inboundmessageports.InboundShipmentFinder
	Support   *capturesupport.Service
}

// BatchProcessor divides a sealed batch into items and suggests where each goes. It runs in
// the worker, which is the process that can read a document's content.
type BatchProcessor struct {
	analyzer      captureports.CaptureAnalyzer
	batches       captureports.CaptureBatchRepository
	documentTypes documenttypeports.DocumentTypeRepository
	items         captureports.CaptureItemRepository
	l             *zap.Logger
	pages         captureports.CapturePageRepository
	profileRepo   captureports.CaptureProfileRepository
	service       *Service
	shipments     inboundmessageports.InboundShipmentFinder
	support       *capturesupport.Service
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func NewBatchProcessor(p BatchProcessorParams) *BatchProcessor {
	return &BatchProcessor{
		analyzer:      p.Analyzer,
		batches:       p.Batches,
		documentTypes: p.DocumentTypes,
		items:         p.Items,
		l:             p.Logger.Named("service.capture-batch"),
		pages:         p.Pages,
		profileRepo:   p.ProfileRepo,
		service:       p.Service,
		shipments:     p.Shipments,
		support:       p.Support,
	}
}
