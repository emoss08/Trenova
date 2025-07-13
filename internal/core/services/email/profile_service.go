package email

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/encryption"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ProfileServiceParams struct {
	fx.In

	Logger            *logger.Logger
	Repository        repositories.EmailProfileRepository
	EncryptionService encryption.Service
}

type profileService struct {
	l                 *zerolog.Logger
	repository        repositories.EmailProfileRepository
	encryptionService encryption.Service
}

// NewProfileService creates a new email profile service
func NewProfileService(p ProfileServiceParams) services.EmailProfileService {
	log := p.Logger.With().
		Str("service", "email_profile").
		Logger()

	return &profileService{
		l:                 &log,
		repository:        p.Repository,
		encryptionService: p.EncryptionService,
	}
}

// Create creates a new email profile
func (s *profileService) Create(
	ctx context.Context,
	profile *email.Profile,
) (*email.Profile, error) {
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

	// Create the profile
	return s.repository.Create(ctx, profile)
}

// Update updates an existing email profile
func (s *profileService) Update(
	ctx context.Context,
	profile *email.Profile,
) (*email.Profile, error) {
	// Validate the profile
	multiErr := errors.NewMultiError()
	profile.Validate(ctx, multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	// Get existing profile to check for password changes
	existing, err := s.repository.Get(ctx, repositories.GetEmailProfileByIDRequest{
		OrgID:     profile.OrganizationID,
		BuID:      profile.BusinessUnitID,
		ProfileID: profile.ID,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get existing profile: %w", err)
	}

	// Only encrypt if the value has changed
	if profile.EncryptedPassword != "" && profile.EncryptedPassword != existing.EncryptedPassword {
		encrypted, err := s.encryptionService.Encrypt( //nolint:govet // We're intentionally ignoring the error here
			profile.EncryptedPassword,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt password: %w", err)
		}
		profile.EncryptedPassword = encrypted
	}

	if profile.EncryptedAPIKey != "" && profile.EncryptedAPIKey != existing.EncryptedAPIKey {
		encrypted, err := s.encryptionService.Encrypt( //nolint:govet // We're intentionally ignoring the error here
			profile.EncryptedAPIKey,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt API key: %w", err)
		}
		profile.EncryptedAPIKey = encrypted
	}

	if profile.OAuth2ClientSecret != "" &&
		profile.OAuth2ClientSecret != existing.OAuth2ClientSecret {
		encrypted, err := s.encryptionService.Encrypt( //nolint:govet // We're intentionally ignoring the error here
			profile.OAuth2ClientSecret,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt OAuth2 client secret: %w", err)
		}
		profile.OAuth2ClientSecret = encrypted
	}

	// Update the profile
	return s.repository.Update(ctx, profile)
}

// Get retrieves an email profile by ID
func (s *profileService) Get(
	ctx context.Context,
	req repositories.GetEmailProfileByIDRequest,
) (*email.Profile, error) {
	return s.repository.Get(ctx, req)
}

// List retrieves a list of email profiles
func (s *profileService) List(
	ctx context.Context,
	req *repositories.ListEmailProfileRequest,
) (*ports.ListResult[*email.Profile], error) {
	return s.repository.List(ctx, req)
}

// Delete deletes an email profile
func (s *profileService) Delete(
	ctx context.Context,
	req repositories.DeleteEmailProfileRequest,
) error {
	// Check if it's the default profile
	profile, err := s.repository.Get(ctx, repositories.GetEmailProfileByIDRequest{
		OrgID:     req.OrgID,
		BuID:      req.BuID,
		ProfileID: req.ProfileID,
	})
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	if profile.IsDefault {
		return eris.New("cannot delete the default email profile")
	}

	return s.repository.Delete(ctx, req)
}

// SetDefault sets a profile as the default for the organization
func (s *profileService) SetDefault(
	ctx context.Context,
	req repositories.GetEmailProfileByIDRequest,
) error {
	profile, err := s.repository.Get(ctx, req)
	if err != nil {
		return fmt.Errorf("failed to get profile: %w", err)
	}

	profile.IsDefault = true
	_, err = s.repository.Update(ctx, profile)
	return err
}

// GetDefault retrieves the default email profile for an organization
func (s *profileService) GetDefault(
	ctx context.Context,
	req repositories.GetEmailProfileByIDRequest,
) (*email.Profile, error) {
	return s.repository.GetDefault(ctx, req.OrgID, req.BuID)
}
