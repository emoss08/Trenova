package models

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/pkg/models/property"
	"github.com/emoss08/trenova/pkg/validator"
	validation "github.com/go-ozzo/ozzo-validation/v4"

	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/google/uuid"
	"github.com/uptrace/bun"
	"golang.org/x/crypto/bcrypt"
)

// TODO(Wolfred): At some point the user should be able to have multiple organizations
// Within the same business unit. This will require a many to many relationship between
// the user and the organization.
// However, we should store the current organization the user is working with in the session or
// in the user model itself.
// This will ensure that the user is only able to access the organization they are currently working with.

type User struct {
	bun.BaseModel `bun:"table:users,alias:u" json:"-"`

	ID            uuid.UUID       `bun:",pk,type:uuid,default:uuid_generate_v4()" json:"id"`
	Status        property.Status `bun:"status,type:status" json:"status"`
	Name          string          `bun:"name" json:"name" queryField:"true"`
	Username      string          `bun:"username,notnull" json:"username"`
	Password      string          `json:"-"`
	Email         string          `bun:"email,notnull,unique" json:"email"`
	Timezone      string          `bun:"timezone,notnull" json:"timezone"`
	ProfilePicURL string          `bun:"profile_pic_url" json:"profilePicUrl"`
	ThumbnailURL  string          `bun:"thumbnail_url" json:"thumbnailUrl"`
	PhoneNumber   string          `bun:"phone_number" json:"phoneNumber"`
	IsAdmin       bool            `bun:"is_admin,default:false" json:"isAdmin"`
	Version       int64           `bun:"type:BIGINT" json:"version"`
	CreatedAt     time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"createdAt"`
	UpdatedAt     time.Time       `bun:",nullzero,notnull,default:current_timestamp" json:"updatedAt"`

	BusinessUnitID uuid.UUID `bun:"type:uuid,notnull" json:"businessUnitId"`
	OrganizationID uuid.UUID `bun:"type:uuid,notnull" json:"organizationId"`

	BusinessUnit *BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
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

func (u *User) BeforeUpdate(_ context.Context) error {
	u.Version++

	return nil
}

func (u *User) OptimisticUpdate(ctx context.Context, tx bun.IDB) error {
	ov := u.Version

	if err := u.BeforeUpdate(ctx); err != nil {
		return err
	}

	result, err := tx.NewUpdate().
		Model(u).
		WherePK().
		Where("version = ?", ov).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return err
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rows == 0 {
		return &validator.BusinessLogicError{
			Message: fmt.Sprintf("Version mismatch. The User (ID: %s) has been updated by another user. Please refresh and try again.", u.ID),
		}
	}

	return nil
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
