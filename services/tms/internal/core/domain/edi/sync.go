package edi

import (
	"context"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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

type ShipmentLink struct {
	bun.BaseModel `json:"-" bun:"table:edi_shipment_links,alias:esl"`

	ID                   pulid.ID           `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID           `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	SourceOrganizationID pulid.ID           `json:"sourceOrganizationId" bun:"source_organization_id,type:VARCHAR(100),notnull"`
	TargetOrganizationID pulid.ID           `json:"targetOrganizationId" bun:"target_organization_id,type:VARCHAR(100),notnull"`
	SourceShipmentID     pulid.ID           `json:"sourceShipmentId"     bun:"source_shipment_id,type:VARCHAR(100),notnull"`
	TargetShipmentID     pulid.ID           `json:"targetShipmentId"     bun:"target_shipment_id,type:VARCHAR(100),notnull"`
	TenderTransferID     pulid.ID           `json:"tenderTransferId"     bun:"tender_transfer_id,type:VARCHAR(100),notnull"`
	OriginatingMessageID pulid.ID           `json:"originatingMessageId" bun:"originating_message_id,type:VARCHAR(100),nullzero"`
	SyncPolicy           ShipmentSyncPolicy `json:"syncPolicy"           bun:"sync_policy,type:edi_shipment_sync_policy_enum,notnull,default:'AutoOperational'"`
	FieldOwnership       map[string]string  `json:"fieldOwnership"       bun:"field_ownership,type:JSONB,notnull,default:'{}'::jsonb"`
	Status               ShipmentLinkStatus `json:"status"               bun:"status,type:edi_shipment_link_status_enum,notnull,default:'Active'"`
	Version              int64              `json:"version"              bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt            int64              `json:"createdAt"            bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64              `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

type TransferChange struct {
	bun.BaseModel `json:"-" bun:"table:edi_transfer_changes,alias:etc"`

	ID                    pulid.ID                     `json:"id"                    bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID        pulid.ID                     `json:"businessUnitId"        bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	ShipmentLinkID        pulid.ID                     `json:"shipmentLinkId"        bun:"shipment_link_id,type:VARCHAR(100),notnull"`
	Direction             TransferChangeDirection      `json:"direction"             bun:"direction,type:edi_transfer_change_direction_enum,notnull"`
	ChangeType            string                       `json:"changeType"            bun:"change_type,type:VARCHAR(100),notnull"`
	Status                TransferChangeStatus         `json:"status"                bun:"status,type:edi_transfer_change_status_enum,notnull,default:'PendingReview'"`
	ConflictStatus        TransferChangeConflictStatus `json:"conflictStatus"        bun:"conflict_status,type:edi_transfer_change_conflict_status_enum,notnull,default:'None'"`
	ConflictReason        string                       `json:"conflictReason"        bun:"conflict_reason,type:TEXT,nullzero"`
	IdempotencyKey        string                       `json:"idempotencyKey"        bun:"idempotency_key,type:VARCHAR(255),notnull"`
	SourceShipmentVersion int64                        `json:"sourceShipmentVersion" bun:"source_shipment_version,type:BIGINT,notnull"`
	TargetShipmentVersion int64                        `json:"targetShipmentVersion" bun:"target_shipment_version,type:BIGINT,notnull"`
	Payload               map[string]any               `json:"payload"               bun:"payload,type:JSONB,notnull,default:'{}'::jsonb"`
	Diff                  map[string]any               `json:"diff"                  bun:"diff,type:JSONB,notnull,default:'{}'::jsonb"`
	ReviewedByID          pulid.ID                     `json:"reviewedById"          bun:"reviewed_by_id,type:VARCHAR(100),nullzero"`
	ReviewedAt            *int64                       `json:"reviewedAt"            bun:"reviewed_at,type:BIGINT,nullzero"`
	AppliedByID           pulid.ID                     `json:"appliedById"           bun:"applied_by_id,type:VARCHAR(100),nullzero"`
	AppliedAt             *int64                       `json:"appliedAt"             bun:"applied_at,type:BIGINT,nullzero"`
	FailureReason         string                       `json:"failureReason"         bun:"failure_reason,type:TEXT,nullzero"`
	SearchVector          string                       `json:"-"                     bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                  string                       `json:"-"                     bun:"rank,type:VARCHAR(100),scanonly"`
	Version               int64                        `json:"version"               bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt             int64                        `json:"createdAt"             bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64                        `json:"updatedAt"             bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (l *ShipmentLink) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
	if l.SyncPolicy == "" {
		l.SyncPolicy = ShipmentSyncPolicyAutoOperational
	}
	if l.Status == "" {
		l.Status = ShipmentLinkStatusActive
	}
	if l.FieldOwnership == nil {
		l.FieldOwnership = DefaultShipmentFieldOwnership()
	}
	switch query.(type) {
	case *bun.InsertQuery:
		if l.ID.IsNil() {
			l.ID = pulid.MustNew("edislink_")
		}
		l.CreatedAt = now
	case *bun.UpdateQuery:
		l.UpdatedAt = now
	}
	return nil
}

func (c *TransferChange) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()
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
	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("editc_")
		}
		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
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
