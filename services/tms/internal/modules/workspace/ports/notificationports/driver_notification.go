package notificationports

import (
	"github.com/emoss08/trenova/internal/modules/workspace/domain/notification"
	"github.com/emoss08/trenova/shared/pulid"
)

// DriverNotificationPreview is a Dash message as the driver would read it,
// rendered by the organization's template. Reachable is false for a driver
// without portal access, whom the message would never reach.
type DriverNotificationPreview struct {
	WorkerID   pulid.ID
	WorkerName string
	Reachable  bool
	// InvitedAt and InvitedEmail are the portal invitation still open for a
	// driver who cannot be reached yet: zero and empty when none is.
	InvitedAt    int64
	InvitedEmail string
	Title        string
	Message      string
	Priority     notification.Priority
	Link         string
	// Text is the message as a text to the driver's phone, set for a driver
	// with no portal login and a number on file. Whether it is sent is the
	// sender's to decide: text messages are part of some plans only.
	Text *DriverSMSPreview
}

// DriverSMSPreview is a text message as it would be sent to a driver's
// phone.
type DriverSMSPreview struct {
	PhoneNumber string
	Message     string
}
