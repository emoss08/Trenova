package worker

import (
	"context"
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*WorkerPTO)(nil)
	_ domaintypes.PostgresSearchable     = (*WorkerPTO)(nil)
	_ pagination.CursorEntity            = (*WorkerPTO)(nil)
	_ validationframework.TenantedEntity = (*WorkerPTO)(nil)
)

// Time-off requests and the days booked against them
type WorkerPTO struct {
	bun.BaseModel             `bun:"table:worker_pto,alias:wpto" json:"-"`
	keel.Table                `                                           keel:"worker_pto,report=Parties,label=Time Off Request"`
	pagination.CursorValueSet `bun:",embed"                      json:"-"`

	ID                 pulid.ID            `json:"id"                 bun:"id,pk,type:VARCHAR(100)"                                                  keel:"id,pk,autoid:wrkpto_"`
	WorkerID           pulid.ID            `json:"workerId"           bun:"worker_id,pk,type:VARCHAR(100),notnull"                                   keel:"worker_id,pk"`
	BusinessUnitID     pulid.ID            `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),notnull,pk"                            keel:"business_unit_id,pk,tenant"`
	OrganizationID     pulid.ID            `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),notnull,pk"                             keel:"organization_id,pk,tenant"`
	ApproverID         pulid.ID            `json:"approverId"         bun:"approver_id,type:VARCHAR(100),nullzero"                                   keel:"approver_id,nullzero"`
	RejectorID         pulid.ID            `json:"rejectorId"         bun:"rejector_id,type:VARCHAR(100),nullzero"                                   keel:"rejector_id,nullzero"`
	CancelledByID      pulid.ID            `json:"cancelledById"      bun:"cancelled_by_id,type:VARCHAR(100),nullzero"                               keel:"cancelled_by_id,nullzero"`
	Status             PTOStatus           `json:"status"             bun:"status,type:worker_pto_status_enum,notnull,default:'Requested'"           keel:"status"`
	Type               PTOType             `json:"type"               bun:"type,type:worker_pto_type_enum,notnull,default:'Vacation'"                keel:"type"`
	StartDate          int64               `json:"startDate"          bun:"start_date,type:BIGINT,notnull"                                           keel:"start_date,label=Starts"`
	EndDate            int64               `json:"endDate"            bun:"end_date,type:BIGINT,notnull"                                             keel:"end_date,label=Ends"`
	Reason             string              `json:"reason"             bun:"reason,type:VARCHAR(255),notnull"                                         keel:"reason"`
	RejectionReason    string              `json:"rejectionReason"    bun:"rejection_reason,type:VARCHAR(255),nullzero"                              keel:"rejection_reason,nullzero"`
	CancellationReason string              `json:"cancellationReason" bun:"cancellation_reason,type:VARCHAR(255),nullzero"                           keel:"cancellation_reason,nullzero"`
	Days               decimal.Decimal     `json:"days"               bun:"days,type:NUMERIC(6,2),notnull,default:0"                                 keel:"days"`
	BalanceAfterDays   decimal.NullDecimal `json:"balanceAfterDays"   bun:"balance_after_days,type:NUMERIC(8,2),nullzero"                            keel:"balance_after_days"`
	AutoApproved       bool                `json:"autoApproved"       bun:"auto_approved,type:BOOLEAN,notnull"                                       keel:"auto_approved"`
	SearchVector       string              `json:"-"                  bun:"search_vector,type:TSVECTOR,scanonly"                                     keel:"search_vector,tsvector,readonly,noselect"`
	Version            int64               `json:"version"            bun:"version,type:BIGINT"                                                      keel:"version,version"`
	CreatedAt          int64               `json:"createdAt"          bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt          int64               `json:"updatedAt"          bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Worker      *Worker      `json:"worker,omitempty"      bun:"rel:belongs-to,join:worker_id=id"       keel:"belongs:worker_id"`
	Approver    *tenant.User `json:"approver,omitempty"    bun:"rel:belongs-to,join:approver_id=id"     keel:"belongs:approver_id,noreport"`
	Rejector    *tenant.User `json:"rejector,omitempty"    bun:"rel:belongs-to,join:rejector_id=id"     keel:"belongs:rejector_id,noreport"`
	CancelledBy *tenant.User `json:"cancelledBy,omitempty" bun:"rel:belongs-to,join:cancelled_by_id=id" keel:"belongs:cancelled_by_id,noreport"`
}

func (wpto *WorkerPTO) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(wpto,
		validation.Field(&wpto.WorkerID,
			validation.Required.Error("Worker is required"),
			validation.By(func(value any) error {
				id, ok := value.(pulid.ID)
				if !ok {
					return errors.New("invalid worker ID type")
				}
				if id.IsNil() {
					return errors.New("Worker is required")
				}
				return nil
			}),
		),
		validation.Field(
			&wpto.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[PTOStatus](
				"status must be one of: Requested, Approved, Rejected, Cancelled",
			),
		),
		validation.Field(
			&wpto.Type,
			validation.Required.Error("Type is required"),
			domainvalidation.ValidEnum[PTOType](
				"type must be one of: Personal, Vacation, Sick, Holiday, Bereavement, Maternity, Paternity",
			),
		),
		validation.Field(&wpto.StartDate,
			validation.Required.Error("Start date is required"),
			validation.Min(int64(1)).Error("Start date must be a positive value"),
		),
		validation.Field(&wpto.EndDate,
			validation.Required.Error("End date is required"),
			validation.Min(int64(1)).Error("End date must be a positive value"),
		),
		validation.Field(&wpto.Reason,
			validation.Required.Error("Reason is required"),
			validation.Length(1, 255).Error("Reason must be between 1 and 255 characters"),
		),
	))

	if wpto.EndDate < wpto.StartDate {
		multiErr.Add("endDate", errortypes.ErrInvalid, "End date cannot be before start date")
	}
}

func (wpto *WorkerPTO) IsOpen() bool {
	return wpto.Status == PTOStatusRequested || wpto.Status == PTOStatusApproved
}

func (wpto *WorkerPTO) GetTableName() string {
	return "worker_pto"
}

func (wpto *WorkerPTO) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if wpto.ID.IsNil() {
			wpto.ID = pulid.MustNew("wrkpto_")
		}
		wpto.CreatedAt = now
	case *bun.UpdateQuery:
		wpto.UpdatedAt = now
	}

	return nil
}

func (wpto *WorkerPTO) GetID() pulid.ID {
	return wpto.ID
}

func (wpto *WorkerPTO) GetCreatedAt() int64 {
	return wpto.CreatedAt
}

func (wpto *WorkerPTO) GetResourceType() string {
	return "worker_pto"
}

func (wpto *WorkerPTO) GetResourceID() string {
	return wpto.ID.String()
}

func (wpto *WorkerPTO) GetBusinessUnitID() pulid.ID {
	return wpto.BusinessUnitID
}

func (wpto *WorkerPTO) GetOrganizationID() pulid.ID {
	return wpto.OrganizationID
}

func (wpto *WorkerPTO) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "wpto",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
			{Name: "type", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
			{
				Name:   "start_date",
				Type:   domaintypes.FieldTypeDate,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "end_date", Type: domaintypes.FieldTypeDate, Weight: domaintypes.SearchWeightB},
			{Name: "reason", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
		},
	}
}
