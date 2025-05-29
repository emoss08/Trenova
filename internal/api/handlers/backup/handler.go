package backup

import (
	"github.com/emoss08/trenova/internal/api/middleware"
	"github.com/emoss08/trenova/internal/core/services/dbbackup"
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/gofiber/fiber/v2"
	"go.uber.org/fx"
)

// HandlerParams are the input parameters for creating a new backup handler
type HandlerParams struct {
	fx.In

	BackupService *dbbackup.Service
	ErrorHandler  *validator.ErrorHandler
}

// Handler handles HTTP requests for database backups
type Handler struct {
	backupService *dbbackup.Service
	errorHandler  *validator.ErrorHandler
}

// NewHandler creates a new backup handler
func NewHandler(p HandlerParams) *Handler {
	return &Handler{
		backupService: p.BackupService,
		errorHandler:  p.ErrorHandler,
	}
}

// RegisterRoutes registers the backup API routes
func (h *Handler) RegisterRoutes(r fiber.Router, rl *middleware.RateLimiter) {
	// Skip registration if backup service is not initialized
	if h.backupService == nil {
		return
	}

	api := r.Group("/database-backups")

	api.Get("/", rl.WithRateLimit(
		[]fiber.Handler{h.listBackups},
		middleware.PerMinute(20), // 20 reads per minute
	)...)

	api.Post("/", rl.WithRateLimit(
		[]fiber.Handler{h.createBackup},
		middleware.PerMinute(5), // 5 writes per minute
	)...)

	api.Get("/:filename/", rl.WithRateLimit(
		[]fiber.Handler{h.downloadBackup},
		middleware.PerMinute(10), // 10 downloads per minute
	)...)

	api.Delete("/:filename/", rl.WithRateLimit(
		[]fiber.Handler{h.deleteBackup},
		middleware.PerMinute(5), // 5 deletes per minute
	)...)

	api.Post("/restore/", rl.WithRateLimit(
		[]fiber.Handler{h.restoreBackup},
		middleware.PerMinute(2), // 2 restores per minute
	)...)

	api.Post("/cleanup/", rl.WithRateLimit(
		[]fiber.Handler{h.cleanupBackups},
		middleware.PerMinute(5), // 5 cleanups per minute
	)...)
}

// listBackups lists all available backups
func (h *Handler) listBackups(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	backups, err := h.backupService.ListBackups(c.UserContext(), dbbackup.ListBackupsRequest{
		UserID: reqCtx.UserID,
		BuID:   reqCtx.BuID,
		OrgID:  reqCtx.OrgID,
	})
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(dbbackup.BackupListResponse{
		Backups: backups,
	})
}

// createBackup creates a new backup
func (h *Handler) createBackup(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	resp, err := h.backupService.CreateBackup(c.UserContext(), dbbackup.CreateBackupRequest{
		UserID: reqCtx.UserID,
		BuID:   reqCtx.BuID,
		OrgID:  reqCtx.OrgID,
	})
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusCreated).JSON(resp)
}

// downloadBackup downloads a backup file
func (h *Handler) downloadBackup(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	filename := c.Params("filename")
	if filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Filename is required",
		})
	}

	backupPath, err := h.backupService.DownloadBackup(c.UserContext(), &dbbackup.DownloadBackupRequest{
		UserID:   reqCtx.UserID,
		BuID:     reqCtx.BuID,
		OrgID:    reqCtx.OrgID,
		Filename: filename,
	})
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	// Set appropriate headers for file download
	c.Set("Content-Disposition", "attachment; filename="+filename)
	c.Set("Content-Type", "application/octet-stream")

	// Serve the file
	return c.SendFile(backupPath)
}

// deleteBackup deletes a backup file
func (h *Handler) deleteBackup(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	filename := c.Params("filename")
	if filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Filename is required",
		})
	}

	// Delete the file
	if err = h.backupService.DeleteBackup(c.UserContext(), &dbbackup.DeleteBackupRequest{
		UserID:   reqCtx.UserID,
		BuID:     reqCtx.BuID,
		OrgID:    reqCtx.OrgID,
		Filename: filename,
	}); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(dbbackup.BackupDeleteResponse{
		Message: "Backup deleted successfully",
	})
}

// restoreBackup restores a database from a backup file
func (h *Handler) restoreBackup(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	// Parse request
	req := new(dbbackup.RestoreRequest)
	req.UserID = reqCtx.UserID
	req.BuID = reqCtx.BuID
	req.OrgID = reqCtx.OrgID

	if err = c.BodyParser(req); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	if req.Filename == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Filename is required",
		})
	}

	if err = h.backupService.RestoreBackup(c.UserContext(), req); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(dbbackup.BackupRestoreResponse{
		Message: "Database restored successfully",
	})
}

// cleanupBackups applies the retention policy
func (h *Handler) cleanupBackups(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	// Parse retention days
	retentionDays := c.QueryInt("retentionDays", 0)

	if err = h.backupService.ApplyRetentionPolicy(c.UserContext(), &dbbackup.ApplyRetentionPolicyRequest{
		UserID:        reqCtx.UserID,
		BuID:          reqCtx.BuID,
		OrgID:         reqCtx.OrgID,
		RetentionDays: retentionDays,
	}); err != nil {
		return h.errorHandler.HandleError(c, err)
	}

	return c.Status(fiber.StatusOK).JSON(fiber.Map{
		"message": "Backup cleanup completed successfully",
	})
}
