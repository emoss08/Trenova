package shipmentdistanceports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

type JurisdictionMileResult struct {
	CountryCode      string  `json:"countryCode"`
	JurisdictionCode string  `json:"jurisdictionCode"`
	Distance         float64 `json:"distance"`
	DistanceUnits    string  `json:"distanceUnits"`
	Loaded           bool    `json:"loaded"`
}

type DistanceMoveResult struct {
	MoveID              pulid.ID                 `json:"moveId,omitempty"`
	MoveIndex           int                      `json:"moveIndex"`
	Distance            float64                  `json:"distance"`
	Source              string                   `json:"source"`
	Provider            string                   `json:"provider,omitempty"`
	RoutingType         string                   `json:"routingType,omitempty"`
	DataVersion         string                   `json:"dataVersion,omitempty"`
	DistanceUnits       string                   `json:"distanceUnits,omitempty"`
	DistanceProfileID   string                   `json:"distanceProfileId,omitempty"`
	DistanceProfileName string                   `json:"distanceProfileName,omitempty"`
	Warnings            []string                 `json:"warnings,omitempty"`
	JurisdictionMiles   []JurisdictionMileResult `json:"jurisdictionMiles,omitempty"`
	CalculatedAt        int64                    `json:"calculatedAt"`
}

type RecalculateMoveJurisdictionMilesRequest struct {
	TenantInfo     pagination.TenantInfo `json:"-"`
	ShipmentMoveID pulid.ID              `json:"shipmentMoveId"`
	UserID         pulid.ID              `json:"userId"`
}

type MoveJurisdictionMilesPlan struct {
	Move *shipment.ShipmentMove
}

type BackfillJurisdictionMilesRequest struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Start      int64                 `json:"start"`
	End        int64                 `json:"end"`
	MaxMoves   int                   `json:"maxMoves"`
	DryRun     bool                  `json:"dryRun"`
	UserID     pulid.ID              `json:"userId"`
}

type BackfillJurisdictionMilesResult struct {
	Started           bool            `json:"started"`
	DryRun            bool            `json:"dryRun"`
	UnattributedMoves int             `json:"unattributedMoves"`
	UnattributedMiles decimal.Decimal `json:"unattributedMiles"`
	WorkflowID        string          `json:"workflowId,omitempty"`
}

type DistanceCalculationResponse struct {
	ShipmentID    pulid.ID             `json:"shipmentId,omitempty"`
	TotalDistance float64              `json:"totalDistance"`
	Moves         []DistanceMoveResult `json:"moves"`
}

type DistanceCalculationService interface {
	ResolveForShipment(
		ctx context.Context,
		entity *shipment.Shipment,
	) (*DistanceCalculationResponse, error)
	RecalculateShipment(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*DistanceCalculationResponse, error)
	PreviewRecalculateShipment(
		ctx context.Context,
		shipmentID pulid.ID,
		tenantInfo pagination.TenantInfo,
	) (*ShipmentDistancePreview, error)
	PlanMoveJurisdictionMiles(
		ctx context.Context,
		req *RecalculateMoveJurisdictionMilesRequest,
	) (*MoveJurisdictionMilesPlan, error)
	RecalculateMoveJurisdictionMiles(
		ctx context.Context,
		req RecalculateMoveJurisdictionMilesRequest,
	) ([]*shipment.ShipmentMoveJurisdictionMile, error)
	BackfillJurisdictionMiles(
		ctx context.Context,
		req BackfillJurisdictionMilesRequest,
	) (*BackfillJurisdictionMilesResult, error)
}

type ShipmentDistancePreview struct {
	Before   *shipment.Shipment
	After    *shipment.Shipment
	Distance *DistanceCalculationResponse
}
