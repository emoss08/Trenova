package user

import (
	"context"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

var _ bun.BeforeAppendModelHook = (*User)(nil)

type User struct {
	bun.BaseModel `bun:"table:users,alias:usr" json:"-"`

	ID                    pulid.ID      `json:"id"                    bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID        pulid.ID      `json:"businessUnitId"        bun:"business_unit_id,type:VARCHAR(100),notnull"`
	CurrentOrganizationID pulid.ID      `json:"currentOrganizationId" bun:"current_organization_id,type:VARCHAR(100),notnull"`
	Status                domain.Status `json:"status"                bun:"status,type:status_enum,notnull,default:'Active'"`
	Name                  string        `json:"name"                  bun:"name,type:VARCHAR(255),notnull"`
	Username              string        `json:"username"              bun:"username,type:VARCHAR(20),notnull"`
	Password              string        `json:"-"                     bun:"password,type:VARCHAR(255),notnull"` // ! We will hide this in the response
	EmailAddress          string        `json:"emailAddress"          bun:"email_address,type:VARCHAR(255),notnull"`
	ProfilePicURL         string        `json:"profilePicUrl"         bun:"profile_pic_url,type:VARCHAR(255)"`
	ThumbnailURL          string        `json:"thumbnailUrl"          bun:"thumbnail_url,type:VARCHAR(255)"`
	Timezone              string        `json:"timezone"              bun:"timezone,type:VARCHAR(50),notnull"`
	TimeFormat            TimeFormat    `json:"timeFormat"            bun:"time_format,type:time_format_enum,notnull,default:'12-hour'"`
	IsLocked              bool          `json:"isLocked"              bun:"is_locked,type:BOOLEAN,notnull,default:false"`
	MustChangePassword    bool          `json:"mustChangePassword"    bun:"must_change_password,type:BOOLEAN,notnull,default:false"`
	LastLoginAt           *int64        `json:"lastLoginAt,omitzero"  bun:"last_login_at,nullzero"`
	Version               int64         `json:"version"               bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt             int64         `json:"createdAt"             bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt             int64         `json:"updatedAt"             bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit        *businessunit.BusinessUnit   `json:"-"                             bun:"rel:belongs-to,join:business_unit_id=id"`
	CurrentOrganization *organization.Organization   `json:"currentOrganization,omitempty" bun:"rel:belongs-to,join:current_organization_id=id"`
	Organizations       []*organization.Organization `json:"organizations,omitempty"       bun:"m2m:user_organizations,join:User=Organization"`
	Roles               []*permission.Role           `json:"roles,omitzero"                bun:"m2m:user_roles,join:User=Role"`
}

// Validate validates the user entity
func (u *User) Validate(multiErr *errors.MultiError) *errors.MultiError {
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
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}

	return multiErr
}

// IsActive returns true if the user is active
func (u *User) IsActive() bool {
	return u.Status == domain.StatusActive
}

func (u *User) GetID() string {
	return u.ID.String()
}

// GeneratePassword generates a hashed password
func (u *User) GeneratePassword(raw string) (string, error) {
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate password hash")
		return "", err
	}

	return string(hashed), nil
}

func (u *User) VerifyCredentials(raw string) error {
	// Validate the status
	if err := u.ValidateStatus(); err != nil {
		return err
	}

	// Verify the password
	if err := u.VerifyPassword(raw); err != nil {
		return err
	}

	return nil
}

// ValidateStatus validates the user's status
func (u *User) ValidateStatus() error {
	if !u.IsActive() {
		return errors.NewAuthorizationError(
			"Your account is not active. Please contact your system administrator.",
		)
	}

	if u.IsLocked {
		return errors.NewAuthorizationError(
			"Your account is locked. Please contact your system administrator.",
		)
	}

	return nil
}

// VerifyPassword verifies the user's password
func (u *User) VerifyPassword(raw string) error {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(raw))
	if err != nil {
		return errors.NewValidationError(
			"password",
			errors.ErrInvalid,
			"Invalid password. Please try again.",
		)
	}

	return nil
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface.
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

//nolint:revive // valid struct name
type UserRole struct {
	bun.BaseModel  `bun:"table:user_roles,alias:ur" json:"-"`
	BusinessUnitID pulid.ID         `bun:"business_unit_id,pk,type:VARCHAR(100),notnull" json:"businessUnitId"`
	OrganizationID pulid.ID         `bun:"organization_id,pk,type:VARCHAR(100),notnull"  json:"organizationId"`
	UserID         pulid.ID         `bun:"user_id,pk,type:VARCHAR(100),notnull"          json:"userId"`
	RoleID         pulid.ID         `bun:"role_id,pk,type:VARCHAR(100),notnull"          json:"roleId"`
	User           *User            `bun:"rel:belongs-to,join:user_id=id"                json:"-"`
	Role           *permission.Role `bun:"rel:belongs-to,join:role_id=id"                json:"-"`
}
