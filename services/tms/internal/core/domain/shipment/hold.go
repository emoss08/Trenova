package shipment

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/holdreason"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type HoldSource string

const (
	HoldSourceUser HoldSource = "User"
	HoldSourceRule HoldSource = "Rule"
	HoldSourceAPI  HoldSource = "API"
	HoldSourceELD  HoldSource = "ELD"
	HoldSourceEDI  HoldSource = "EDI"
)

type ShipmentHold struct {
	bun.BaseModel `bun:"table:shipment_holds,alias:shh" json:"-"`

	ID                pulid.ID                `json:"id"                bun:"id,pk,type:VARCHAR(100),notnull"`
	ShipmentID        pulid.ID                `json:"shipmentId"        bun:"shipment_id,type:VARCHAR(100),notnull"`
	BusinessUnitID    pulid.ID                `json:"businessUnitId"    bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID    pulid.ID                `json:"organizationId"    bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	HoldReasonID      *pulid.ID               `json:"holdReasonId"      bun:"hold_reason_id,type:VARCHAR(100),nullzero"`
	Type              holdreason.HoldType     `json:"type"              bun:"type,type:hold_type_enum,notnull"`
	Severity          holdreason.HoldSeverity `json:"severity"          bun:"severity,type:hold_severity_enum,notnull,default:'Advisory'"`
	ReasonCode        string                  `json:"reasonCode"        bun:"reason_code,type:VARCHAR(100),nullzero"`
	Notes             string                  `json:"notes"             bun:"notes,type:TEXT,nullzero"`
	Source            HoldSource              `json:"source"            bun:"source,type:hold_source_enum,notnull,default:'User'"`
	BlocksDispatch    bool                    `json:"blocksDispatch"    bun:"blocks_dispatch,type:BOOLEAN,notnull,default:false"`
	BlocksDelivery    bool                    `json:"blocksDelivery"    bun:"blocks_delivery,type:BOOLEAN,notnull,default:false"`
	BlocksBilling     bool                    `json:"blocksBilling"     bun:"blocks_billing,type:BOOLEAN,notnull,default:false"`
	VisibleToCustomer bool                    `json:"visibleToCustomer" bun:"visible_to_customer,type:BOOLEAN,notnull,default:false"`
	StartedAt         int64                   `json:"startedAt"         bun:"started_at,type:BIGINT,notnull"`
	CreatedAt         int64                   `json:"createdAt"         bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt         int64                   `json:"updatedAt"         bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	Version           int64                   `json:"version"           bun:"version,type:BIGINT,notnull,default:0"`
	ReleasedAt        *int64                  `json:"releasedAt"        bun:"released_at,type:BIGINT,nullzero"`
	CreatedByID       *pulid.ID               `json:"createdById"       bun:"created_by_id,type:VARCHAR(100),nullzero"`
	ReleasedByID      *pulid.ID               `json:"releasedById"      bun:"released_by_id,type:VARCHAR(100),nullzero"`

	HoldReason   *holdreason.HoldReason `json:"holdReason,omitempty"   bun:"rel:belongs-to,join:hold_reason_id=id,join:organization_id=organization_id"`
	CreatedBy    *tenant.User           `json:"createdBy,omitempty"    bun:"rel:belongs-to,join:created_by_id=id"`
	ReleasedBy   *tenant.User           `json:"releasedBy,omitempty"   bun:"rel:belongs-to,join:released_by_id=id"`
	Shipment     *Shipment              `json:"-"                      bun:"rel:belongs-to,join:shipment_id=id"`
	BusinessUnit *tenant.BusinessUnit   `json:"-"                      bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization   `json:"-"                      bun:"rel:belongs-to,join:organization_id=id"`
}

func (h *ShipmentHold) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		h,
		validation.Field(&h.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(&h.OrganizationID, validation.Required.Error("Organization ID is required")),
		validation.Field(&h.BusinessUnitID, validation.Required.Error("Business unit ID is required")),
		validation.Field(&h.Type,
			validation.Required.Error("Hold type is required"),
			validation.In(
				holdreason.HoldTypeOperational,
				holdreason.HoldTypeCompliance,
				holdreason.HoldTypeCustomer,
				holdreason.HoldTypeFinance,
			).Error("Invalid hold type"),
		),
		validation.Field(&h.Severity,
			validation.Required.Error("Severity is required"),
			validation.In(
				holdreason.HoldSeverityInformational,
				holdreason.HoldSeverityAdvisory,
				holdreason.HoldSeverityBlocking,
			).Error("Invalid hold severity"),
		),
		validation.Field(&h.Source,
			validation.Required.Error("Source is required"),
			validation.In(HoldSourceUser, HoldSourceRule, HoldSourceAPI, HoldSourceELD, HoldSourceEDI).
				Error("Invalid hold source"),
		),
		validation.Field(&h.StartedAt,
			validation.Required.Error("Started At is required"),
			validation.Min(int64(1)).Error("Started At must be greater than zero"),
		),
		validation.Field(&h.ReleasedAt,
			validation.By(func(_ any) error {
				if h.ReleasedAt != nil && *h.ReleasedAt < h.StartedAt {
					return errors.New("Released At must be greater than or equal to Started At")
				}
				return nil
			}),
		),
		validation.Field(&h.CreatedByID,
			validation.When(h.Source == HoldSourceUser,
				validation.Required.Error("Created By is required when source is user"),
			),
		),
		validation.Field(&h.BlocksDispatch,
			validation.By(func(_ any) error {
				if h.Severity == holdreason.HoldSeverityBlocking && !(h.BlocksDispatch || h.BlocksDelivery || h.BlocksBilling) {
					return errors.New("Blocking holds must block dispatch, delivery, or billing")
				}
				return nil
			}),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (h *ShipmentHold) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if h.ID.IsNil() {
			h.ID = pulid.MustNew("shh_")
		}
		h.CreatedAt = now
	case *bun.UpdateQuery:
		h.UpdatedAt = now
	}

	return nil
}

func (h *ShipmentHold) IsActive() bool {
	return h != nil && h.ReleasedAt == nil
}

func (h *ShipmentHold) IsBlockedForDispatch() bool {
	return h != nil &&
		h.Severity == holdreason.HoldSeverityBlocking &&
		h.BlocksDispatch &&
		h.ReleasedAt == nil
}

func (h *ShipmentHold) IsBlockedForDelivery() bool {
	return h != nil &&
		h.Severity == holdreason.HoldSeverityBlocking &&
		h.BlocksDelivery &&
		h.ReleasedAt == nil
}

func (h *ShipmentHold) IsBlockedForBilling() bool {
	return h != nil &&
		h.Severity == holdreason.HoldSeverityBlocking &&
		h.BlocksBilling &&
		h.ReleasedAt == nil
}

func (h *ShipmentHold) GetID() pulid.ID {
	return h.ID
}

func (h *ShipmentHold) GetOrganizationID() pulid.ID {
	return h.OrganizationID
}

func (h *ShipmentHold) GetBusinessUnitID() pulid.ID {
	return h.BusinessUnitID
}
