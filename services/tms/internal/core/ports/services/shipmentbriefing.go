package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
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
}

type ShipmentBriefingReader interface {
	Briefing(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		timezone string,
	) (*ShipmentBriefing, error)
}
