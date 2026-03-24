package tenant

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

var _ bun.BeforeAppendModelHook = (*OrganizationMembership)(nil)

type OrganizationMembership struct {
	bun.BaseModel `bun:"table:user_organization_memberships,alias:uom" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	IsDefault      bool     `json:"isDefault"      bun:"is_default,default:false"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	JoinedAt       int64    `json:"joinedAt"       bun:"joined_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	GrantedByID    pulid.ID `json:"grantedByID"    bun:"granted_by_id,type:VARCHAR(100)"`
	ExpiresAt      *int64   `json:"expiresAt"      bun:"expires_at"`

	// Relationships
	User         *User         `json:"user,omitempty"         bun:"rel:belongs-to,join:user_id=id"`
	GrantedBy    *User         `json:"grantedBy,omitempty"    bun:"rel:belongs-to,join:granted_by_id=id"`
	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (om *OrganizationMembership) BeforeAppendModel(_ context.Context, q bun.Query) error {
	if _, ok := q.(*bun.InsertQuery); ok {
		if om.ID.IsNil() {
			om.ID = pulid.MustNew("uom_")
		}

		if om.JoinedAt == 0 {
			om.JoinedAt = timeutils.NowUnix()
		}
	}
	return nil
}

var (
	_ bun.BeforeAppendModelHook          = (*User)(nil)
	_ domaintypes.PostgresSearchable     = (*User)(nil)
	_ validationframework.TenantedEntity = (*User)(nil)
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:usr" json:"-"`

	ID                    pulid.ID               `json:"id"                    bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID        pulid.ID               `json:"businessUnitId"        bun:"business_unit_id,type:VARCHAR(100),notnull"`
	CurrentOrganizationID pulid.ID               `json:"currentOrganizationId" bun:"current_organization_id,type:VARCHAR(100),notnull"`
	Status                domaintypes.Status     `json:"status"                bun:"status,type:status_enum,notnull,default:'Active'"`
	Name                  string                 `json:"name"                  bun:"name,type:VARCHAR(255),notnull"`
	Username              string                 `json:"username"              bun:"username,type:VARCHAR(20),notnull"`
	TimeFormat            domaintypes.TimeFormat `json:"timeFormat"            bun:"time_format,type:time_format_enum,notnull,default:'12-hour'"`
	Password              string                 `json:"-"                     bun:"password,type:VARCHAR(255),notnull"` // Hidden in responses
	EmailAddress          string                 `json:"emailAddress"          bun:"email_address,type:VARCHAR(255),notnull"`
	ProfilePicURL         string                 `json:"profilePicUrl"         bun:"profile_pic_url,type:VARCHAR(255)"`
	ThumbnailURL          string                 `json:"thumbnailUrl"          bun:"thumbnail_url,type:VARCHAR(255)"`
	Timezone              string                 `json:"timezone"              bun:"timezone,type:VARCHAR(50),notnull"`
	IsLocked              bool                   `json:"isLocked"              bun:"is_locked,type:BOOLEAN,notnull,default:false"`
	MustChangePassword    bool                   `json:"mustChangePassword"    bun:"must_change_password,type:BOOLEAN,notnull,default:false"`
	IsPlatformAdmin       bool                   `json:"isPlatformAdmin"       bun:"is_platform_admin,type:BOOLEAN,notnull,default:false"`
	Version               int64                  `json:"version"               bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt             int64                  `json:"createdAt"             bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64                  `json:"updatedAt"             bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	LastLoginAt           *int64                 `json:"lastLoginAt,omitzero"  bun:"last_login_at,nullzero"`

	BusinessUnit        *BusinessUnit                    `json:"-"                     bun:"rel:belongs-to,join:business_unit_id=id"`
	CurrentOrganization *Organization                    `json:"-"                     bun:"rel:belongs-to,join:current_organization_id=id"`
	Memberships         []*OrganizationMembership        `json:"memberships,omitempty" bun:"rel:has-many,join:id=user_id"`
	Assignments         []*permission.UserRoleAssignment `json:"assignments,omitempty" bun:"rel:has-many,join:id=user_id"`
}

func (u *User) GeneratePassword(raw string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashed), nil
}

func (u *User) verifyPassword(raw string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(raw))
	if err != nil {
		return errortypes.NewValidationError(
			"password",
			errortypes.ErrInvalid,
			"Invalid password. Please try again",
		)
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

func (u *User) VerifyCredentials(raw string) error {
	if err := u.ValidateStatus(); err != nil {
		return err
	}
	if err := u.verifyPassword(raw); err != nil {
		return err
	}

	return nil
}

func (u *User) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

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

func (u *User) GetID() pulid.ID {
	return u.ID
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

func (u *User) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(u,
		validation.Field(&u.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 255).Error("Name must be between 1 and 255 characters"),
		),
		validation.Field(&u.Username,
			validation.Required.Error("Username is required"),
			validation.Length(1, 20).Error("Username must be between 1 and 20 characters"),
		),
		validation.Field(&u.EmailAddress,
			validation.Required.Error("Email address is required"),
			is.Email.Error("Email address must be a valid email"),
		),
		validation.Field(&u.Timezone,
			validation.Required.Error("Timezone is required"),
		),
	)
	if err != nil {
		multiErr.AddOzzoError(err)
	}
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
		Relationships: []*domaintypes.RelationshipDefintion{
			{
				Field:              "roles",
				Type:               dbtype.RelationshipTypeManyToMany,
				TargetEntity:       (*permission.Role)(nil),
				TargetTable:        "roles",
				ReferenceKey:       "id",
				Alias:              "r",
				Queryable:          true,
				JoinTable:          "user_roles",
				JoinTableAlias:     "ur",
				JoinTableSourceKey: "user_id",
				JoinTableTargetKey: "role_id",
				CustomJoinPath: []domaintypes.JoinStep{
					{
						Table:     "user_roles",
						Alias:     "ur",
						Condition: "usr.id = ur.user_id",
						JoinType:  dbtype.JoinTypeLeft,
					},
					{
						Table:     "roles",
						Alias:     "r",
						Condition: "ur.role_id = r.id",
						JoinType:  dbtype.JoinTypeLeft,
					},
				},
			},
		},
	}
}
