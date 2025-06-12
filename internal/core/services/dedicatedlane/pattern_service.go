package dedicatedlane

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/rs/zerolog"
	"github.com/shopspring/decimal"
	"go.uber.org/fx"
)

type PatternServiceParams struct {
	fx.In

	Logger            *logger.Logger
	ShipmentRepo      repositories.ShipmentRepository
	DedicatedLaneRepo repositories.DedicatedLaneRepository
	PatternConfigRepo repositories.PatternConfigRepository
	PermService       services.PermissionService
	AuditService      services.AuditService
	SuggestionRepo    repositories.DedicatedLaneSuggestionRepository
	LocationRepo      repositories.LocationRepository
}

type PatternService struct {
	l            *zerolog.Logger
	shipmentRepo repositories.ShipmentRepository
	dlRepo       repositories.DedicatedLaneRepository
	pcRepo       repositories.PatternConfigRepository
	suggRepo     repositories.DedicatedLaneSuggestionRepository
	permService  services.PermissionService
	auditService services.AuditService
	locationRepo repositories.LocationRepository
}

// NewPatternService creates a new instance of the PatternService, which is responsible for
// analyzing shipment data to detect recurring patterns and suggest potential dedicated lanes.
//
// Parameters:
//   - p: PatternServiceParams containing all the dependencies for the service.
//
// Returns:
//   - *PatternService: A new PatternService instance.
//
//nolint:gocritic // This is a constructor
func NewPatternService(p PatternServiceParams) *PatternService {
	log := p.Logger.With().
		Str("service", "dedicated_lane_pattern").
		Logger()

	return &PatternService{
		l:            &log,
		shipmentRepo: p.ShipmentRepo,
		dlRepo:       p.DedicatedLaneRepo,
		pcRepo:       p.PatternConfigRepo,
		suggRepo:     p.SuggestionRepo,
		locationRepo: p.LocationRepo,
		permService:  p.PermService,
		auditService: p.AuditService,
	}
}

func (ps *PatternService) GetPatternConfig(
	ctx context.Context,
	req repositories.GetPatternConfigRequest,
) (*dedicatedlane.PatternConfig, error) {
	if err := ps.checkPermission(
		ctx,
		permission.ActionRead,
		req.UserID,
		req.BuID,
		req.OrgID,
	); err != nil {
		return nil, err
	}

	return ps.pcRepo.GetByOrgID(ctx, req)
}

// AnalyzePatterns performs pattern analysis on shipments to identify potential dedicated lanes.
// It processes each organization with a pattern configuration, detects patterns,
// filters them based on frequency and confidence, and creates suggestions for qualified patterns.
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: The pattern analysis request containing the date range and other filters.
//
// Returns:
//   - *dedicatedlane.PatternAnalysisResult: The result of the analysis, including statistics and detected patterns.
//   - error: An error if the analysis fails.
func (ps *PatternService) AnalyzePatterns(
	ctx context.Context,
	req *dedicatedlane.PatternAnalysisRequest,
) (*dedicatedlane.PatternAnalysisResult, error) {
	startTime := time.Now()

	log := ps.l.With().
		Str("operation", "AnalyzePatterns").
		Logger()

	log.Info().Msg("starting pattern analysis")

	// * Get all pattern configs for all organizations
	patternConfigs, err := ps.pcRepo.GetAll(ctx)
	if err != nil {
		log.Error().Err(err).Msg("failed to get pattern configs")
		return nil, fmt.Errorf("fetch pattern configs: %w", err)
	}

	log.Info().
		Int("organizationConfigs", len(patternConfigs)).
		Msg("processing organizations with configs")

	// * Process each organization individually
	allPatterns := make([]*dedicatedlane.PatternMatch, 0)
	allConfigsUsed := make([]*dedicatedlane.PatternDetectionConfig, 0, len(patternConfigs))
	var suggestionsCreated, suggestionsSkipped, totalPatternsDetected int64
	var organizationsSkipped int64

	for _, patternConfig := range patternConfigs {
		// * Check if pattern analysis is enabled for this organization
		if !patternConfig.Enabled {
			organizationsSkipped++
			log.Info().
				Str("organizationId", patternConfig.OrganizationID.String()).
				Str("organizationName", func() string {
					if patternConfig.Organization != nil {
						return patternConfig.Organization.Name
					}
					return "unknown"
				}()).
				Msg("skipping organization - pattern analysis disabled")
			continue
		}

		patterns, created, skipped, processErr := ps.processOrganization(ctx, patternConfig, req)
		if processErr != nil {
			log.Error().
				Err(processErr).
				Str("organizationId", patternConfig.OrganizationID.String()).
				Msg("failed to process organization")
			continue
		}

		config := patternConfig.ToPatternDetectionConfig()
		allConfigsUsed = append(allConfigsUsed, config)
		allPatterns = append(allPatterns, patterns...)
		suggestionsCreated += created
		suggestionsSkipped += skipped
		totalPatternsDetected += int64(len(patterns))
	}

	result := &dedicatedlane.PatternAnalysisResult{
		TotalPatternsDetected:  totalPatternsDetected,
		PatternsAboveThreshold: int64(len(allPatterns)),
		SuggestionsCreated:     suggestionsCreated,
		SuggestionsSkipped:     suggestionsSkipped,
		AnalysisStartDate:      req.StartDate,
		AnalysisEndDate:        req.EndDate,
		ConfigsUsed:            allConfigsUsed,
		Patterns:               allPatterns,
		ProcessingTimeMs:       time.Since(startTime).Milliseconds(),
	}

	log.Info().
		Int64("suggestionsCreated", suggestionsCreated).
		Int64("processingTimeMs", result.ProcessingTimeMs).
		Int("organizationsProcessed", len(patternConfigs)-int(organizationsSkipped)).
		Int64("organizationsSkipped", organizationsSkipped).
		Int("totalOrganizationsWithConfigs", len(patternConfigs)).
		Msg("pattern analysis completed")

	return result, nil
}

// processOrganization processes a single organization's pattern analysis.
// It converts the pattern config to a detection config, fetches shipments, groups them into patterns,
// filters them based on frequency and confidence, and creates suggestions for qualified patterns.
//
// Parameters:
//   - ctx: The context for the operation.
//   - patternConfig: The pattern configuration for the organization.
//   - req: The pattern analysis request containing the date range and other filters.
//
// Returns:
//   - []*dedicatedlane.PatternMatch: A slice of detected patterns.
//   - int64: The number of suggestions created.
//   - int64: The number of suggestions skipped.
//   - error: An error if the operation fails.
func (ps *PatternService) processOrganization(
	ctx context.Context,
	patternConfig *dedicatedlane.PatternConfig,
	req *dedicatedlane.PatternAnalysisRequest,
) (qualifiedPatterns []*dedicatedlane.PatternMatch, suggestionsCreated, suggestionsSkipped int64, err error) {
	orgLog := ps.l.With().
		Str("organizationId", patternConfig.OrganizationID.String()).
		Str("organizationName", func() string {
			if patternConfig.Organization != nil {
				return patternConfig.Organization.Name
			}
			return "unknown"
		}()).
		Logger()

	orgLog.Info().Msg("processing organization pattern analysis")

	// * Convert pattern config to detection config
	config := patternConfig.ToPatternDetectionConfig()

	// * Get shipments for this organization
	shipments, err := ps.getShipmentsForOrganization(ctx, req, patternConfig.OrganizationID)
	if err != nil {
		orgLog.Error().Err(err).Msg("failed to get shipments for organization")
		return nil, 0, 0, err
	}

	orgLog.Info().Int("shipmentCount", len(shipments)).Msg("analyzing organization shipments")

	// * Group shipments by pattern for this organization
	patterns := ps.groupShipmentsByPattern(shipments, config)

	orgLog.Info().Int("patternCount", len(patterns)).Msg("patterns detected for organization")

	// * Log detailed pattern analysis results
	for i, pattern := range patterns {
		orgLog.Info().
			Int("patternIndex", i).
			Int64("frequencyCount", pattern.FrequencyCount).
			Str("confidenceScore", pattern.ConfidenceScore.String()).
			Str("customerId", pattern.CustomerID.String()).
			Str("originLocationId", pattern.OriginLocationID.String()).
			Str("destinationLocationId", pattern.DestinationLocationID.String()).
			Msg("detected pattern details")
	}

	// * Filter patterns by frequency and confidence
	qualifiedPatterns = ps.filterPatterns(patterns, config)

	orgLog.Info().
		Int("qualifiedPatterns", len(qualifiedPatterns)).
		Int64("minFrequency", config.MinFrequency).
		Str("minConfidenceScore", config.MinConfidenceScore.String()).
		Msg("patterns above threshold for organization")

	// * Log details about patterns that didn't qualify
	for _, pattern := range patterns {
		qualified := pattern.FrequencyCount >= config.MinFrequency &&
			pattern.ConfidenceScore.GreaterThanOrEqual(config.MinConfidenceScore)
		if !qualified {
			orgLog.Info().
				Int64("frequencyCount", pattern.FrequencyCount).
				Str("confidenceScore", pattern.ConfidenceScore.String()).
				Bool("meetsFrequency", pattern.FrequencyCount >= config.MinFrequency).
				Bool("meetsConfidence", pattern.ConfidenceScore.GreaterThanOrEqual(config.MinConfidenceScore)).
				Str("customerId", pattern.CustomerID.String()).
				Str("originLocationId", pattern.OriginLocationID.String()).
				Str("destinationLocationId", pattern.DestinationLocationID.String()).
				Msg("pattern did not qualify - below threshold")
		}
	}

	// * Check for existing dedicated lanes and suggestions
	if req.ExcludeExisting {
		qualifiedPatterns = ps.excludeExistingLanes(ctx, qualifiedPatterns)
	}

	// * Create suggestions for this organization
	for _, pattern := range qualifiedPatterns {
		suggestion := ps.createSuggestionFromPattern(ctx, pattern, req, config)

		_, err = ps.suggRepo.Create(ctx, suggestion)
		if err != nil {
			orgLog.Error().Err(err).Msg("failed to save suggestion")
			suggestionsSkipped++
			continue
		}

		suggestionsCreated++
	}

	orgLog.Info().
		Int("orgSuggestionsCreated", len(qualifiedPatterns)).
		Msg("completed organization pattern analysis")

	return qualifiedPatterns, suggestionsCreated, suggestionsSkipped, nil
}

// getShipmentsForOrganization fetches and filters shipments for a specific organization
// based on the analysis request.
//
// Parameters:
//   - ctx: The context for the operation.
//   - req: The pattern analysis request containing the date range and customer filters.
//   - orgID: The ID of the organization to fetch shipments for.
//
// Returns:
//   - []*shipment.Shipment: A slice of shipments that match the criteria.
//   - error: An error if fetching shipments fails.
func (ps *PatternService) getShipmentsForOrganization(
	ctx context.Context,
	req *dedicatedlane.PatternAnalysisRequest,
	orgID pulid.ID,
) ([]*shipment.Shipment, error) {
	log := ps.l.With().
		Str("operation", "getShipmentsForOrganization").
		Str("organizationId", orgID.String()).
		Logger()

	log.Info().Msg("fetching shipments for organization pattern analysis")

	result, err := ps.shipmentRepo.GetByOrgID(ctx, orgID)
	if err != nil {
		log.Error().Err(err).Msg("failed to fetch shipments for organization")
		return nil, fmt.Errorf("fetch organization shipments: %w", err)
	}

	log.Info().Int("totalShipments", result.Total).Msg("fetched shipments for organization")

	// * Filter by date range and customer if specified
	var filteredShipments []*shipment.Shipment
	for _, shp := range result.Items {
		// * Check date range
		if shp.CreatedAt < req.StartDate || shp.CreatedAt > req.EndDate {
			continue
		}

		// * Filter by customer if specified
		if req.CustomerID != nil && !pulid.Equals(shp.CustomerID, *req.CustomerID) {
			continue
		}

		filteredShipments = append(filteredShipments, shp)
	}

	log.Info().
		Int("filteredShipments", len(filteredShipments)).
		Int64("dateRangeStart", req.StartDate).
		Int64("dateRangeEnd", req.EndDate).
		Msg("filtered shipments for organization pattern analysis")

	return filteredShipments, nil
}

// validateShipmentForPattern ensures a shipment contains the minimum required information
// to be considered in pattern analysis. This includes customer, service type, shipment type,
// and valid origin/destination locations.
//
// It returns the origin and destination location IDs and a boolean indicating if the shipment is valid.
func (ps *PatternService) validateShipmentForPattern(
	shp *shipment.Shipment,
) (originLocationID, destLocationID pulid.ID, isValid bool) {
	// ! Skip shipments without required data
	if shp.CustomerID.IsNil() {
		return pulid.Nil, pulid.Nil, false
	}

	if len(shp.Moves) == 0 {
		return pulid.Nil, pulid.Nil, false
	}

	// ! Skip shipments without required service type and shipment type
	if shp.ServiceTypeID.IsNil() {
		return pulid.Nil, pulid.Nil, false
	}
	if shp.ShipmentTypeID.IsNil() {
		return pulid.Nil, pulid.Nil, false
	}

	// * Get origin and destination from first and last moves
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

// initializePattern creates a new PatternMatch object from the first shipment that establishes
// a new pattern. It populates the pattern with key information from the shipment.
//
// Parameters:
//   - shp: The shipment to initialize the pattern from.
//   - originLocationID: The ID of the origin location.
//   - destLocationID: The ID of the destination location.
//
// Returns:
//   - A new *dedicatedlane.PatternMatch instance.
func (ps *PatternService) initializePattern(
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

// updatePatternMetrics updates an existing pattern with data from a new matching shipment.
// It increments the frequency count, updates the date range, and aggregates the total freight value.
//
// Parameters:
//   - pattern: The pattern to update.
//   - shp: The new shipment to incorporate into the pattern.
func (ps *PatternService) updatePatternMetrics(
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

// groupShipmentsByPattern orchestrates the process of grouping shipments into patterns.
// It iterates over shipments, validates them, and then uses helper functions to create
// or update pattern matches. After grouping, it calculates final metrics like
// average freight charge and confidence score for each pattern.
//
// Parameters:
//   - shipments: A slice of shipments to be grouped.
//   - config: The pattern detection configuration.
//
// Returns:
//   - []*dedicatedlane.PatternMatch: A slice of detected patterns with their calculated metrics.
func (ps *PatternService) groupShipmentsByPattern(
	shipments []*shipment.Shipment,
	config *dedicatedlane.PatternDetectionConfig,
) []*dedicatedlane.PatternMatch {
	patternMap := make(map[string]*dedicatedlane.PatternMatch)

	log := ps.l.With().
		Str("operation", "groupShipmentsByPattern").
		Logger()

	log.Info().
		Int("totalShipments", len(shipments)).
		Msg("starting shipment pattern grouping")

	var validShipments, invalidShipments int

	// * Loop through shipments and group them by pattern
	for i, shp := range shipments {
		originLocationID, destLocationID, isValid := ps.validateShipmentForPattern(shp)
		if !isValid {
			invalidShipments++
			log.Debug().
				Int("shipmentIndex", i).
				Str("shipmentId", shp.ID.String()).
				Bool("hasCustomer", !shp.CustomerID.IsNil()).
				Bool("hasMoves", len(shp.Moves) > 0).
				Bool("hasServiceType", !shp.ServiceTypeID.IsNil()).
				Bool("hasShipmentType", !shp.ShipmentTypeID.IsNil()).
				Bool("hasValidOrigin", !originLocationID.IsNil()).
				Bool("hasValidDestination", !destLocationID.IsNil()).
				Msg("shipment failed validation - skipping")
			continue
		}

		validShipments++
		key := ps.createPatternKey(shp, originLocationID, destLocationID, config)

		pattern, exists := patternMap[key]
		if !exists {
			pattern = ps.initializePattern(shp, originLocationID, destLocationID)
			patternMap[key] = pattern
			log.Debug().
				Str("patternKey", key).
				Str("shipmentId", shp.ID.String()).
				Str("customerId", shp.CustomerID.String()).
				Str("originLocationId", originLocationID.String()).
				Str("destinationLocationId", destLocationID.String()).
				Msg("created new pattern")
		} else {
			log.Debug().
				Str("patternKey", key).
				Str("shipmentId", shp.ID.String()).
				Int64("currentFrequency", pattern.FrequencyCount).
				Msg("added shipment to existing pattern")
		}

		ps.updatePatternMetrics(pattern, shp)
	}

	log.Info().
		Int("validShipments", validShipments).
		Int("invalidShipments", invalidShipments).
		Int("uniquePatterns", len(patternMap)).
		Msg("shipment validation and grouping summary")

	// * Convert map to slice and calculate final metrics
	patterns := make([]*dedicatedlane.PatternMatch, 0, len(patternMap))
	for _, pattern := range patternMap {
		// * Calculate average freight charge
		if pattern.TotalFreightValue.Valid && pattern.FrequencyCount > 0 {
			avgCharge := pattern.TotalFreightValue.Decimal.Div(
				decimal.NewFromInt(pattern.FrequencyCount),
			)
			pattern.AverageFreightCharge = &decimal.NullDecimal{
				Decimal: avgCharge,
				Valid:   true,
			}
		}

		// * Calculate confidence score
		pattern.ConfidenceScore = ps.calculateConfidenceScore(pattern, config)

		log.Debug().
			Str("customerId", pattern.CustomerID.String()).
			Str("originLocationId", pattern.OriginLocationID.String()).
			Str("destinationLocationId", pattern.DestinationLocationID.String()).
			Int64("frequencyCount", pattern.FrequencyCount).
			Str("confidenceScore", pattern.ConfidenceScore.String()).
			Str("avgFreightCharge", func() string {
				if pattern.AverageFreightCharge.Valid {
					return pattern.AverageFreightCharge.Decimal.String()
				}
				return "N/A"
			}()).
			Msg("final pattern metrics calculated")

		patterns = append(patterns, pattern)
	}

	log.Info().
		Int("finalPatternCount", len(patterns)).
		Msg("completed pattern grouping and metrics calculation")

	return patterns
}

// createPatternKey generates a unique key for a shipment pattern based on its attributes.
// The key includes organization, customer, origin, and destination, and can optionally include
// service and equipment types if an exact match is required.
//
// Parameters:
//   - shp: The shipment for which to create the key.
//   - originID: The origin location ID.
//   - destID: The destination location ID.
//   - config: The pattern detection configuration, which determines if an exact match is needed.
//
// Returns:
//   - string: The generated pattern key.
func (ps *PatternService) createPatternKey(
	shp *shipment.Shipment,
	originID, destID pulid.ID,
	config *dedicatedlane.PatternDetectionConfig,
) string {
	// * Include organization ID in pattern key so patterns are grouped by organization
	key := fmt.Sprintf("%s|%s|%s|%s",
		shp.OrganizationID.String(),
		shp.CustomerID.String(),
		originID.String(),
		destID.String())

	// * Include equipment/service types if exact match required
	if config.RequireExactMatch {
		if shp.ServiceTypeID != pulid.Nil {
			key += "|" + shp.ServiceTypeID.String()
		}
		if shp.ShipmentTypeID != pulid.Nil {
			key += "|" + shp.ShipmentTypeID.String()
		}
		if shp.TrailerTypeID != nil && !shp.TrailerTypeID.IsNil() {
			key += "|" + shp.TrailerTypeID.String()
		}
		if shp.TractorTypeID != nil && !shp.TractorTypeID.IsNil() {
			key += "|" + shp.TractorTypeID.String()
		}
	}

	return key
}

// calculateConfidenceScore computes a confidence score for a given pattern.
// The score is based on factors like frequency, recency, consistency, and total
// freight value. This score helps in identifying high-quality, reliable patterns.
//
// Parameters:
//   - pattern: The pattern for which to calculate the score.
//   - config: The pattern detection configuration, which may influence scoring logic.
//
// Returns:
//   - decimal.Decimal: The calculated confidence score, ranging from 0.0 to 1.0.
func (ps *PatternService) calculateConfidenceScore(
	pattern *dedicatedlane.PatternMatch,
	config *dedicatedlane.PatternDetectionConfig,
) decimal.Decimal {
	score := decimal.NewFromFloat(0.0)

	// * Base score from frequency, normalized to a maximum of 0.4.
	// * This gives more weight to patterns that occur more often.
	frequencyScore := decimal.NewFromInt(pattern.FrequencyCount).Div(decimal.NewFromInt(10))
	if frequencyScore.GreaterThan(decimal.NewFromFloat(0.4)) {
		frequencyScore = decimal.NewFromFloat(0.4)
	}
	score = score.Add(frequencyScore)

	// * Recency bonus, up to 0.3.
	// * This rewards patterns that have been active recently.
	if config.WeightRecentShipments {
		daysSinceLastShipment := (timeutils.NowUnix() - pattern.LastShipmentDate) / 86400
		switch {
		case daysSinceLastShipment <= 7:
			score = score.Add(decimal.NewFromFloat(0.3))
		case daysSinceLastShipment <= 30:
			score = score.Add(decimal.NewFromFloat(0.2))
		case daysSinceLastShipment <= 60:
			score = score.Add(decimal.NewFromFloat(0.1))
		}
	}

	// * Consistency bonus, up to 0.2.
	// * This rewards patterns with regular, predictable shipment intervals.
	timeSpan := pattern.LastShipmentDate - pattern.FirstShipmentDate
	if timeSpan > 0 {
		avgDaysBetween := timeSpan / (86400 * (pattern.FrequencyCount - 1))
		if avgDaysBetween <= 30 { // * Regular monthly pattern
			score = score.Add(decimal.NewFromFloat(0.2))
		} else if avgDaysBetween <= 60 {
			score = score.Add(decimal.NewFromFloat(0.1))
		}
	}

	// * Value bonus, up to 0.1.
	// * This gives a small boost to high-value patterns.
	if pattern.TotalFreightValue.Valid &&
		pattern.TotalFreightValue.Decimal.GreaterThan(decimal.NewFromFloat(10000)) {
		score = score.Add(decimal.NewFromFloat(0.1))
	}

	// * Cap at 1.0 to ensure the score is a normalized value between 0 and 1.
	if score.GreaterThan(decimal.NewFromFloat(1.0)) {
		score = decimal.NewFromFloat(1.0)
	}

	return score
}

// filterPatterns filters a list of patterns to include only those that meet the
// minimum frequency and confidence score thresholds. The filtered patterns are then
// sorted by confidence score in descending order.
//
// Parameters:
//   - patterns: A slice of patterns to filter.
//   - config: The pattern detection configuration containing the thresholds.
//
// Returns:
//   - []*dedicatedlane.PatternMatch: A slice of qualified and sorted patterns.
func (ps *PatternService) filterPatterns(
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

	// * Sort by confidence score descending
	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ConfidenceScore.GreaterThan(filtered[j].ConfidenceScore)
	})

	return filtered
}

// excludeExistingLanes filters out patterns for which a dedicated lane or a pending
// suggestion already exists. This prevents the creation of duplicate suggestions
// for the same lane.
//
// Parameters:
//   - ctx: The context for the operation.
//   - patterns: The slice of patterns to check.
//
// Returns:
//   - []*dedicatedlane.PatternMatch: A slice of patterns that do not have existing lanes or suggestions.
func (ps *PatternService) excludeExistingLanes(
	ctx context.Context,
	patterns []*dedicatedlane.PatternMatch,
) []*dedicatedlane.PatternMatch {
	log := ps.l.With().
		Str("operation", "excludeExistingLanes").
		Logger()

	log.Info().
		Int("inputPatterns", len(patterns)).
		Msg("checking patterns against existing lanes and suggestions")

	var filteredPatterns []*dedicatedlane.PatternMatch

	for _, pattern := range patterns {
		// ! Skip patterns without required service type and shipment type IDs
		if pattern.ServiceTypeID == nil || pattern.ServiceTypeID.IsNil() ||
			pattern.ShipmentTypeID == nil || pattern.ShipmentTypeID.IsNil() {
			log.Warn().
				Str("customerId", pattern.CustomerID.String()).
				Str("originLocationId", pattern.OriginLocationID.String()).
				Str("destinationLocationId", pattern.DestinationLocationID.String()).
				Msg("skipping pattern - missing required service type or shipment type")
			continue
		}

		// * Check if a dedicated lane already exists for this pattern
		existingLane, err := ps.dlRepo.FindByShipment(
			ctx,
			&repositories.FindDedicatedLaneByShipmentRequest{
				OrganizationID:        pattern.OrganizationID,
				BusinessUnitID:        pattern.BusinessUnitID,
				CustomerID:            pattern.CustomerID,
				ServiceTypeID:         *pattern.ServiceTypeID,
				ShipmentTypeID:        *pattern.ShipmentTypeID,
				OriginLocationID:      pattern.OriginLocationID,
				DestinationLocationID: pattern.DestinationLocationID,
				TrailerTypeID:         pattern.TrailerTypeID,
				TractorTypeID:         pattern.TractorTypeID,
			},
		)

		if err == nil && existingLane != nil {
			log.Info().
				Str("existingLaneId", existingLane.ID.String()).
				Str("customerId", pattern.CustomerID.String()).
				Str("originLocationId", pattern.OriginLocationID.String()).
				Str("destinationLocationId", pattern.DestinationLocationID.String()).
				Msg("skipping pattern - dedicated lane already exists")
			continue
		}

		// * Check if a pending suggestion already exists for this pattern
		existingSuggestion, err := ps.suggRepo.CheckForDuplicatePattern(
			ctx,
			pattern.CustomerID,
			pattern.OriginLocationID,
			pattern.DestinationLocationID,
			pattern.OrganizationID,
			pattern.BusinessUnitID,
		)

		if err != nil {
			log.Error().Err(err).
				Str("customerId", pattern.CustomerID.String()).
				Msg("failed to check for duplicate suggestion - including pattern")
			// ! Include pattern on error to avoid missing valid suggestions
			filteredPatterns = append(filteredPatterns, pattern)
			continue
		}

		if existingSuggestion != nil {
			log.Info().
				Str("existingSuggestionId", existingSuggestion.ID.String()).
				Str("customerId", pattern.CustomerID.String()).
				Str("originLocationId", pattern.OriginLocationID.String()).
				Str("destinationLocationId", pattern.DestinationLocationID.String()).
				Msg("skipping pattern - pending suggestion already exists")
			continue
		}

		// * Pattern is unique, include it
		filteredPatterns = append(filteredPatterns, pattern)
	}

	log.Info().
		Int("inputPatterns", len(patterns)).
		Int("filteredPatterns", len(filteredPatterns)).
		Int("excludedPatterns", len(patterns)-len(filteredPatterns)).
		Msg("completed pattern exclusion check")

	return filteredPatterns
}

// createSuggestionFromPattern creates a DedicatedLaneSuggestion from a qualified pattern.
// The suggestion includes all relevant details from the pattern, along with metadata
// from the analysis process.
//
// Parameters:
//   - pattern: The pattern to convert into a suggestion.
//   - req: The original analysis request, used for start and end dates.
//   - config: The pattern detection configuration, used for suggestion TTL and other metadata.
//
// Returns:
//   - *dedicatedlane.DedicatedLaneSuggestion: The newly created suggestion.
func (ps *PatternService) createSuggestionFromPattern(
	ctx context.Context,
	pattern *dedicatedlane.PatternMatch,
	req *dedicatedlane.PatternAnalysisRequest,
	config *dedicatedlane.PatternDetectionConfig,
) *dedicatedlane.DedicatedLaneSuggestion {
	now := timeutils.NowUnix()
	expiresAt := now + (config.SuggestionTTLDays * 86400)

	suggestedName := ps.generateSuggestedName(ctx, pattern)

	// * Use pattern's organization and business unit IDs for organization-specific analysis
	orgID := pattern.OrganizationID
	buID := pattern.BusinessUnitID

	suggestion := &dedicatedlane.DedicatedLaneSuggestion{
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
		AnalysisStartDate:     req.StartDate,
		AnalysisEndDate:       req.EndDate,
		SuggestedName:         suggestedName,
		PatternDetails: map[string]any{
			"shipmentIds":    pattern.ShipmentIDs,
			"analysisConfig": config,
			"detectionTime":  now,
			"analysisType":   "organization-specific",
		},
		ExpiresAt: expiresAt,
	}

	return suggestion
}

func (ps *PatternService) UpdatePatternConfig(
	ctx context.Context,
	pc *dedicatedlane.PatternConfig,
	userID pulid.ID,
) (*dedicatedlane.PatternConfig, error) {
	log := ps.l.With().
		Str("operation", "UpdatePatternConfig").
		Str("orgID", pc.OrganizationID.String()).
		Str("buID", pc.BusinessUnitID.String()).
		Logger()

	if err := ps.checkPermission(ctx, permission.ActionUpdate, userID, pc.BusinessUnitID, pc.OrganizationID); err != nil {
		return nil, err
	}

	original, err := ps.pcRepo.GetByOrgID(ctx, repositories.GetPatternConfigRequest{
		OrgID: pc.OrganizationID,
		BuID:  pc.BusinessUnitID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get original pattern config")
		return nil, err
	}

	updatedEntity, err := ps.pcRepo.Update(ctx, pc)
	if err != nil {
		log.Error().Err(err).Msg("failed to update pattern config")
		return nil, err
	}

	// * Log the action
	err = ps.auditService.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourcePatternConfig,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Pattern Config updated"),
		audit.WithDiff(original, updatedEntity),
		audit.WithCritical(),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log action")
		return nil, err
	}

	return updatedEntity, nil
}

// generateSuggestedName creates a human-readable name for a dedicated lane suggestion
// based on the origin and destination location codes.
//
// If location details cannot be fetched, it returns an empty string, which will be
// handled by the calling function.
//
// Parameters:
//   - ctx: The context for the operation.
//   - pattern: The pattern for which to generate a name.
//
// Returns:
//   - string: The suggested name for the lane (e.g., "Lane-ORIG-DEST").
func (ps *PatternService) generateSuggestedName(
	ctx context.Context,
	pattern *dedicatedlane.PatternMatch,
) string {
	log := ps.l.With().
		Str("operation", "generateSuggestedName").
		Logger()

	// * Fetch each location to get its code for the name.
	originLocation, err := ps.locationRepo.GetByID(ctx, repositories.GetLocationByIDOptions{
		ID:    pattern.OriginLocationID,
		OrgID: pattern.OrganizationID,
		BuID:  pattern.BusinessUnitID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get origin location")
		return ""
	}

	destLocation, err := ps.locationRepo.GetByID(ctx, repositories.GetLocationByIDOptions{
		ID:    pattern.DestinationLocationID,
		OrgID: pattern.OrganizationID,
		BuID:  pattern.BusinessUnitID,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to get destination location")
		return ""
	}

	// * Generate the name using the location codes.
	return fmt.Sprintf("Lane-%s-to-%s", originLocation.Code, destLocation.Code)
}

func (ps *PatternService) checkPermission(
	ctx context.Context,
	action permission.Action,
	userID, buID, orgID pulid.ID,
) error {
	log := ps.l.With().
		Str("operation", "checkPermission").
		Str("action", string(action)).
		Str("userID", userID.String()).
		Str("buID", buID.String()).
		Str("orgID", orgID.String()).
		Logger()

	result, err := ps.permService.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:   userID,
			Resource: permission.ResourcePatternConfig,
			Action:   action,
		},
	})

	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return err
	}

	if !result.Allowed {
		return errors.NewAuthorizationError(
			fmt.Sprintf("You do not have permission to %s pattern config",
				string(action),
			),
		)
	}

	return nil
}
