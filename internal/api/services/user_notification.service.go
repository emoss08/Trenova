package services

import (
	"context"
	"log"

	"github.com/emoss08/trenova/internal/api"
	"github.com/emoss08/trenova/internal/ent"
	"github.com/emoss08/trenova/internal/ent/organization"
	"github.com/emoss08/trenova/internal/ent/user"
	"github.com/emoss08/trenova/internal/ent/usernotification"
	"github.com/emoss08/trenova/internal/util"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type UserNotificationService struct {
	Client *ent.Client
	Logger *zerolog.Logger
}

// NewUserNotificationService creates a new user notification service.
func NewUserNotificationService(s *api.Server) *UserNotificationService {
	return &UserNotificationService{
		Client: s.Client,
		Logger: s.Logger,
	}
}

// GetUserNotifications gets the user notifications for a user.
func (r *UserNotificationService) GetUserNotifications(
	ctx context.Context, limit int, userID, buID, orgID uuid.UUID,
) ([]*ent.UserNotification, int, error) {
	entityCount, countErr := r.Client.UserNotification.Query().Where(
		usernotification.HasOrganizationWith(
			organization.IDEQ(orgID),
			organization.BusinessUnitIDEQ(buID),
		),
		usernotification.IsReadEQ(false),
		usernotification.HasUserWith(
			user.IDEQ(userID),
		),
	).Count(ctx)

	if countErr != nil {
		return nil, 0, countErr
	}

	entities, err := r.Client.UserNotification.Query().
		Limit(limit).
		Where(
			usernotification.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
			usernotification.IsReadEQ(false),
			usernotification.HasUserWith(
				user.IDEQ(userID),
			),
		).Order(
		ent.Desc(
			usernotification.FieldCreatedAt,
		),
	).All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return entities, entityCount, nil
}

func (r *UserNotificationService) CreateUserNotification(
	ctx context.Context, orgID, buID, userID uuid.UUID, title, description, actionURL string,
) error {
	err := util.WithTx(ctx, r.Client, func(tx *ent.Tx) error {
		var err error
		err = r.createUserNotificationEntity(ctx, tx, orgID, buID, userID, title, description, actionURL)
		if err != nil {
			return err
		}

		return nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (r *UserNotificationService) createUserNotificationEntity(
	ctx context.Context, tx *ent.Tx, orgID, buID, userID uuid.UUID, title, description, actionURL string,
) error {
	err := tx.UserNotification.Create().
		SetOrganizationID(orgID).
		SetBusinessUnitID(buID).
		SetUserID(userID).
		SetDescription(description).
		SetActionURL(actionURL).
		SetTitle(title).
		Exec(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (r *UserNotificationService) MarkNotificationsAsRead(
	ctx context.Context, orgID, buID, userID uuid.UUID,
) error {
	log.Printf("Marking notifications as read for user %s in organization %s", userID, orgID)
	asRead, err := r.Client.UserNotification.Update().
		Where(
			usernotification.HasOrganizationWith(
				organization.IDEQ(orgID),
				organization.BusinessUnitIDEQ(buID),
			),
			usernotification.HasUserWith(
				user.IDEQ(userID),
			),
		).SetIsRead(true).
		Save(ctx)
	if err != nil {
		log.Printf("Failed to delete user notifications: %v", err)
		return err
	}

	log.Printf("Marked %d notifications as read", asRead)

	return nil
}
