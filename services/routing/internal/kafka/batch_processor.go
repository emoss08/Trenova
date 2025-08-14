/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package kafka

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/emoss08/routing/internal/graph"
	"github.com/rs/zerolog"
	"github.com/sourcegraph/conc"
)

// RouteCalculator interface for route calculation
type RouteCalculator interface {
	CalculateRoute(
		ctx context.Context,
		originZip, destZip, vehicleType string,
	) (float64, float64, error)
}

// BatchRouteProcessor processes batch route calculation requests
type BatchRouteProcessor struct {
	calculator    RouteCalculator
	logger        zerolog.Logger
	httpClient    *http.Client
	maxConcurrent int
}

// NewBatchRouteProcessor creates a new batch route processor
func NewBatchRouteProcessor(
	calculator RouteCalculator,
	logger zerolog.Logger,
	maxConcurrent int,
) *BatchRouteProcessor {
	return &BatchRouteProcessor{
		calculator:    calculator,
		logger:        logger,
		httpClient:    &http.Client{Timeout: 30 * time.Second},
		maxConcurrent: maxConcurrent,
	}
}

// BatchResult represents the result of a batch calculation
type BatchResult struct {
	BatchID   string        `json:"batch_id"`
	Timestamp time.Time     `json:"timestamp"`
	Status    string        `json:"status"` // completed, partial, failed
	Results   []RouteResult `json:"results"`
	Errors    []RouteError  `json:"errors,omitempty"`
	Stats     BatchStats    `json:"stats"`
}

// RouteResult represents a single route calculation result
type RouteResult struct {
	ID            string    `json:"id"`
	OriginZip     string    `json:"origin_zip"`
	DestZip       string    `json:"dest_zip"`
	DistanceMiles float64   `json:"distance_miles"`
	TimeMinutes   float64   `json:"time_minutes"`
	Status        string    `json:"status"` // success, error
	Error         string    `json:"error,omitempty"`
	CalculatedAt  time.Time `json:"calculated_at"`
}

// RouteError represents an error for a specific route
type RouteError struct {
	ID      string `json:"id"`
	Message string `json:"message"`
}

// BatchStats contains statistics for the batch processing
type BatchStats struct {
	TotalRoutes      int           `json:"total_routes"`
	SuccessfulRoutes int           `json:"successful_routes"`
	FailedRoutes     int           `json:"failed_routes"`
	TotalTime        time.Duration `json:"total_time"`
	AverageTime      time.Duration `json:"average_time"`
}

// ProcessBatch processes a batch of route calculations
func (p *BatchRouteProcessor) ProcessBatch(
	ctx context.Context,
	request BatchCalculationRequest,
) error {
	startTime := time.Now()

	p.logger.Info().
		Str("batch_id", request.BatchID).
		Int("route_count", len(request.Routes)).
		Msg("Starting batch processing")

	// Create result structure
	result := BatchResult{
		BatchID:   request.BatchID,
		Timestamp: time.Now(),
		Results:   make([]RouteResult, 0, len(request.Routes)),
		Errors:    make([]RouteError, 0),
	}

	// Process routes concurrently with limit
	var (
		mu        sync.Mutex
		wg        = conc.NewWaitGroup()
		semaphore = make(chan struct{}, p.maxConcurrent)
	)

	for _, route := range request.Routes {
		route := route // Capture loop variable

		wg.Go(func() {
			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Calculate route
			routeResult := p.calculateSingleRoute(ctx, route)

			// Add to results
			mu.Lock()
			result.Results = append(result.Results, routeResult)
			if routeResult.Status == "error" {
				result.Errors = append(result.Errors, RouteError{
					ID:      route.ID,
					Message: routeResult.Error,
				})
			}
			mu.Unlock()
		})
	}

	// Wait for all routes to complete
	wg.Wait()

	// Calculate statistics
	result.Stats = p.calculateStats(result, startTime)

	// Determine overall status
	if result.Stats.FailedRoutes == 0 {
		result.Status = "completed"
	} else if result.Stats.SuccessfulRoutes > 0 {
		result.Status = "partial"
	} else {
		result.Status = "failed"
	}

	p.logger.Info().
		Str("batch_id", request.BatchID).
		Str("status", result.Status).
		Int("successful", result.Stats.SuccessfulRoutes).
		Int("failed", result.Stats.FailedRoutes).
		Dur("duration", result.Stats.TotalTime).
		Msg("Batch processing completed")

	// Send callback if provided
	if request.CallbackURL != "" {
		if err := p.sendCallback(ctx, request.CallbackURL, result); err != nil {
			p.logger.Error().
				Err(err).
				Str("batch_id", request.BatchID).
				Str("callback_url", request.CallbackURL).
				Msg("Failed to send callback")
		}
	}

	return nil
}

// calculateSingleRoute calculates a single route
func (p *BatchRouteProcessor) calculateSingleRoute(
	ctx context.Context,
	route RouteRequest,
) RouteResult {
	result := RouteResult{
		ID:           route.ID,
		OriginZip:    route.OriginZip,
		DestZip:      route.DestZip,
		CalculatedAt: time.Now(),
	}

	// Set default vehicle type if not specified
	vehicleType := route.VehicleType
	if vehicleType == "" {
		vehicleType = "truck"
	}

	// Calculate route using the calculator interface
	distanceMiles, timeMinutes, err := p.calculator.CalculateRoute(
		ctx,
		route.OriginZip,
		route.DestZip,
		vehicleType,
	)
	if err != nil {
		result.Status = "error"
		result.Error = err.Error()
		return result
	}

	result.DistanceMiles = distanceMiles
	result.TimeMinutes = timeMinutes
	result.Status = "success"

	return result
}

// calculateStats calculates batch processing statistics
func (p *BatchRouteProcessor) calculateStats(result BatchResult, startTime time.Time) BatchStats {
	stats := BatchStats{
		TotalRoutes: len(result.Results),
		TotalTime:   time.Since(startTime),
	}

	for _, r := range result.Results {
		if r.Status == "success" {
			stats.SuccessfulRoutes++
		} else {
			stats.FailedRoutes++
		}
	}

	if stats.SuccessfulRoutes > 0 {
		stats.AverageTime = stats.TotalTime / time.Duration(stats.TotalRoutes)
	}

	return stats
}

// sendCallback sends the batch result to the callback URL
func (p *BatchRouteProcessor) sendCallback(
	ctx context.Context,
	callbackURL string,
	result BatchResult,
) error {
	// Marshal result to JSON
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshaling result: %w", err)
	}

	// Create request
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, callbackURL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Batch-ID", result.BatchID)

	// Send request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("sending request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("callback returned status %d", resp.StatusCode)
	}

	return nil
}

// GraphUpdateService handles graph updates from Kafka
type GraphUpdateService struct {
	storage *graph.Router
	logger  zerolog.Logger
	mu      sync.RWMutex
}

// NewGraphUpdateService creates a new graph update service
func NewGraphUpdateService(router *graph.Router, logger zerolog.Logger) *GraphUpdateService {
	return &GraphUpdateService{
		storage: router,
		logger:  logger,
	}
}

// UpdateOSMData handles OSM data updates
func (s *GraphUpdateService) UpdateOSMData(ctx context.Context, update OSMUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info().
		Str("update_id", update.UpdateID).
		Str("region", update.Region).
		Int("nodes_added", update.NodesAdded).
		Int("edges_added", update.EdgesAdded).
		Msg("Processing OSM data update")

	// In a real implementation, this would:
	// 1. Download the updated OSM data for the region
	// 2. Parse and validate the data
	// 3. Update the graph in memory
	// 4. Trigger cache invalidation for affected routes

	// For now, we just log the update
	s.logger.Info().
		Str("update_id", update.UpdateID).
		Msg("OSM data update processed (simulated)")

	return nil
}

// UpdateRestrictions handles restriction updates
func (s *GraphUpdateService) UpdateRestrictions(
	ctx context.Context,
	update RestrictionUpdate,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logger.Info().
		Str("update_id", update.UpdateID).
		Str("restriction_type", update.RestrictionType).
		Int("affected_edges", len(update.EdgeIDs)).
		Msg("Processing restriction update")

	// In a real implementation, this would:
	// 1. Look up the affected edges in the graph
	// 2. Update their restriction properties
	// 3. Mark affected routes for cache invalidation

	// For now, we just log the update
	s.logger.Info().
		Str("update_id", update.UpdateID).
		Msg("Restriction update processed (simulated)")

	return nil
}
