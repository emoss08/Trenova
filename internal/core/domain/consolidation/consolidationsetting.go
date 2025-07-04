package consolidation

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*ConsolidationSettings)(nil)
	_ domain.Validatable        = (*ConsolidationSettings)(nil)
)

//nolint:revive // valid struct name
type ConsolidationSettings struct {
	bun.BaseModel `bun:"table:consolidation_settings,alias:cs"`

	ID             pulid.ID `json:"id"             bun:",pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`

	// Maximum distance in miles between pickup locations for shipments to be considered for consolidation
	// Example: If set to 25, only shipments with pickups within 25 miles of each other will be grouped
	MaxPickupDistance float64 `json:"maxPickupDistance" bun:"max_pickup_distance,type:FLOAT,notnull,default:25"`

	// Maximum distance in miles between delivery locations for shipments to be considered for consolidation
	// Example: If set to 25, only shipments with deliveries within 25 miles of each other will be grouped
	MaxDeliveryDistance float64 `json:"maxDeliveryDistance" bun:"max_delivery_distance,type:FLOAT,notnull,default:25"`

	// Maximum percentage increase in total route distance that's acceptable for consolidation
	// Example: If set to 15, consolidation is only suggested if the combined route is max 15% longer than separate routes
	MaxRouteDetour float64 `json:"maxRouteDetour" bun:"max_route_detour,type:FLOAT,notnull,default:15"`

	// Maximum time gap in minutes between shipments' planned pickup/delivery windows for consolidation
	// Example: If set to 240 (4 hours), shipments must have overlapping or close time windows within 4 hours
	MaxTimeWindowGap int64 `json:"maxTimeWindowGap" bun:"max_time_window_gap,type:BIGINT,notnull,default:240"`

	// Minimum time buffer in minutes required between stops when consolidating shipments
	// Example: If set to 30, there must be at least 30 minutes between planned departure and next pickup/delivery
	MinTimeBuffer int64 `json:"minTimeBuffer" bun:"min_time_buffer,type:BIGINT,notnull,default:30"`

	// Maximum number of shipments that can be consolidated into a single group
	// Example: If set to 3, the system will never suggest consolidating more than 3 shipments together
	MaxShipmentsPerGroup int `json:"maxShipmentsPerGroup" bun:"max_shipments_per_group,type:INTEGER,notnull,default:3"`

	// Metadata
	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (cs *ConsolidationSettings) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(
		ctx,
		cs,
		validation.Field(
			&cs.MaxPickupDistance,
			validation.Required.Error("Max. pickup distance is required"),
			validation.Min(float64(1)).Error("Max. pickup distance must be greater or equal to 1"),
		),
		validation.Field(
			&cs.MaxDeliveryDistance,
			validation.Required.Error("Max. delivery distance is required"),
			validation.Min(float64(1)).
				Error("Max. delivery distance must be greater or equal to 1"),
		),
		validation.Field(&cs.MaxRouteDetour,
			validation.Required.Error("Max. route detour is required"),
			validation.Min(float64(1)).Error("Max. router detour must be greater or equal to 1"),
		),
		validation.Field(&cs.MaxTimeWindowGap,
			validation.Required.Error("Max. time window gap is required"),
			validation.Min(1).Error("Max. time window gap must be greater or equal to 1"),
		),
		validation.Field(&cs.MinTimeBuffer,
			validation.Required.Error("Min. time buffer is required"),
			validation.Min(1).Error("Min. Time buffer must be greater or equal to 1"),
		),
		validation.Field(
			&cs.MaxShipmentsPerGroup,
			validation.Required.Error("Max. shipments per group is required"),
			validation.Min(1).
				Error("Max. shipments per group must be greater or equal to 1"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (cs *ConsolidationSettings) GetID() string {
	return cs.ID.String()
}

func (cs *ConsolidationSettings) GetTableName() string {
	return "consolidation_settings"
}

func (cs *ConsolidationSettings) GetVersion() int64 {
	return cs.Version
}

func (cs *ConsolidationSettings) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if cs.ID.IsNil() {
			cs.ID = pulid.MustNew("cs_")
		}

		cs.CreatedAt = now
	case *bun.UpdateQuery:
		cs.UpdatedAt = now
	}

	return nil
}
