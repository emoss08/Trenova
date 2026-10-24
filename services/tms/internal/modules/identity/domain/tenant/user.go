package tenant

import (
	"context"
	"fmt"
	"sync"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/dbtype"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/passwordutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"golang.org/x/crypto/bcrypt"
)

type OrganizationMembership struct {
	keel.Table `keel:"user_organization_memberships"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:uom_"`
	IsDefault      bool     `json:"isDefault" keel:"is_default,nullzero"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,tenant"`
	UserID         pulid.ID `json:"userId" keel:"user_id"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,tenant"`
	JoinedAt       int64    `json:"joinedAt" keel:"joined_at,nullzero"`
	GrantedByID    pulid.ID `json:"grantedByID" keel:"granted_by_id,nullzero"`
	ExpiresAt      *int64   `json:"expiresAt" keel:"expires_at"`
	// PositionID is the title the user holds in this organisation. It sits on
	// the membership rather than the user because a position belongs to one
	// organisation and the same person may hold different titles in two.
	PositionID pulid.ID `json:"positionId" keel:"position_id,nullzero"`

	// Relationships
	User         *User         `json:"user,omitempty" keel:"belongs:user_id"`
	GrantedBy    *User         `json:"grantedBy,omitempty" keel:"belongs:granted_by_id"`
	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty"`
	Organization *Organization `json:"organization,omitempty" keel:"belongs:organization_id"`
}

func (om *OrganizationMembership) BeforeCreate(ctx context.Context) error {
	if om.JoinedAt == 0 {
		om.JoinedAt = timeutils.NowUnix()
	}

	return nil
}

var (
	_ domaintypes.PostgresSearchable     = (*User)(nil)
	_ validationframework.TenantedEntity = (*User)(nil)
)

const (
	UserIDPrefix   = "usr_"
	SystemUsername = "system"
)

type User struct {
	keel.Table `keel:"users"`

	ID                    pulid.ID               `json:"id" keel:"id,pk,autoid:usr_"`
	BusinessUnitID        pulid.ID               `json:"businessUnitId" keel:"business_unit_id,tenant"`
	CurrentOrganizationID pulid.ID               `json:"currentOrganizationId" keel:"current_organization_id"`
	Status                domaintypes.Status     `json:"status" keel:"status"`
	Name                  string                 `json:"name" keel:"name"`
	Username              string                 `json:"username" keel:"username"`
	TimeFormat            domaintypes.TimeFormat `json:"timeFormat" keel:"time_format"`
	Password              string                 `json:"-" keel:"password"` // Hidden in responses
	EmailAddress          string                 `json:"emailAddress" keel:"email_address"`
	ProfilePicURL         string                 `json:"profilePicUrl" keel:"profile_pic_url,nullzero"`
	ThumbnailURL          string                 `json:"thumbnailUrl" keel:"thumbnail_url,nullzero"`
	Timezone              string                 `json:"timezone" keel:"timezone"`
	Locale                string                 `json:"locale" keel:"locale"`
	IsLocked              bool                   `json:"isLocked" keel:"is_locked"`
	MustChangePassword    bool                   `json:"mustChangePassword" keel:"must_change_password"`
	Version               int64                  `json:"version" keel:"version,version"`
	CreatedAt             int64                  `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt             int64                  `json:"updatedAt" keel:"updated_at,autoupdate"`
	LastLoginAt           *int64                 `json:"lastLoginAt,omitzero" keel:"last_login_at"`
	JobTitle              string                 `json:"jobTitle" keel:"job_title"`
	PhoneNumber           string                 `json:"phoneNumber" keel:"phone_number"`
	AvatarStyle           AvatarStyle            `json:"avatarStyle" keel:"avatar_style"`
	AvailabilityStatus    AvailabilityStatus     `json:"availabilityStatus" keel:"availability_status"`
	FocusUntil            *int64                 `json:"focusUntil" keel:"focus_until"`
	AwayBackOn            *int64                 `json:"awayBackOn" keel:"away_back_on"`
	AwayDelegateUserID    *pulid.ID              `json:"awayDelegateUserId" keel:"away_delegate_user_id"`
	AwayNote              string                 `json:"awayNote" keel:"away_note"`
	PasswordChangedAt     *int64                 `json:"passwordChangedAt" keel:"password_changed_at"`
	AssistantHidden       bool                   `json:"assistantHidden" keel:"assistant_hidden"`

	BusinessUnit        *BusinessUnit                    `json:"-"`
	CurrentOrganization *Organization                    `json:"-" keel:"belongs:current_organization_id"`
	Memberships         []*OrganizationMembership        `json:"memberships,omitempty" keel:"many:user_id"`
	Assignments         []*permission.UserRoleAssignment `json:"assignments,omitempty" keel:"many:user_id"`
}

func (u *User) BeforeCreate(_ context.Context) error {
	if u.Status == "" {
		u.Status = domaintypes.StatusActive
	}
	if u.TimeFormat == "" {
		u.TimeFormat = domaintypes.TimeFormat12Hour
	}
	if u.Locale == "" {
		u.Locale = defaultUserLocale
	}
	if u.AvatarStyle == "" {
		u.AvatarStyle = AvatarStyleAuto
	}
	if u.AvailabilityStatus == "" {
		u.AvailabilityStatus = AvailabilityAvailable
	}

	return nil
}

func (u *User) GeneratePassword(raw string) (string, error) {
	if len(raw) > passwordutils.MaxBytes {
		return "", errortypes.NewValidationError(
			"password",
			errortypes.ErrInvalid,
			"Password must be 72 bytes or fewer",
		)
	}
	hashed, err := bcrypt.GenerateFromPassword([]byte(raw), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hashed), nil
}

var dummyPasswordHash = sync.OnceValue(func() []byte {
	hashed, err := bcrypt.GenerateFromPassword(
		[]byte(pulid.MustNew("dpw_").String()),
		bcrypt.DefaultCost,
	)
	if err != nil {
		panic(fmt.Sprintf("hash the dummy password: %v", err))
	}
	return hashed
})

func DiscardPasswordCheck(raw string) {
	_ = bcrypt.CompareHashAndPassword(dummyPasswordHash(), []byte(raw))
}

func (u *User) VerifyPassword(raw string) error {
	return u.verifyPassword(raw)
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

func NewDuplicateEmailAddressError() *errortypes.Error {
	return errortypes.NewValidationError(
		"emailAddress",
		errortypes.ErrDuplicate,
		"A user with this email address already exists",
	)
}

func NewNoOrganizationMembershipsError() *errortypes.Error {
	return errortypes.NewValidationError(
		"organizationIds",
		errortypes.ErrRequired,
		"A user must keep access to at least one organization. Deactivate the user to remove all access",
	)
}

func NewDuplicateUsernameError() *errortypes.Error {
	return errortypes.NewValidationError(
		"username",
		errortypes.ErrDuplicate,
		"A user with this username already exists",
	)
}

func (u *User) AwaitingAccountSetup() bool {
	return u.LastLoginAt == nil && u.PasswordChangedAt == nil
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

func (u *User) GetID() pulid.ID {
	return u.ID
}

func (u *User) GetCreatedAt() int64 {
	return u.CreatedAt
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
		validation.Field(&u.Locale,
			validation.By(domainvalidation.ValidateLocale),
		),
		validation.Field(&u.Status,
			validation.Required.Error("Status is required"),
			validation.In(domaintypes.StatusActive, domaintypes.StatusInactive).
				Error("Status must be either Active or Inactive"),
		),
		validation.Field(&u.TimeFormat,
			validation.In(domaintypes.TimeFormat12Hour, domaintypes.TimeFormat24Hour).
				Error("Time format must be either 12-hour or 24-hour"),
		),
		validation.Field(&u.JobTitle,
			validation.Length(0, 100).Error("Job title must be at most 100 characters"),
		),
		validation.Field(&u.PhoneNumber,
			validation.Length(0, 30).Error("Phone number must be at most 30 characters"),
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
			{Field: "roles", Queryable: true},
		},
	}
}

func (m *OrganizationMembership) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&m.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&m.UserID, validation.Required.Error("User is required")),
		// A membership that expires before it began grants nothing while still
		// appearing in the user's organization list.
		validation.Field(&m.ExpiresAt,
			validation.Min(m.JoinedAt+1).Error("Expiry must be after the membership began"),
		),
	))
}
