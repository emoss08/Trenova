package tenant

import (
	"testing"

	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/uptrace/bun"
)

func validUser() *User {
	return &User{
		ID:                    pulid.MustNew("usr_"),
		BusinessUnitID:        pulid.MustNew("bu_"),
		CurrentOrganizationID: pulid.MustNew("org_"),
		Status:                domaintypes.StatusActive,
		Name:                  "John Doe",
		Username:              "johndoe",
		EmailAddress:          "john@example.com",
		Timezone:              "America/New_York",
		Password:              "hashedpassword",
	}
}

func TestUser_Validate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modify  func(u *User)
		wantErr bool
	}{
		{
			name:    "valid entity passes",
			modify:  func(_ *User) {},
			wantErr: false,
		},
		{
			name: "missing name fails",
			modify: func(u *User) {
				u.Name = ""
			},
			wantErr: true,
		},
		{
			name: "missing username fails",
			modify: func(u *User) {
				u.Username = ""
			},
			wantErr: true,
		},
		{
			name: "username too long fails",
			modify: func(u *User) {
				u.Username = "abcdefghijklmnopqrstu"
			},
			wantErr: true,
		},
		{
			name: "username at max length passes",
			modify: func(u *User) {
				u.Username = "abcdefghijklmnopqrst"
			},
			wantErr: false,
		},
		{
			name: "missing email fails",
			modify: func(u *User) {
				u.EmailAddress = ""
			},
			wantErr: true,
		},
		{
			name: "invalid email format fails",
			modify: func(u *User) {
				u.EmailAddress = "notanemail"
			},
			wantErr: true,
		},
		{
			name: "valid email passes",
			modify: func(u *User) {
				u.EmailAddress = "user@example.com"
			},
			wantErr: false,
		},
		{
			name: "missing timezone fails",
			modify: func(u *User) {
				u.Timezone = ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u := validUser()
			tt.modify(u)

			multiErr := errortypes.NewMultiError()
			u.Validate(multiErr)

			if tt.wantErr {
				assert.True(t, multiErr.HasErrors())
			} else {
				assert.False(t, multiErr.HasErrors())
			}
		})
	}
}

func TestUser_GetTableName(t *testing.T) {
	t.Parallel()

	u := &User{}
	assert.Equal(t, "users", u.GetTableName())
}

func TestUser_GeneratePassword(t *testing.T) {
	t.Parallel()

	t.Run("generates a hashed password", func(t *testing.T) {
		t.Parallel()

		u := &User{}
		hashed, err := u.GeneratePassword("mysecretpassword")
		require.NoError(t, err)

		assert.NotEmpty(t, hashed)
		assert.NotEqual(t, "mysecretpassword", hashed)
	})

	t.Run("different calls produce different hashes", func(t *testing.T) {
		t.Parallel()

		u := &User{}
		h1, err := u.GeneratePassword("password123")
		require.NoError(t, err)

		h2, err := u.GeneratePassword("password123")
		require.NoError(t, err)

		assert.NotEqual(t, h1, h2)
	})
}

func TestUser_VerifyCredentials(t *testing.T) {
	t.Parallel()

	t.Run("correct password succeeds for active user", func(t *testing.T) {
		t.Parallel()

		u := validUser()
		hashed, err := u.GeneratePassword("correctpassword")
		require.NoError(t, err)
		u.Password = hashed

		err = u.VerifyCredentials("correctpassword")
		assert.NoError(t, err)
	})

	t.Run("wrong password fails", func(t *testing.T) {
		t.Parallel()

		u := validUser()
		hashed, err := u.GeneratePassword("correctpassword")
		require.NoError(t, err)
		u.Password = hashed

		err = u.VerifyCredentials("wrongpassword")
		assert.Error(t, err)
	})

	t.Run("inactive user fails", func(t *testing.T) {
		t.Parallel()

		u := validUser()
		u.Status = domaintypes.StatusInactive
		hashed, err := u.GeneratePassword("password")
		require.NoError(t, err)
		u.Password = hashed

		err = u.VerifyCredentials("password")
		assert.Error(t, err)
	})

	t.Run("locked user fails", func(t *testing.T) {
		t.Parallel()

		u := validUser()
		u.IsLocked = true
		hashed, err := u.GeneratePassword("password")
		require.NoError(t, err)
		u.Password = hashed

		err = u.VerifyCredentials("password")
		assert.Error(t, err)
	})
}

func TestUser_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and CreatedAt", func(t *testing.T) {
		t.Parallel()

		u := &User{}
		require.True(t, u.ID.IsNil())

		err := u.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.False(t, u.ID.IsNil())
		assert.NotZero(t, u.CreatedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()

		existingID := pulid.MustNew("usr_")
		u := &User{ID: existingID}

		err := u.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)

		assert.Equal(t, existingID, u.ID)
	})

	t.Run("update sets UpdatedAt", func(t *testing.T) {
		t.Parallel()

		u := &User{}

		err := u.BeforeAppendModel(t.Context(), (*bun.UpdateQuery)(nil))
		require.NoError(t, err)

		assert.NotZero(t, u.UpdatedAt)
	})
}

func TestUser_GetID(t *testing.T) {
	t.Parallel()
	id := pulid.MustNew("usr_")
	u := &User{ID: id}
	assert.Equal(t, id, u.GetID())
}

func TestUser_GetOrganizationID(t *testing.T) {
	t.Parallel()
	orgID := pulid.MustNew("org_")
	u := &User{CurrentOrganizationID: orgID}
	assert.Equal(t, orgID, u.GetOrganizationID())
}

func TestUser_GetBusinessUnitID(t *testing.T) {
	t.Parallel()
	buID := pulid.MustNew("bu_")
	u := &User{BusinessUnitID: buID}
	assert.Equal(t, buID, u.GetBusinessUnitID())
}

func TestOrganizationMembership_BeforeAppendModel(t *testing.T) {
	t.Parallel()

	t.Run("insert sets ID and JoinedAt", func(t *testing.T) {
		t.Parallel()
		om := &OrganizationMembership{}
		err := om.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)
		assert.False(t, om.ID.IsNil())
		assert.NotZero(t, om.JoinedAt)
	})

	t.Run("insert does not overwrite existing ID", func(t *testing.T) {
		t.Parallel()
		existingID := pulid.MustNew("uom_")
		om := &OrganizationMembership{ID: existingID}
		err := om.BeforeAppendModel(t.Context(), (*bun.InsertQuery)(nil))
		require.NoError(t, err)
		assert.Equal(t, existingID, om.ID)
	})
}

func TestUser_IsActive(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status domaintypes.Status
		want   bool
	}{
		{name: "active status returns true", status: domaintypes.StatusActive, want: true},
		{name: "inactive status returns false", status: domaintypes.StatusInactive, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			u := &User{Status: tt.status}
			assert.Equal(t, tt.want, u.IsActive())
		})
	}
}
