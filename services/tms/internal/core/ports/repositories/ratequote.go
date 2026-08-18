package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/domain/ratequote"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetRateQuoteByIDRequest struct {
	RateQuoteID pulid.ID              `json:"rateQuoteId"`
	TenantInfo  pagination.TenantInfo `json:"-"`
}

type ListRateQuotesRequest struct {
	Filter *pagination.QueryOptions `json:"filter"`
}

type ListRateQuoteConnectionRequest struct {
	Filter           *pagination.QueryOptions `json:"filter"`
	Cursor           pagination.CursorInfo    `json:"-"`
	RateQuoteColumns []string                 `json:"-"`
}

type GetShipmentRateQuoteRequest struct {
	ShipmentID pulid.ID                `json:"shipmentId"`
	PartyType  rateagreement.PartyType `json:"partyType"`
	TenantInfo pagination.TenantInfo   `json:"-"`
}

type ListShipmentRateQuotesRequest struct {
	ShipmentID pulid.ID              `json:"shipmentId"`
	TenantInfo pagination.TenantInfo `json:"-"`
	Limit      int                   `json:"limit"`
}

type RateQuoteRepository interface {
	GetByID(ctx context.Context, req *GetRateQuoteByIDRequest) (*ratequote.RateQuote, error)
	List(
		ctx context.Context,
		req *ListRateQuotesRequest,
	) (*pagination.ListResult[*ratequote.RateQuote], error)
	ListConnection(
		ctx context.Context,
		req *ListRateQuoteConnectionRequest,
	) (*pagination.CursorListResult[*ratequote.RateQuote], error)

	// GetAppliedForShipment returns the quote currently governing a shipment.
	GetAppliedForShipment(
		ctx context.Context,
		req *GetShipmentRateQuoteRequest,
	) (*ratequote.RateQuote, error)
	// ListForShipment returns a shipment's rating history, newest first.
	ListForShipment(
		ctx context.Context,
		req *ListShipmentRateQuotesRequest,
	) ([]*ratequote.RateQuote, error)

	// Record writes a quote and, when it governs a shipment, supersedes the one
	// it replaces. The two happen together so a shipment is never left with two
	// applied quotes or none.
	Record(ctx context.Context, entity *ratequote.RateQuote) (*ratequote.RateQuote, error)
}
