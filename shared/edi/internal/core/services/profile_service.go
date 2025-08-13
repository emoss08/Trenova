package services

import (
	"context"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/shared/edi/internal/core/domain"
	"github.com/emoss08/trenova/shared/edi/internal/core/ports"
	"github.com/emoss08/trenova/shared/edi/internal/profiles"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ProfileService struct {
	logger     *zap.Logger
	repository ports.EDIPartnerProfileRepository
}

type ProfileServiceParams struct {
	fx.In
	Logger     *zap.Logger
	Repository ports.EDIPartnerProfileRepository
}

func NewProfileService(params ProfileServiceParams) *ProfileService {
	return &ProfileService{
		logger:     params.Logger,
		repository: params.Repository,
	}
}

// SaveProfile saves an EDIPartnerProfile to the database
func (s *ProfileService) SaveProfile(ctx context.Context, profile *domain.EDIPartnerProfile) error {
	// Validate inputs
	if profile.PartnerID == "" {
		return fmt.Errorf("partner_id is required")
	}
	if profile.PartnerName == "" {
		profile.PartnerName = profile.PartnerID
	}

	// Check if profile exists
	existing, err := s.repository.GetByPartnerID(ctx, profile.PartnerID)
	if err == nil && existing != nil {
		// Update existing profile - preserve the ID
		profile.ID = existing.ID
		profile.CreatedAt = existing.CreatedAt
		
		if err := s.repository.Update(ctx, profile); err != nil {
			return fmt.Errorf("failed to update profile: %w", err)
		}
		
		s.logger.Info("profile updated",
			zap.String("partner_id", profile.PartnerID),
			zap.String("partner_name", profile.PartnerName),
		)
	} else {
		// Create new profile
		if err := s.repository.Create(ctx, profile); err != nil {
			return fmt.Errorf("failed to create profile: %w", err)
		}
		
		s.logger.Info("profile created",
			zap.String("partner_id", profile.PartnerID),
			zap.String("partner_name", profile.PartnerName),
		)
	}

	return nil
}

// GetProfile retrieves a profile from the database
func (s *ProfileService) GetProfile(ctx context.Context, partnerID string) (*domain.EDIPartnerProfile, error) {
	dbProfile, err := s.repository.GetByPartnerID(ctx, partnerID)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	if !dbProfile.Active {
		s.logger.Warn("profile is inactive",
			zap.String("partner_id", partnerID),
		)
	}

	return dbProfile, nil
}

// ListProfiles lists all profiles from the database
func (s *ProfileService) ListProfiles(ctx context.Context, activeOnly bool) ([]*domain.EDIPartnerProfile, error) {
	dbProfiles, err := s.repository.List(ctx, activeOnly)
	if err != nil {
		return nil, fmt.Errorf("failed to list profiles: %w", err)
	}

	return dbProfiles, nil
}

// DeleteProfile deletes a profile from the database
func (s *ProfileService) DeleteProfile(ctx context.Context, partnerID string) error {
	if err := s.repository.Delete(ctx, partnerID); err != nil {
		return fmt.Errorf("failed to delete profile: %w", err)
	}

	s.logger.Info("profile deleted",
		zap.String("partner_id", partnerID),
	)

	return nil
}

// ImportProfile imports a profile from JSON data that includes metadata
func (s *ProfileService) ImportProfile(ctx context.Context, jsonData []byte) (string, error) {
	// Parse the full JSON to extract both metadata and profile
	var fullData map[string]interface{}
	if err := sonic.Unmarshal(jsonData, &fullData); err != nil {
		return "", fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	// Extract metadata fields
	partnerID, _ := fullData["partner_id"].(string)
	partnerName, _ := fullData["partner_name"].(string)
	active, _ := fullData["active"].(bool)
	description, _ := fullData["description"].(string)
	
	// Remove metadata fields to leave only profile configuration
	delete(fullData, "partner_id")
	delete(fullData, "partner_name")
	delete(fullData, "active")
	delete(fullData, "description")
	
	// Marshal the cleaned data back to JSON
	profileJSON, err := sonic.Marshal(fullData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal profile data: %w", err)
	}
	
	// Validate the profile structure by unmarshaling
	var profile profiles.PartnerProfile
	if err := sonic.Unmarshal(profileJSON, &profile); err != nil {
		return "", fmt.Errorf("failed to unmarshal profile: %w", err)
	}

	// Process the profile (validate delimiters, set defaults, etc.)
	if err := s.processProfile(&profile); err != nil {
		return "", fmt.Errorf("invalid profile: %w", err)
	}

	// Set defaults for metadata
	if partnerID == "" {
		return "", fmt.Errorf("partner_id is required in import data")
	}
	if partnerName == "" {
		partnerName = partnerID
	}

	// Create the database model
	dbProfile := &domain.EDIPartnerProfile{
		PartnerID:     partnerID,
		PartnerName:   partnerName,
		Active:        active,
		Description:   description,
		Configuration: profile,
	}

	// Save to database
	if err := s.SaveProfile(ctx, dbProfile); err != nil {
		return "", err
	}

	return partnerID, nil
}

// processProfile validates and sets defaults for a profile
func (s *ProfileService) processProfile(profile *profiles.PartnerProfile) error {
	// Validate delimiters
	if profile.Format.Delimiters.Element == "" {
		return fmt.Errorf("element delimiter is required")
	}
	if profile.Format.Delimiters.Segment == "" {
		return fmt.Errorf("segment delimiter is required")
	}

	// Set defaults
	if profile.Format.Encoding == "" {
		profile.Format.Encoding = "UTF-8"
	}

	if profile.ValidationConfig.Strictness == "" {
		profile.ValidationConfig.Strictness = "strict"
	}

	return nil
}

// GetProfileForProcessing retrieves a profile optimized for EDI processing
// This method caches frequently used profiles in memory
func (s *ProfileService) GetProfileForProcessing(ctx context.Context, partnerID string) (*domain.EDIPartnerProfile, error) {
	// For now, just get from database
	// TODO: Add caching layer for frequently accessed profiles
	return s.GetProfile(ctx, partnerID)
}