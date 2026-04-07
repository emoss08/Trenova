package billingqueue

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*BillingQueueItem)(nil)
	_ validationframework.TenantedEntity = (*BillingQueueItem)(nil)
	_ domaintypes.PostgresSearchable     = (*BillingQueueItem)(nil)
)

type BillingQueueItem struct {
	bun.BaseModel `bun:"table:billing_queue_items,alias:bqi" json:"-"`

	ID                  pulid.ID             `json:"id"                  bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID      pulid.ID             `json:"organizationId"      bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID             `json:"businessUnitId"      bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	ShipmentID          pulid.ID             `json:"shipmentId"          bun:"shipment_id,type:VARCHAR(100),notnull"`
	AssignedBillerID    *pulid.ID            `json:"assignedBillerId"    bun:"assigned_biller_id,type:VARCHAR(100),nullzero"`
	Status              Status               `json:"status"              bun:"status,type:billing_queue_status,notnull,default:'ReadyForReview'"`
	BillType            BillType             `json:"billType"            bun:"bill_type,type:billing_type,notnull,default:'Invoice'"`
	ExceptionReasonCode *ExceptionReasonCode `json:"exceptionReasonCode" bun:"exception_reason_code,type:VARCHAR(50),nullzero"`
	ReviewNotes         string               `json:"reviewNotes"         bun:"review_notes,type:TEXT,nullzero"`
	ExceptionNotes      string               `json:"exceptionNotes"      bun:"exception_notes,type:TEXT,nullzero"`
	ReviewStartedAt     *int64               `json:"reviewStartedAt"     bun:"review_started_at,type:BIGINT,nullzero"`
	ReviewCompletedAt   *int64               `json:"reviewCompletedAt"   bun:"review_completed_at,type:BIGINT,nullzero"`
	CanceledByID        *pulid.ID            `json:"canceledById"        bun:"canceled_by_id,type:VARCHAR(100),nullzero"`
	CanceledAt          *int64               `json:"canceledAt"          bun:"canceled_at,type:BIGINT,nullzero"`
	CancelReason        string               `json:"cancelReason"        bun:"cancel_reason,type:VARCHAR(100),nullzero"`
	Version             int64                `json:"version"             bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt           int64                `json:"createdAt"           bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64                `json:"updatedAt"           bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Shipment       *shipment.Shipment `json:"shipment,omitempty"       bun:"rel:belongs-to,join:shipment_id=id,join:organization_id=organization_id,join:business_unit_id=business_unit_id"`
	AssignedBiller *tenant.User       `json:"assignedBiller,omitempty" bun:"rel:belongs-to,join:assigned_biller_id=id"`
	CanceledBy     *tenant.User       `json:"canceledBy,omitempty"     bun:"rel:belongs-to,join:canceled_by_id=id"`
}

func (b *BillingQueueItem) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		b,
		validation.Field(
			&b.ShipmentID,
			validation.Required.Error("Shipment ID is required"),
		),
		validation.Field(
			&b.OrganizationID,
			validation.Required.Error("Organization ID is required"),
		),
		validation.Field(
			&b.BusinessUnitID,
			validation.Required.Error("Business unit ID is required"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	b.validateStatusConstraints(multiErr)
}

func (b *BillingQueueItem) validateStatusConstraints(multiErr *errortypes.MultiError) {
	switch b.Status {
	case StatusInReview:
		if b.AssignedBillerID == nil || b.AssignedBillerID.IsNil() {
			multiErr.Add(
				"assignedBillerId",
				errortypes.ErrRequired,
				"Assigned biller is required when status is InReview",
			)
		}
	case StatusSentBackToOps, StatusException:
		if b.ExceptionReasonCode == nil {
			multiErr.Add(
				"exceptionReasonCode",
				errortypes.ErrRequired,
				"Exception reason code is required",
			)
		} else if !b.ExceptionReasonCode.IsValid() {
			multiErr.Add("exceptionReasonCode", errortypes.ErrInvalid, "Invalid exception reason code")
		}

		notesRequired := b.Status == StatusException ||
			(b.ExceptionReasonCode != nil && *b.ExceptionReasonCode == ExceptionOther)
		if notesRequired && b.ExceptionNotes == "" {
			multiErr.Add("exceptionNotes", errortypes.ErrRequired, "Exception notes are required")
		}
	case StatusCanceled:
		if b.CanceledByID == nil || b.CanceledByID.IsNil() {
			multiErr.Add(
				"canceledById",
				errortypes.ErrRequired,
				"Canceled by is required when status is Canceled",
			)
		}
		if b.CancelReason == "" {
			multiErr.Add(
				"cancelReason",
				errortypes.ErrRequired,
				"Cancel reason is required when status is Canceled",
			)
		}
	}
}

func (b *BillingQueueItem) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if b.ID.IsNil() {
			b.ID = pulid.MustNew("bqi_")
		}
		b.CreatedAt = now
	case *bun.UpdateQuery:
		b.UpdatedAt = now
	}

	return nil
}

func (b *BillingQueueItem) GetID() pulid.ID {
	return b.ID
}

func (b *BillingQueueItem) GetOrganizationID() pulid.ID {
	return b.OrganizationID
}

func (b *BillingQueueItem) GetBusinessUnitID() pulid.ID {
	return b.BusinessUnitID
}

func (b *BillingQueueItem) GetTableName() string {
	return "billing_queue_items"
}

func (b *BillingQueueItem) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "bqi",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
			{
				Name:   "review_notes",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "exception_notes",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "cancel_reason",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:        "shipment",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*shipment.Shipment)(nil),
				TargetTable:  "shipments",
				ForeignKey:   "shipment_id",
				ReferenceKey: "id",
				Alias:        "sp",
				Queryable:    true,
			},
			{
				Field:        "assigned_biller",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*tenant.User)(nil),
				TargetTable:  "users",
				ForeignKey:   "assigned_biller_id",
				ReferenceKey: "id",
				Alias:        "usr",
				Queryable:    true,
			},
		},
	}
}
