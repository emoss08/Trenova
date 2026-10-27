package billingtransfer

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const MaxRunShipments = 5000

type BillingTransferRun struct {
	keel.Table `keel:"billing_transfer_runs"`

	ID                          pulid.ID              `json:"id"                          keel:"id,pk,autoid:btr_"`
	BusinessUnitID              pulid.ID              `json:"businessUnitId"              keel:"business_unit_id,pk,tenant"`
	OrganizationID              pulid.ID              `json:"organizationId"              keel:"organization_id,pk,tenant"`
	RequestedByID               pulid.ID              `json:"requestedById"               keel:"requested_by_id"`
	SourceRunID                 pulid.ID              `json:"sourceRunId"                 keel:"source_run_id,nullzero"`
	Status                      RunStatus             `json:"status"                      keel:"status"`
	Scope                       RunScope              `json:"scope"                       keel:"scope"`
	SearchQuery                 string                `json:"searchQuery"                 keel:"search_query,nullzero"`
	ShipmentStatus              shipment.Status       `json:"shipmentStatus"              keel:"shipment_status,nullzero"`
	BillType                    billingqueue.BillType `json:"billType"                    keel:"bill_type"`
	MarkCompletedReadyToInvoice bool                  `json:"markCompletedReadyToInvoice" keel:"mark_completed_ready_to_invoice"`
	TotalCount                  int                   `json:"totalCount"                  keel:"total_count"`
	ProcessedCount              int                   `json:"processedCount"              keel:"processed_count"`
	TransferredCount            int                   `json:"transferredCount"            keel:"transferred_count"`
	NotTransferredCount         int                   `json:"notTransferredCount"         keel:"not_transferred_count"`
	SkippedCount                int                   `json:"skippedCount"                keel:"skipped_count"`
	MarkedReadyToInvoiceCount   int                   `json:"markedReadyToInvoiceCount"   keel:"marked_ready_to_invoice_count"`
	RetryableCount              int                   `json:"retryableCount"              keel:"retryable_count"`
	UnmatchedCount              int                   `json:"unmatchedCount"              keel:"unmatched_count"`
	FailureMessage              string                `json:"failureMessage"              keel:"failure_message,nullzero"`
	CancelRequestedAt           *int64                `json:"cancelRequestedAt"           keel:"cancel_requested_at"`
	CancelRequestedByID         pulid.ID              `json:"cancelRequestedById"         keel:"cancel_requested_by_id,nullzero"`
	TemporalWorkflowID          string                `json:"temporalWorkflowId"          keel:"temporal_workflow_id,nullzero"`
	TemporalRunID               string                `json:"temporalRunId"               keel:"temporal_run_id,nullzero"`
	QueuedAt                    int64                 `json:"queuedAt"                    keel:"queued_at"`
	StartedAt                   *int64                `json:"startedAt"                   keel:"started_at"`
	CompletedAt                 *int64                `json:"completedAt"                 keel:"completed_at"`
	Version                     int64                 `json:"version"                     keel:"version,version"`
	CreatedAt                   int64                 `json:"createdAt"                   keel:"created_at,autocreate"`
	UpdatedAt                   int64                 `json:"updatedAt"                   keel:"updated_at,autoupdate"`

	RequestedBy *tenant.User `json:"requestedBy,omitempty" keel:"belongs:requested_by_id"`
}

func (r *BillingTransferRun) BeforeCreate(ctx context.Context) error {
	now := timeutils.NowUnix()
	if r.QueuedAt == 0 {
		r.QueuedAt = now
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
			validation.Min(0).Error("Total cannot be negative"),
			validation.Max(MaxRunShipments).
				Error("A transfer can include at most 5000 shipments"),
			// An AllMatching run is created before anything is counted: the
			// workflow resolves its candidates. Only a run that named its
			// shipments up front can be judged empty at this point.
			validation.When(r.Scope == RunScopeSelected || r.Scope == RunScopeAutomatic,
				validation.Min(1).Error("A transfer must include at least one shipment"),
			),
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
	keel.Table `keel:"billing_transfer_run_items"`

	ID                   pulid.ID             `json:"id"                   keel:"id,pk,autoid:btri_"`
	BusinessUnitID       pulid.ID             `json:"businessUnitId"       keel:"business_unit_id,pk,tenant"`
	OrganizationID       pulid.ID             `json:"organizationId"       keel:"organization_id,pk,tenant"`
	RunID                pulid.ID             `json:"runId"                keel:"run_id"`
	ShipmentID           pulid.ID             `json:"shipmentId"           keel:"shipment_id"`
	Sequence             int                  `json:"sequence"             keel:"sequence"`
	ProNumber            string               `json:"proNumber"            keel:"pro_number,nullzero"`
	Status               ItemStatus           `json:"status"               keel:"status"`
	FailureCode          FailureCode          `json:"failureCode"          keel:"failure_code,nullzero"`
	ErrorMessage         string               `json:"errorMessage"         keel:"error_message,nullzero"`
	MarkedReadyToInvoice bool                 `json:"markedReadyToInvoice" keel:"marked_ready_to_invoice"`
	BillingQueueItemID   pulid.ID             `json:"billingQueueItemId"   keel:"billing_queue_item_id,nullzero"`
	BillingQueueNumber   string               `json:"billingQueueNumber"   keel:"billing_queue_number,nullzero"`
	BillingQueueStatus   billingqueue.Status  `json:"billingQueueStatus"   keel:"billing_queue_status,nullzero,sqltype=billing_queue_status"`
	MissingRequirements  []MissingRequirement `json:"missingRequirements"  keel:"missing_requirements,jsonb"`
	ValidationFailures   []ValidationFailure  `json:"validationFailures"   keel:"validation_failures,jsonb"`
	ProcessedAt          *int64               `json:"processedAt"          keel:"processed_at"`
	CreatedAt            int64                `json:"createdAt"            keel:"created_at,autocreate"`
	UpdatedAt            int64                `json:"updatedAt"            keel:"updated_at,autoupdate"`
}

func (i *BillingTransferRunItem) BeforeCreate(ctx context.Context) error {
	if i.MissingRequirements == nil {
		i.MissingRequirements = []MissingRequirement{}
	}
	if i.ValidationFailures == nil {
		i.ValidationFailures = []ValidationFailure{}
	}

	return nil
}

func (i *BillingTransferRunItem) GetID() pulid.ID { return i.ID }

func (i *BillingTransferRunItem) GetCreatedAt() int64 { return i.CreatedAt }

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
