package emailprofile

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/core/services/encryption"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/validator"
	"github.com/emoss08/trenova/pkg/validator/emailvalidator"

	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	Repo              repositories.EmailProfileRepository
	AuditService      services.AuditService
	EncryptionService encryption.Service
	Validator         *emailvalidator.EmailProfileValidator
}

type service struct {
	l                 *zap.Logger
	repo              repositories.EmailProfileRepository
	as                services.AuditService
	encryptionService encryption.Service
	v                 *emailvalidator.EmailProfileValidator
}

func NewService(p Params) services.EmailProfileService {
	return &service{
		l:                 p.Logger.With(zap.String("service", "email_profile")),
		repo:              p.Repo,
		encryptionService: p.EncryptionService,
		as:                p.AuditService,
		v:                 p.Validator,
	}
}

func (s *service) List(
	ctx context.Context,
	req *repositories.ListEmailProfileRequest,
) (*pagination.ListResult[*email.EmailProfile], error) {
	return s.repo.List(ctx, req)
}

func (s *service) Get(
	ctx context.Context,
	req repositories.GetEmailProfileByIDRequest,
) (*email.EmailProfile, error) {
	return s.repo.Get(ctx, req)
}

func (s *service) Create(
	ctx context.Context,
	profile *email.EmailProfile,
	userID pulid.ID,
) (*email.EmailProfile, error) {
	log := s.l.With(
		zap.String("operation", "Create"),
		zap.Any("profile", profile),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, profile); err != nil {
		return nil, err
	}

	if err := s.encryptSensitiveFields(profile, nil); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, profile)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceEmailProfile,
			ResourceID:     createdEntity.ID.String(),
			Operation:      permission.OpCreate,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			UserID:         userID,
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Email profile created"),
		audit.WithMetadata(map[string]any{
			"name": createdEntity.Name,
		}),
	)
	if err != nil {
		log.Error("failed to log email profile creation", zap.Error(err))
	}

	return createdEntity, nil
}

func (s *service) Update(
	ctx context.Context,
	profile *email.EmailProfile,
	userID pulid.ID,
) (*email.EmailProfile, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("profileID", profile.ID.String()),
	)

	valCtx := &validator.ValidationContext{
		IsCreate: false,
		IsUpdate: true,
	}

	if err := s.v.Validate(ctx, valCtx, profile); err != nil {
		return nil, err
	}

	existing, err := s.repo.Get(ctx, repositories.GetEmailProfileByIDRequest{
		OrgID:     profile.OrganizationID,
		BuID:      profile.BusinessUnitID,
		ProfileID: profile.ID,
	})
	if err != nil {
		log.Error("failed to get existing profile", zap.Error(err))
		return nil, err
	}

	if err = s.encryptSensitiveFields(profile, existing); err != nil {
		log.Error("failed to encrypt sensitive fields", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, profile)
	if err != nil {
		log.Error("failed to update email profile", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceEmailProfile,
			ResourceID:     updatedEntity.ID.String(),
			Operation:      permission.OpUpdate,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
			UserID:         userID,
		},
		audit.WithComment("Email profile updated"),
		audit.WithDiff(existing, updatedEntity),
	)
	if err != nil {
		log.Error("failed to log email profile update", zap.Error(err))
	}

	return updatedEntity, nil
}

func (s *service) SetDefault(
	ctx context.Context,
	req repositories.GetEmailProfileByIDRequest,
) error {
	profile, err := s.repo.Get(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	profile.IsDefault = true
	_, err = s.repo.Update(ctx, profile)
	return err
}

func (s *service) GetDefault(
	ctx context.Context,
	req repositories.GetEmailProfileByIDRequest,
) (*email.EmailProfile, error) {
	return s.repo.GetDefault(ctx, req.OrgID, req.BuID)
}

func (s *service) encryptSensitiveFields(profile, existing *email.EmailProfile) error {
	if profile.EncryptedPassword != "" {
		if existing == nil || profile.EncryptedPassword != existing.EncryptedPassword {
			encrypted, err := s.encryptionService.Encrypt(profile.EncryptedPassword)
			if err != nil {
				return fmt.Errorf("failed to encrypt password: %w", err)
			}
			profile.EncryptedPassword = encrypted
		}
	}

	if profile.EncryptedAPIKey != "" {
		if existing == nil || profile.EncryptedAPIKey != existing.EncryptedAPIKey {
			encrypted, err := s.encryptionService.Encrypt(profile.EncryptedAPIKey)
			if err != nil {
				return fmt.Errorf("failed to encrypt API key: %w", err)
			}
			profile.EncryptedAPIKey = encrypted
		}
	}

	if profile.OAuth2ClientSecret != "" {
		if existing == nil || profile.OAuth2ClientSecret != existing.OAuth2ClientSecret {
			encrypted, err := s.encryptionService.Encrypt(profile.OAuth2ClientSecret)
			if err != nil {
				return fmt.Errorf("failed to encrypt OAuth2 client secret: %w", err)
			}
			profile.OAuth2ClientSecret = encrypted
		}
	}

	return nil
}
