/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package docker

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/external/dockerhub"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

type HandlerParams struct {
	fx.In

	DockerService services.DockerService
	ErrorHandler  *validator.ErrorHandler
}

// Handler handles HTTP requests for Docker management
type Handler struct {
	dockerService services.DockerService
	errorHandler  *validator.ErrorHandler
}

// NewHandler creates a new Docker handler
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		dockerService: p.DockerService,
		errorHandler:  p.ErrorHandler,
	}
}

// RegisterRoutes registers Docker routes
func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	api := r.Group("/docker")

	// Container endpoints
	api.Get("/containers", rl.WithRateLimit(
		[]fiber.Handler{h.listContainers},
		middleware.PerSecond(10),
	)...)

	api.Get("/containers/:id", rl.WithRateLimit(
		[]fiber.Handler{h.inspectContainer},
		middleware.PerSecond(10),
	)...)

	api.Post("/containers/:id/start", rl.WithRateLimit(
		[]fiber.Handler{h.startContainer},
		middleware.PerSecond(5),
	)...)

	api.Post("/containers/:id/stop", rl.WithRateLimit(
		[]fiber.Handler{h.stopContainer},
		middleware.PerSecond(5),
	)...)

	api.Post("/containers/:id/restart", rl.WithRateLimit(
		[]fiber.Handler{h.restartContainer},
		middleware.PerSecond(5),
	)...)

	api.Delete("/containers/:id", rl.WithRateLimit(
		[]fiber.Handler{h.removeContainer},
		middleware.PerSecond(5),
	)...)

	api.Get("/containers/:id/logs", rl.WithRateLimit(
		[]fiber.Handler{h.getContainerLogs},
		middleware.PerSecond(10),
	)...)

	api.Get("/containers/:id/stats", rl.WithRateLimit(
		[]fiber.Handler{h.getContainerStats},
		middleware.PerSecond(10),
	)...)

	// SSE endpoint for real-time stats
	api.Get("/containers/:id/stats/stream", rl.WithRateLimit(
		[]fiber.Handler{h.streamContainerStats},
		middleware.PerSecond(5),
	)...)

	// Image endpoints
	api.Get("/images", rl.WithRateLimit(
		[]fiber.Handler{h.listImages},
		middleware.PerSecond(10),
	)...)

	api.Post("/images/pull", rl.WithRateLimit(
		[]fiber.Handler{h.pullImage},
		middleware.PerSecond(2),
	)...)

	api.Delete("/images/:id", rl.WithRateLimit(
		[]fiber.Handler{h.removeImage},
		middleware.PerSecond(5),
	)...)

	// Volume endpoints
	api.Get("/volumes", rl.WithRateLimit(
		[]fiber.Handler{h.listVolumes},
		middleware.PerSecond(10),
	)...)

	api.Post("/volumes", rl.WithRateLimit(
		[]fiber.Handler{h.createVolume},
		middleware.PerSecond(5),
	)...)

	api.Delete("/volumes/:id", rl.WithRateLimit(
		[]fiber.Handler{h.removeVolume},
		middleware.PerSecond(5),
	)...)

	// Network endpoints
	api.Get("/networks", rl.WithRateLimit(
		[]fiber.Handler{h.listNetworks},
		middleware.PerSecond(10),
	)...)

	api.Get("/networks/:id", rl.WithRateLimit(
		[]fiber.Handler{h.inspectNetwork},
		middleware.PerSecond(10),
	)...)

	// System endpoints
	api.Get("/system/info", rl.WithRateLimit(
		[]fiber.Handler{h.getSystemInfo},
		middleware.PerSecond(5),
	)...)

	api.Get("/system/disk-usage", rl.WithRateLimit(
		[]fiber.Handler{h.getDiskUsage},
		middleware.PerSecond(5),
	)...)

	api.Post("/system/prune", rl.WithRateLimit(
		[]fiber.Handler{h.pruneSystem},
		middleware.PerSecond(1),
	)...)
}

// Container handlers

func (h *Handler) listContainers(c *fiber.Ctx) error {
	// Get request context for permissions
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	all := c.QueryBool("all", false)

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	// Pass user context for request handling
	containers, err := h.dockerService.ListContainers(c.UserContext(), req, all)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"containers": containers,
	})
}

func (h *Handler) inspectContainer(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	containerID := c.Params("id")
	if containerID == "" {
		return h.errorHandler.HandleError(c, errors.NewValidationError(
			"id",
			"required",
			"Container ID is required",
		))
	}

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	container, err := h.dockerService.InspectContainer(c.UserContext(), req, containerID)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(container)
}

func (h *Handler) startContainer(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	containerID := c.Params("id")
	if containerID == "" {
		return h.errorHandler.HandleError(c, errors.NewValidationError(
			"id",
			"required",
			"Container ID is required",
		))
	}

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	if err := h.dockerService.StartContainer(c.UserContext(), req, containerID); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Container started successfully",
	})
}

func (h *Handler) stopContainer(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	containerID := c.Params("id")
	if containerID == "" {
		return h.errorHandler.HandleError(c, errors.NewValidationError(
			"id",
			"required",
			"Container ID is required",
		))
	}

	// Get optional timeout parameter
	var timeout *int
	if t := c.Query("timeout"); t != "" {
		if val, err := strconv.Atoi(t); err == nil {
			timeout = &val
		}
	}

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	if err := h.dockerService.StopContainer(c.UserContext(), req, containerID, timeout); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Container stopped successfully",
	})
}

func (h *Handler) restartContainer(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	containerID := c.Params("id")
	if containerID == "" {
		return h.errorHandler.HandleError(c, errors.NewValidationError(
			"id",
			"required",
			"Container ID is required",
		))
	}

	// Get optional timeout parameter
	var timeout *int
	if t := c.Query("timeout"); t != "" {
		if val, err := strconv.Atoi(t); err == nil {
			timeout = &val
		}
	}

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	if err := h.dockerService.RestartContainer(c.UserContext(), req, containerID, timeout); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Container restarted successfully",
	})
}

func (h *Handler) removeContainer(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	containerID := c.Params("id")
	if containerID == "" {
		return h.errorHandler.HandleError(c, errors.NewValidationError(
			"id",
			"required",
			"Container ID is required",
		))
	}

	force := c.QueryBool("force", false)

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	if err := h.dockerService.RemoveContainer(c.UserContext(), req, containerID, force); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Container removed successfully",
	})
}

func (h *Handler) getContainerLogs(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	containerID := c.Params("id")
	if containerID == "" {
		return h.errorHandler.HandleError(c, errors.NewValidationError(
			"id",
			"required",
			"Container ID is required",
		))
	}

	tail := c.Query("tail", "100")
	follow := c.QueryBool("follow", false)

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	logs, err := h.dockerService.GetContainerLogs(c.UserContext(), req, containerID, tail, follow)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}
	defer logs.Close()

	// Stream logs for SSE or return as JSON
	if follow {
		c.Set("Content-Type", "text/event-stream")
		c.Set("Cache-Control", "no-cache")
		c.Set("Connection", "keep-alive")

		scanner := bufio.NewScanner(logs)
		for scanner.Scan() {
			line := scanner.Text()
			// Docker logs have a header we need to strip
			if len(line) > 8 {
				line = line[8:] // Remove the 8-byte header
			}
			fmt.Fprintf(c, "data: %s\n\n", line)
			if flusher, ok := c.Response().BodyWriter().(interface{ Flush() }); ok {
				flusher.Flush()
			}
		}
	} else {
		// Read all logs and return as JSON
		var logLines []string
		scanner := bufio.NewScanner(logs)
		for scanner.Scan() {
			line := scanner.Text()
			// Docker logs have a header we need to strip
			if len(line) > 8 {
				line = line[8:] // Remove the 8-byte header
			}
			logLines = append(logLines, line)
		}

		return c.Status(fiber.StatusOK).JSON(fiber.Map{
			"logs": logLines,
		})
	}

	return nil
}

func (h *Handler) getContainerStats(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	containerID := c.Params("id")
	if containerID == "" {
		return h.errorHandler.HandleError(c, errors.NewValidationError(
			"id",
			"required",
			"Container ID is required",
		))
	}

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	stats, err := h.dockerService.GetContainerStats(c.UserContext(), req, containerID)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(stats)
}

func (h *Handler) streamContainerStats(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	containerID := c.Params("id")
	if containerID == "" {
		return h.errorHandler.HandleError(c, errors.NewValidationError(
			"id",
			"required",
			"Container ID is required",
		))
	}

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // Disable proxy buffering

	// Create context for cancellation
	ctx, cancel := context.WithCancel(c.UserContext())
	defer cancel()

	// Create channels for stats streaming
	statsChan := make(chan *dockerhub.ContainerStatsResponse)
	errChan := make(chan error)

	// Create Docker operation request
	dockerReq := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	// Start streaming stats in background
	go func() {
		err := h.dockerService.StreamContainerStats(ctx, dockerReq, containerID, statsChan)
		if err != nil && err != context.Canceled {
			errChan <- err
		}
	}()

	// Stream stats to client
	ticker := time.NewTicker(30 * time.Second) // Keepalive ticker
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
			
		case stats := <-statsChan:
			// Marshal stats to JSON
			data, err := json.Marshal(stats)
			if err != nil {
				fmt.Fprintf(c, "event: error\ndata: {\"error\":\"Failed to marshal stats\"}\n\n")
				if flusher, ok := c.Response().BodyWriter().(interface{ Flush() }); ok {
					flusher.Flush()
				}
				continue
			}
			
			// Send SSE event
			fmt.Fprintf(c, "event: stats\ndata: %s\n\n", data)
			if flusher, ok := c.Response().BodyWriter().(interface{ Flush() }); ok {
				flusher.Flush()
			}
			
		case err := <-errChan:
			// Send error event
			errorData, _ := json.Marshal(fiber.Map{"error": err.Error()})
			fmt.Fprintf(c, "event: error\ndata: %s\n\n", errorData)
			if flusher, ok := c.Response().BodyWriter().(interface{ Flush() }); ok {
				flusher.Flush()
			}
			return nil
			
		case <-ticker.C:
			// Send keepalive ping
			fmt.Fprintf(c, ": keepalive\n\n")
			if flusher, ok := c.Response().BodyWriter().(interface{ Flush() }); ok {
				flusher.Flush()
			}
		}
	}
}

// Image handlers

func (h *Handler) listImages(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	images, err := h.dockerService.ListImages(c.UserContext(), req)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"images": images,
	})
}

func (h *Handler) pullImage(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	var req struct {
		ImageName string `json:"imageName" validate:"required"`
	}

	if err := c.BodyParser(&req); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	// Create Docker operation request
	dockerReq := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	result, err := h.dockerService.PullImage(c.UserContext(), dockerReq, req.ImageName)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": result,
	})
}

func (h *Handler) removeImage(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	imageID := c.Params("id")
	if imageID == "" {
		return h.errorHandler.HandleError(c, errors.NewValidationError(
			"id",
			"required",
			"Image ID is required",
		))
	}

	force := c.QueryBool("force", false)

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	if err := h.dockerService.RemoveImage(c.UserContext(), req, imageID, force); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Image removed successfully",
	})
}

// Volume handlers

func (h *Handler) listVolumes(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	volumes, err := h.dockerService.ListVolumes(c.UserContext(), req)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(volumes)
}

func (h *Handler) createVolume(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	var req struct {
		Name   string            `json:"name" validate:"required"`
		Driver string            `json:"driver"`
		Labels map[string]string `json:"labels"`
	}

	if err := c.BodyParser(&req); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	if req.Driver == "" {
		req.Driver = "local"
	}

	// Create Docker operation request
	dockerReq := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	volume, err := h.dockerService.CreateVolume(c.UserContext(), dockerReq, req.Name, req.Driver, req.Labels)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(volume)
}

func (h *Handler) removeVolume(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	volumeID := c.Params("id")
	if volumeID == "" {
		return h.errorHandler.HandleError(c, errors.NewValidationError(
			"id",
			"required",
			"Volume ID is required",
		))
	}

	force := c.QueryBool("force", false)

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	if err := h.dockerService.RemoveVolume(c.UserContext(), req, volumeID, force); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Volume removed successfully",
	})
}

// Network handlers

func (h *Handler) listNetworks(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	networks, err := h.dockerService.ListNetworks(c.UserContext(), req)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"networks": networks,
	})
}

func (h *Handler) inspectNetwork(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	networkID := c.Params("id")
	if networkID == "" {
		return h.errorHandler.HandleError(c, errors.NewValidationError(
			"id",
			"required",
			"Network ID is required",
		))
	}

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	network, err := h.dockerService.InspectNetwork(c.UserContext(), req, networkID)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(network)
}

// System handlers

func (h *Handler) getSystemInfo(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	info, err := h.dockerService.GetSystemInfo(c.UserContext(), req)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(info)
}

func (h *Handler) getDiskUsage(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	usage, err := h.dockerService.GetDiskUsage(c.UserContext(), req)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(usage)
}

func (h *Handler) pruneSystem(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	// Create Docker operation request
	req := &services.DockerOperationRequest{
		UserID:         reqCtx.UserID,
		OrganizationID: reqCtx.OrgID,
		BusinessUnitID: reqCtx.BuID,
	}

	report, err := h.dockerService.PruneSystem(c.UserContext(), req)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(report)
}
