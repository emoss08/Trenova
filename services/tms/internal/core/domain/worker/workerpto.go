package worker

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*WorkerPTO)(nil)
	_ domaintypes.PostgresSearchable = (*WorkerPTO)(nil)
	_ domain.Validatable             = (*WorkerPTO)(nil)
	_ framework.TenantedEntity       = (*WorkerPTO)(nil)
)

//nolint:revive // struct should keep this name
type WorkerPTO struct {
	bun.BaseModel `bun:"table:worker_pto,alias:wpto" json:"-"`

	ID             pulid.ID  `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	WorkerID       pulid.ID  `json:"workerId"       bun:"worker_id,type:VARCHAR(100),notnull,pk"`
	ApproverID     *pulid.ID `json:"approverId"     bun:"approver_id,type:VARCHAR(100),nullzero"`
	RejectorID     *pulid.ID `json:"rejectorId"     bun:"rejector_id,type:VARCHAR(100),nullzero"`
	Status         PTOStatus `json:"status"         bun:"status,type:worker_pto_status_enum,notnull,default:'Requested'"`
	Type           PTOType   `json:"type"           bun:"type,type:worker_pto_type_enum,notnull,default:'Vacation'"`
	StartDate      int64     `json:"startDate"      bun:"start_date,type:BIGINT,notnull"`
	EndDate        int64     `json:"endDate"        bun:"end_date,type:BIGINT,notnull"`
	Reason         string    `json:"reason"         bun:"reason,type:VARCHAR(255),notnull"`
	Version        int64     `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64     `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64     `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	Worker       *Worker              `json:"worker,omitempty"       bun:"rel:belongs-to,join:worker_id=id"`
	Approver     *tenant.User         `json:"approver,omitempty"     bun:"rel:belongs-to,join:approver_id=id"`
	Rejector     *tenant.User         `json:"rejector,omitempty"     bun:"rel:belongs-to,join:rejector_id=id"`
}

func (w *WorkerPTO) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(w,
		validation.Field(&w.Status,
			validation.Required.Error("Status is required"),
			validation.In(
				PTOStatusRequested,
				PTOStatusApproved,
				PTOStatusRejected,
				PTOStatusCancelled,
			).Error("Status must be a valid PTO status"),
		),
		validation.Field(&w.ApproverID,
			validation.When(w.Status == PTOStatusApproved,
				validation.Required.Error("Approver ID is required when the status is approved")),
		),
		validation.Field(&w.Type,
			validation.Required.Error("Type is required"),
			validation.In(
				PTOTypeVacation,
				PTOTypeSick,
				PTOTypeHoliday,
				PTOTypeBereavement,
				PTOTypeMaternity,
				PTOTypePaternity,
			).Error("Type must be a valid PTO type"),
		),
		validation.Field(&w.StartDate,
			validation.Required.Error("Start date is required"),
			validation.Max(w.EndDate).Error("Start date cannot be after end date"),
		),
		validation.Field(&w.EndDate,
			validation.Required.Error("End date is required"),
			validation.Min(w.StartDate).Error("End date must be after start date"),
		),
		validation.Field(&w.Reason,
			validation.When(
				w.Status == PTOStatusCancelled || w.Status == PTOStatusRejected,
				validation.Required.Error(
					"Reason is required when the status is cancelled or rejected",
				),
			),
			validation.When(
				w.Status != PTOStatusCancelled && w.Status != PTOStatusRejected,
				validation.Empty.Error(
					"Reason cannot be input when the status is not cancelled or rejected",
				),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (w *WorkerPTO) GetTableName() string {
	return "worker_pto"
}

func (w *WorkerPTO) GetID() string {
	return w.ID.String()
}

func (w *WorkerPTO) GetBusinessUnitID() pulid.ID {
	return w.BusinessUnitID
}

func (w *WorkerPTO) GetOrganizationID() pulid.ID {
	return w.OrganizationID
}

func (w *WorkerPTO) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
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

func (w *WorkerPTO) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if w.ID.IsNil() {
			w.ID = pulid.MustNew("pto_")
		}

		w.CreatedAt = now
	case *bun.UpdateQuery:
		w.UpdatedAt = now
	}

	return nil
}
