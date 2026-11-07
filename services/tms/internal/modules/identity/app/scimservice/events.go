package scimservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/domain/iam"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/scimports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	statusSucceeded = "succeeded"
	statusFailed    = "failed"
	statusRefused   = "refused"
	statusWithheld  = "withheld"

	resourceTypeUser  = "User"
	resourceTypeGroup = "Group"
	resourceTypeRole  = "RoleGrant"

	maxRecordedMessage = 500
)

type event struct {
	action       iam.ProvisioningAction
	resourceType string
	resourceID   pulid.ID
	externalID   string
	status       string
	message      string
	comment      string
	before       any
	after        any
	metadata     map[string]any
}

func (s *Service) record(ctx context.Context, dir *scimports.Directory, e *event) {
	record := &iam.ProvisioningAuditRecord{
		OrganizationID: dir.OrganizationID,
		DirectoryID:    dir.ID,
		Action:         e.action,
		ResourceType:   e.resourceType,
		ExternalID:     truncate(e.externalID, 255),
		ResourceID:     e.resourceID,
		Status:         e.status,
		ErrorMessage:   truncate(e.message, maxRecordedMessage),
	}
	if err := s.repo.RecordProvisioning(scopedTo(ctx, dir), record); err != nil {
		s.l.Error("failed to record a provisioning event",
			zap.String("directoryId", dir.ID.String()),
			zap.String("resourceType", e.resourceType),
			zap.String("status", e.status),
			zap.Error(err))
	}

	if e.status != statusSucceeded {
		return
	}

	metadata := map[string]any{
		"scimDirectoryId": dir.ID.String(),
		"scimTokenId":     dir.TokenID.String(),
		"resourceType":    e.resourceType,
		"resourceId":      e.resourceID.String(),
	}
	if e.externalID != "" {
		metadata["externalId"] = e.externalID
	}
	for key, value := range e.metadata {
		metadata[key] = value
	}

	s.auditor.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceSCIMDirectory,
		ResourceID:     dir.ID.String(),
		Operation:      permission.OpUpdate,
		Actor:          permissionports.SystemAuditActor(),
		OrganizationID: dir.OrganizationID,
		BusinessUnitID: dir.BusinessUnitID,
		Before:         e.before,
		After:          e.after,
		Comment:        e.comment,
		Metadata:       metadata,
	})
}

func (s *Service) recordFailure(
	ctx context.Context,
	dir *scimports.Directory,
	e *event,
	err error,
) {
	e.status = statusFailed
	if errortypes.IsRefusal(err) || errortypes.IsAuthorizationError(err) {
		e.status = statusRefused
	}
	e.message = errortypes.Summary(err)
	s.record(ctx, dir, e)
}

func truncate(value string, limit int) string {
	runes := []rune(value)
	if len(runes) <= limit {
		return value
	}

	return string(runes[:limit])
}
