package scim

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
)

type User struct {
	keel.Table `keel:"scim_users"`

	ID                     pulid.ID `json:"id"                     keel:"id,pk,autoid:scu_"`
	OrganizationID         pulid.ID `json:"organizationId"         keel:"organization_id,pk,tenant"`
	BusinessUnitID         pulid.ID `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	DirectoryID            pulid.ID `json:"directoryId"            keel:"directory_id"`
	UserID                 pulid.ID `json:"userId"                 keel:"user_id"`
	UserName               string   `json:"userName"               keel:"user_name"`
	ExternalID             string   `json:"externalId"             keel:"external_id,nullzero"`
	Active                 bool     `json:"active"                 keel:"active"`
	Provisioned            bool     `json:"provisioned"            keel:"provisioned"`
	RolesVersion           int64    `json:"rolesVersion"           keel:"roles_version"`
	RolesReconciledVersion int64    `json:"rolesReconciledVersion" keel:"roles_reconciled_version"`
	CreatedAt              int64    `json:"createdAt"              keel:"created_at,autocreate"`
	UpdatedAt              int64    `json:"updatedAt"              keel:"updated_at,autoupdate"`

	User *tenant.User `json:"-" keel:"belongs:user_id"`
}

func (User) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_scim_users_user_name",
			Message: "A user with this userName already exists in this directory",
		},
		{
			Name:    "idx_scim_users_external_id",
			Message: "A user with this externalId already exists in this directory",
		},
		{
			Name:    "uq_scim_users_directory_user",
			Field:   "userId",
			Message: "This person is already linked to this directory",
		},
	}
}

type Group struct {
	keel.Table `keel:"scim_groups"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:scg_"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	DirectoryID    pulid.ID `json:"directoryId"    keel:"directory_id"`
	DisplayName    string   `json:"displayName"    keel:"display_name"`
	ExternalID     string   `json:"externalId"     keel:"external_id,nullzero"`
	Version        int64    `json:"version"        keel:"version,version"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (Group) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_scim_groups_display_name",
			Message: "A group with this displayName already exists in this directory",
		},
		{
			Name:    "idx_scim_groups_external_id",
			Message: "A group with this externalId already exists in this directory",
		},
	}
}

type GroupMember struct {
	keel.Table `keel:"scim_group_members"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:sgm_"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	DirectoryID    pulid.ID `json:"directoryId"    keel:"directory_id"`
	GroupID        pulid.ID `json:"groupId"        keel:"group_id"`
	UserID         pulid.ID `json:"userId"         keel:"user_id"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`

	User *tenant.User `json:"-" keel:"belongs:user_id"`
}

func (GroupMember) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "fk_scim_group_members_user",
			Field:   "userId",
			Message: "A group member must be a user this directory provisioned",
		},
	}
}

type RoleGrant struct {
	keel.Table `keel:"scim_role_grants"`

	ID             pulid.ID `json:"id"             keel:"id,pk,autoid:srg_"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	DirectoryID    pulid.ID `json:"directoryId"    keel:"directory_id"`
	UserID         pulid.ID `json:"userId"         keel:"user_id"`
	RoleID         pulid.ID `json:"roleId"         keel:"role_id"`
	AssignmentID   pulid.ID `json:"assignmentId"   keel:"assignment_id"`
	GrantedByID    pulid.ID `json:"grantedById"    keel:"granted_by_id"`
	CreatedAt      int64    `json:"createdAt"      keel:"created_at,autocreate"`
}

func (RoleGrant) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_scim_role_grants_user_role",
			Field:   "roleId",
			Message: "This directory already grants this role to this person",
		},
	}
}

type RoleSource struct {
	RoleID      pulid.ID `json:"roleId"`
	MappingID   pulid.ID `json:"mappingId"`
	SavedByID   pulid.ID `json:"savedById"`
	GroupID     pulid.ID `json:"groupId"`
	DisplayName string   `json:"displayName"`
}
