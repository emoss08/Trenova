package classification

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/ailog"
	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/temporaljobs/ailogjobs"
	"github.com/emoss08/trenova/internal/infrastructure/redis"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/openai/openai-go/v2"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type ServiceParams struct {
	fx.In

	Logger                     *zap.Logger
	OpenAIClient               openai.Client
	LocationCategoryRepository repositories.LocationCategoryRepository
	TemporalClient             client.Client
	Cache                      *redis.Connection
}

type Service struct {
	l                          *zap.Logger
	openAIClient               openai.Client
	locationCategoryRepository repositories.LocationCategoryRepository
	temporalClient             client.Client
	cache                      *redis.Connection
}

//nolint:gocritic // This is a constructor
func NewService(p ServiceParams) *Service {
	return &Service{
		l:                          p.Logger.Named("service.ai"),
		openAIClient:               p.OpenAIClient,
		locationCategoryRepository: p.LocationCategoryRepository,
		temporalClient:             p.TemporalClient,
		cache:                      p.Cache,
	}
}

func (s *Service) ClassifyLocation(
	ctx context.Context,
	req *LocationClassificationRequest,
) (*LocationClassificationResponse, error) {
	log := s.l.With(
		zap.String("operation", "ClassifyLocation"),
		zap.Any("req", req),
	)

	if cachedResponse := s.getFromCache(ctx, req, log); cachedResponse != nil {
		return cachedResponse, nil
	}

	categories, err := s.fetchLocationCategories(ctx, req.TenantOpts, log)
	if err != nil {
		return nil, err
	}

	prompt := s.buildStructuredPromptWithCategories(req, categories)
	result, err := s.callOpenAI(ctx, prompt, log)
	if err != nil {
		return nil, err
	}

	response, err := s.parseOpenAIResponse(result, log)
	if err != nil {
		return nil, err
	}

	s.cacheResponse(ctx, req, response, log)

	s.logAIOperation(req, result, prompt, log)

	return response, nil
}

func (s *Service) getFromCache(
	ctx context.Context,
	req *LocationClassificationRequest,
	log *zap.Logger,
) *LocationClassificationResponse {
	var response *LocationClassificationResponse
	cacheKey := s.buildCacheKey(req)

	err := s.cache.GetJSON(ctx, cacheKey, &response)
	if err == nil && response != nil && response.CategoryID != "" {
		log.Debug("retrieved classification from cache", zap.String("key", cacheKey))
		return response
	}

	return nil
}

func (s *Service) fetchLocationCategories(
	ctx context.Context,
	tenantOpts pagination.TenantOptions,
	log *zap.Logger,
) ([]*location.LocationCategory, error) {
	categories, err := s.locationCategoryRepository.List(
		ctx,
		&repositories.ListLocationCategoryRequest{
			Filter: &pagination.QueryOptions{
				TenantOpts: tenantOpts,
			},
		},
	)
	if err != nil {
		log.Error("failed to list location categories", zap.Error(err))
		return nil, err
	}

	return categories.Items, nil
}

func (s *Service) callOpenAI(
	ctx context.Context,
	prompt string,
	log *zap.Logger,
) (*openai.ChatCompletion, error) {
	schemaParam := openai.ResponseFormatJSONSchemaJSONSchemaParam{
		Name:        "location_classification",
		Description: openai.String("Location classification with category and facility type"),
		Schema:      LocationResponseSchema,
		Strict:      openai.Bool(true),
	}

	result, err := s.openAIClient.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:           locationClassificationModel,
		ReasoningEffort: openai.ReasoningEffortMinimal,
		Messages: []openai.ChatCompletionMessageParamUnion{
			openai.SystemMessage(locationClassificationSystemPrompt),
			openai.UserMessage(prompt),
		},
		ResponseFormat: openai.ChatCompletionNewParamsResponseFormatUnion{
			OfJSONSchema: &openai.ResponseFormatJSONSchemaParam{JSONSchema: schemaParam},
		},
	})
	if err != nil {
		log.Error("failed to complete chat completion", zap.Error(err))
		return nil, err
	}

	return result, nil
}

func (s *Service) parseOpenAIResponse(
	result *openai.ChatCompletion,
	log *zap.Logger,
) (*LocationClassificationResponse, error) {
	var structuredResp StructuredLocationResponse
	err := sonic.Unmarshal([]byte(result.Choices[0].Message.Content), &structuredResp)
	if err != nil {
		log.Error("failed to unmarshal structured response", zap.Error(err))
		return nil, err
	}

	return s.convertToResponse(&structuredResp), nil
}

func (s *Service) cacheResponse(
	ctx context.Context,
	req *LocationClassificationRequest,
	response *LocationClassificationResponse,
	log *zap.Logger,
) {
	cacheKey := s.buildCacheKey(req)
	if err := s.cache.SetJSON(ctx, cacheKey, response, 24*time.Hour); err != nil {
		log.Error("failed to cache classification response", zap.Error(err))
	}
}

func (s *Service) logAIOperation(
	req *LocationClassificationRequest,
	result *openai.ChatCompletion,
	prompt string,
	log *zap.Logger,
) {
	payload := &ailogjobs.InsertAILogPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: req.TenantOpts.OrgID,
			BusinessUnitID: req.TenantOpts.BuID,
			UserID:         req.TenantOpts.UserID,
			Timestamp:      time.Now().Unix(),
			Metadata:       make(map[string]any),
		},
		Log: &ailog.AILog{
			OrganizationID:   req.TenantOpts.OrgID,
			BusinessUnitID:   req.TenantOpts.BuID,
			UserID:           req.TenantOpts.UserID,
			Prompt:           prompt,
			Response:         result.Choices[0].Message.Content,
			Operation:        ailog.OperationClassifyLocation,
			Model:            ailog.Model(locationClassificationModel),
			Object:           string(result.Object),
			ServiceTier:      locationClassificationServiceTier,
			PromptTokens:     result.Usage.PromptTokens,
			CompletionTokens: result.Usage.CompletionTokens,
			TotalTokens:      result.Usage.TotalTokens,
			ReasoningTokens:  result.Usage.CompletionTokensDetails.ReasoningTokens,
		},
	}

	workflowID := fmt.Sprintf("ailog-classification-%s-%d",
		req.TenantOpts.UserID.String(),
		time.Now().UnixNano())

	if _, err := s.temporalClient.ExecuteWorkflow(context.Background(), client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: ailogjobs.AILogTaskQueue,
	}, ailogjobs.InsertAILogWorkflow, payload); err != nil {
		log.Error("failed to start ai log workflow",
			zap.Error(err),
			zap.String("workflowID", workflowID))
	}
}

func (s *Service) buildCacheKey(req *LocationClassificationRequest) string {
	parts := []string{"location_classification", req.Name}
	if req.Description != nil {
		parts = append(parts, *req.Description)
	}
	if req.Address != nil {
		parts = append(parts, *req.Address)
	}
	if req.City != nil {
		parts = append(parts, *req.City)
	}
	if req.State != nil {
		parts = append(parts, *req.State)
	}
	if req.PostalCode != nil {
		parts = append(parts, *req.PostalCode)
	}
	if req.Code != nil {
		parts = append(parts, *req.Code)
	}
	return strings.Join(parts, ":")
}

func (s *Service) buildStructuredPromptWithCategories(
	req *LocationClassificationRequest,
	categories []*location.LocationCategory,
) string {
	var categoriesSection strings.Builder

	for _, cat := range categories {
		categoriesSection.WriteString(fmt.Sprintf("• %s (ID: %s) - %s\n",
			cat.Name,
			cat.ID.String(),
			cat.Description))
	}

	var locationInfo strings.Builder
	locationInfo.WriteString(fmt.Sprintf("Name: %s\n", req.Name))

	if req.Code != nil && *req.Code != "" {
		locationInfo.WriteString(fmt.Sprintf("Location Code: %s\n", *req.Code))
	}
	if req.Description != nil && *req.Description != "" {
		locationInfo.WriteString(fmt.Sprintf("Description: %s\n", *req.Description))
	}
	if req.Address != nil && *req.Address != "" {
		locationInfo.WriteString(fmt.Sprintf("Street Address: %s\n", *req.Address))
	}
	if req.City != nil && *req.City != "" {
		locationInfo.WriteString(fmt.Sprintf("City: %s\n", *req.City))
	}
	if req.State != nil && *req.State != "" {
		locationInfo.WriteString(fmt.Sprintf("State: %s\n", *req.State))
	}
	if req.PostalCode != nil && *req.PostalCode != "" {
		locationInfo.WriteString(fmt.Sprintf("Postal Code: %s\n", *req.PostalCode))
	}
	if req.Latitude != nil && req.Longitude != nil {
		locationInfo.WriteString(
			fmt.Sprintf("Coordinates: %.6f, %.6f\n", *req.Latitude, *req.Longitude),
		)
	}
	if req.PlaceID != nil && *req.PlaceID != "" {
		locationInfo.WriteString(fmt.Sprintf("Google Place ID: %s\n", *req.PlaceID))
	}

	prompt := fmt.Sprintf(
		`Classify this location based on its primary logistics function:

%s
Select from these categories (MUST use exact name and ID):
%s
Instructions:
- Match location to category based on its actual operational purpose
- Corporate campuses often serve as distribution centers
- Location codes (DC, WH, TERM) indicate facility type
- Consider industry context (pharma/medical → may need cold storage)
- Set confidence 0.7+ for clear matches, 0.4-0.7 for probable, <0.4 for uncertain`,
		locationInfo.String(),
		categoriesSection.String(),
	)

	return prompt
}

func (s *Service) convertToResponse(
	structured *StructuredLocationResponse,
) *LocationClassificationResponse {
	response := &LocationClassificationResponse{
		Category:              structured.Category,
		CategoryID:            structured.CategoryID,
		Confidence:            structured.Confidence,
		Reasoning:             structured.Reasoning,
		AlternativeCategories: structured.AlternativeCategories,
	}

	if structured.FacilityType != "" {
		response.FacilityType = &structured.FacilityType
	}

	return response
}
