package factory

import (
	"context"
	"database/sql"
	"errors"
	"log"

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

	user := new(models.User)
	err = u.db.NewSelect().Model(user).Where("username = ?", "admin").Scan(ctx)

	if err == nil {
		// User exists, return the existing user
		return user, nil
	}

	if !errors.Is(err, sql.ErrNoRows) {
		// An unexpected error occurred
		return nil, err
	}

	// User does not exist, create a new one
	log.Printf("User does not exist, creating user")
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin"), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	newUser := &models.User{
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

	_, err = u.db.NewInsert().Model(newUser).Exec(ctx)
	if err != nil {
		return nil, err
	}

	return newUser, nil
}
