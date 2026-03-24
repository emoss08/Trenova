package worker

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*WorkerPTO)(nil)
	_ domaintypes.PostgresSearchable     = (*WorkerPTO)(nil)
	_ validationframework.TenantedEntity = (*WorkerPTO)(nil)
)

type WorkerPTO struct {
	bun.BaseModel `bun:"table:worker_pto,alias:wpto" json:"-"`

	ID             pulid.ID  `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	WorkerID       pulid.ID  `json:"workerId"       bun:"worker_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	ApproverID     pulid.ID  `json:"approverId"     bun:"approver_id,type:VARCHAR(100),nullzero"`
	RejectorID     pulid.ID  `json:"rejectorId"     bun:"rejector_id,type:VARCHAR(100),nullzero"`
	Status         PTOStatus `json:"status"         bun:"status,type:worker_pto_status_enum,notnull,default:'Requested'"`
	Type           PTOType   `json:"type"           bun:"type,type:worker_pto_type_enum,notnull,default:'Vacation'"`
	StartDate      int64     `json:"startDate"      bun:"start_date,type:BIGINT,notnull"`
	EndDate        int64     `json:"endDate"        bun:"end_date,type:BIGINT,notnull"`
	Reason         string    `json:"reason"         bun:"reason,type:VARCHAR(255),notnull"`
	SearchVector   string    `json:"-"              bun:"search_vector,type:TSVECTOR,scanonly"`
	Version        int64     `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64     `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64     `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Worker   *Worker      `json:"worker,omitempty"   bun:"rel:belongs-to,join:worker_id=id"`
	Approver *tenant.User `json:"approver,omitempty" bun:"rel:belongs-to,join:approver_id=id"`
	Rejector *tenant.User `json:"rejector,omitempty" bun:"rel:belongs-to,join:rejector_id=id"`
}

func (wpto *WorkerPTO) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(wpto,
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
		validation.Field(&wpto.Status,
			validation.Required.Error("Status is required"),
			validation.By(func(value any) error {
				s, ok := value.(PTOStatus)
				if !ok {
					return errors.New("invalid PTO status type")
				}
				if !s.IsValid() {
					return errors.New(
						"status must be one of: Requested, Approved, Rejected, Cancelled",
					)
				}
				return nil
			}),
		),
		validation.Field(&wpto.Type,
			validation.Required.Error("Type is required"),
			validation.By(func(value any) error {
				t, ok := value.(PTOType)
				if !ok {
					return errors.New("invalid PTO type")
				}
				if !t.IsValid() {
					return errors.New(
						"type must be one of: Personal, Vacation, Sick, Holiday, Bereavement, Maternity, Paternity",
					)
				}
				return nil
			}),
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
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	if wpto.EndDate <= wpto.StartDate {
		multiErr.Add("endDate", errortypes.ErrInvalid, "End date must be after start date")
	}
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
