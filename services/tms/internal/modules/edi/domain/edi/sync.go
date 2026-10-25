package edi

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type ShipmentSyncPolicy string

const (
	ShipmentSyncPolicyManualReview    = ShipmentSyncPolicy("ManualReview")
	ShipmentSyncPolicyAutoOperational = ShipmentSyncPolicy("AutoOperational")
	ShipmentSyncPolicyAutoAllSafe     = ShipmentSyncPolicy("AutoAllSafe")
	ShipmentSyncPolicyReadOnly        = ShipmentSyncPolicy("ReadOnly")
)

type ShipmentLinkStatus string

const (
	ShipmentLinkStatusActive    = ShipmentLinkStatus("Active")
	ShipmentLinkStatusSuspended = ShipmentLinkStatus("Suspended")
	ShipmentLinkStatusClosed    = ShipmentLinkStatus("Closed")
)

type TransferChangeDirection string

const (
	TransferChangeDirectionSourceToTarget = TransferChangeDirection("SourceToTarget")
	TransferChangeDirectionTargetToSource = TransferChangeDirection("TargetToSource")
)

type TransferChangeStatus string

const (
	TransferChangeStatusPendingReview = TransferChangeStatus("PendingReview")
	TransferChangeStatusApplied       = TransferChangeStatus("Applied")
	TransferChangeStatusRejected      = TransferChangeStatus("Rejected")
	TransferChangeStatusFailed        = TransferChangeStatus("Failed")
	TransferChangeStatusIgnored       = TransferChangeStatus("Ignored")
)

type TransferChangeConflictStatus string

const (
	TransferChangeConflictNone     = TransferChangeConflictStatus("None")
	TransferChangeConflictConflict = TransferChangeConflictStatus("Conflict")
	TransferChangeConflictResolved = TransferChangeConflictStatus("Resolved")
)

const (
	TransferChangeTypeShipmentStatus214    = "ShipmentStatus214"
	TransferChangeTypeShipmentLifecycle214 = "ShipmentLifecycle214"
	TransferChangeTypeShipmentCancel214    = "ShipmentCancel214"
)

const (
	InternalEDIMirroredFromEventIDKey = "ediMirroredFromEventId"
	InternalEDIShipmentLinkIDKey      = "ediShipmentLinkId"
)

type ShipmentLink struct {
	keel.Table `keel:"edi_shipment_links"`

	ID                   pulid.ID           `json:"id"                   keel:"id,pk,autoid:edislink_"`
	BusinessUnitID       pulid.ID           `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	SourceOrganizationID pulid.ID           `json:"sourceOrganizationId" keel:"source_organization_id"`
	TargetOrganizationID pulid.ID           `json:"targetOrganizationId" keel:"target_organization_id"`
	SourceShipmentID     pulid.ID           `json:"sourceShipmentId"     keel:"source_shipment_id"`
	TargetShipmentID     pulid.ID           `json:"targetShipmentId"     keel:"target_shipment_id"`
	TenderTransferID     pulid.ID           `json:"tenderTransferId"     keel:"tender_transfer_id"`
	OriginatingMessageID pulid.ID           `json:"originatingMessageId" keel:"originating_message_id,nullzero"`
	SyncPolicy           ShipmentSyncPolicy `json:"syncPolicy"           keel:"sync_policy"`
	FieldOwnership       map[string]string  `json:"fieldOwnership"       keel:"field_ownership,jsonb"`
	Status               ShipmentLinkStatus `json:"status"               keel:"status"`
	Version              int64              `json:"version"              keel:"version,version"`
	CreatedAt            int64              `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64              `json:"updatedAt"            keel:"updated_at,autoupdate"`
}

func (l *ShipmentLink) BeforeUpdate(ctx context.Context) error {
	if err := l.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (l *ShipmentLink) BeforeCreate(ctx context.Context) error {
	if err := l.BeforeSave(ctx); err != nil {
		return err
	}
	if l.SyncPolicy == "" {
		l.SyncPolicy = ShipmentSyncPolicyAutoOperational
	}
	if l.Status == "" {
		l.Status = ShipmentLinkStatusActive
	}

	return nil
}

type TransferChange struct {
	keel.Table `keel:"edi_transfer_changes"`

	ID                    pulid.ID                     `json:"id"                    keel:"id,pk,autoid:editc_"`
	BusinessUnitID        pulid.ID                     `json:"businessUnitId"        keel:"business_unit_id,pk,tenant"`
	ShipmentLinkID        pulid.ID                     `json:"shipmentLinkId"        keel:"shipment_link_id"`
	Direction             TransferChangeDirection      `json:"direction"             keel:"direction"`
	ChangeType            string                       `json:"changeType"            keel:"change_type"`
	Status                TransferChangeStatus         `json:"status"                keel:"status"`
	ConflictStatus        TransferChangeConflictStatus `json:"conflictStatus"        keel:"conflict_status"`
	ConflictReason        string                       `json:"conflictReason"        keel:"conflict_reason,nullzero"`
	IdempotencyKey        string                       `json:"idempotencyKey"        keel:"idempotency_key"`
	SourceShipmentVersion int64                        `json:"sourceShipmentVersion" keel:"source_shipment_version"`
	TargetShipmentVersion int64                        `json:"targetShipmentVersion" keel:"target_shipment_version"`
	Payload               map[string]any               `json:"payload"               keel:"payload,jsonb"`
	Diff                  map[string]any               `json:"diff"                  keel:"diff,jsonb"`
	ReviewedByID          pulid.ID                     `json:"reviewedById"          keel:"reviewed_by_id,nullzero"`
	ReviewedAt            *int64                       `json:"reviewedAt"            keel:"reviewed_at"`
	AppliedByID           pulid.ID                     `json:"appliedById"           keel:"applied_by_id,nullzero"`
	AppliedAt             *int64                       `json:"appliedAt"             keel:"applied_at"`
	FailureReason         string                       `json:"failureReason"         keel:"failure_reason,nullzero"`
	SearchVector          string                       `json:"-"                     keel:"search_vector,tsvector,readonly,noselect"`
	Rank                  string                       `json:"-"                     keel:"rank,computed"`
	Version               int64                        `json:"version"               keel:"version,version"`
	CreatedAt             int64                        `json:"createdAt"             keel:"created_at,autocreate"`
	UpdatedAt             int64                        `json:"updatedAt"             keel:"updated_at,autoupdate"`
}

func (c *TransferChange) BeforeUpdate(ctx context.Context) error {
	if err := c.BeforeSave(ctx); err != nil {
		return err
	}

	return nil
}

func (c *TransferChange) BeforeCreate(ctx context.Context) error {
	if err := c.BeforeSave(ctx); err != nil {
		return err
	}
	if c.Status == "" {
		c.Status = TransferChangeStatusPendingReview
	}
	if c.ConflictStatus == "" {
		c.ConflictStatus = TransferChangeConflictNone
	}

	return nil
}

func (l *ShipmentLink) BeforeSave(_ context.Context) error {
	if l.SyncPolicy == "" {
		l.SyncPolicy = ShipmentSyncPolicyAutoOperational
	}
	if l.Status == "" {
		l.Status = ShipmentLinkStatusActive
	}
	if l.FieldOwnership == nil {
		l.FieldOwnership = DefaultShipmentFieldOwnership()
	}
	return nil
}

func (c *TransferChange) BeforeSave(_ context.Context) error {
	if c.Payload == nil {
		c.Payload = map[string]any{}
	}
	if c.Diff == nil {
		c.Diff = map[string]any{}
	}
	if c.Status == "" {
		c.Status = TransferChangeStatusPendingReview
	}
	if c.ConflictStatus == "" {
		c.ConflictStatus = TransferChangeConflictNone
	}
	return nil
}

func (l *ShipmentLink) GetID() pulid.ID { return l.ID }

func (l *ShipmentLink) GetOrganizationID() pulid.ID { return l.SourceOrganizationID }

func (l *ShipmentLink) GetBusinessUnitID() pulid.ID { return l.BusinessUnitID }

func (l *ShipmentLink) GetTableName() string { return "edi_shipment_links" }

func (l *ShipmentLink) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "esl",
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "source_shipment_id",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "target_shipment_id",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
			{
				Name:   "sync_policy",
				Type:   domaintypes.FieldTypeEnum,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}

func (c *TransferChange) GetID() pulid.ID { return c.ID }

func (c *TransferChange) GetOrganizationID() pulid.ID { return pulid.Nil }

func (c *TransferChange) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *TransferChange) GetTableName() string { return "edi_transfer_changes" }

func (c *TransferChange) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "etc",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "change_type",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
			{Name: "direction", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightC},
			{
				Name:   "conflict_reason",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
		},
	}
}

func DefaultShipmentFieldOwnership() map[string]string {
	return map[string]string{
		"customer":                   "source",
		"billing":                    "source",
		"rating":                     "source",
		"charges":                    "source",
		"originalTenderRequirements": "source",
		"commodities":                "source",
		"tenderReferences":           "source",
		"stopActuals":                "target",
		"dispatchExecution":          "target",
		"equipmentAssignment":        "target",
		"driverAssignment":           "target",
		"operationalNotes":           "target",
		"arrivalsDepartures":         "target",
	}
}

func (v ShipmentSyncPolicy) IsValid() bool {
	switch v {
	case ShipmentSyncPolicyManualReview,
		ShipmentSyncPolicyAutoOperational,
		ShipmentSyncPolicyAutoAllSafe,
		ShipmentSyncPolicyReadOnly:
		return true
	default:
		return false
	}
}

func (v ShipmentLinkStatus) IsValid() bool {
	switch v {
	case ShipmentLinkStatusActive,
		ShipmentLinkStatusSuspended,
		ShipmentLinkStatusClosed:
		return true
	default:
		return false
	}
}

func (v TransferChangeDirection) IsValid() bool {
	switch v {
	case TransferChangeDirectionSourceToTarget,
		TransferChangeDirectionTargetToSource:
		return true
	default:
		return false
	}
}

func (v TransferChangeConflictStatus) IsValid() bool {
	switch v {
	case TransferChangeConflictNone,
		TransferChangeConflictConflict,
		TransferChangeConflictResolved:
		return true
	default:
		return false
	}
}

func (l *ShipmentLink) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(l,
		validation.Field(&l.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&l.SourceOrganizationID,
			validation.Required.Error("Source organization is required"),
		),
		validation.Field(&l.TargetOrganizationID,
			validation.Required.Error("Target organization is required"),
		),
		// The link is what keeps two copies of one load in step; either side
		// missing would leave changes with nowhere to go.
		validation.Field(&l.SourceShipmentID,
			validation.Required.Error("Source shipment is required"),
		),
		validation.Field(&l.TargetShipmentID,
			validation.Required.Error("Target shipment is required"),
			validation.NotIn(l.SourceShipmentID).
				Error("A link cannot join a shipment to itself"),
		),
		validation.Field(&l.TenderTransferID,
			validation.Required.Error("Tender transfer is required"),
		),
		validation.Field(&l.SyncPolicy,
			validation.Required.Error("Sync policy is required"),
			domainvalidation.ValidEnum[ShipmentSyncPolicy]("Sync policy is invalid"),
		),
		validation.Field(&l.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ShipmentLinkStatus]("Status is invalid"),
		),
	))
}

func (c *TransferChange) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&c.ShipmentLinkID,
			validation.Required.Error("Shipment link is required"),
		),
		validation.Field(&c.Direction,
			validation.Required.Error("Direction is required"),
			domainvalidation.ValidEnum[TransferChangeDirection]("Direction is invalid"),
		),
		validation.Field(&c.ChangeType,
			validation.Required.Error("Change type is required"),
			validation.Length(1, maxChangeTypeLength).
				Error("Change type cannot be longer than 100 characters"),
		),
		validation.Field(&c.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[TransferChangeStatus]("Status is invalid"),
		),
		validation.Field(&c.ConflictStatus,
			validation.Required.Error("Conflict status is required"),
			domainvalidation.ValidEnum[TransferChangeConflictStatus](
				"Conflict status is invalid",
			),
		),
		validation.Field(&c.ConflictReason,
			validation.When(
				c.ConflictStatus == TransferChangeConflictConflict,
				validation.Required.Error("A conflict must record what conflicted"),
			),
		),
		// The idempotency key is what stops one change being applied twice when
		// a delivery is retried.
		validation.Field(&c.IdempotencyKey,
			validation.Required.Error("Idempotency key is required"),
			validation.Length(1, maxIdempotencyKeyLength).
				Error("Idempotency key cannot be longer than 255 characters"),
		),
		validation.Field(&c.SourceShipmentVersion,
			validation.Min(int64(0)).Error("Source shipment version cannot be negative"),
		),
		validation.Field(&c.TargetShipmentVersion,
			validation.Min(int64(0)).Error("Target shipment version cannot be negative"),
		),
		validation.Field(&c.ReviewedAt,
			validation.When(c.ReviewedByID.IsNotNil(), validation.Required.Error(
				"A reviewed change must record when it was reviewed",
			)),
		),
		validation.Field(&c.AppliedAt,
			validation.When(c.AppliedByID.IsNotNil(), validation.Required.Error(
				"An applied change must record when it was applied",
			)),
		),
	))
}

const (
	maxChangeTypeLength     = 100
	maxIdempotencyKeyLength = 255
)

func (v TransferChangeStatus) IsValid() bool {
	switch v {
	case TransferChangeStatusPendingReview,
		TransferChangeStatusApplied,
		TransferChangeStatusRejected,
		TransferChangeStatusFailed,
		TransferChangeStatusIgnored:
		return true
	default:
		return false
	}
}
