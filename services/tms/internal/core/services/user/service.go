package user

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.UserRepository
	AuditService services.AuditService
	AuthService  services.AuthService
	EmailService services.EmailService
}

type Service struct {
	l    *zap.Logger
	repo repositories.UserRepository
	as   services.AuditService
	auth services.AuthService
	es   services.EmailService
}

//nolint:gocritic // This is a constructor
func NewService(p ServiceParams) *Service {
	return &Service{
		l:    p.Logger.Named("service.user"),
		repo: p.Repo,
		as:   p.AuditService,
		auth: p.AuthService,
		es:   p.EmailService,
	}
}

func (s *Service) GetOption(
	ctx context.Context,
	req repositories.GetUserByIDRequest,
) (*tenant.User, error) {
	return s.repo.GetOption(ctx, req)
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req repositories.UserSelectOptionsRequest,
) ([]*repositories.UserSelectOptionResponse, error) {
	return s.repo.SelectOptions(ctx, req)
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListUserRequest,
) (*pagination.ListResult[*tenant.User], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetUserByIDRequest,
) (*tenant.User, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Create(
	ctx context.Context,
	entity *tenant.User,
	userID pulid.ID,
) (*tenant.User, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.String("buID", entity.BusinessUnitID.String()),
		zap.String("orgID", entity.CurrentOrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	temporaryPassword := utils.GenerateSecurePassword(12)

	hashed, err := entity.GeneratePassword(temporaryPassword)
	if err != nil {
		return nil, err
	}

	entity.Password = hashed
	entity.MustChangePassword = true

	createdEntity, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	err = s.es.SendSystemEmail(
		ctx,
		&services.SendSystemEmailRequest{
			TemplateKey: services.TemplateUserWelcome,
			To:          []string{entity.EmailAddress},
			Variables: map[string]any{
				"UserName":          entity.Name,
				"EmailAddress":      entity.EmailAddress,
				"TemporaryPassword": temporaryPassword,
				"Year":              utils.GetCurrentYear(),
			},
			OrgID:  entity.CurrentOrganizationID,
			BuID:   entity.BusinessUnitID,
			UserID: userID,
		},
	)
	if err != nil {
		log.Error("failed to send welcome email", zap.Error(err))
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceUser,
			ResourceID:     createdEntity.GetID(),
			Operation:      permission.OpCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.CurrentOrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("User created"),
	)
	if err != nil {
		log.Error("failed to log user creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *Service) UpdateMe(
	ctx context.Context,
	req *repositories.UpdateMeRequest,
) (*tenant.User, error) {
	return s.repo.UpdateMe(ctx, req)
}

func (s *Service) Update(
	ctx context.Context,
	u *tenant.User,
	userID pulid.ID,
) (*tenant.User, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("buID", u.BusinessUnitID.String()),
		zap.String("orgID", u.CurrentOrganizationID.String()),
		zap.String("userID", userID.String()),
	)

	original, err := s.repo.GetByID(ctx, repositories.GetUserByIDRequest{
		OrgID:        u.CurrentOrganizationID,
		BuID:         u.BusinessUnitID,
		UserID:       userID,
		IncludeRoles: true,
		IncludeOrgs:  true,
	})
	if err != nil {
		log.Error("failed to get user", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, u)
	if err != nil {
		log.Error("failed to update user", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceUser,
			ResourceID:     updatedEntity.GetID(),
			Operation:      permission.OpUpdate,
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
		log.Error("failed to log user update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *Service) ChangePassword(
	ctx context.Context,
	req *repositories.ChangePasswordRequest,
) (*tenant.User, error) {
	log := s.l.With(
		zap.String("operation", "ChangePassword"),
		zap.String("userID", req.UserID.String()),
	)

	currentUser, err := s.repo.GetByID(ctx, repositories.GetUserByIDRequest{
		OrgID:        req.OrgID,
		BuID:         req.BuID,
		UserID:       req.UserID,
		IncludeRoles: true,
		IncludeOrgs:  true,
	})
	if err != nil {
		log.Error("failed to get user", zap.Error(err))
		return nil, err
	}

	if err = currentUser.VerifyPassword(req.CurrentPassword); err != nil {
		return nil, errortypes.NewValidationError(
			"currentPassword",
			errortypes.ErrInvalid,
			"Current password is incorrect",
		)
	}

	if req.NewPassword != req.ConfirmPassword {
		return nil, errortypes.NewValidationError(
			"confirmPassword",
			errortypes.ErrInvalid,
			"New password and confirm password do not match",
		)
	}

	hashedPassword, err := currentUser.GeneratePassword(req.NewPassword)
	if err != nil {
		log.Error("failed to generate password", zap.Error(err))
		return nil, err
	}

	req.HashedPassword = hashedPassword

	updatedUser, err := s.repo.ChangePassword(ctx, req)
	if err != nil {
		log.Error("failed to update user", zap.Error(err))
		return nil, err
	}

	return updatedUser, nil
}

func (s *Service) SwitchOrganization(
	ctx context.Context,
	userID, newOrgID, sessionID pulid.ID,
) (*tenant.User, error) {
	log := s.l.With(
		zap.String("operation", "SwitchOrganization"),
		zap.String("userID", userID.String()),
		zap.String("newOrgID", newOrgID.String()),
		zap.String("sessionID", sessionID.String()),
	)

	originalUser, err := s.repo.GetByID(ctx, repositories.GetUserByIDRequest{
		UserID:       userID,
		OrgID:        newOrgID,
		BuID:         userID,
		IncludeOrgs:  true,
		IncludeRoles: true,
	})
	if err != nil {
		log.Error("failed to get original user", zap.Error(err))
		return nil, err
	}

	updatedUser, err := s.repo.SwitchOrganization(ctx, userID, newOrgID)
	if err != nil {
		log.Error("failed to switch organization", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceUser,
			ResourceID:     updatedUser.GetID(),
			Operation:      permission.OpUpdate,
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
		log.Error("failed to log organization switch", zap.Error(err))
		// ! Don't fail the operation if audit logging fails
	}

	if sessionID.IsNotNil() {
		if err = s.auth.UpdateSessionOrganization(ctx, sessionID, newOrgID); err != nil {
			log.Error("failed to update session organization", zap.Error(err))
			// ! Don't fail the operation if session update fails, but log it
		}
	}

	return updatedUser, nil
}
