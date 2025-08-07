/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package user

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/user"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/core/services/auth"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/utils/stringutils"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.UserRepository
	AuditService services.AuditService
	PermService  services.PermissionService
	AuthService  *auth.Service
	EmailService services.EmailService
}

type Service struct {
	repo repositories.UserRepository
	l    *zerolog.Logger
	ps   services.PermissionService
	as   services.AuditService
	auth *auth.Service
	es   services.EmailService
}

// NewService creates a new user service
//
//nolint:gocritic // params are dependency injected
func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "user").
		Logger()

	return &Service{
		repo: p.Repo,
		l:    &log,
		ps:   p.PermService,
		as:   p.AuditService,
		auth: p.AuthService,
		es:   p.EmailService,
	}
}

func (s *Service) SelectOptions(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, repositories.ListUserRequest{
		Filter: opts,
	})
	if err != nil {
		return nil, eris.Wrap(err, "select users")
	}

	options := make([]*types.SelectOption, len(result.Items))
	for i, u := range result.Items {
		options[i] = &types.SelectOption{
			Value: u.ID.String(),
			Label: u.Name,
		}
	}

	return options, nil
}

func (s *Service) List(
	ctx context.Context,
	opts repositories.ListUserRequest,
) (*ports.ListResult[*user.User], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceUser,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.Filter.TenantOpts.BuID,
				OrganizationID: opts.Filter.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read users")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list users")
		return nil, err
	}

	return &ports.ListResult[*user.User]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	opts repositories.GetUserByIDOptions,
) (*user.User, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("shipmentID", opts.UserID.String()).
		Logger()

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get user")
		return nil, err
	}

	return entity, nil
}

func (s *Service) Create(
	ctx context.Context,
	u *user.User,
	userID pulid.ID,
) (*user.User, error) {
	log := s.l.With().
		Str("operation", "Create").
		Interface("user", u).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         userID,
			Resource:       permission.ResourceUser,
			Action:         permission.ActionCreate,
			BusinessUnitID: u.BusinessUnitID,
			OrganizationID: u.CurrentOrganizationID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to create a user")
	}

	temporaryPassword := stringutils.GenerateSecurePassword(12)

	hashed, err := u.GeneratePassword(temporaryPassword)
	if err != nil {
		return nil, err
	}

	u.Password = hashed
	u.MustChangePassword = true

	createdEntity, err := s.repo.Create(ctx, u)
	if err != nil {
		return nil, err
	}

	emailHTMLBody := fmt.Sprintf(`
		<html>
			<body>
				<h2>Welcome to Trenova</h2>
				<p>Your account has been created successfully.</p>
				<p><strong>Email:</strong> %s</p>
				<p><strong>Temporary Password:</strong> %s</p>
				<p>Please log in and change your password immediately.</p>
			</body>
		</html>
	`, u.EmailAddress, temporaryPassword)

	emailTextBody := fmt.Sprintf(`
Welcome to Trenova

Your account has been created successfully.

Email: %s
Temporary Password: %s

Please log in and change your password immediately.
	`, u.EmailAddress, temporaryPassword)

	_, err = s.es.SendEmail(ctx, &services.SendEmailRequest{
		OrganizationID: createdEntity.CurrentOrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
		UserID:         userID,
		Subject:        "Welcome to Trenova - Account Created",
		To:             []string{u.EmailAddress},
		HTMLBody:       emailHTMLBody,
		TextBody:       emailTextBody,
		Priority:       email.PriorityHigh,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to send email")
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceUser,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.CurrentOrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("User created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log user creation")
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	u *user.User,
	userID pulid.ID,
) (*user.User, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("userID", u.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         userID,
			Resource:       permission.ResourceUser,
			Action:         permission.ActionUpdate,
			BusinessUnitID: u.BusinessUnitID,
			OrganizationID: u.CurrentOrganizationID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this user",
		)
	}

	original, err := s.repo.GetByID(ctx, repositories.GetUserByIDOptions{
		OrgID:        u.CurrentOrganizationID,
		BuID:         u.BusinessUnitID,
		UserID:       u.ID,
		IncludeRoles: true,
	})
	if err != nil {
		log.Error().Err(err).Str("userID", u.ID.String()).Msg("failed to get user")
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, u)
	if err != nil {
		log.Error().Err(err).Interface("user", u).Msg("failed to update user")
		return nil, err
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceUser,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.CurrentOrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("User updated"),
		audit.WithCritical(),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log user update")
	}

	return updatedEntity, nil
}

func (s *Service) ChangePassword(
	ctx context.Context,
	req *repositories.ChangePasswordRequest,
) (*user.User, error) {
	log := s.l.With().
		Str("operation", "ChangePassword").
		Str("userID", req.UserID.String()).
		Logger()

	currentUser, err := s.repo.GetByID(ctx, repositories.GetUserByIDOptions{
		OrgID:        req.OrgID,
		BuID:         req.BuID,
		UserID:       req.UserID,
		IncludeRoles: true,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get user")
		return nil, err
	}

	if err = currentUser.VerifyPassword(req.CurrentPassword); err != nil {
		return nil, errors.NewValidationError(
			"currentPassword",
			errors.ErrInvalid,
			"Current password is incorrect",
		)
	}

	if req.NewPassword != req.ConfirmPassword {
		return nil, errors.NewValidationError(
			"confirmPassword",
			errors.ErrInvalid,
			"New password and confirm password do not match",
		)
	}

	hashedPassword, err := currentUser.GeneratePassword(req.NewPassword)
	if err != nil {
		log.Error().Err(err).Msg("failed to generate password")
		return nil, err
	}

	req.HashedPassword = hashedPassword

	updatedUser, err := s.repo.ChangePassword(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to update user")
		return nil, err
	}

	return updatedUser, nil
}

func (s *Service) SwitchOrganization(
	ctx context.Context,
	userID, newOrgID, sessionID pulid.ID,
) (*user.User, error) {
	log := s.l.With().
		Str("operation", "SwitchOrganization").
		Str("userID", userID.String()).
		Str("newOrgID", newOrgID.String()).
		Str("sessionID", sessionID.String()).
		Logger()

	// * Get the original user for audit logging
	originalUser, err := s.repo.GetByID(ctx, repositories.GetUserByIDOptions{
		UserID:      userID,
		IncludeOrgs: true,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get original user")
		return nil, err
	}

	// * Switch the organization
	updatedUser, err := s.repo.SwitchOrganization(ctx, userID, newOrgID)
	if err != nil {
		log.Error().Err(err).Msg("failed to switch organization")
		return nil, err
	}

	log.Info().
		Str("userID", userID.String()).
		Str("newOrgID", newOrgID.String()).
		Msg("organization switched successfully")

	// * Log the organization switch
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceUser,
			ResourceID:     updatedUser.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedUser),
			PreviousState:  jsonutils.MustToJSON(originalUser),
			OrganizationID: updatedUser.CurrentOrganizationID,
			BusinessUnitID: updatedUser.BusinessUnitID,
		},
		audit.WithComment("Organization switched"),
		audit.WithCritical(),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log organization switch")
		// ! Don't fail the operation if audit logging fails
	}

	// * Update the session with the new organization ID
	if sessionID.IsNotNil() {
		if err = s.auth.UpdateSessionOrganization(ctx, sessionID, newOrgID); err != nil {
			log.Error().Err(err).Msg("failed to update session organization")
			// ! Don't fail the operation if session update fails, but log it
		} else {
			log.Debug().Msg("session organization updated successfully")
		}
	}

	log.Info().
		Str("previousOrgID", originalUser.CurrentOrganizationID.String()).
		Str("newOrgID", updatedUser.CurrentOrganizationID.String()).
		Msg("organization switched successfully")

	return updatedUser, nil
}
