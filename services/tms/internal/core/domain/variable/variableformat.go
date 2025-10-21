package variable

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*VariableFormat)(nil)
	_ domain.Validatable        = (*VariableFormat)(nil)
)

//nolint:revive // valid struct name
type VariableFormat struct {
	bun.BaseModel `bun:"table:variable_formats,alias:vf" json:"-"`

	ID             pulid.ID  `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,pk,notnull,type:VARCHAR(100)"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,pk,notnull,type:VARCHAR(100)"`
	Name           string    `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Description    string    `json:"description"    bun:"description,type:TEXT"`
	ValueType      ValueType `json:"valueType"      bun:"value_type,type:variable_value_type_enum,notnull"`
	FormatSQL      string    `json:"formatSql"      bun:"format_sql,type:TEXT,notnull"`
	IsActive       bool      `json:"isActive"       bun:"is_active,type:BOOLEAN,default:true"`
	IsSystem       bool      `json:"isSystem"       bun:"is_system,type:BOOLEAN,default:false"`
	Version        int64     `json:"version"        bun:"version,type:BIGINT"`
	CreatedAt      int64     `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64     `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
}

func (f *VariableFormat) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(f,
		validation.Field(&f.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&f.ValueType,
			validation.Required.Error("Value type is required"),
			validation.By(validateValueType),
		),
		validation.Field(&f.FormatSQL,
			validation.Required.Error("Format SQL is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (f *VariableFormat) GetID() string {
	return f.ID.String()
}

func (f *VariableFormat) GetTableName() string {
	return "variable_formats"
}

func (f *VariableFormat) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if f.ID.IsNil() {
			f.ID = pulid.MustNew("vfm_")
		}
		f.CreatedAt = now
	case *bun.UpdateQuery:
		f.UpdatedAt = now
	}

	return nil
}

func (f *VariableFormat) CanEdit() bool {
	return !f.IsSystem
}

func (f *VariableFormat) CanDelete() bool {
	return !f.IsSystem
}
