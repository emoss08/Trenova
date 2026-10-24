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
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/passwordutils"
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
	keel.Table    `keel:"user_organization_memberships"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:uom_"`
	IsDefault      bool     `json:"isDefault"      bun:"is_default" keel:"is_default,nullzero"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull" keel:"business_unit_id,tenant"`
	UserID         pulid.ID `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull" keel:"user_id"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,tenant"`
	JoinedAt       int64    `json:"joinedAt"       bun:"joined_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"joined_at,nullzero"`
	GrantedByID    pulid.ID `json:"grantedByID"    bun:"granted_by_id,type:VARCHAR(100)" keel:"granted_by_id,nullzero"`
	ExpiresAt      *int64   `json:"expiresAt"      bun:"expires_at" keel:"expires_at"`
	// PositionID is the title the user holds in this organisation. It sits on
	// the membership rather than the user because a position belongs to one
	// organisation and the same person may hold different titles in two.
	PositionID pulid.ID `json:"positionId" bun:"position_id,type:VARCHAR(100),nullzero" keel:"position_id,nullzero"`

	// Relationships
	User         *User         `json:"user,omitempty"         bun:"rel:belongs-to,join:user_id=id" keel:"belongs:user_id"`
	GrantedBy    *User         `json:"grantedBy,omitempty"    bun:"rel:belongs-to,join:granted_by_id=id" keel:"belongs:granted_by_id"`
	BusinessUnit *BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id" keel:"belongs:organization_id"`
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

const (
	UserIDPrefix   = "usr_"
	SystemUsername = "system"
)

type User struct {
	bun.BaseModel `bun:"table:users,alias:usr" json:"-"`
	keel.Table    `keel:"users"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                    pulid.ID               `json:"id"                    bun:"id,pk,type:VARCHAR(100)" keel:"id,pk"`
	BusinessUnitID        pulid.ID               `json:"businessUnitId"        bun:"business_unit_id,type:VARCHAR(100),notnull" keel:"business_unit_id,tenant"`
	CurrentOrganizationID pulid.ID               `json:"currentOrganizationId" bun:"current_organization_id,type:VARCHAR(100),notnull" keel:"current_organization_id"`
	Status                domaintypes.Status     `json:"status"                bun:"status,type:status_enum,notnull,default:'Active'" keel:"status"`
	Name                  string                 `json:"name"                  bun:"name,type:VARCHAR(255),notnull" keel:"name"`
	Username              string                 `json:"username"              bun:"username,type:VARCHAR(20),notnull" keel:"username"`
	TimeFormat            domaintypes.TimeFormat `json:"timeFormat"            bun:"time_format,type:time_format_enum,notnull,default:'12-hour'" keel:"time_format"`
	Password              string                 `json:"-"                     bun:"password,type:VARCHAR(255),notnull" keel:"password"` // Hidden in responses
	EmailAddress          string                 `json:"emailAddress"          bun:"email_address,type:VARCHAR(255),notnull" keel:"email_address"`
	ProfilePicURL         string                 `json:"profilePicUrl"         bun:"profile_pic_url,type:VARCHAR(255)" keel:"profile_pic_url,nullzero"`
	ThumbnailURL          string                 `json:"thumbnailUrl"          bun:"thumbnail_url,type:VARCHAR(255)" keel:"thumbnail_url,nullzero"`
	Timezone              string                 `json:"timezone"              bun:"timezone,type:VARCHAR(50),notnull" keel:"timezone"`
	Locale                string                 `json:"locale"                bun:"locale,type:VARCHAR(10),notnull,default:'en'" keel:"locale"`
	IsLocked              bool                   `json:"isLocked"              bun:"is_locked,type:BOOLEAN,notnull" keel:"is_locked"`
	MustChangePassword    bool                   `json:"mustChangePassword"    bun:"must_change_password,type:BOOLEAN,notnull" keel:"must_change_password"`
	Version               int64                  `json:"version"               bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt             int64                  `json:"createdAt"             bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt             int64                  `json:"updatedAt"             bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
	LastLoginAt           *int64                 `json:"lastLoginAt,omitzero"  bun:"last_login_at,nullzero" keel:"last_login_at"`
	JobTitle              string                 `json:"jobTitle"              bun:"job_title,type:VARCHAR(100),notnull" keel:"job_title"`
	PhoneNumber           string                 `json:"phoneNumber"           bun:"phone_number,type:VARCHAR(30),notnull" keel:"phone_number"`
	AvatarStyle           AvatarStyle            `json:"avatarStyle"           bun:"avatar_style,type:VARCHAR(20),notnull,default:'auto'" keel:"avatar_style"`
	AvailabilityStatus    AvailabilityStatus     `json:"availabilityStatus"    bun:"availability_status,type:VARCHAR(20),notnull,default:'available'" keel:"availability_status"`
	FocusUntil            *int64                 `json:"focusUntil"            bun:"focus_until,nullzero" keel:"focus_until"`
	AwayBackOn            *int64                 `json:"awayBackOn"            bun:"away_back_on,nullzero" keel:"away_back_on"`
	AwayDelegateUserID    *pulid.ID              `json:"awayDelegateUserId"    bun:"away_delegate_user_id,type:VARCHAR(100),nullzero" keel:"away_delegate_user_id"`
	AwayNote              string                 `json:"awayNote"              bun:"away_note,type:VARCHAR(280),notnull" keel:"away_note"`
	PasswordChangedAt     *int64                 `json:"passwordChangedAt"     bun:"password_changed_at,nullzero" keel:"password_changed_at"`
	AssistantHidden       bool                   `json:"assistantHidden"       bun:"assistant_hidden,type:BOOLEAN,notnull,default:false" keel:"assistant_hidden"`

	BusinessUnit        *BusinessUnit                    `json:"-"                     bun:"rel:belongs-to,join:business_unit_id=id"`
	CurrentOrganization *Organization                    `json:"-"                     bun:"rel:belongs-to,join:current_organization_id=id" keel:"belongs:current_organization_id"`
	Memberships         []*OrganizationMembership        `json:"memberships,omitempty" bun:"rel:has-many,join:id=user_id" keel:"many:user_id"`
	Assignments         []*permission.UserRoleAssignment `json:"assignments,omitempty" bun:"rel:has-many,join:id=user_id" keel:"many:user_id"`
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

func (u *User) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if u.ID.IsNil() {
			u.ID = pulid.MustNew(UserIDPrefix)
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
