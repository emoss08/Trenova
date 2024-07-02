package models

import (
	"context"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

// UserPermission is a type for user permissions
type UserPermission string

const (
	// PermissionUserView is the permission to view user details
	PermissionUserView = UserPermission("user.view")

	// PermissionUserEdit is the permission to edit user details
	PermissionUserEdit = UserPermission("user.edit")

	// PermissionUserAdd is the permission to add a new user
	PermissionUserAdd = UserPermission("user.add")

	// PermissionUserDelete is the permission to delete an user
	PermissionUserDelete = UserPermission("user.delete")
)

// String returns the string representation of the UserPermission
func (p UserPermission) String() string {
	return string(p)
}

// TODO(Wolfred): At some point the user should be able to have multiple organizations
// Within the same business unit. This will require a many to many relationship between
// the user and the organization.
// However, we should store the current organization the user is working with in the session or
// in the user model itself.
// This will ensure that the user is only able to access the organization they are currently working with.

// User is the model for the user.
type User struct {
	bun.BaseModel  `bun:"table:users,alias:u" json:"-"`
	CreatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt      time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	ID             uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status         property.Status `bun:"status,type:status" json:"status"`
	Name           string          `bun:"name" json:"name" queryField:"true"`
	Username       string          `bun:"username,notnull" json:"username"`
	Password       string          `json:"-"`
	Email          string          `bun:"email,notnull,unique" json:"email"`
	Timezone       string          `bun:"timezone,notnull" json:"timezone"`
	ProfilePicURL  string          `bun:"profile_pic_url" json:"profilePicUrl"`
	ThumbnailURL   string          `bun:"thumbnail_url" json:"thumbnailUrl"`
	PhoneNumber    string          `bun:"phone_number" json:"phoneNumber"`
	IsAdmin        bool            `bun:"is_admin,default:false" json:"isAdmin"`
	BusinessUnitID uuid.UUID       `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID       `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
	Roles        []*Role       `bun:"m2m:user_roles,join:User=Role" json:"roles"`
}

func (u User) Validate() error {
	return validation.ValidateStruct(&u,
		validation.Field(&u.Name, validation.Required),
		validation.Field(&u.Username, validation.Required),
		validation.Field(&u.Email, validation.Required, is.Email),
		validation.Field(&u.Timezone, validation.Required),
		validation.Field(&u.BusinessUnitID, validation.Required),
		validation.Field(&u.OrganizationID, validation.Required),
	)
}

// Generate return a hashed password.
func (u *User) GeneratePassword(raw string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(raw), 10)
	if err != nil {
		panic(err)
	}

	return string(hash)
}

// Verify the users current password with the raw password.
func (u *User) VerifyPassword(raw string) error {
	return bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(raw))
}

var _ bun.BeforeAppendModelHook = (*User)(nil)

func (u *User) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		u.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		u.UpdatedAt = time.Now()
	}
	return nil
}

// UserRole is the model for the user roles.
type UserRole struct {
	bun.BaseModel `bun:"table:user_roles" json:"-"`
	CreatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`
	// `UserID` is the foreign key to the user.
	UserID uuid.UUID `bun:",pk,type:uuid" json:"userId"`
	// `User` is the user for the role.
	User *User `bun:"rel:belongs-to,join:user_id=id" json:"-"`
	// `RoleID` is the foreign key to the role.
	RoleID uuid.UUID `bun:",pk,type:uuid" json:"roleId"`
	// `Role` is the role for the user.
	Role *Role `bun:"rel:belongs-to,join:role_id=id" json:"-"`
}

var _ bun.BeforeAppendModelHook = (*UserRole)(nil)

func (c *UserRole) BeforeAppendModel(_ context.Context, query bun.Query) error {
	switch query.(type) {
	case *bun.InsertQuery:
		c.CreatedAt = time.Now()
	case *bun.UpdateQuery:
		c.UpdatedAt = time.Now()
	}
	return nil
}
