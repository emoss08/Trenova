package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*ShipmentControl)(nil)

type ShipmentControl struct {
	bun.BaseModel `bun:"table:shipment_controls,alias:sc" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),pk,notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull" json:"organizationId"`

	// Service Failure Related Fields
	RecordServiceFailures     bool  `json:"recordServiceFailures" bun:"record_service_failures,notnull,default:true"`
	ServiceFailureGracePeriod int64 `json:"serviceFailureGracePeriod" bun:"service_failure_grace_period,notnull,default:10"` // In minutes

	// Delay Shipment Related Fields
	AutoDelayShipments          bool  `json:"autoDelayShipments" bun:"auto_delay_shipments,notnull,default:true"`
	AutoDelayShipmentsThreshold int64 `json:"autoDelayShipmentsThreshold" bun:"auto_delay_shipments_threshold,notnull,default:10"` // In minutes

	// Compliance Controls
	EnforceHOSCompliance bool `json:"enforceHOSCompliance" bun:"enforce_hos_compliance,notnull,default:true"`

	// Detentiion Tracking
	TrackDetentionTime bool  `json:"trackDetentionTime" bun:"track_detention_time,notnull,default:true"`
	DetentionThreshold int64 `json:"detentionThreshold" bun:"detention_threshold,notnull,default:10"` // In minutes

	// Misc....
	CheckForDuplicateBOLs bool `json:"checkForDuplicateBOLs" bun:"check_for_duplicate_bols,notnull,default:true"`

	// Metadata
	Version   int64 `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (sc *ShipmentControl) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if sc.ID.IsNil() {
			sc.ID = pulid.MustNew("sc_")
		}

		sc.CreatedAt = now
	case *bun.UpdateQuery:
		sc.UpdatedAt = now
	}

	return nil
}
