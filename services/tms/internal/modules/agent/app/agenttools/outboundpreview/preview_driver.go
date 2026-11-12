package outboundpreview

import (
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentpreview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercredential"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/notificationports"
	"github.com/emoss08/trenova/pkg/remedy"
)

// driverMessageSend is a Dash message to one driver, as the driver would
// read it.
func DriverMessageSend(
	notice *notificationports.DriverNotificationPreview,
) *agentpreview.RecordChange {
	return toolpreview.Send(
		toolpreview.Record{
			Resource: permission.ResourceWorker,
			ID:       notice.WorkerID,
			Label:    DriverName(notice),
		},
		&agentpreview.MessagePreview{
			Channel: agentpreview.MessageChannelDash,
			To:      []string{DriverName(notice)},
			Subject: notice.Title,
			Body:    notice.Message,
		},
	)
}

func WarnDriverUnreachable(
	preview *agentpreview.ToolPreview,
	notice *notificationports.DriverNotificationPreview,
) {
	if notice == nil || notice.Reachable {
		return
	}

	// The argument is what a person reads in the warning: the driver's name,
	// never the record's id. With no name on file the warning says "the
	// driver" and carries no argument.
	args := make([]string, 0, 1)
	if name := strings.TrimSpace(notice.WorkerName); name != "" {
		args = append(args, name)
	}
	if preview == nil {
		return
	}
	invite := agentpreview.PreviewRemedy{
		Kind:        string(remedy.InviteDriverToPortal),
		RecordKind:  string(permission.ResourceWorker),
		RecordID:    notice.WorkerID.String(),
		RecordLabel: DriverName(notice),
	}
	if notice.InvitedAt > 0 {
		invitedAt := notice.InvitedAt
		invite.TakenAt = &invitedAt
		invite.TakenNote = notice.InvitedEmail
	}
	preview.AddWarning(&agentpreview.PreviewWarning{
		Code:     agentpreview.PreviewWarningDriverUnreachable,
		Args:     args,
		Message:  DriverName(notice) + " has no Dash access, so the message would not reach them.",
		Remedies: []agentpreview.PreviewRemedy{invite},
	})
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
