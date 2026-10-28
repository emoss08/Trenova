package common

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
)

// driverMessageSend is a Dash message to one driver, as the driver would
// read it.
func DriverMessageSend(notice *notificationports.DriverNotificationPreview) *agent.RecordChange {
	return toolpreview.Send(
		toolpreview.Record{
			Resource: permission.ResourceWorker,
			ID:       notice.WorkerID,
			Label:    DriverName(notice),
		},
		&agent.MessagePreview{
			Channel: agent.MessageChannelDash,
			To:      []string{DriverName(notice)},
			Subject: notice.Title,
			Body:    notice.Message,
		},
	)
}

func WarnDriverUnreachable(
	preview *agent.ToolPreview,
	notice *notificationports.DriverNotificationPreview,
) {
	if notice == nil || notice.Reachable {
		return
	}

	toolpreview.Warn(preview, agent.PreviewWarningDriverUnreachable,
		DriverName(notice)+" has no Dash access, so the message would not reach them.",
		notice.WorkerID.String())
}

func DriverName(notice *notificationports.DriverNotificationPreview) string {
	if notice == nil || strings.TrimSpace(notice.WorkerName) == "" {
		return "the driver"
	}

	return notice.WorkerName
}

func CredentialLabel(credential *workercredential.WorkerCredential) string {
	name := "a credential"
	if credential.CredentialType != nil && credential.CredentialType.Name != "" {
		name = credential.CredentialType.Name
	}
	if credential.ExpiresAt == nil {
		return name
	}

	return name + " (expires " + time.Unix(*credential.ExpiresAt, 0).
		UTC().
		Format("2006-01-02") +
		")"
}
