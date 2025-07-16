package email

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/infrastructure/encryption"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

type ProfileServiceParams struct {
	fx.In

	Logger            *logger.Logger
	PermService       services.PermissionService
	Repo              repositories.EmailProfileRepository
	AuditService      services.AuditService
	EncryptionService encryption.Service
}

type profileService struct {
	l                 *zerolog.Logger
	repo              repositories.EmailProfileRepository
	ps                services.PermissionService
	as                services.AuditService
	encryptionService encryption.Service
}

// NewProfileService creates a new email profile service
func NewProfileService(p ProfileServiceParams) services.EmailProfileService {
	log := p.Logger.With().
		Str("service", "email_profile").
		Logger()

	return &profileService{
		l:                 &log,
		repo:              p.Repo,
		encryptionService: p.EncryptionService,
		ps:                p.PermService,
		as:                p.AuditService,
	}
}

// List retrieves a list of email profiles
func (s *profileService) List(
	ctx context.Context,
	req *repositories.ListEmailProfileRequest,
) (*ports.ListResult[*email.Profile], error) {
	log := s.l.With().
		Str("operation", "List").
		Str("orgID", req.Filter.TenantOpts.OrgID.String()).
		Str("buID", req.Filter.TenantOpts.BuID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				Resource:       permission.ResourceEmailProfile,
				Action:         permission.ActionRead,
				BusinessUnitID: req.Filter.TenantOpts.BuID,
				OrganizationID: req.Filter.TenantOpts.OrgID,
				UserID:         req.Filter.TenantOpts.UserID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read email profiles",
		)
	}

	return s.repo.List(ctx, req)
}

// Get retrieves an email profile by ID
func (s *profileService) Get(
	ctx context.Context,
	req repositories.GetEmailProfileByIDRequest,
) (*email.Profile, error) {
	log := s.l.With().
		Str("operation", "Get").
		Str("profileID", req.ProfileID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				Resource:       permission.ResourceEmailProfile,
				Action:         permission.ActionRead,
				BusinessUnitID: req.BuID,
				OrganizationID: req.OrgID,
				UserID:         req.UserID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, err
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read this email profile",
		)
	}

	return s.repo.Get(ctx, req)
}

// Create creates a new email profile
func (s *profileService) Create(
	ctx context.Context,
	profile *email.Profile,
	userID pulid.ID,
) (*email.Profile, error) {
	log := s.l.With().
		Str("operation", "Create").
		Interface("profile", profile).
		Logger()

	// Validate the profile
	multiErr := errors.NewMultiError()
	profile.Validate(ctx, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	// Encrypt sensitive fields
	if profile.EncryptedPassword != "" {
		encrypted, err := s.encryptionService.Encrypt(profile.EncryptedPassword)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
		profile.EncryptedPassword = encrypted
	}

	if profile.EncryptedAPIKey != "" {
		encrypted, err := s.encryptionService.Encrypt(profile.EncryptedAPIKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt API key: %w", err)
		}
		profile.EncryptedAPIKey = encrypted
	}

	if profile.OAuth2ClientSecret != "" {
		encrypted, err := s.encryptionService.Encrypt(profile.OAuth2ClientSecret)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt OAuth2 client secret: %w", err)
		}
		profile.OAuth2ClientSecret = encrypted
	}

	createdEntity, err := s.repo.Create(ctx, profile)
	if err != nil {
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceEmailProfile,
			ResourceID:     createdEntity.ID.String(),
			Action:         permission.ActionCreate,
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
		log.Error().Err(err).Msg("failed to log email profile creation")
	}

	// Create the profile
	return createdEntity, nil
}

// Update updates an existing email profile
func (s *profileService) Update(
	ctx context.Context,
	profile *email.Profile,
	userID pulid.ID,
) (*email.Profile, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("profileID", profile.ID.String()).
		Logger()

	// Validate the profile
	multiErr := errors.NewMultiError()
	profile.Validate(ctx, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	// Get existing profile to check for password changes
	existing, err := s.repo.Get(ctx, repositories.GetEmailProfileByIDRequest{
		OrgID:     profile.OrganizationID,
		BuID:      profile.BusinessUnitID,
		ProfileID: profile.ID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get existing profile")
		return nil, oops.In("email_profile_service").
			Tags("operation", "update").
			Tags("profileID", profile.ID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get existing profile")
	}

	// Only encrypt if the value has changed
	if profile.EncryptedPassword != "" && profile.EncryptedPassword != existing.EncryptedPassword {
		encrypted, err := s.encryptionService.Encrypt( //nolint:govet // We're intentionally ignoring the error here
			profile.EncryptedPassword,
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to encrypt password")
			return nil, oops.In("email_profile_service").
				Tags("operation", "update").
				Tags("profileID", profile.ID.String()).
				Time(time.Now()).
				Wrapf(err, "failed to encrypt password")
		}
		profile.EncryptedPassword = encrypted
	}

	if profile.EncryptedAPIKey != "" && profile.EncryptedAPIKey != existing.EncryptedAPIKey {
		encrypted, err := s.encryptionService.Encrypt( //nolint:govet // We're intentionally ignoring the error here
			profile.EncryptedAPIKey,
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to encrypt API key")
			return nil, oops.In("email_profile_service").
				Tags("operation", "update").
				Tags("profileID", profile.ID.String()).
				Time(time.Now()).
				Wrapf(err, "failed to encrypt API key")
		}
		profile.EncryptedAPIKey = encrypted
	}

	if profile.OAuth2ClientSecret != "" &&
		profile.OAuth2ClientSecret != existing.OAuth2ClientSecret {
		encrypted, err := s.encryptionService.Encrypt( //nolint:govet // We're intentionally ignoring the error here
			profile.OAuth2ClientSecret,
		)
		if err != nil {
			log.Error().Err(err).Msg("failed to encrypt OAuth2 client secret")
			return nil, oops.In("email_profile_service").
				Tags("operation", "update").
				Tags("profileID", profile.ID.String()).
				Time(time.Now()).
				Wrapf(err, "failed to encrypt OAuth2 client secret")
		}
		profile.OAuth2ClientSecret = encrypted
	}

	updatedEntity, err := s.repo.Update(ctx, profile)
	if err != nil {
		log.Error().Err(err).Msg("failed to update email profile")
		return nil, oops.In("email_profile_service").
			Tags("operation", "update").
			Tags("profileID", profile.ID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to update email profile")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceEmailProfile,
			ResourceID:     updatedEntity.ID.String(),
			Action:         permission.ActionUpdate,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(existing),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Email profile updated"),
		audit.WithDiff(existing, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log email profile update")
	}

	return updatedEntity, nil
}

// SetDefault sets a profile as the default for the organization
func (s *profileService) SetDefault(
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

// GetDefault retrieves the default email profile for an organization
func (s *profileService) GetDefault(
	ctx context.Context,
	req repositories.GetEmailProfileByIDRequest,
) (*email.Profile, error) {
	return s.repo.GetDefault(ctx, req.OrgID, req.BuID)
}
