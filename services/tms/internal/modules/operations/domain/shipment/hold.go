package shipment

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/reference/domain/holdreason"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type ShipmentHold struct {
	keel.Table `keel:"shipment_holds"`

	ID                pulid.ID                `json:"id"                keel:"id,pk,autoid:shh_"`
	ShipmentID        pulid.ID                `json:"shipmentId"        keel:"shipment_id"`
	BusinessUnitID    pulid.ID                `json:"businessUnitId"    keel:"business_unit_id,pk,tenant"`
	OrganizationID    pulid.ID                `json:"organizationId"    keel:"organization_id,pk,tenant"`
	HoldReasonID      *pulid.ID               `json:"holdReasonId"      keel:"hold_reason_id"`
	Type              holdreason.HoldType     `json:"type"              keel:"type"`
	Severity          holdreason.HoldSeverity `json:"severity"          keel:"severity"`
	ReasonCode        string                  `json:"reasonCode"        keel:"reason_code,nullzero"`
	Notes             string                  `json:"notes"             keel:"notes,nullzero"`
	Source            HoldSource              `json:"source"            keel:"source"`
	BlocksDispatch    bool                    `json:"blocksDispatch"    keel:"blocks_dispatch"`
	BlocksDelivery    bool                    `json:"blocksDelivery"    keel:"blocks_delivery"`
	BlocksBilling     bool                    `json:"blocksBilling"     keel:"blocks_billing"`
	VisibleToCustomer bool                    `json:"visibleToCustomer" keel:"visible_to_customer"`
	StartedAt         int64                   `json:"startedAt"         keel:"started_at"`
	CreatedAt         int64                   `json:"createdAt"         keel:"created_at,autocreate"`
	UpdatedAt         int64                   `json:"updatedAt"         keel:"updated_at,autoupdate"`
	Version           int64                   `json:"version"           keel:"version,version"`
	ReleasedAt        *int64                  `json:"releasedAt"        keel:"released_at"`
	CreatedByID       *pulid.ID               `json:"createdById"       keel:"created_by_id"`
	ReleasedByID      *pulid.ID               `json:"releasedById"      keel:"released_by_id"`

	HoldReason   *holdreason.HoldReason `json:"holdReason,omitempty" keel:"belongs:hold_reason_id"`
	CreatedBy    *tenant.User           `json:"createdBy,omitempty"  keel:"belongs:created_by_id"`
	ReleasedBy   *tenant.User           `json:"releasedBy,omitempty" keel:"belongs:released_by_id"`
	Shipment     *Shipment              `json:"-"                    keel:"belongs:shipment_id"`
	BusinessUnit *tenant.BusinessUnit   `json:"-"`
	Organization *tenant.Organization   `json:"-"`
}

func (h *ShipmentHold) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		h,
		validation.Field(&h.ShipmentID, validation.Required.Error("Shipment ID is required")),
		validation.Field(
			&h.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&h.BusinessUnitID,
			validation.Required.Error("Business unit ID is required"),
		),
		validation.Field(
			&h.Type,
			validation.Required.Error("Hold type is required"),
			validation.In(
				holdreason.HoldTypeOperational,
				holdreason.HoldTypeCompliance,
				holdreason.HoldTypeCustomer,
				holdreason.HoldTypeFinance,
			).Error("Invalid hold type"),
		),
		validation.Field(
			&h.Severity,
			validation.Required.Error("Severity is required"),
			validation.In(
				holdreason.HoldSeverityInformational,
				holdreason.HoldSeverityAdvisory,
				holdreason.HoldSeverityBlocking,
			).Error("Invalid hold severity"),
		),
		validation.Field(
			&h.Source,
			validation.Required.Error("Source is required"),
			validation.In(HoldSourceUser, HoldSourceRule, HoldSourceAPI, HoldSourceELD, HoldSourceEDI).
				Error("Invalid hold source"),
		),
		validation.Field(
			&h.StartedAt,
			validation.Required.Error("Started At is required"),
			validation.Min(int64(1)).Error("Started At must be greater than zero"),
		),
		validation.Field(
			&h.ReleasedAt,
			validation.By(func(_ any) error {
				if h.ReleasedAt != nil && *h.ReleasedAt < h.StartedAt {
					return errors.New("released at must be greater than or equal to started at")
				}
				return nil
			}),
		),
		validation.Field(
			&h.CreatedByID,
			validation.When(
				h.Source == HoldSourceUser,
				validation.Required.Error("Created By is required when source is user"),
			),
		),
		validation.Field(
			&h.BlocksDispatch,
			validation.By(func(_ any) error {
				if h.Severity == holdreason.HoldSeverityBlocking &&
					(!h.BlocksDispatch && !h.BlocksDelivery && !h.BlocksBilling) {
					return errors.New("blocking holds must block dispatch, delivery, or billing")
				}
				return nil
			}),
		),
	))
}

func (h *ShipmentHold) BeforeCreate(ctx context.Context) error {
	if h.Severity == "" {
		h.Severity = holdreason.HoldSeverityAdvisory
	}
	if h.Source == "" {
		h.Source = HoldSourceUser
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

func (ShipmentHold) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_shipment_holds_shipment",
			Message: "Shipment does not exist in your organization",
		},
		{
			Name:    "ux_shipment_holds_active_by_type",
			Message: "Shipment already has an active hold of this type",
		},
	}
}
