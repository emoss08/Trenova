package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/shipmentbrief"
	"github.com/emoss08/trenova/pkg/pagination"
)

type ShipmentBriefingSegment struct {
	Text   string                `json:"text"`
	Filter *shipment.QuickFilter `json:"filter,omitempty"`
}

type ShipmentBriefing struct {
	Segments    []ShipmentBriefingSegment `json:"segments"`
	Narrated    bool                      `json:"narrated"`
	GeneratedAt int64                     `json:"generatedAt"`
	Generation  int                       `json:"generation"`
}

type ShipmentBriefingReader interface {
	Briefing(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		timezone string,
	) (*ShipmentBriefing, error)
}

type WriteShipmentBriefRequest struct {
	TenantInfo pagination.TenantInfo
	Trigger    shipmentbrief.Trigger
}

type ShipmentBriefWriter interface {
	WriteBrief(ctx context.Context, req *WriteShipmentBriefRequest) (*shipmentbrief.Brief, error)
}

type ShipmentSuggestionCandidates interface {
	Candidates(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		timezone string,
	) ([]*ShipmentSuggestion, error)
}
