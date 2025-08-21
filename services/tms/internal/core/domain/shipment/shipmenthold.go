package shipment

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*ShipmentHold)(nil)

type ShipmentHold struct {
	bun.BaseModel `bun:"table:shipment_holds,alias:sh" json:"-"`

	ID                pulid.ID       `json:"id"                bun:"id,type:VARCHAR(100),pk,notnull"`
	ShipmentID        pulid.ID       `json:"shipmentId"        bun:"shipment_id,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID       `json:"businessUnitId"    bun:"business_unit_id,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID       `json:"organizationId"    bun:"organization_id,type:VARCHAR(100),notnull"`
	Type              HoldType       `json:"type"              bun:"type,type:hold_type_enum,notnull"`
	Severity          HoldSeverity   `json:"severity"          bun:"severity,type:VARCHAR(32),notnull,default:'Advisory'"`
	ReasonCode        string         `json:"reasonCode"        bun:"reason_code,type:VARCHAR(100),nullzero"` // e.g. ELD_OOS, APPT_PENDING
	Notes             string         `json:"notes"             bun:"notes,type:TEXT,nullzero"`
	Source            HoldSource     `json:"source"            bun:"source,type:hold_source_enum,notnull,default:'User'"`
	BlocksDispatch    bool           `json:"blocksDispatch"    bun:"blocks_dispatch,type:BOOLEAN,notnull,default:false"`
	BlocksDelivery    bool           `json:"blocksDelivery"    bun:"blocks_delivery,type:BOOLEAN,notnull,default:false"`
	BlocksBilling     bool           `json:"blocksBilling"     bun:"blocks_billing,type:BOOLEAN,notnull,default:false"`
	VisibleToCustomer bool           `json:"visibleToCustomer" bun:"visible_to_customer,type:BOOLEAN,notnull,default:false"`
	Metadata          map[string]any `json:"metadata"          bun:"metadata,type:JSONB,nullzero"` // freeform (e.g., apptId, doc refs)
	StartedAt         int64          `json:"startedAt"         bun:"started_at,type:BIGINT,notnull"`
	CreatedAt         int64          `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64          `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	Version           int64          `json:"version"           bun:"version,type:BIGINT"`
	ReleasedAt        *int64         `json:"releasedAt"        bun:"released_at,type:BIGINT,nullzero"`
	CreatedByID       *pulid.ID      `json:"createdById"       bun:"created_by_id,type:VARCHAR(100),nullzero"`
	ReleasedByID      *pulid.ID      `json:"releasedById"      bun:"released_by_id,type:VARCHAR(100),nullzero"`

	CreatedBy  *user.User `json:"createdBy,omitempty"  bun:"rel:belongs-to,join:created_by_id=id"`
	ReleasedBy *user.User `json:"releasedBy,omitempty" bun:"rel:belongs-to,join:released_by_id=id"`
}

func (sh *ShipmentHold) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, sh,
		validation.Field(&sh.Type,
			validation.Required.Error("Hold Type is required"),
			validation.In(
				HoldOperational,
				HoldCompliance,
				HoldCustomer,
				HoldFinance,
			).Error("Hold Type must be a valid value"),
		),
		validation.Field(&sh.Severity,
			validation.Required.Error("Severity is required"),
			validation.In(
				SeverityInfo,
				SeverityAdvisory,
				SeverityBlocking,
			).Error("Severity must be a valid value"),
		),
		validation.Field(&sh.Source,
			validation.Required.Error("Source is required"),
			validation.In(
				SourceUser,
				SourceRule,
				SourceAPI,
				SourceELD,
				SourceEDI,
			).Error("Source must be a valid value"),
		),
		validation.Field(&sh.ReasonCode,
			validation.Length(0, 100).Error("Reason Code must be at most 100 characters"),
		),
		validation.Field(&sh.StartedAt,
			validation.Required.Error("Started At is required"),
			validation.Min(int64(1)).Error("Started At must be a Unix timestamp > 0"),
		),
		validation.Field(&sh.ReleasedAt,
			validation.By(func(value any) error {
				if sh.ReleasedAt != nil && *sh.ReleasedAt < sh.StartedAt {
					return validation.NewError(
						"released_at_lt_started_at",
						"Released At must be >= Started At",
					)
				}
				return nil
			}),
		),
		validation.Field(&sh.CreatedByID,
			validation.When(
				sh.Source == SourceUser,
				validation.Required.Error("Created By is required when Source is user"),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (sh *ShipmentHold) IsBlockedForDispatch() bool {
	return sh.Severity == SeverityBlocking &&
		sh.BlocksDispatch &&
		sh.ReleasedAt == nil
}

func (sh *ShipmentHold) IsBlockedForBilling() bool {
	return sh.Severity == SeverityBlocking &&
		sh.BlocksBilling &&
		sh.ReleasedAt == nil
}

func (sh *ShipmentHold) IsBlockedForDelivery() bool {
	return sh.Severity == SeverityBlocking &&
		sh.BlocksDelivery &&
		sh.ReleasedAt == nil
}

func (sh *ShipmentHold) GetID() string {
	return sh.ID.String()
}

func (sh *ShipmentHold) GetTableName() string {
	return "shipment_holds"
}

func (sh *ShipmentHold) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if sh.ID.IsNil() {
			sh.ID = pulid.MustNew("sh_")
		}

		sh.CreatedAt = now
	case *bun.UpdateQuery:
		sh.UpdatedAt = now
	}

	return nil
}
