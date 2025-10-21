package tenant

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*DataRetention)(nil)
	_ domain.Validatable        = (*DataRetention)(nil)
	_ framework.TenantedEntity  = (*DataRetention)(nil)
)

type DataRetention struct {
	bun.BaseModel `bun:"table:data_retention,alias:dr" json:"-"`

	ID                   pulid.ID `json:"id"                   bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID       pulid.ID `json:"businessUnitId"       bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID       pulid.ID `json:"organizationId"       bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	AuditRetentionPeriod int      `json:"auditRetentionPeriod" bun:"audit_retention_period,type:INTEGER,notnull,default:120"` // In days
	Version              int64    `json:"version"              bun:"version,type:BIGINT"`
	CreatedAt            int64    `json:"createdAt"            bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64    `json:"updatedAt"            bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (dr *DataRetention) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(dr,
		validation.Field(&dr.AuditRetentionPeriod,
			validation.Required.Error("Audit retention period is required"),
			validation.Min(1).Error("Audit retention period must be greater than 0"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (dr *DataRetention) GetID() string {
	return dr.ID.String()
}

func (dr *DataRetention) GetTableName() string {
	return "data_retention"
}

func (dr *DataRetention) GetOrganizationID() pulid.ID {
	return dr.OrganizationID
}

func (dr *DataRetention) GetBusinessUnitID() pulid.ID {
	return dr.BusinessUnitID
}

func (dr *DataRetention) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if dr.ID.IsNil() {
			dr.ID = pulid.MustNew("dr_")
		}

		dr.CreatedAt = now
	case *bun.UpdateQuery:
		dr.UpdatedAt = now
	}

	return nil
}
