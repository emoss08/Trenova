package services

import (
	"github.com/emoss08/trenova/internal/core/domain/notification"
	"github.com/emoss08/trenova/shared/pulid"
)

// DriverNotificationPreview is a Dash message as the driver would read it,
// rendered by the organization's template. Reachable is false for a driver
// without portal access, whom the message would never reach.
type DriverNotificationPreview struct {
	WorkerID   pulid.ID
	WorkerName string
	Reachable  bool
	Title      string
	Message    string
	Priority   notification.Priority
	Link       string
}

// DriverSMSPreview is a text message as it would be sent to a driver's
// phone.
type DriverSMSPreview struct {
	PhoneNumber string
	Message     string
}
