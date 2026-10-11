package customerports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
)

var ErrCacheMiss = errors.New("cache miss")

type WorkerCacheRepository interface {
	GetByID(ctx context.Context, req workerports.GetWorkerByIDRequest) (*worker.Worker, error)
}

type CustomerCacheRepository interface {
	GetByID(ctx context.Context, req GetCustomerByIDRequest) (*customer.Customer, error)
}

type ShipmentCacheRepository interface {
	GetByID(
		ctx context.Context,
		req *shipmentports.GetShipmentByIDRequest,
	) (*shipment.Shipment, error)
}

type DocumentCacheRepository interface {
	GetByID(
		ctx context.Context,
		req documentports.GetDocumentByIDRequest,
	) (*document.Document, error)
}

type SearchRepository interface {
	Enabled() bool
	Search(ctx context.Context, req SearchRequest) ([]map[string]any, error)
}

type SearchRequest struct {
	Index  string
	Query  string
	Limit  int
	Filter string
}
