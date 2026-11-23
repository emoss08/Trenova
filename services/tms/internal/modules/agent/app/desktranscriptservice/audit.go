package desktranscriptservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/desktranscript"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/shared/pulid"
)

type look int

const (
	lookOpened look = iota + 1
	lookDownloaded
)

const (
	actionOpened           = "Opened another person's Desk conversation"
	actionDownloaded       = "Downloaded another person's Desk conversation"
	actionExportRequested  = "Requested an export of Desk conversations"
	actionExportDownloaded = "Downloaded an export of Desk conversations"
)

type lookSubject struct {
	ThreadID    pulid.ID
	OwnerUserID pulid.ID
	OwnerName   string
	AgentName   string
	Title       string
}

func subjectOf(conversation *desktranscript.Conversation) lookSubject {
	return lookSubject{
		ThreadID:    conversation.ID,
		OwnerUserID: conversation.UserID,
		OwnerName:   conversation.OwnerName,
		AgentName:   conversation.AgentName,
		Title:       conversation.Title,
	}
}

func (s *Service) recordLook(
	ctx context.Context,
	actor *permissionports.RequestActor,
	subject lookSubject,
	kind look,
) {
	action := actionOpened
	operation := permission.OpRead
	if kind == lookDownloaded {
		action = actionDownloaded
		operation = permission.OpExport
	}

	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceDeskTranscript,
		ResourceID:     subject.ThreadID.String(),
		Operation:      operation,
		Action:         action,
		Actor:          actor.AuditActor(),
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		After: map[string]any{
			"conversationId": subject.ThreadID.String(),
			"ownerUserId":    subject.OwnerUserID.String(),
			"ownerName":      subject.OwnerName,
			"agentName":      subject.AgentName,
			"title":          subject.Title,
		},
		Metadata: map[string]any{
			"ownerUserId":    subject.OwnerUserID.String(),
			"conversationId": subject.ThreadID.String(),
		},
	})
}

func (s *Service) recordExport(
	ctx context.Context,
	actor *permissionports.RequestActor,
	export *desktranscript.Export,
	action string,
) {
	coverage := export.Coverage
	if coverage == nil {
		coverage = &desktranscript.Coverage{Kind: export.Kind}
	}
	state := map[string]any{
		"exportId":          export.ID.String(),
		"kind":              string(export.Kind),
		"conversationCount": export.ConversationCount,
		"ownerCount":        coverage.OwnerCount,
		"ownerNames":        coverage.OwnerNames,
	}
	if coverage.Timeframe != nil {
		state["timeframe"] = string(coverage.Timeframe.Preset)
		state["fromDate"] = coverage.Timeframe.FromDate
		state["toDate"] = coverage.Timeframe.ToDate
		state["windowFrom"] = coverage.Window.From
		state["windowBefore"] = coverage.Window.Before
	}
	if coverage.PersonName != "" {
		state["personName"] = coverage.PersonName
	}
	if coverage.AgentName != "" {
		state["agentName"] = coverage.AgentName
	}
	if len(coverage.Origins) > 0 {
		state["origins"] = coverage.Origins
	}

	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceDeskTranscript,
		ResourceID:     export.ID.String(),
		Operation:      permission.OpExport,
		Action:         action,
		Actor:          actor.AuditActor(),
		OrganizationID: actor.OrganizationID,
		BusinessUnitID: actor.BusinessUnitID,
		After:          state,
	})
}
