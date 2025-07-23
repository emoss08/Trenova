// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package websocket

import (
	"github.com/emoss08/trenova/internal/pkg/appctx"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	"github.com/gofiber/fiber/v2"
)

// TestNotification sends a test notification to the current user
func (h *Handler) TestNotification(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get request context",
		})
	}

	// Send notification directly via websocket for testing
	testNotif := map[string]any{
		"id":             pulid.MustNew("notif_").String(),
		"organizationId": reqCtx.OrgID.String(),
		"targetUserId":   reqCtx.UserID.String(),
		"eventType":      "system.info",
		"priority":       "medium",
		"channel":        "user",
		"title":          "Test WebSocket Notification",
		"message":        "This is a test notification to verify WebSocket delivery is working correctly.",
		"data": map[string]any{
			"timestamp": timeutils.NowUnix(),
			"test":      true,
		},
		"source":         "websocket_test",
		"deliveryStatus": "delivered",
		"createdAt":      timeutils.NowUnix(),
		"updatedAt":      timeutils.NowUnix(),
	}

	// Send notification via websocket
	h.webSocketService.BroadcastToUser(reqCtx.UserID.String(), testNotif)

	return c.JSON(fiber.Map{
		"status":          "success",
		"message":         "Test notification sent successfully",
		"notification_id": testNotif["id"],
		"target_user_id":  reqCtx.UserID.String(),
	})
}

// TestOrgNotification sends a test notification to all users in the organization
func (h *Handler) TestOrgNotification(c *fiber.Ctx) error {
	reqCtx, err := appctx.WithRequestContext(c)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to get request context",
		})
	}

	// Broadcast to organization
	h.webSocketService.BroadcastToOrg(reqCtx.OrgID.String(), map[string]any{
		"id":        pulid.MustNew("notif_").String(),
		"type":      "org_broadcast",
		"title":     "Organization Broadcast",
		"message":   "This is a test broadcast to all users in the organization.",
		"priority":  "low",
		"timestamp": timeutils.NowUnix(),
	})

	return c.JSON(fiber.Map{
		"status":  "success",
		"message": "Organization broadcast sent successfully",
		"org_id":  reqCtx.OrgID.String(),
	})
}
