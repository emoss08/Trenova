package patternconfig

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/repositories/dberror"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/utils/jsonutils"
	"github.com/samber/lo"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	Repo              repositories.PatternConfigRepository
	ShipmentRepo      repositories.ShipmentRepository
	SuggestionRepo    repositories.DedicatedLaneSuggestionRepository
	DedicatedLaneRepo repositories.DedicatedLaneRepository
	LocationRepo      repositories.LocationRepository
	UserRepo          repositories.UserRepository
	AuditService      services.AuditService
}

type Service struct {
	l                 *zap.Logger
	repo              repositories.PatternConfigRepository
	shipmentRepo      repositories.ShipmentRepository
	suggestionRepo    repositories.DedicatedLaneSuggestionRepository
	dedicatedLaneRepo repositories.DedicatedLaneRepository
	locationRepo      repositories.LocationRepository
	userRepo          repositories.UserRepository
	as                services.AuditService
}

//nolint:gocritic // This is a constructor
func NewService(p Params) *Service {
	return &Service{
		l:                 p.Logger.Named("service.patternconfig"),
		repo:              p.Repo,
		shipmentRepo:      p.ShipmentRepo,
		suggestionRepo:    p.SuggestionRepo,
		dedicatedLaneRepo: p.DedicatedLaneRepo,
		userRepo:          p.UserRepo,
		locationRepo:      p.LocationRepo,
		as:                p.AuditService,
	}
}

func (s *Service) GetByOrgID(
	ctx context.Context,
	req repositories.GetPatternConfigRequest,
) (*dedicatedlane.PatternConfig, error) {
	return s.repo.GetByOrgID(ctx, req)
}

func (s *Service) AnalyzePatterns(
	ctx context.Context,
	req *dedicatedlane.PatternAnalysisRequest,
) (*dedicatedlane.PatternAnalysisResult, error) {
	startTime := time.Now()
	log := s.l.With(
		zap.String("operation", "AnalyzePatterns"),
	)

	patternConfigs, err := s.repo.GetAll(ctx)
	if err != nil {
		log.Error("failed to get pattern configs", zap.Error(err))
		return nil, fmt.Errorf("fetch pattern configs: %w", err)
	}

	allPatterns := make([]*dedicatedlane.PatternMatch, 0)
	allConfigsUsed := make([]*dedicatedlane.PatternDetectionConfig, 0, len(patternConfigs))
	var totalPatternsDetected int64
	var organizationsSkipped int64

	for _, patternConfig := range patternConfigs {
		if !patternConfig.Enabled {
			organizationsSkipped++
			continue
		}

		processedResult, processErr := s.processOrganization(ctx, patternConfig, req)
		if processErr != nil {
			continue
		}

		config := patternConfig.ToPatternDetectionConfig()
		allConfigsUsed = append(allConfigsUsed, config)
		allPatterns = append(allPatterns, processedResult.QualifiedPatterns...)
		totalPatternsDetected += int64(len(processedResult.QualifiedPatterns))
	}

	result := &dedicatedlane.PatternAnalysisResult{
		TotalPatternsDetected:  totalPatternsDetected,
		PatternsAboveThreshold: int64(len(allPatterns)),
		ConfigsUsed:            allConfigsUsed,
		Patterns:               allPatterns,
		ProcessingTimeMs:       time.Since(startTime).Milliseconds(),
	}

	return result, nil
}

type ProcessedOrganizationResult struct {
	QualifiedPatterns []*dedicatedlane.PatternMatch
	SuggestionsResult *ProcessQualifiedPatternsResult
}

func (s *Service) processOrganization(
	ctx context.Context,
	patternConfig *dedicatedlane.PatternConfig,
	req *dedicatedlane.PatternAnalysisRequest,
) (*ProcessedOrganizationResult, error) {
	config := patternConfig.ToPatternDetectionConfig()

	shipments, err := s.shipmentRepo.GetByOrgID(ctx, patternConfig.OrganizationID)
	if err != nil {
		return nil, err
	}

	if shipments.Total == 0 {
		return nil, errors.New("no shipments found for organization")
	}

	patterns := s.groupShipmentsByPattern(shipments.Items, config)

	qualifiedPatterns := s.filterPatterns(patterns, config)

	if req.ExcludeExisting {
		qualifiedPatterns = s.excludeExistingLanes(ctx, qualifiedPatterns)
	}

	suggestionsResult, err := s.processQualifiedPatterns(ctx, qualifiedPatterns, req)
	if err != nil {
		return nil, err
	}

	return &ProcessedOrganizationResult{
		QualifiedPatterns: qualifiedPatterns,
		SuggestionsResult: suggestionsResult,
	}, nil
}

type ProcessQualifiedPatternsResult struct {
	SuggestionsCreated int64
	SuggestionsSkipped int64
}

func (s *Service) processQualifiedPatterns(
	ctx context.Context,
	patterns []*dedicatedlane.PatternMatch,
	req *dedicatedlane.PatternAnalysisRequest,
) (*ProcessQualifiedPatternsResult, error) {
	log := s.l.With(
		zap.String("operation", "processQualifiedPatterns"),
	)

	result := new(ProcessQualifiedPatternsResult)

	sysUser, err := s.userRepo.GetSystemUser(ctx)
	if err != nil {
		return nil, err
	}

	for _, pattern := range patterns {
		suggestion := s.createSuggestionFromPattern(ctx, pattern, req)

		duplicateReq := &repositories.FindDedicatedLaneByShipmentRequest{
			OrganizationID:        pattern.OrganizationID,
			BusinessUnitID:        pattern.BusinessUnitID,
			CustomerID:            pattern.CustomerID,
			ServiceTypeID:         pattern.ServiceTypeID,
			ShipmentTypeID:        pattern.ShipmentTypeID,
			OriginLocationID:      pattern.OriginLocationID,
			DestinationLocationID: pattern.DestinationLocationID,
			TrailerTypeID:         pattern.TrailerTypeID,
			TractorTypeID:         pattern.TractorTypeID,
		}

		existingSuggestion, dupErr := s.suggestionRepo.CheckForDuplicatePattern(ctx, duplicateReq)
		if dupErr == nil && existingSuggestion != nil {
			result.SuggestionsSkipped++
			continue
		}

		cs, csErr := s.suggestionRepo.Create(ctx, suggestion)
		if csErr != nil {
			log.Error("failed to create dedicated lane suggestion", zap.Error(csErr))
			result.SuggestionsSkipped++
			continue
		}

		result.SuggestionsCreated++

		err = s.as.LogAction(
			&services.LogActionParams{
				Resource:       permission.ResourceDedicatedLaneSuggestion,
				ResourceID:     cs.GetID(),
				Operation:      permission.OpCreate,
				UserID:         sysUser.ID,
				CurrentState:   jsonutils.MustToJSON(cs),
				OrganizationID: cs.OrganizationID,
				BusinessUnitID: cs.BusinessUnitID,
			},
			audit.WithComment("Dedicated lane suggestion created"),
			audit.WithTags(
				"dedicated-lane-suggestion-creation",
				fmt.Sprintf("customer-%s", pattern.CustomerID.String()),
			),
			audit.WithCritical(),
			audit.WithCategory("operations"),
		)
		if err != nil {
			log.Error("failed to log dedicated lane suggestion creation", zap.Error(err))
		}
	}

	return result, nil
}

func (s *Service) groupShipmentsByPattern(
	shipments []*shipment.Shipment,
	config *dedicatedlane.PatternDetectionConfig,
) []*dedicatedlane.PatternMatch {
	patternMap := make(map[string]*dedicatedlane.PatternMatch)

	var validShipments, invalidShipments int

	for _, shp := range shipments {
		originLocationID, destLocationID, isValid := s.validateShipmentForPattern(shp)
		if !isValid {
			invalidShipments++
			continue
		}

		validShipments++
		key := s.createPatternKey(shp, originLocationID, destLocationID, config)

		pattern, exists := patternMap[key]
		if !exists {
			pattern = s.initializePattern(shp, originLocationID, destLocationID)
			patternMap[key] = pattern
		}

		s.updatePatternMetrics(pattern, shp)
	}

	patterns := make([]*dedicatedlane.PatternMatch, 0, len(patternMap))
	for _, pattern := range patternMap {
		if pattern.TotalFreightValue.Valid && pattern.FrequencyCount > 0 {
			avgCharge := pattern.TotalFreightValue.Decimal.Div(
				decimal.NewFromInt(pattern.FrequencyCount),
			)
			pattern.AverageFreightCharge = &decimal.NullDecimal{
				Decimal: avgCharge,
				Valid:   true,
			}
		}

		pattern.ConfidenceScore = s.calculateConfidenceScore(pattern, config)
		patterns = append(patterns, pattern)
	}

	return patterns
}

func (s *Service) validateShipmentForPattern(
	shp *shipment.Shipment,
) (originLocationID, destLocationID pulid.ID, isValid bool) {
	if shp.CustomerID.IsNil() {
		return pulid.Nil, pulid.Nil, false
	}

	if len(shp.Moves) == 0 {
		return pulid.Nil, pulid.Nil, false
	}

	if shp.ServiceTypeID.IsNil() {
		return pulid.Nil, pulid.Nil, false
	}
	if shp.ShipmentTypeID.IsNil() {
		return pulid.Nil, pulid.Nil, false
	}

	if len(shp.Moves) > 0 && len(shp.Moves[0].Stops) > 0 {
		originLocationID = shp.Moves[0].Stops[0].LocationID
	}
	if len(shp.Moves) > 0 && len(shp.Moves[len(shp.Moves)-1].Stops) > 0 {
		lastMove := shp.Moves[len(shp.Moves)-1]
		destLocationID = lastMove.Stops[len(lastMove.Stops)-1].LocationID
	}

	if originLocationID.IsNil() || destLocationID.IsNil() {
		return pulid.Nil, pulid.Nil, false
	}

	return originLocationID, destLocationID, true
}

func (s *Service) createPatternKey(
	shp *shipment.Shipment,
	originID, destID pulid.ID,
	config *dedicatedlane.PatternDetectionConfig,
) string {
	var builder strings.Builder

	builder.WriteString(shp.OrganizationID.String())
	builder.WriteString("|")
	builder.WriteString(shp.CustomerID.String())
	builder.WriteString("|")
	builder.WriteString(originID.String())
	builder.WriteString("|")
	builder.WriteString(destID.String())

	if config.RequireExactMatch {
		if shp.ServiceTypeID != pulid.Nil {
			builder.WriteString("|")
			builder.WriteString(shp.ServiceTypeID.String())
		}
		if shp.ShipmentTypeID != pulid.Nil {
			builder.WriteString("|")
			builder.WriteString(shp.ShipmentTypeID.String())
		}
		if shp.TrailerTypeID != nil && !shp.TrailerTypeID.IsNil() {
			builder.WriteString("|")
			builder.WriteString(shp.TrailerTypeID.String())
		}
		if shp.TractorTypeID != nil && !shp.TractorTypeID.IsNil() {
			builder.WriteString("|")
			builder.WriteString(shp.TractorTypeID.String())
		}
	}

	return builder.String()
}

func (s *Service) initializePattern(
	shp *shipment.Shipment,
	originLocationID, destLocationID pulid.ID,
) *dedicatedlane.PatternMatch {
	return &dedicatedlane.PatternMatch{
		OrganizationID:        shp.OrganizationID,
		BusinessUnitID:        shp.BusinessUnitID,
		CustomerID:            shp.CustomerID,
		OriginLocationID:      originLocationID,
		DestinationLocationID: destLocationID,
		ServiceTypeID:         &shp.ServiceTypeID,
		ShipmentTypeID:        &shp.ShipmentTypeID,
		TrailerTypeID:         shp.TrailerTypeID,
		TractorTypeID:         shp.TractorTypeID,
		FrequencyCount:        0,
		ShipmentIDs:           make([]pulid.ID, 0),
		PatternDetails:        make(map[string]any),
		AverageFreightCharge:  &decimal.NullDecimal{},
		TotalFreightValue:     &decimal.NullDecimal{},
	}
}

func (s *Service) updatePatternMetrics(
	pattern *dedicatedlane.PatternMatch,
	shp *shipment.Shipment,
) {
	pattern.FrequencyCount++
	pattern.ShipmentIDs = append(pattern.ShipmentIDs, shp.ID)

	if pattern.FirstShipmentDate == 0 || shp.CreatedAt < pattern.FirstShipmentDate {
		pattern.FirstShipmentDate = shp.CreatedAt
	}
	if shp.CreatedAt > pattern.LastShipmentDate {
		pattern.LastShipmentDate = shp.CreatedAt
	}

	if shp.FreightChargeAmount.Valid {
		if !pattern.TotalFreightValue.Valid {
			pattern.TotalFreightValue = &decimal.NullDecimal{
				Decimal: shp.FreightChargeAmount.Decimal,
				Valid:   true,
			}
		} else {
			pattern.TotalFreightValue.Decimal = pattern.TotalFreightValue.Decimal.Add(shp.FreightChargeAmount.Decimal)
		}
	}
}

func (s *Service) calculateConfidenceScore(
	pattern *dedicatedlane.PatternMatch,
	config *dedicatedlane.PatternDetectionConfig,
) decimal.Decimal {
	score := decimal.NewFromFloat(0.0)

	// Base score from frequency, normalized to a maximum of 0.4.
	// This gives more weight to patterns that occur more often.
	frequencyScore := decimal.NewFromInt(pattern.FrequencyCount).Div(decimal.NewFromInt(10))
	if frequencyScore.GreaterThan(decimal.NewFromFloat(0.4)) {
		frequencyScore = decimal.NewFromFloat(0.4)
	}
	score = score.Add(frequencyScore)

	now := utils.NowUnix()
	oneDaySeconds := utils.DaysToSeconds(1)

	if config.WeightRecentShipments {
		daysSinceLastShipment := (now - pattern.LastShipmentDate) / oneDaySeconds
		switch {
		case daysSinceLastShipment <= 7:
			score = score.Add(decimal.NewFromFloat(0.3))
		case daysSinceLastShipment <= 30:
			score = score.Add(decimal.NewFromFloat(0.2))
		case daysSinceLastShipment <= 60:
			score = score.Add(decimal.NewFromFloat(0.1))
		}
	}

	// Consistency bonus, up to 0.2.
	// This rewards patterns with regular, predictable shipment intervals.
	timeSpan := pattern.LastShipmentDate - pattern.FirstShipmentDate
	if timeSpan > 0 {
		avgDaysBetween := timeSpan / (oneDaySeconds * (pattern.FrequencyCount - 1))
		if avgDaysBetween <= 30 {
			score = score.Add(decimal.NewFromFloat(0.2))
		} else if avgDaysBetween <= 60 {
			score = score.Add(decimal.NewFromFloat(0.1))
		}
	}

	// Value bonus, up to 0.1.
	// This gives a small boost to high-value patterns.
	if pattern.TotalFreightValue.Valid &&
		pattern.TotalFreightValue.Decimal.GreaterThan(decimal.NewFromFloat(10000)) {
		score = score.Add(decimal.NewFromFloat(0.1))
	}

	// Cap at 1.0 to ensure the score is a normalized value between 0 and 1.
	if score.GreaterThan(decimal.NewFromFloat(1.0)) {
		score = decimal.NewFromFloat(1.0)
	}

	return score
}

func (s *Service) filterPatterns(
	patterns []*dedicatedlane.PatternMatch,
	config *dedicatedlane.PatternDetectionConfig,
) []*dedicatedlane.PatternMatch {
	var filtered []*dedicatedlane.PatternMatch

	for _, pattern := range patterns {
		if pattern.FrequencyCount >= config.MinFrequency &&
			pattern.ConfidenceScore.GreaterThanOrEqual(config.MinConfidenceScore) {
			filtered = append(filtered, pattern)
		}
	}

	// Sort by confidence score descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ConfidenceScore.GreaterThan(filtered[j].ConfidenceScore)
	})

	return filtered
}

func (s *Service) excludeExistingLanes(
	ctx context.Context,
	patterns []*dedicatedlane.PatternMatch,
) []*dedicatedlane.PatternMatch {
	log := s.l.With(
		zap.String("operation", "excludeExistingLanes"),
		zap.Any("patterns", patterns),
	)

	filteredPatterns := lo.Filter(patterns, func(pattern *dedicatedlane.PatternMatch, _ int) bool {
		req := &repositories.FindDedicatedLaneByShipmentRequest{
			OrganizationID:        pattern.OrganizationID,
			BusinessUnitID:        pattern.BusinessUnitID,
			CustomerID:            pattern.CustomerID,
			ServiceTypeID:         pattern.ServiceTypeID,
			ShipmentTypeID:        pattern.ShipmentTypeID,
			OriginLocationID:      pattern.OriginLocationID,
			DestinationLocationID: pattern.DestinationLocationID,
			TrailerTypeID:         pattern.TrailerTypeID,
			TractorTypeID:         pattern.TractorTypeID,
		}

		existingLane, err := s.dedicatedLaneRepo.FindByShipment(ctx, req)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				// ! If the error is not found, include the pattern
				return true
			}

			// ! For other errors, log them but err on the side of caution - exclude the pattern
			return false
		}

		if existingLane != nil {
			return false
		}

		existingSuggestion, err := s.suggestionRepo.CheckForDuplicatePattern(ctx, req)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				log.Error("failed to check for duplicate pattern", zap.Error(err))
				return true
			}

			return false
		}

		if existingSuggestion != nil {
			return false
		}

		return true
	})

	return filteredPatterns
}

func (s *Service) createSuggestionFromPattern(
	ctx context.Context,
	pattern *dedicatedlane.PatternMatch,
	req *dedicatedlane.PatternAnalysisRequest,
) *dedicatedlane.Suggestion {
	now := utils.NowUnix()
	expiresAt := now + (req.Config.SuggestionTTLDays * utils.DaysToSeconds(1))

	suggestedName := s.generateSuggestedName(ctx, pattern)

	orgID := pattern.OrganizationID
	buID := pattern.BusinessUnitID

	suggestion := &dedicatedlane.Suggestion{
		BusinessUnitID:        buID,
		OrganizationID:        orgID,
		Status:                dedicatedlane.SuggestionStatusPending,
		CustomerID:            pattern.CustomerID,
		OriginLocationID:      pattern.OriginLocationID,
		DestinationLocationID: pattern.DestinationLocationID,
		ServiceTypeID:         pattern.ServiceTypeID,
		ShipmentTypeID:        pattern.ShipmentTypeID,
		TrailerTypeID:         pattern.TrailerTypeID,
		TractorTypeID:         pattern.TractorTypeID,
		ConfidenceScore:       pattern.ConfidenceScore,
		FrequencyCount:        pattern.FrequencyCount,
		AverageFreightCharge:  pattern.AverageFreightCharge,
		TotalFreightValue:     pattern.TotalFreightValue,
		LastShipmentDate:      pattern.LastShipmentDate,
		FirstShipmentDate:     pattern.FirstShipmentDate,
		SuggestedName:         suggestedName,
		PatternDetails: map[string]any{
			"shipmentIds":    pattern.ShipmentIDs,
			"analysisConfig": req.Config,
			"detectionTime":  now,
			"analysisType":   "organization-specific",
		},
		ExpiresAt: expiresAt,
	}

	return suggestion
}

func (s *Service) generateSuggestedName(
	ctx context.Context,
	pattern *dedicatedlane.PatternMatch,
) string {
	log := s.l.With(
		zap.String("operation", "generateSuggestedName"),
		zap.String("originLocationId", pattern.OriginLocationID.String()),
		zap.String("destinationLocationId", pattern.DestinationLocationID.String()),
	)

	originLocation, err := s.locationRepo.GetByID(ctx, repositories.GetLocationByIDRequest{
		ID:    pattern.OriginLocationID,
		OrgID: pattern.OrganizationID,
		BuID:  pattern.BusinessUnitID,
	})
	if err != nil {
		log.Error("failed to get origin location", zap.Error(err))
		return ""
	}

	destLocation, err := s.locationRepo.GetByID(ctx, repositories.GetLocationByIDRequest{
		ID:    pattern.DestinationLocationID,
		OrgID: pattern.OrganizationID,
		BuID:  pattern.BusinessUnitID,
	})
	if err != nil {
		log.Error("failed to get destination location", zap.Error(err))
		return ""
	}

	return fmt.Sprintf("Lane-%s-to-%s", originLocation.Code, destLocation.Code)
}

func (s *Service) Update(
	ctx context.Context,
	entity *dedicatedlane.PatternConfig,
	userID pulid.ID,
) (*dedicatedlane.PatternConfig, error) {
	log := s.l.With(
		zap.String("operation", "Update"),
		zap.String("patternConfigID", entity.ID.String()),
		zap.String("userID", userID.String()),
	)

	original, err := s.repo.GetByOrgID(ctx, repositories.GetPatternConfigRequest{
		OrgID: entity.OrganizationID,
		BuID:  entity.BusinessUnitID,
	})
	if err != nil {
		log.Error("failed to get pattern config", zap.Error(err))
		return nil, err
	}

	updatedEntity, err := s.repo.Update(ctx, entity)
	if err != nil {
		log.Error("failed to update pattern config", zap.Error(err))
		return nil, err
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourcePatternConfig,
			ResourceID:     updatedEntity.ID.String(),
			Operation:      permission.OpUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Pattern config updated"),
		audit.WithDiff(original, updatedEntity),
		audit.WithCritical(),
	)
	if err != nil {
		log.Error("failed to log pattern config update", zap.Error(err))
		return nil, err
	}

	return updatedEntity, nil
}
