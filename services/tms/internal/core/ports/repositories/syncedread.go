package repositories

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/worker"
)

var ErrCacheMiss = errors.New("cache miss")

type WorkerCacheRepository interface {
	GetByID(ctx context.Context, req GetWorkerByIDRequest) (*worker.Worker, error)
}

type CustomerCacheRepository interface {
	GetByID(ctx context.Context, req GetCustomerByIDRequest) (*customer.Customer, error)
}

type ShipmentCacheRepository interface {
	GetByID(ctx context.Context, req *GetShipmentByIDRequest) (*shipment.Shipment, error)
}

type DocumentCacheRepository interface {
	GetByID(ctx context.Context, req GetDocumentByIDRequest) (*document.Document, error)
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
