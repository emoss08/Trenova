package iam

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*SCIMGroupRoleMapping)(nil)
	_ domaintypes.PostgresSearchable = (*SCIMGroupRoleMapping)(nil)
)

type SCIMGroupRoleMapping struct {
	bun.BaseModel             `bun:"table:scim_group_role_mappings,alias:sgrm" json:"-"`
	keel.Table                `keel:"scim_group_role_mappings"`
	pagination.CursorValueSet `bun:",embed"                                    json:"-"`

	ID              pulid.ID `json:"id"              bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:sgr_"`
	OrganizationID  pulid.ID `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),notnull,pk" keel:"organization_id,pk,tenant"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),notnull,pk" keel:"business_unit_id,pk,tenant"`
	DirectoryID     pulid.ID `json:"directoryId"     bun:"directory_id,type:VARCHAR(100),notnull" keel:"directory_id"`
	ExternalGroupID string   `json:"externalGroupId" bun:"external_group_id,type:VARCHAR(255),notnull" keel:"external_group_id"`
	DisplayName     string   `json:"displayName"     bun:"display_name,type:VARCHAR(255),notnull" keel:"display_name"`
	RoleID          pulid.ID `json:"roleId"          bun:"role_id,type:VARCHAR(100),notnull" keel:"role_id"`

	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly" keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly" keel:"rank,computed"`
	Version      int64  `json:"version"   bun:"version,type:BIGINT" keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	Role *permission.Role `json:"role,omitempty" bun:"rel:belongs-to,join:role_id=id" keel:"belongs:role_id"`
}

func (m *SCIMGroupRoleMapping) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		m,
		validation.Field(&m.RoleID, validation.Required.Error("Role is required")),
		validation.Field(
			&m.ExternalGroupID,
			validation.Required.Error("External Group ID is required"),
		),
		validation.Field(
			&m.DirectoryID,
			validation.Required.Error("Directory ID is required"),
		),
	))
}

func (m *SCIMGroupRoleMapping) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if m.ID.IsNil() {
			m.ID = pulid.MustNew("sgr_")
		}
		m.CreatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = now
	}

	return nil
}

func (m *SCIMGroupRoleMapping) GetID() pulid.ID {
	return m.ID
}

func (m *SCIMGroupRoleMapping) GetCreatedAt() int64 {
	return m.CreatedAt
}

func (m *SCIMGroupRoleMapping) GetOrganizationID() pulid.ID {
	return m.OrganizationID
}

func (m *SCIMGroupRoleMapping) GetBusinessUnitID() pulid.ID {
	return m.BusinessUnitID
}

func (m *SCIMGroupRoleMapping) GetTableName() string {
	return "scim_group_role_mappings"
}

func (m *SCIMGroupRoleMapping) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "sgrm",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "display_name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "external_group_id",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
		},
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:        "role",
				Type:         dbtype.RelationshipTypeBelongsTo,
				TargetEntity: (*permission.Role)(nil),
				TargetTable:  "roles",
				ForeignKey:   "role_id",
				ReferenceKey: "id",
				Alias:        "r",
				Queryable:    true,
			},
		},
	}
}
