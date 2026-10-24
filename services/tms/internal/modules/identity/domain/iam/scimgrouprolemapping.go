package iam

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ domaintypes.PostgresSearchable = (*SCIMGroupRoleMapping)(nil)
)

type SCIMGroupRoleMapping struct {
	keel.Table      `keel:"scim_group_role_mappings"`
	ID              pulid.ID `json:"id" keel:"id,pk,autoid:sgr_"`
	OrganizationID  pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID  pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	DirectoryID     pulid.ID `json:"directoryId" keel:"directory_id"`
	ExternalGroupID string   `json:"externalGroupId" keel:"external_group_id"`
	DisplayName     string   `json:"displayName" keel:"display_name"`
	RoleID          pulid.ID `json:"roleId" keel:"role_id"`

	SearchVector string `json:"-" keel:"search_vector,tsvector,readonly,noselect"`
	Rank         string `json:"-" keel:"rank,computed"`
	Version      int64  `json:"version" keel:"version,version"`
	CreatedAt    int64  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt    int64  `json:"updatedAt" keel:"updated_at,autoupdate"`

	Role *permission.Role `json:"role,omitempty" keel:"belongs:role_id"`
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
			{Field: "role", Queryable: true},
		},
	}
}
