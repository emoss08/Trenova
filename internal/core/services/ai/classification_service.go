package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/ai"
	locationDomain "github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/external/ai/claude"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

type classificationServiceImpl struct {
	logger               *logger.Logger
	claudeClient         *claude.Client
	locationCategoryRepo repositories.LocationCategoryRepository
	cache                infra.CacheClient
}

type ServiceParams struct {
	fx.In

	Logger               *logger.Logger
	ClaudeClient         *claude.Client
	LocationCategoryRepo repositories.LocationCategoryRepository
	Cache                infra.CacheClient
}

// NewClassificationService creates a new AI classification service
func NewClassificationService(params ServiceParams) services.AIClassificationService {
	return &classificationServiceImpl{
		logger:               params.Logger,
		claudeClient:         params.ClaudeClient,
		cache:                params.Cache,
		locationCategoryRepo: params.LocationCategoryRepo,
	}
}

func (s *classificationServiceImpl) ClassifyLocation(
	ctx context.Context,
	req *ai.ClassificationRequest,
) (*ai.ClassificationResponse, error) {
	log := s.logger.With().
		Str("operation", "ClassifyLocation").
		Str("location", req.Name).
		Logger()

	cacheKey := s.buildCacheKey(req)
	cachedResult, err := s.cache.Get(ctx, cacheKey)
	if err == nil && cachedResult != "" {
		var response ai.ClassificationResponse
		if err := sonic.Unmarshal([]byte(cachedResult), &response); err == nil {
			s.logger.Debug().Str("location", req.Name).Msg("Returning cached classification")
			return &response, nil
		}
	}

	categories, err := s.locationCategoryRepo.List(ctx, &ports.LimitOffsetQueryOptions{
		TenantOpts: &ports.TenantOptions{
			BuID:   req.TenantOpts.BuID,
			OrgID:  req.TenantOpts.OrgID,
			UserID: req.TenantOpts.UserID,
		},
	})
	if err != nil {
		return nil, oops.In("classification_service").
			Time(time.Now()).
			With("req", req).
			Wrapf(err, "failed to fetch location categories")
	}

	prompt, err := s.buildClassificationPromptWithCategories(req, categories.Items)
	if err != nil {
		return nil, oops.In("classification_service").
			Time(time.Now()).
			With("req", req).
			Wrapf(err, "failed to build classification prompt")
	}

	result, err := s.claudeClient.Complete(ctx, prompt, "claude-3-haiku-20240307")
	if err != nil {
		log.Error().Err(err).Msg("Failed to classify location")
		return nil, oops.In("classification_service").
			Time(time.Now()).
			With("req", req).
			Wrapf(err, "failed to classify location")
	}

	response, err := s.parseClassificationResponse(result)
	if err != nil {
		log.Error().Err(err).Str("raw_response", result).Msg("Failed to parse Claude response")
		return &ai.ClassificationResponse{
			Category:              locationDomain.CategoryCustomerLocation,
			Confidence:            0.3,
			Reasoning:             "Unable to classify with high confidence due to parsing error",
			AlternativeCategories: []ai.AlternativeCategory{},
		}, nil
	}

	if responseJSON, err := sonic.Marshal(response); err == nil {
		_ = s.cache.Set(ctx, cacheKey, string(responseJSON), time.Hour)
	}

	return response, nil
}

func (s *classificationServiceImpl) ClassifyLocationBatch(
	ctx context.Context,
	req *ai.BatchClassificationRequest,
) (*ai.BatchClassificationResponse, error) {
	log := s.logger.With().
		Str("operation", "ClassifyLocationBatch").
		Int("num_locations", len(req.Locations)).
		Logger()

	results := make([]ai.ClassificationResponse, 0, len(req.Locations))

	for _, location := range req.Locations {
		classReq := &ai.ClassificationRequest{
			Name:        location.Name,
			Description: location.Description,
			Address:     location.Address,
		}

		response, err := s.ClassifyLocation(ctx, classReq)
		if err != nil {
			log.Error().
				Err(err).
				Str("location", location.Name).
				Msg("Failed to classify location")
			response = &ai.ClassificationResponse{
				Category:              locationDomain.CategoryCustomerLocation,
				Confidence:            0.3,
				Reasoning:             fmt.Sprintf("Classification failed: %v", err),
				AlternativeCategories: []ai.AlternativeCategory{},
			}
		}

		results = append(results, *response)
	}

	return &ai.BatchClassificationResponse{Results: results}, nil
}

func (s *classificationServiceImpl) buildCacheKey(req *ai.ClassificationRequest) string {
	parts := []string{"location_classification", req.Name}
	if req.Description != nil {
		parts = append(parts, *req.Description)
	}
	if req.Address != nil {
		parts = append(parts, *req.Address)
	}
	return strings.Join(parts, ":")
}

func (s *classificationServiceImpl) buildClassificationPromptWithCategories(
	req *ai.ClassificationRequest,
	categories []*locationDomain.LocationCategory,
) (string, error) {
	var categoriesSection strings.Builder
	categoryMap := make(map[string]string)

	categoriesSection.WriteString("Available categories (choose from these only):\n")
	for _, cat := range categories {
		categoryMap[cat.Name] = cat.ID.String()
		categoriesSection.WriteString(fmt.Sprintf("- %s (ID: %s): %s\n",
			cat.Name,
			cat.ID.String(),
			cat.Description))
	}

	prompt := `Classify this transportation location using ONLY the categories below:

%s

Location: %s
%s
%s

Return JSON with these exact fields:
{
    "category": "exact category name from list",
    "categoryId": "exact ID from list",
    "facilityType": "ONE of: CrossDock, StorageWarehouse, ColdStorage, HazmatFacility, IntermodalFacility" or null,
    "confidence": 0.0-1.0,
    "reasoning": "brief explanation",
    "alternativeCategories": []
}`

	descriptionPart := ""
	if req.Description != nil {
		descriptionPart = fmt.Sprintf("Description: %s", *req.Description)
	}

	addressPart := ""
	if req.Address != nil {
		addressPart = fmt.Sprintf("Address: %s", *req.Address)
	}

	return fmt.Sprintf(prompt,
		categoriesSection.String(),
		req.Name,
		descriptionPart,
		addressPart), nil
}

func (s *classificationServiceImpl) parseClassificationResponse(
	rawResponse string,
) (*ai.ClassificationResponse, error) {
	jsonStr := jsonutils.ExtractJSON(rawResponse)
	if jsonStr == "" {
		return nil, oops.In("classification_service").
			With("raw_response", rawResponse).
			New("no JSON found in response")
	}

	var parsed struct {
		Category              string  `json:"category"`
		CategoryID            string  `json:"categoryId"`
		FacilityType          any     `json:"facilityType"`
		Confidence            float64 `json:"confidence"`
		Reasoning             string  `json:"reasoning"`
		AlternativeCategories []struct {
			Category   string  `json:"category"`
			CategoryID string  `json:"categoryId"`
			Confidence float64 `json:"confidence"`
		} `json:"alternativeCategories"`
	}

	if err := sonic.Unmarshal([]byte(jsonStr), &parsed); err != nil {
		return nil, oops.In("classification_service").
			With("raw_response", rawResponse).
			Wrapf(err, "failed to parse JSON response")
	}

	response := &ai.ClassificationResponse{
		Category:              locationDomain.Category(parsed.Category),
		CategoryID:            parsed.CategoryID,
		Confidence:            parsed.Confidence,
		Reasoning:             parsed.Reasoning,
		AlternativeCategories: make([]ai.AlternativeCategory, len(parsed.AlternativeCategories)),
	}

	if parsed.FacilityType != nil {
		switch ft := parsed.FacilityType.(type) {
		case string:
			if ft != "" && ft != "null" {
				facilityType := locationDomain.FacilityType(ft)
				response.FacilityType = &facilityType
			}
		case []any:
			if len(ft) > 0 {
				if facilityStr, ok := ft[0].(string); ok {
					facilityType := locationDomain.FacilityType(facilityStr)
					response.FacilityType = &facilityType
				}
			}
		}
	}

	for i, alt := range parsed.AlternativeCategories {
		response.AlternativeCategories[i] = ai.AlternativeCategory{
			Category:   locationDomain.Category(alt.Category),
			CategoryID: alt.CategoryID,
			Confidence: alt.Confidence,
		}
	}

	return response, nil
}
