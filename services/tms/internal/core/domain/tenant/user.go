package tenant

import (
	"context"
	"errors"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

type ComputedPermissions struct {
	OrganizationID pulid.ID               `json:"organizationId"`
	ResourceMap    *ResourcePermissionMap `json:"resourceMap"` // Optimized permission structure
	BloomFilter    []byte                 `json:"-"`           // For quick negative checks
	Checksum       string                 `json:"checksum"`    // For integrity
}

type ResourcePermissionMap struct {
	StandardResources map[string]StandardPermission `json:"standardResources"`
	ExtendedResources map[string]ExtendedPermission `json:"extendedResources"`
	GlobalFlags       GlobalCapabilities            `json:"globalFlags"`
}

type StandardPermission struct {
	Operations uint32               `json:"operations"` // Bitfield for up to 32 ops
	DataScope  permission.DataScope `json:"dataScope"`
	QuickCheck uint64               `json:"quickCheck"` // Pre-computed common checks
}

type ExtendedPermission struct {
	StandardOps uint32               `json:"standardOps"`
	CustomOps   map[string]bool      `json:"customOps"`
	DataScope   permission.DataScope `json:"dataScope"`
	Conditions  []CompiledCondition  `json:"conditions"` // Pre-compiled for speed
}

type CompiledCondition struct {
	Type       string         `json:"type"`
	Expression string         `json:"expression"`
	Parameters map[string]any `json:"parameters"`
}

type GlobalCapabilities struct {
	IsSuperAdmin   bool `json:"isSuperAdmin"`
	IsOrgAdmin     bool `json:"isOrgAdmin"`
	CanSeeAllData  bool `json:"canSeeAllData"`
	CanExportData  bool `json:"canExportData"`
	CanManageUsers bool `json:"canManageUsers"`
	CanManageRoles bool `json:"canManageRoles"`
	CanViewReports bool `json:"canViewReports"`
}

type OrganizationMembership struct {
	bun.BaseModel `bun:"table:user_organization_memberships,alias:uom" json:"-"`

	ID             pulid.ID   `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID   `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	UserID         pulid.ID   `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID   `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	RoleIDs        []pulid.ID `json:"roleIds"        bun:"role_ids,type:TEXT[]"`
	DirectPolicies []pulid.ID `json:"directPolicies" bun:"direct_policies,type:TEXT[]"`
	JoinedAt       int64      `json:"joinedAt"       bun:"joined_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	GrantedByID    pulid.ID   `json:"grantedByID"    bun:"granted_by_id,type:VARCHAR(100)"`
	ExpiresAt      *int64     `json:"expiresAt"      bun:"expires_at"`
	IsDefault      bool       `json:"isDefault"      bun:"is_default,default:false"`

	// Relationships
	User         *User         `json:"user,omitempty"         bun:"rel:belongs-to,join:user_id=id"`
	GrantedBy    *User         `json:"grantedBy,omitempty"    bun:"rel:belongs-to,join:granted_by_id=id"`
	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (om *OrganizationMembership) BeforeAppendModel(_ context.Context, q bun.Query) error {
	switch q.(type) {
	case *bun.InsertQuery:
		if om.ID.IsNil() {
			om.ID = pulid.MustNew("uom_")
		}
		if om.JoinedAt == 0 {
			om.JoinedAt = utils.NowUnix()
		}
	}
	return nil
}

var (
	_ bun.BeforeAppendModelHook      = (*User)(nil)
	_ bun.BeforeAppendModelHook      = (*OrganizationMembership)(nil)
	_ domaintypes.PostgresSearchable = (*User)(nil)
	_ domain.Validatable             = (*User)(nil)
	_ framework.TenantedEntity       = (*User)(nil)
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:usr" json:"-"`

	ID                      pulid.ID                 `json:"id"                                bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID          pulid.ID                 `json:"businessUnitId"                    bun:"business_unit_id,type:VARCHAR(100),notnull"`
	CurrentOrganizationID   pulid.ID                 `json:"currentOrganizationId"             bun:"current_organization_id,type:VARCHAR(100),notnull"`
	Status                  domaintypes.Status       `json:"status"                            bun:"status,type:status_enum,notnull,default:'Active'"`
	Name                    string                   `json:"name"                              bun:"name,type:VARCHAR(255),notnull"`
	Username                string                   `json:"username"                          bun:"username,type:VARCHAR(20),notnull"`
	Password                string                   `json:"-"                                 bun:"password,type:VARCHAR(255),notnull"` // Hidden in responses
	EmailAddress            string                   `json:"emailAddress"                      bun:"email_address,type:VARCHAR(255),notnull"`
	ProfilePicURL           string                   `json:"profilePicUrl"                     bun:"profile_pic_url,type:VARCHAR(255)"`
	ThumbnailURL            string                   `json:"thumbnailUrl"                      bun:"thumbnail_url,type:VARCHAR(255)"`
	Timezone                string                   `json:"timezone"                          bun:"timezone,type:VARCHAR(50),notnull"`
	SearchVector            string                   `json:"-"                                 bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                    string                   `json:"-"                                 bun:"rank,type:VARCHAR(100),scanonly"`
	TimeFormat              domaintypes.TimeFormat   `json:"timeFormat"                        bun:"time_format,type:time_format_enum,notnull,default:'12-hour'"`
	IsLocked                bool                     `json:"isLocked"                          bun:"is_locked,type:BOOLEAN,notnull,default:false"`
	MustChangePassword      bool                     `json:"mustChangePassword"                bun:"must_change_password,type:BOOLEAN,notnull,default:false"`
	Version                 int64                    `json:"version"                           bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt               int64                    `json:"createdAt"                         bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt               int64                    `json:"updatedAt"                         bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	LastLoginAt             *int64                   `json:"lastLoginAt,omitzero"              bun:"last_login_at,nullzero"`
	OrganizationMemberships []OrganizationMembership `json:"organizationMemberships,omitempty" bun:"rel:has-many,join:id=user_id"`

	BusinessUnit        *BusinessUnit `json:"-"                             bun:"rel:belongs-to,join:business_unit_id=id"`
	CurrentOrganization *Organization `json:"currentOrganization,omitempty" bun:"rel:belongs-to,join:current_organization_id=id"`
}

func (u *User) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(u,
		validation.
			Field(
				&u.Name,
				validation.Required.Error("Name is required. Please try again"),
				validation.Length(1, 255).
					Error("Name must be between 1 and 255 characters. Please try again"),
				validation.Match(regexp.MustCompile(`^[a-zA-Z]+(\s[a-zA-Z]+)*$`)).
					Error("Name can only contain letters and spaces. Please try again"),
			),

		validation.
			Field(
				&u.Username,
				validation.Required.Error("Username is required. Please try again"),
				validation.Length(1, 20).
					Error("Username must be between 1 and 20 characters. Please try again"),
				is.Alphanumeric.Error("Username must be alphanumeric. Please try again"),
			),

		validation.Field(
			&u.Timezone,
			validation.Required.Error("Timezone is required. Please try again"),
			validation.Length(1, 50).
				Error("Timezone must be between 1 and 50 characters. Please try again"),
		),

		validation.
			Field(&u.Password,
				validation.Required.Error("Password is required. Please try again")),

		validation.
			Field(&u.EmailAddress,
				validation.Required.Error("Email address is required. Please try again"),
				is.EmailFormat.Error("Invalid email format. Please try again")),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (u *User) GetTableName() string {
	return "users"
}

func (u *User) GetOrganizationID() pulid.ID {
	return u.CurrentOrganizationID
}

func (u *User) GetBusinessUnitID() pulid.ID {
	return u.BusinessUnitID
}

func (u *User) GetID() string {
	return u.ID.String()
}

func (u *User) GeneratePassword(raw string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashed), nil
}

func (u *User) VerifyCredentials(raw string) error {
	if err := u.ValidateStatus(); err != nil {
		return err
	}

	if err := u.VerifyPassword(raw); err != nil {
		return err
	}

	return nil
}

func (u *User) IsActive() bool {
	return u.Status == domaintypes.StatusActive
}

func (u *User) ValidateStatus() error {
	if !u.IsActive() {
		return errortypes.NewAuthorizationError(
			"Your account is not active. Please contact your system administrator.",
		)
	}

	if u.IsLocked {
		return errortypes.NewAuthorizationError(
			"Your account is locked. Please contact your system administrator.",
		)
	}

	return nil
}

func (u *User) VerifyPassword(raw string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(raw))
	if err != nil {
		return errortypes.NewValidationError(
			"password",
			errortypes.ErrInvalid,
			"Invalid password. Please try again.",
		)
	}

	return nil
}

func (u *User) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "usr",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "username", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "email_address",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
		},
		Relationships: []*domaintypes.RelationshipDefinition{
			{
				Field:              "roles",
				Type:               domaintypes.RelationshipTypeManyToMany,
				TargetEntity:       (*permission.Role)(nil),
				TargetTable:        "roles",
				ReferenceKey:       "id",
				Alias:              "r",
				Queryable:          true,
				JoinTable:          "user_roles",
				JoinTableAlias:     "ur",
				JoinTableSourceKey: "user_id",
				JoinTableTargetKey: "role_id",
			},
		},
	}
}

func (u *User) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := utils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if u.ID.IsNil() {
			u.ID = pulid.MustNew("usr_")
		}

		u.CreatedAt = now
	case *bun.UpdateQuery:
		u.UpdatedAt = now
	}

	return nil
}
