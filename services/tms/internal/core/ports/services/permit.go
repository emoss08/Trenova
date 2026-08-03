package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/jurisdictionrule"
	"github.com/emoss08/trenova/internal/core/domain/permit"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

// AssessedMeasurements is the load as it was presented to every jurisdiction on
// the route. Returning it alongside the requirements means the panel renders
// the same numbers the derivation used rather than recomputing them from the
// commodity rows and risking a quiet disagreement with the server.
type AssessedMeasurements struct {
	WidthFeet    float64 `json:"widthFeet"`
	HeightFeet   float64 `json:"heightFeet"`
	LengthFeet   float64 `json:"lengthFeet"`
	WeightPounds int64   `json:"weightPounds"`
}

func (m AssessedMeasurements) IsZero() bool {
	return m.WidthFeet == 0 && m.HeightFeet == 0 && m.LengthFeet == 0 && m.WeightPounds == 0
}

// AssessedJurisdiction is one state on the route with its resolved limits and
// the load's headroom against them. Every jurisdiction is returned, including
// the ones the load clears, because "you have two inches of width headroom in
// Tennessee" is the warning that prevents the next load from being a surprise.
// Systems that only report states already exceeded give an operator no way to
// see how close a legal load runs.
type AssessedJurisdiction struct {
	StateID           pulid.ID                           `json:"stateId"`
	StateCode         string                             `json:"stateCode"`
	StateName         string                             `json:"stateName"`
	Sequence          int16                              `json:"sequence"`
	VerificationState jurisdictionrule.VerificationState `json:"verificationState"`
	SourceNote        string                             `json:"sourceNote,omitempty"`
	SourceURL         string                             `json:"sourceUrl,omitempty"`

	MaxWidthFeet    float64 `json:"maxWidthFeet"`
	MaxHeightFeet   float64 `json:"maxHeightFeet"`
	MaxLengthFeet   float64 `json:"maxLengthFeet"`
	MaxWeightPounds int64   `json:"maxWeightPounds"`

	// Headroom is the limit minus the measured value. A negative number is an
	// exceedance, so the same field carries both "how much room is left" and
	// "how far over" without the client needing a second shape.
	WidthHeadroomFeet    float64 `json:"widthHeadroomFeet"`
	HeightHeadroomFeet   float64 `json:"heightHeadroomFeet"`
	LengthHeadroomFeet   float64 `json:"lengthHeadroomFeet"`
	WeightHeadroomPounds int64   `json:"weightHeadroomPounds"`

	RequiresPermit bool                               `json:"requiresPermit"`
	Restrictions   []jurisdictionrule.RestrictionKind `json:"restrictions,omitempty"`
}

// PermitAssessment is the answer to "what does this load need, where, and by
// when" for one shipment.
type PermitAssessment struct {
	Measurements           AssessedMeasurements   `json:"measurements"`
	Jurisdictions          []AssessedJurisdiction `json:"jurisdictions"`
	Requirements           []*permit.Requirement  `json:"requirements"`
	Open                   []*permit.Requirement  `json:"open"`
	ExpiringBeforeDelivery []*permit.Requirement  `json:"expiringBeforeDelivery"`
	TotalEscorts           int                    `json:"totalEscorts"`
	TotalEstimatedFee      decimal.Decimal        `json:"totalEstimatedFee"`
	MaxLeadTimeDays        int16                  `json:"maxLeadTimeDays"`
	EarliestPickup         int64                  `json:"earliestPickup"`
	// RouteResolved distinguishes "this route needs no permits" from "we could
	// not tell which states this route crosses". Both produce an empty
	// requirement list, and conflating them would let an unmapped route read as
	// a clean one.
	RouteResolved bool `json:"routeResolved"`
	// FeeIsBaseOnly records that per-mile permit fees were not included because
	// per-jurisdiction mileage is unavailable. Presenting a mileage-derived fee
	// without state-level distance would be fabricated precision on a number
	// carriers bill from.
	FeeIsBaseOnly bool `json:"feeIsBaseOnly"`
}

func (a *PermitAssessment) HasOpen() bool {
	return a != nil && len(a.Open) > 0
}

func (a *PermitAssessment) RequiresEscorts() bool {
	return a != nil && a.TotalEscorts > 0
}

type WaiveRequirementRequest struct {
	TenantInfo    pagination.TenantInfo
	RequirementID pulid.ID
	WaivedByID    pulid.ID
	Reason        string
}

type PermitService interface {
	// Assess derives requirements without writing, so validation can call it.
	Assess(ctx context.Context, entity *shipment.Shipment) (*PermitAssessment, error)

	// Sync persists the derivation, reconciles the dispatch hold, and emits
	// derived charges when the profile enables them.
	Sync(
		ctx context.Context,
		entity *shipment.Shipment,
		actor *RequestActor,
	) (*PermitAssessment, error)

	ListPermits(ctx context.Context, shipmentID pulid.ID, tenantInfo pagination.TenantInfo) (
		[]*permit.Permit, error,
	)

	CreatePermit(ctx context.Context, entity *permit.Permit) (*permit.Permit, error)

	UpdatePermit(ctx context.Context, entity *permit.Permit) (*permit.Permit, error)

	WaiveRequirement(
		ctx context.Context,
		req *WaiveRequirementRequest,
	) (*permit.Requirement, error)
}
