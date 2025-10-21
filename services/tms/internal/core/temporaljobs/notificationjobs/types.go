package notificationjobs

import (
	"github.com/emoss08/trenova/internal/core/ports/services"
)

type (
	SendNotificationPayload                    = services.JobCompletionNotificationRequest
	SendConfigurationCopiedNotificationPayload = services.ConfigurationCopiedNotificationRequest
)
