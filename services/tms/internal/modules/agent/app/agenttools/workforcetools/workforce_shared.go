package workforcetools

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const wfNoteChars = 2000

const wfShortChars = 255

const wfParamDocument = "documentId"

const wfParamOccurred = "occurredAt"

const wfParamDueDate = "dueDate"

const wfFieldNotes = "notes"

const wfFieldWorkerID = "workerId"

const wfFieldDocument = "documentId"

const wfFieldCompletedAt = "completedAt"

const wfFieldClosedAt = "closedAt"

const wfFieldRequestedAt = "requestedAt"

const wfFieldCertRequestedAt = "certificationRequestedAt"

const wfFieldStartedAt = "startedAt"

const wfFieldApproverID = "approverId"

const wfFieldExpiresAt = "expiresAt"

const wfFieldResolvedByID = "resolvedById"

const wfFieldStartDate = "startDate"

const wfFieldEndDate = "endDate"

func wfSpec(
	name, description, rationale string,
	resource permission.Resource,
	operation permission.Operation,
) *toolspec.ReceivableSpec {
	return &toolspec.ReceivableSpec{
		Name:        name,
		Description: description,
		Rationale:   rationale,
		Resource:    resource,
		Operation:   operation,
		Egress:      agent.EgressInternal,
		DefaultTier: agent.TierPropose,
		MaxTier:     agent.TierActWithApproval,
		Reversible:  true,
		Artifact:    changepreview.WorkerRecordEntity,
	}
}

func personOnly(spec *toolspec.ReceivableSpec) *toolspec.ReceivableSpec {
	spec.DefaultTier = agent.TierPropose
	spec.MaxTier = agent.TierPropose
	spec.PersonOnly = true

	return spec
}

func withSchema(
	spec *toolspec.ReceivableSpec,
	properties map[string]any,
	required ...string,
) *toolspec.ReceivableSpec {
	spec.Properties = properties
	spec.Required = required

	return spec
}

func targeting(
	spec *toolspec.ReceivableSpec,
	key string,
	resource permission.Resource,
) *toolspec.ReceivableSpec {
	spec.Target = func(params map[string]any) (agenttoolports.ToolTarget, bool) {
		return toolparams.TargetOf(params, key, resource)
	}

	return spec
}

func wfDocumentProperty() map[string]any {
	return agenttoolschema.RecordIDText(
		permission.ResourceDocument,
		"A document already filed on the worker, from document_list. "+
			"Never guess one.",
	)
}

func wfNoteProperty(description string) map[string]any {
	return toolparams.StringProperty(description, wfNoteChars)
}

func optionalNullDecimal(params map[string]any, key string) (decimal.NullDecimal, error) {
	value, present, err := toolparams.OptionalDecimal(params, key)
	if err != nil || !present {
		return decimal.NullDecimal{}, err
	}
	if value.IsNegative() {
		return decimal.NullDecimal{}, fmt.Errorf("parameter %q cannot be negative", key)
	}

	return decimal.NewNullDecimal(value), nil
}

func optionalBoolParam(params map[string]any, key string, fallback bool) (bool, error) {
	value, err := toolparams.OptionalBoolPointer(params, key)
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
		record.Version = changepreview.PinnedVersion(version)
	}

	return record
}

var wfRefs = map[string]permission.Resource{
	wfFieldWorkerID:            permission.ResourceWorker,
	toolparams.ParamShipmentID: permission.ResourceShipment,
	wfFieldDocument:            permission.ResourceDocument,
	"recordedById":             permission.ResourceUser,
	"closedById":               permission.ResourceUser,
	"awardedById":              permission.ResourceUser,
	"orderedById":              permission.ResourceUser,
	"assignedById":             permission.ResourceUser,
	"reviewerId":               permission.ResourceUser,
	"requestedById":            permission.ResourceUser,
	"completedById":            permission.ResourceUser,
	"archivedById":             permission.ResourceUser,
	"verifiedById":             permission.ResourceUser,
	wfFieldResolvedByID:        permission.ResourceUser,
	"reviewedById":             permission.ResourceUser,
	"generatedById":            permission.ResourceUser,
	wfFieldApproverID:          permission.ResourceUser,
	"drawnById":                permission.ResourceUser,
	"safetyEventId":            permission.ResourceWorkerSafetyEvent,
	"courseId":                 permission.ResourceTrainingCourse,
	"templateId":               permission.ResourceWorkerChecklistTemplate,
	"credentialTypeId":         permission.ResourceWorkerCredentialType,
}

var wfTimes = map[string]assistantartifact.DisplayType{
	wfParamOccurred:                assistantartifact.DisplayDateTime,
	wfFieldClosedAt:                assistantartifact.DisplayDateTime,
	"scheduledAt":                  assistantartifact.DisplayDateTime,
	"collectedAt":                  assistantartifact.DisplayDateTime,
	"finalizedAt":                  assistantartifact.DisplayDateTime,
	"notifiedAt":                   assistantartifact.DisplayDateTime,
	"drawnAt":                      assistantartifact.DisplayDateTime,
	"reportedAt":                   assistantartifact.DisplayDateTime,
	wfFieldCompletedAt:             assistantartifact.DisplayDate,
	"assignedAt":                   assistantartifact.DisplayDateTime,
	"archivedAt":                   assistantartifact.DisplayDateTime,
	"verifiedAt":                   assistantartifact.DisplayDateTime,
	wfFieldRequestedAt:             assistantartifact.DisplayDate,
	"responseReceivedAt":           assistantartifact.DisplayDate,
	"lastFollowUpAt":               assistantartifact.DisplayDateTime,
	wfFieldCertRequestedAt:         assistantartifact.DisplayDateTime,
	"certificationDueAt":           assistantartifact.DisplayDate,
	"pointsExpireAt":               assistantartifact.DisplayDate,
	"dueAt":                        assistantartifact.DisplayDate,
	wfFieldExpiresAt:               assistantartifact.DisplayDate,
	"issuedAt":                     assistantartifact.DisplayDate,
	"startsAt":                     assistantartifact.DisplayDate,
	"endsAt":                       assistantartifact.DisplayDate,
	"usedOn":                       assistantartifact.DisplayDate,
	wfFieldStartedAt:               assistantartifact.DisplayDate,
	"employedFrom":                 assistantartifact.DisplayDate,
	"employedTo":                   assistantartifact.DisplayDate,
	"returnedToWorkAt":             assistantartifact.DisplayDate,
	"claimFiledAt":                 assistantartifact.DisplayDate,
	changepreview.FieldResolvedAt:  assistantartifact.DisplayDateTime,
	changepreview.FieldReviewedAt:  assistantartifact.DisplayDateTime,
	changepreview.FieldVoidedAt:    assistantartifact.DisplayDateTime,
	"generatedAt":                  assistantartifact.DisplayDateTime,
	"effectiveAt":                  assistantartifact.DisplayDate,
	changepreview.FieldPeriodStart: assistantartifact.DisplayDate,
	changepreview.FieldPeriodEnd:   assistantartifact.DisplayDate,
	wfFieldStartDate:               assistantartifact.DisplayDate,
	wfFieldEndDate:                 assistantartifact.DisplayDate,
}

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
		Record: changepreview.RecordOf(changepreview.WorkerRecordEntity, workerID),
	}
	if !workerID.IsNil() {
		result.IDs[toolparams.ParamWorkerID] = workerID.String()
	}

	return result
}
