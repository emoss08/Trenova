package agenttoolservice

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolschema"
	"github.com/emoss08/trenova/internal/core/services/toolpreview"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const (
	wfNoteChars            = 2000
	wfShortChars           = 255
	wfParamDocument        = "documentId"
	wfParamOccurred        = "occurredAt"
	wfParamDueDate         = "dueDate"
	wfFieldStatus          = "status"
	wfFieldNotes           = "notes"
	wfFieldWorkerID        = "workerId"
	wfFieldDocument        = "documentId"
	wfFieldCreatedAt       = "createdAt"
	wfFieldCompletedAt     = "completedAt"
	wfFieldClosedAt        = "closedAt"
	wfFieldRequestedAt     = "requestedAt"
	wfFieldCertRequestedAt = "certificationRequestedAt"
	wfFieldStartedAt       = "startedAt"
	wfFieldApproverID      = "approverId"
	wfFieldExpiresAt       = "expiresAt"
	wfFieldResolvedByID    = "resolvedById"
	wfFieldStartDate       = "startDate"
	wfFieldEndDate         = "endDate"
)

func wfSpec(
	name, description, rationale string,
	resource permission.Resource,
	operation permission.Operation,
) *receivableSpec {
	return &receivableSpec{
		name:        name,
		description: description,
		rationale:   rationale,
		resource:    resource,
		operation:   operation,
		egress:      agent.EgressInternal,
		defaultTier: agent.TierPropose,
		maxTier:     agent.TierActWithApproval,
		reversible:  true,
		artifact:    workerRecordEntity,
	}
}

func personOnly(spec *receivableSpec) *receivableSpec {
	spec.defaultTier = agent.TierPropose
	spec.maxTier = agent.TierPropose
	spec.personOnly = true

	return spec
}

func withSchema(
	spec *receivableSpec,
	properties map[string]any,
	required ...string,
) *receivableSpec {
	spec.properties = properties
	spec.required = required

	return spec
}

func targeting(spec *receivableSpec, key string, resource permission.Resource) *receivableSpec {
	spec.target = func(params map[string]any) (serviceports.ToolTarget, bool) {
		return targetOf(params, key, resource)
	}

	return spec
}

func wfDocumentProperty() map[string]any {
	return agenttoolschema.RecordIDText(
		permission.ResourceDocument,
		"A document already filed on the worker, from search_documents. "+
			"Never guess one.",
	)
}

func wfNoteProperty(description string) map[string]any {
	return stringProperty(description, wfNoteChars)
}

func optionalID(params map[string]any, key string) (pulid.ID, error) {
	id, err := optionalPulidParam(params, key)
	if err != nil || id == nil {
		return pulid.Nil, err
	}

	return *id, nil
}

func optionalNullDecimal(params map[string]any, key string) (decimal.NullDecimal, error) {
	value, present, err := optionalDecimal(params, key)
	if err != nil || !present {
		return decimal.NullDecimal{}, err
	}
	if value.IsNegative() {
		return decimal.NullDecimal{}, fmt.Errorf("parameter %q cannot be negative", key)
	}

	return decimal.NewNullDecimal(value), nil
}

func optionalIntInRange(
	params map[string]any,
	key string,
	minimum, maximum int,
) (*int, error) {
	if raw, given := params[key]; !given || raw == nil {
		return nil, nil //nolint:nilnil // an absent number is no change and no error
	}
	value, err := requireIntInRange(params, key, minimum, maximum)
	if err != nil {
		return nil, err
	}

	return &value, nil
}

func optionalBoolParam(params map[string]any, key string, fallback bool) (bool, error) {
	value, err := optionalBoolPointer(params, key)
	if err != nil || value == nil {
		return fallback, err
	}

	return *value, nil
}

func wfRecord(
	resource permission.Resource,
	id pulid.ID,
	label string,
	version int64,
) toolpreview.Record {
	record := toolpreview.Record{Resource: resource, ID: id, Label: label}
	if !id.IsNil() {
		record.Version = pinnedVersion(version)
	}

	return record
}

var (
	wfRefs = map[string]permission.Resource{
		wfFieldWorkerID:     permission.ResourceWorker,
		paramShipmentID:     permission.ResourceShipment,
		wfFieldDocument:     permission.ResourceDocument,
		"recordedById":      permission.ResourceUser,
		"closedById":        permission.ResourceUser,
		"awardedById":       permission.ResourceUser,
		"orderedById":       permission.ResourceUser,
		"assignedById":      permission.ResourceUser,
		"reviewerId":        permission.ResourceUser,
		"requestedById":     permission.ResourceUser,
		"completedById":     permission.ResourceUser,
		"archivedById":      permission.ResourceUser,
		"verifiedById":      permission.ResourceUser,
		wfFieldResolvedByID: permission.ResourceUser,
		"reviewedById":      permission.ResourceUser,
		"generatedById":     permission.ResourceUser,
		wfFieldApproverID:   permission.ResourceUser,
		"drawnById":         permission.ResourceUser,
		"safetyEventId":     permission.ResourceWorkerSafetyEvent,
		"courseId":          permission.ResourceTrainingCourse,
		"templateId":        permission.ResourceWorkerChecklistTemplate,
		"credentialTypeId":  permission.ResourceWorkerCredentialType,
	}
	wfTimes = map[string]assistantartifact.DisplayType{
		wfParamOccurred:        assistantartifact.DisplayDateTime,
		wfFieldClosedAt:        assistantartifact.DisplayDateTime,
		"scheduledAt":          assistantartifact.DisplayDateTime,
		"collectedAt":          assistantartifact.DisplayDateTime,
		"finalizedAt":          assistantartifact.DisplayDateTime,
		"notifiedAt":           assistantartifact.DisplayDateTime,
		"drawnAt":              assistantartifact.DisplayDateTime,
		"reportedAt":           assistantartifact.DisplayDateTime,
		wfFieldCompletedAt:     assistantartifact.DisplayDate,
		"assignedAt":           assistantartifact.DisplayDateTime,
		"archivedAt":           assistantartifact.DisplayDateTime,
		"verifiedAt":           assistantartifact.DisplayDateTime,
		wfFieldRequestedAt:     assistantartifact.DisplayDate,
		"responseReceivedAt":   assistantartifact.DisplayDate,
		"lastFollowUpAt":       assistantartifact.DisplayDateTime,
		wfFieldCertRequestedAt: assistantartifact.DisplayDateTime,
		"certificationDueAt":   assistantartifact.DisplayDate,
		"pointsExpireAt":       assistantartifact.DisplayDate,
		"dueAt":                assistantartifact.DisplayDate,
		wfFieldExpiresAt:       assistantartifact.DisplayDate,
		"issuedAt":             assistantartifact.DisplayDate,
		"startsAt":             assistantartifact.DisplayDate,
		"endsAt":               assistantartifact.DisplayDate,
		"usedOn":               assistantartifact.DisplayDate,
		wfFieldStartedAt:       assistantartifact.DisplayDate,
		"employedFrom":         assistantartifact.DisplayDate,
		"employedTo":           assistantartifact.DisplayDate,
		"returnedToWorkAt":     assistantartifact.DisplayDate,
		"claimFiledAt":         assistantartifact.DisplayDate,
		fieldResolvedAt:        assistantartifact.DisplayDateTime,
		fieldReviewedAt:        assistantartifact.DisplayDateTime,
		fieldVoidedAt:          assistantartifact.DisplayDateTime,
		"generatedAt":          assistantartifact.DisplayDateTime,
		"effectiveAt":          assistantartifact.DisplayDate,
		fieldPeriodStart:       assistantartifact.DisplayDate,
		fieldPeriodEnd:         assistantartifact.DisplayDate,
		wfFieldStartDate:       assistantartifact.DisplayDate,
		wfFieldEndDate:         assistantartifact.DisplayDate,
	}
)

func wfOptions(fields ...string) []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(fields...),
		toolpreview.WithRefs(wfRefs),
		toolpreview.Types(wfTimes),
	}
}

func wfResult(
	action, kind, idParam string,
	id, workerID pulid.ID,
) *agent.ToolExecutionResult {
	result := &agent.ToolExecutionResult{
		Action: action,
		Kind:   kind,
		IDs:    map[string]string{idParam: id.String()},
		Record: recordOf(workerRecordEntity, workerID),
	}
	if !workerID.IsNil() {
		result.IDs[paramWorkerID] = workerID.String()
	}

	return result
}
