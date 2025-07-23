// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package api

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"sync"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/routing/internal/graph"
	"github.com/emoss08/routing/internal/kafka"
	"github.com/emoss08/routing/internal/storage"
	"github.com/gofiber/fiber/v2"
	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"
)

// Handler handles routing API requests
type Handler struct {
	storage       *storage.PostgresStorage
	cache         *redis.Client
	logger        zerolog.Logger
	router        *graph.Router
	metrics       *Metrics
	kafkaProducer *kafka.Producer
	mu            sync.RWMutex
}

// NewHandler creates a new API handler
func NewHandler(
	storage *storage.PostgresStorage,
	cache *redis.Client,
	logger zerolog.Logger,
	kafkaProducer *kafka.Producer,
) *Handler {
	return &Handler{
		storage:       storage,
		cache:         cache,
		logger:        logger,
		router:        nil, // Initialized lazily
		metrics:       NewMetrics(),
		kafkaProducer: kafkaProducer,
	}
}

// Metrics returns the metrics instance
func (h *Handler) Metrics() *Metrics {
	return h.metrics
}

const defaultVehicleType = "truck"

var (
	// zipCodeRegex validates US zip codes (5 digits or 5+4 format)
	zipCodeRegex = regexp.MustCompile(`^\d{5}(-\d{4})?$`)
)

// RouteDistanceRequest represents the request for route distance calculation
type RouteDistanceRequest struct {
	OriginZip   string `query:"origin_zip"   validate:"required,len=5"`
	DestZip     string `query:"dest_zip"     validate:"required,len=5"`
	VehicleType string `query:"vehicle_type" validate:"omitempty,oneof=truck car"`
	Visualize   bool   `query:"visualize"    validate:"omitempty"`
}

// RouteDistanceResponse represents the response for route distance calculation
type RouteDistanceResponse struct {
	DistanceMiles float64   `json:"distance_miles"`
	TimeMinutes   float64   `json:"time_minutes"`
	CalculatedAt  time.Time `json:"calculated_at"`
	CacheHit      bool      `json:"cache_hit"`
}

// GetRouteDistance calculates the distance between two zip codes
func (h *Handler) GetRouteDistance(c *fiber.Ctx) error {
	var req RouteDistanceRequest
	if err := c.QueryParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request parameters",
			"code":  "INVALID_REQUEST",
		})
	}

	// _ Validate zip codes
	if !zipCodeRegex.MatchString(req.OriginZip) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid origin zip code format",
			"code":  "INVALID_ZIP_CODE",
			"field": "origin_zip",
		})
	}
	if !zipCodeRegex.MatchString(req.DestZip) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid destination zip code format",
			"code":  "INVALID_ZIP_CODE",
			"field": "dest_zip",
		})
	}

	// _ Default to truck if not specified
	if req.VehicleType == "" {
		req.VehicleType = defaultVehicleType
	}

	ctx := c.Context()

	// _ If visualization is requested, skip cache and calculate fresh
	if req.Visualize {
		viz, err := h.calculateRouteWithVisualization(ctx, req)
		if err != nil {
			return h.handleRouteError(c, err)
		}
		return c.JSON(viz)
	}

	// _ Check Redis cache first (only for non-visualization requests)
	cacheKey := fmt.Sprintf("route:%s:%s:%s", req.OriginZip, req.DestZip, req.VehicleType)
	cached, err := h.cache.Get(ctx, cacheKey).Result()
	if err == nil && cached != "" {
		var resp RouteDistanceResponse
		if err := sonic.UnmarshalString(cached, &resp); err == nil {
			resp.CacheHit = true
			return c.JSON(resp)
		}
	}

	// _ Check PostgreSQL cache
	distance, travelTime, found, err := h.storage.GetCachedRoute(ctx, req.OriginZip, req.DestZip)
	if err != nil {
		h.logger.Error().Err(err).Msg("Error checking PostgreSQL cache")
	}

	if found {
		resp := RouteDistanceResponse{
			DistanceMiles: distance,
			TimeMinutes:   travelTime,
			CalculatedAt:  time.Now(),
			CacheHit:      true,
		}

		// _ Cache in Redis
		h.cacheResponse(ctx, cacheKey, resp)
		return c.JSON(resp)
	}

	// _ Calculate route
	resp, err := h.calculateRoute(ctx, req)
	if err != nil {
		return h.handleRouteError(c, err)
	}

	// _ Save to caches
	h.cacheResponse(ctx, cacheKey, resp)
	if err := h.storage.SaveCachedRoute(ctx, req.OriginZip, req.DestZip, resp.DistanceMiles, resp.TimeMinutes); err != nil {
		h.logger.Error().Err(err).Msg("Error saving to PostgreSQL cache")
	}

	return c.JSON(resp)
}

// ensureRouter ensures the router is initialized
func (h *Handler) ensureRouter(ctx context.Context) error {
	h.mu.RLock()
	if h.router != nil {
		h.mu.RUnlock()
		return nil
	}
	h.mu.RUnlock()

	h.mu.Lock()
	defer h.mu.Unlock()

	// _ Double-check after acquiring write lock
	if h.router != nil {
		return nil
	}

	// _ Load graph for California region (simplified for now)
	g, err := h.storage.LoadGraphForRegion(ctx, 32.0, -125.0, 42.0, -114.0)
	if err != nil {
		return fmt.Errorf("loading graph: %w", err)
	}

	h.router = graph.NewRouter(g)
	return nil
}

func (h *Handler) calculateRoute(
	ctx context.Context,
	req RouteDistanceRequest,
) (RouteDistanceResponse, error) {
	startTime := time.Now()

	// _ Get node IDs for zip codes
	originNode, err := h.storage.GetNodeIDForZip(ctx, req.OriginZip)
	if err != nil {
		return RouteDistanceResponse{}, fmt.Errorf("getting origin node: %w", err)
	}

	destNode, err := h.storage.GetNodeIDForZip(ctx, req.DestZip)
	if err != nil {
		return RouteDistanceResponse{}, fmt.Errorf("getting destination node: %w", err)
	}

	// _ Ensure router is initialized
	if err := h.ensureRouter(ctx); err != nil {
		return RouteDistanceResponse{}, err
	}

	// _ Set up routing options
	opts := graph.PathOptions{
		TruckOnly:        req.VehicleType == "truck",
		Algorithm:        graph.AlgorithmAStar,
		OptimizationType: graph.OptimizeFastest,
	}

	// _ Calculate route with timeout
	routeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	result, err := h.router.FindRoute(routeCtx, originNode, destNode, opts)
	if err != nil {
		return RouteDistanceResponse{}, fmt.Errorf("pathfinding failed: %w", err)
	}

	computeTimeMS := time.Since(startTime).Milliseconds()

	h.logger.Info().
		Float64("compute_seconds", result.ComputeTime).
		Int("path_nodes", len(result.Path)).
		Str("algorithm", result.Algorithm).
		Msg("Route calculated")

	// _ Convert to response
	resp := RouteDistanceResponse{
		DistanceMiles: result.Distance * 0.000621371, // meters to miles
		TimeMinutes:   result.TravelTime / 60,        // seconds to minutes
		CalculatedAt:  time.Now(),
		CacheHit:      false,
	}

	// _ Publish route calculation event to Kafka
	if h.kafkaProducer != nil {
		event := kafka.RouteCalculatedEvent{
			OriginZip:        req.OriginZip,
			DestZip:          req.DestZip,
			VehicleType:      req.VehicleType,
			DistanceMiles:    resp.DistanceMiles,
			TimeMinutes:      resp.TimeMinutes,
			Algorithm:        result.Algorithm,
			OptimizationType: getOptimizationTypeString(opts.OptimizationType),
			ComputeTimeMS:    computeTimeMS,
			CacheHit:         false,
		}

		// _ Publish asynchronously
		go func() {
			if err := h.kafkaProducer.PublishRouteCalculated(context.Background(), event); err != nil {
				h.logger.Error().Err(err).Msg("Failed to publish route event to Kafka")
			}
		}()
	}

	return resp, nil
}

// calculateRouteWithVisualization calculates a route and returns visualization data
func (h *Handler) calculateRouteWithVisualization(
	ctx context.Context,
	req RouteDistanceRequest,
) (interface{}, error) {
	// _ Get node IDs for zip codes
	originNode, err := h.storage.GetNodeIDForZip(ctx, req.OriginZip)
	if err != nil {
		return nil, NewZipCodeError(req.OriginZip)
	}

	destNode, err := h.storage.GetNodeIDForZip(ctx, req.DestZip)
	if err != nil {
		return nil, NewZipCodeError(req.DestZip)
	}

	// _ Ensure router is initialized
	if err := h.ensureRouter(ctx); err != nil {
		return nil, err
	}

	// _ Set up routing options
	opts := graph.PathOptions{
		TruckOnly: req.VehicleType == "truck",
		Algorithm: graph.AlgorithmAStar,
	}

	// _ Get route with visualization
	routeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	viz, err := h.router.GetRouteVisualization(routeCtx, originNode, destNode, opts)
	if err != nil {
		return nil, fmt.Errorf("route visualization failed: %w", err)
	}

	// _ Convert distances for response
	response := struct {
		*graph.RouteVisualization
		DistanceMiles float64   `json:"distance_miles"`
		TimeMinutes   float64   `json:"time_minutes"`
		CalculatedAt  time.Time `json:"calculated_at"`
	}{
		RouteVisualization: viz,
		DistanceMiles:      viz.Distance * 0.000621371,
		TimeMinutes:        viz.TravelTime / 60,
		CalculatedAt:       time.Now(),
	}

	return response, nil
}

// handleRouteError handles route calculation errors appropriately
func (h *Handler) handleRouteError(c *fiber.Ctx, err error) error {
	h.logger.Error().Err(err).Msg("Route calculation error")

	// _ Check for specific error types
	var clientErr ClientError
	if errors.As(err, &clientErr) {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error":   clientErr.Message,
			"code":    clientErr.Code,
			"details": clientErr.Details,
		})
	}

	// _ Check for known errors
	switch {
	case errors.Is(err, graph.ErrNodeNotFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "One or both zip codes not found in routing network",
			"code":  "ZIP_NOT_ROUTABLE",
		})
	case errors.Is(err, graph.ErrNoPathFound):
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "No route available between these locations",
			"code":  "NO_ROUTE_FOUND",
		})
	case errors.Is(err, graph.ErrTimeout):
		return c.Status(fiber.StatusGatewayTimeout).JSON(fiber.Map{
			"error": "Route calculation timed out",
			"code":  "TIMEOUT",
		})
	case errors.Is(err, graph.ErrSearchSpaceLimit):
		return c.Status(fiber.StatusUnprocessableEntity).JSON(fiber.Map{
			"error": "Route too complex to calculate",
			"code":  "ROUTE_TOO_COMPLEX",
		})
	default:
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to calculate route",
			"code":  "INTERNAL_ERROR",
		})
	}
}

func (h *Handler) cacheResponse(ctx context.Context, key string, resp RouteDistanceResponse) {
	data, err := sonic.Marshal(resp)
	if err != nil {
		h.logger.Error().Err(err).Msg("Error marshaling response for cache")
		return
	}

	if err := h.cache.Set(ctx, key, data, 24*time.Hour).Err(); err != nil {
		h.logger.Error().Err(err).Msg("Error caching response in Redis")
	}
}

// getOptimizationTypeString converts OptimizationType to string
func getOptimizationTypeString(opt graph.OptimizationType) string {
	switch opt {
	case graph.OptimizeShortest:
		return "shortest"
	case graph.OptimizeFastest:
		return "fastest"
	case graph.OptimizePractical:
		return "practical"
	default:
		return "unknown"
	}
}

// HealthCheck returns the health status of the service
func (h *Handler) HealthCheck(c *fiber.Ctx) error {
	ctx := c.Context()

	health := fiber.Map{
		"status": "healthy",
		"time":   time.Now(),
		"checks": fiber.Map{},
	}
	checks := health["checks"].(fiber.Map)

	// _ Check database
	if _, _, _, err := h.storage.GetCachedRoute(ctx, "test", "test"); err != nil {
		checks["database"] = fiber.Map{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "unhealthy"
	} else {
		checks["database"] = fiber.Map{
			"status": "healthy",
		}
	}

	// _ Check Redis
	if err := h.cache.Ping(ctx).Err(); err != nil {
		checks["redis"] = fiber.Map{
			"status": "unhealthy",
			"error":  err.Error(),
		}
		health["status"] = "unhealthy"
	} else {
		checks["redis"] = fiber.Map{
			"status": "healthy",
		}
	}

	// _ Check Kafka if configured
	if h.kafkaProducer != nil {
		stats := h.kafkaProducer.Stats()
		checks["kafka"] = fiber.Map{
			"status": "healthy",
			"stats": fiber.Map{
				"messages": stats.Messages,
				"bytes":    stats.Bytes,
				"errors":   stats.Errors,
			},
		}
	} else {
		checks["kafka"] = fiber.Map{
			"status": "disabled",
		}
	}

	if health["status"] == "unhealthy" {
		return c.Status(fiber.StatusServiceUnavailable).JSON(health)
	}

	return c.JSON(health)
}
