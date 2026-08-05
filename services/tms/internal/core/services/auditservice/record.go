package auditservice

import (
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// RecordParams describes one audited change.
//
// Grouped into a struct rather than passed positionally because the call takes
// a resource, an operation, an actor, two states, a comment and metadata — a
// nine-argument call where several neighbours share a type is the kind that
// silently transposes previous and current.
type RecordParams struct {
	Resource   permission.Resource
	ResourceID string
	Operation  permission.Operation
	Actor      services.AuditActor

	// OrganizationID and BusinessUnitID stay zero for globally scoped
	// resources. Stamping the acting user's tenant onto a change every tenant
	// sees would imply the change was scoped to them.
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID

	// Critical marks entries that must outlive ordinary retention trimming.
	// Compliance evidence — permits, waivers, limits an operator relies on — is
	// exactly what gets asked about long after the fact.
	Critical bool

	Previous any
	Current  any
	Comment  string
	Metadata map[string]any
}

// Record writes one audit entry, diffing previous against current when both are
// present.
//
// Every service that audits was assembling the same LogActionParams, marshalling
// the same two states and appending the same options. Sharing it means a change
// to what an audit entry carries lands in one place rather than being applied to
// each caller and missed on one.
//
// A logging failure is reported rather than returned: the change being audited
// has already happened, and failing the caller would report an error for work
// that succeeded. Callers pass a logger so the failure is attributable.
func Record(
	auditService services.AuditService,
	logger *zap.Logger,
	params *RecordParams,
) {
	if auditService == nil || params == nil {
		return
	}

	logParams := &services.LogActionParams{
		Resource:       params.Resource,
		ResourceID:     params.ResourceID,
		Operation:      params.Operation,
		UserID:         params.Actor.UserID,
		APIKeyID:       params.Actor.APIKeyID,
		PrincipalType:  params.Actor.PrincipalType,
		PrincipalID:    params.Actor.PrincipalID,
		OrganizationID: params.OrganizationID,
		BusinessUnitID: params.BusinessUnitID,
		Critical:       params.Critical,
	}
	if params.Current != nil {
		logParams.CurrentState = jsonutils.MustToJSON(params.Current)
	}
	if params.Previous != nil {
		logParams.PreviousState = jsonutils.MustToJSON(params.Previous)
	}

	opts := make([]services.LogOption, 0, 3)
	if params.Comment != "" {
		opts = append(opts, WithComment(params.Comment))
	}
	if len(params.Metadata) > 0 {
		opts = append(opts, WithMetadata(params.Metadata))
	}
	if params.Previous != nil && params.Current != nil {
		opts = append(opts, WithDiff(params.Previous, params.Current))
	}

	if err := auditService.LogAction(logParams, opts...); err != nil {
		logger.Error("failed to record audit entry",
			zap.String("resource", params.Resource.String()),
			zap.String("resourceId", params.ResourceID),
			zap.Error(err),
		)
	}
}
