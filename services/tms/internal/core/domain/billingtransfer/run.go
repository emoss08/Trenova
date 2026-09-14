package billingtransfer

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const MaxRunShipments = 5000

var (
	_ bun.BeforeAppendModelHook = (*BillingTransferRun)(nil)
	_ bun.BeforeAppendModelHook = (*BillingTransferRunItem)(nil)
)

type BillingTransferRun struct {
	bun.BaseModel `bun:"table:billing_transfer_runs,alias:btr" json:"-"`

	ID                          pulid.ID              `json:"id"                          bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID              pulid.ID              `json:"businessUnitId"              bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID              pulid.ID              `json:"organizationId"              bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RequestedByID               pulid.ID              `json:"requestedById"               bun:"requested_by_id,type:VARCHAR(100),notnull"`
	SourceRunID                 pulid.ID              `json:"sourceRunId"                 bun:"source_run_id,type:VARCHAR(100),nullzero"`
	Status                      RunStatus             `json:"status"                      bun:"status,type:VARCHAR(20),notnull"`
	Scope                       RunScope              `json:"scope"                       bun:"scope,type:VARCHAR(20),notnull"`
	SearchQuery                 string                `json:"searchQuery"                 bun:"search_query,type:VARCHAR(200),nullzero"`
	ShipmentStatus              shipment.Status       `json:"shipmentStatus"              bun:"shipment_status,type:VARCHAR(50),nullzero"`
	BillType                    billingqueue.BillType `json:"billType"                    bun:"bill_type,type:VARCHAR(50),notnull"`
	MarkCompletedReadyToInvoice bool                  `json:"markCompletedReadyToInvoice" bun:"mark_completed_ready_to_invoice,type:BOOLEAN,notnull"`
	TotalCount                  int                   `json:"totalCount"                  bun:"total_count,type:INTEGER,notnull"`
	ProcessedCount              int                   `json:"processedCount"              bun:"processed_count,type:INTEGER,notnull"`
	TransferredCount            int                   `json:"transferredCount"            bun:"transferred_count,type:INTEGER,notnull"`
	NotTransferredCount         int                   `json:"notTransferredCount"         bun:"not_transferred_count,type:INTEGER,notnull"`
	SkippedCount                int                   `json:"skippedCount"                bun:"skipped_count,type:INTEGER,notnull"`
	MarkedReadyToInvoiceCount   int                   `json:"markedReadyToInvoiceCount"   bun:"marked_ready_to_invoice_count,type:INTEGER,notnull"`
	RetryableCount              int                   `json:"retryableCount"              bun:"retryable_count,type:INTEGER,notnull"`
	UnmatchedCount              int                   `json:"unmatchedCount"              bun:"unmatched_count,type:INTEGER,notnull"`
	FailureMessage              string                `json:"failureMessage"              bun:"failure_message,type:TEXT,nullzero"`
	CancelRequestedAt           *int64                `json:"cancelRequestedAt"           bun:"cancel_requested_at,type:BIGINT,nullzero"`
	CancelRequestedByID         pulid.ID              `json:"cancelRequestedById"         bun:"cancel_requested_by_id,type:VARCHAR(100),nullzero"`
	TemporalWorkflowID          string                `json:"temporalWorkflowId"          bun:"temporal_workflow_id,type:VARCHAR(255),nullzero"`
	TemporalRunID               string                `json:"temporalRunId"               bun:"temporal_run_id,type:VARCHAR(255),nullzero"`
	QueuedAt                    int64                 `json:"queuedAt"                    bun:"queued_at,type:BIGINT,notnull"`
	StartedAt                   *int64                `json:"startedAt"                   bun:"started_at,type:BIGINT,nullzero"`
	CompletedAt                 *int64                `json:"completedAt"                 bun:"completed_at,type:BIGINT,nullzero"`
	Version                     int64                 `json:"version"                     bun:"version,type:BIGINT,notnull"`
	CreatedAt                   int64                 `json:"createdAt"                   bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt                   int64                 `json:"updatedAt"                   bun:"updated_at,type:BIGINT,notnull"`

	RequestedBy *tenant.User `json:"requestedBy,omitempty" bun:"rel:belongs-to,join:requested_by_id=id"`
}

func (r *BillingTransferRun) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if r.ID.IsNil() {
			r.ID = pulid.MustNew("btr_")
		}
		r.CreatedAt = now
		r.UpdatedAt = now
		if r.QueuedAt == 0 {
			r.QueuedAt = now
		}
	case *bun.UpdateQuery:
		r.UpdatedAt = now
	}

	return nil
}

func (r *BillingTransferRun) GetID() pulid.ID { return r.ID }

func (r *BillingTransferRun) GetOrganizationID() pulid.ID { return r.OrganizationID }

func (r *BillingTransferRun) GetBusinessUnitID() pulid.ID { return r.BusinessUnitID }

func (r *BillingTransferRun) GetCreatedAt() int64 { return r.CreatedAt }

func (r *BillingTransferRun) GetTableName() string { return "billing_transfer_runs" }

func (r *BillingTransferRun) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "btr",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "status", Type: domaintypes.FieldTypeText},
			{Name: "scope", Type: domaintypes.FieldTypeText},
		},
	}
}

func (r *BillingTransferRun) IsCancelRequested() bool {
	return r.CancelRequestedAt != nil
}

func (r *BillingTransferRun) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&r.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&r.RequestedByID, validation.Required.Error("Requester is required")),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[RunStatus]("Status is invalid"),
		),
		validation.Field(&r.Scope,
			validation.Required.Error("Scope is required"),
			domainvalidation.ValidEnum[RunScope]("Scope is invalid"),
		),
		validation.Field(&r.SourceRunID,
			validation.When(
				r.Scope == RunScopeRetry,
				validation.Required.Error("A retry must name the transfer it retries"),
			),
		),
		validation.Field(&r.ShipmentStatus,
			validation.When(
				r.ShipmentStatus != "",
				validation.By(func(any) error {
					if r.ShipmentStatus.IsBillingTransferCandidate() {
						return nil
					}
					return validation.NewError(
						"invalid_status",
						"Only completed or ready to invoice shipments can be transferred to billing",
					)
				}),
			),
		),
		validation.Field(&r.BillType, validation.Required.Error("Bill type is required")),
		validation.Field(&r.TotalCount,
			validation.Min(1).Error("A transfer must include at least one shipment"),
			validation.Max(MaxRunShipments).
				Error("A transfer can include at most 5000 shipments"),
		),
		validation.Field(&r.SearchQuery,
			validation.Length(0, 200).Error("Search cannot be longer than 200 characters"),
		),
	))
}

type MissingRequirement struct {
	DocumentTypeID   string `json:"documentTypeId"`
	DocumentTypeCode string `json:"documentTypeCode"`
	DocumentTypeName string `json:"documentTypeName"`
}

type ValidationFailure struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type BillingTransferRunItem struct {
	bun.BaseModel `bun:"table:billing_transfer_run_items,alias:btri" json:"-"`

	ID                   pulid.ID             `json:"id"                   bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID       pulid.ID             `json:"businessUnitId"       bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID       pulid.ID             `json:"organizationId"       bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	RunID                pulid.ID             `json:"runId"                bun:"run_id,type:VARCHAR(100),notnull"`
	ShipmentID           pulid.ID             `json:"shipmentId"           bun:"shipment_id,type:VARCHAR(100),notnull"`
	Sequence             int                  `json:"sequence"             bun:"sequence,type:INTEGER,notnull"`
	ProNumber            string               `json:"proNumber"            bun:"pro_number,type:VARCHAR(100),nullzero"`
	Status               ItemStatus           `json:"status"               bun:"status,type:VARCHAR(20),notnull"`
	FailureCode          FailureCode          `json:"failureCode"          bun:"failure_code,type:VARCHAR(50),nullzero"`
	ErrorMessage         string               `json:"errorMessage"         bun:"error_message,type:TEXT,nullzero"`
	MarkedReadyToInvoice bool                 `json:"markedReadyToInvoice" bun:"marked_ready_to_invoice,type:BOOLEAN,notnull"`
	BillingQueueItemID   pulid.ID             `json:"billingQueueItemId"   bun:"billing_queue_item_id,type:VARCHAR(100),nullzero"`
	BillingQueueNumber   string               `json:"billingQueueNumber"   bun:"billing_queue_number,type:VARCHAR(100),nullzero"`
	BillingQueueStatus   billingqueue.Status  `json:"billingQueueStatus"   bun:"billing_queue_status,type:VARCHAR(50),nullzero"`
	MissingRequirements  []MissingRequirement `json:"missingRequirements"  bun:"missing_requirements,type:JSONB,notnull"`
	ValidationFailures   []ValidationFailure  `json:"validationFailures"   bun:"validation_failures,type:JSONB,notnull"`
	ProcessedAt          *int64               `json:"processedAt"          bun:"processed_at,type:BIGINT,nullzero"`
	CreatedAt            int64                `json:"createdAt"            bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt            int64                `json:"updatedAt"            bun:"updated_at,type:BIGINT,notnull"`
}

func (i *BillingTransferRunItem) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if i.ID.IsNil() {
			i.ID = pulid.MustNew("btri_")
		}
		if i.MissingRequirements == nil {
			i.MissingRequirements = []MissingRequirement{}
		}
		if i.ValidationFailures == nil {
			i.ValidationFailures = []ValidationFailure{}
		}
		i.CreatedAt = now
		i.UpdatedAt = now
	case *bun.UpdateQuery:
		i.UpdatedAt = now
	}

	return nil
}

func (i *BillingTransferRunItem) GetID() pulid.ID { return i.ID }

func (i *BillingTransferRunItem) GetOrganizationID() pulid.ID { return i.OrganizationID }

func (i *BillingTransferRunItem) GetBusinessUnitID() pulid.ID { return i.BusinessUnitID }

func (i *BillingTransferRunItem) GetTableName() string { return "billing_transfer_run_items" }

func (i *BillingTransferRunItem) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "btri",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "pro_number", Type: domaintypes.FieldTypeText},
		},
	}
}
