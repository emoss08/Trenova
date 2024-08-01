package factory

import (
	"context"
	"errors"

	"golang.org/x/crypto/bcrypt"

	"github.com/emoss08/trenova/pkg/models"
	"github.com/uptrace/bun"
)

type UserFactory struct {
	db *bun.DB
}

func NewUserFactory(db *bun.DB) *UserFactory {
	return &UserFactory{db: db}
}

func (u *UserFactory) CreateOrGetUser(ctx context.Context) (*models.User, error) {
	org, err := NewOrganizationFactory(u.db).MustCreateOrganization(ctx)
	if err != nil {
		return nil, err
	}

	exists, err := u.db.NewSelect().Model((*models.User)(nil)).Where("username = ?", "test_admin").Exists(ctx)
	if err != nil {
		return nil, err
	}

	if !exists {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}

		user := &models.User{
			OrganizationID: org.ID,
			Organization:   org,
			BusinessUnitID: org.BusinessUnitID,
			BusinessUnit:   org.BusinessUnit,
			Status:         "Active",
			Username:       "admin",
			Password:       string(hashedPassword),
			Email:          "admin@trenova.app",
			Name:           "System Administrator",
			IsAdmin:        true,
			Timezone:       "America/New_York",
		}

		_, err = u.db.NewInsert().Model(user).Exec(ctx)
		if err != nil {
			return nil, err
		}

		return user, nil
	}

	return nil, errors.New("cannot get user")

}
