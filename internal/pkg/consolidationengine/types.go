/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package consolidationengine

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/consolidation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/shopspring/decimal"
)

type EstimatedSavings struct {
	DistanceSavings decimal.Decimal `json:"distanceSavings"` // miles saved
	TimeSavings     int64           `json:"timeSavings"`     // minutes saved
	CostSavings     decimal.Decimal `json:"costSavings"`     // estimated $ saved (optional)
}

type ConsolidationOpportunity struct {
	ID               string               `json:"id"`
	ShipmentIDs      []string             `json:"shipmentIds"`
	Shipments        []*shipment.Shipment `json:"shipments"`
	Score            float64              `json:"score"`
	EstimatedSavings EstimatedSavings     `json:"estimatedSavings"`
	Constraints      []string             `json:"constraints"` // Why these can be consolidated
	Warnings         []string             `json:"warnings"`    // Potential issues to review
	CreatedAt        int64                `json:"createdAt"`
}

type FindConsolidationRequest struct {
	BuID   pulid.ID `json:"buId"`
	OrgID  pulid.ID `json:"orgId"`
	UserID pulid.ID `json:"userId"`
}

type Engine interface {
	List(
		ctx context.Context,
		req *repositories.ListConsolidationRequest,
	) (*ports.ListResult[*consolidation.ConsolidationGroup], error)
	FindConsolidationOpportunities(
		ctx context.Context,
		req FindConsolidationRequest,
	) ([]*ConsolidationOpportunity, error)
}
