package workforcekit

import (
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/changepreview"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolspec"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/app/toolpreview"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentrecord"
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
)

const NoteChars = 2000

const ShortChars = 255

const ParamDocument = "documentId"

const ParamOccurred = "occurredAt"

const ParamDueDate = "dueDate"

const FieldNotes = "notes"

const FieldWorkerID = "workerId"

const FieldDocument = "documentId"

const FieldCompletedAt = "completedAt"

const FieldClosedAt = "closedAt"

const FieldRequestedAt = "requestedAt"

const FieldCertRequestedAt = "certificationRequestedAt"

const FieldStartedAt = "startedAt"

const FieldApproverID = "approverId"

const FieldExpiresAt = "expiresAt"

const FieldResolvedByID = "resolvedById"

const FieldStartDate = "startDate"

const FieldEndDate = "endDate"

func Spec(
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

func PersonOnly(spec *toolspec.ReceivableSpec) *toolspec.ReceivableSpec {
	spec.DefaultTier = agent.TierPropose
	spec.MaxTier = agent.TierPropose
	spec.PersonOnly = true

	return spec
}

func WithSchema(
	spec *toolspec.ReceivableSpec,
	properties map[string]any,
	required ...string,
) *toolspec.ReceivableSpec {
	spec.Properties = properties
	spec.Required = required

	return spec
}

func Targeting(
	spec *toolspec.ReceivableSpec,
	key string,
	resource permission.Resource,
) *toolspec.ReceivableSpec {
	spec.Target = func(params map[string]any) (agenttoolports.ToolTarget, bool) {
		return toolparams.TargetOf(params, key, resource)
	}

	return spec
}

func DocumentProperty() map[string]any {
	return agenttoolschema.RecordIDText(
		permission.ResourceDocument,
		"A document already filed on the worker, from document_list. "+
			"Never guess one.",
	)
}

func NoteProperty(description string) map[string]any {
	return toolparams.StringProperty(description, NoteChars)
}

func OptionalNullDecimal(params map[string]any, key string) (decimal.NullDecimal, error) {
	value, present, err := toolparams.OptionalDecimal(params, key)
	if err != nil || !present {
		return decimal.NullDecimal{}, err
	}
	if value.IsNegative() {
		return decimal.NullDecimal{}, fmt.Errorf("parameter %q cannot be negative", key)
	}

	return decimal.NewNullDecimal(value), nil
}

func OptionalBoolParam(params map[string]any, key string, fallback bool) (bool, error) {
	value, err := toolparams.OptionalBoolPointer(params, key)
	if err != nil || value == nil {
		return fallback, err
	}

	return *value, nil
}

func Record(
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
	FieldWorkerID:              permission.ResourceWorker,
	toolparams.ParamShipmentID: permission.ResourceShipment,
	FieldDocument:              permission.ResourceDocument,
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
	FieldResolvedByID:          permission.ResourceUser,
	"reviewedById":             permission.ResourceUser,
	"generatedById":            permission.ResourceUser,
	FieldApproverID:            permission.ResourceUser,
	"drawnById":                permission.ResourceUser,
	"safetyEventId":            permission.ResourceWorkerSafetyEvent,
	"courseId":                 permission.ResourceTrainingCourse,
	"templateId":               permission.ResourceWorkerChecklistTemplate,
	"credentialTypeId":         permission.ResourceWorkerCredentialType,
}

var wfTimes = map[string]assistantartifact.DisplayType{
	ParamOccurred:                  assistantartifact.DisplayDateTime,
	FieldClosedAt:                  assistantartifact.DisplayDateTime,
	"scheduledAt":                  assistantartifact.DisplayDateTime,
	"collectedAt":                  assistantartifact.DisplayDateTime,
	"finalizedAt":                  assistantartifact.DisplayDateTime,
	"notifiedAt":                   assistantartifact.DisplayDateTime,
	"drawnAt":                      assistantartifact.DisplayDateTime,
	"reportedAt":                   assistantartifact.DisplayDateTime,
	FieldCompletedAt:               assistantartifact.DisplayDate,
	"assignedAt":                   assistantartifact.DisplayDateTime,
	"archivedAt":                   assistantartifact.DisplayDateTime,
	"verifiedAt":                   assistantartifact.DisplayDateTime,
	FieldRequestedAt:               assistantartifact.DisplayDate,
	"responseReceivedAt":           assistantartifact.DisplayDate,
	"lastFollowUpAt":               assistantartifact.DisplayDateTime,
	FieldCertRequestedAt:           assistantartifact.DisplayDateTime,
	"certificationDueAt":           assistantartifact.DisplayDate,
	"pointsExpireAt":               assistantartifact.DisplayDate,
	"dueAt":                        assistantartifact.DisplayDate,
	FieldExpiresAt:                 assistantartifact.DisplayDate,
	"issuedAt":                     assistantartifact.DisplayDate,
	"startsAt":                     assistantartifact.DisplayDate,
	"endsAt":                       assistantartifact.DisplayDate,
	"usedOn":                       assistantartifact.DisplayDate,
	FieldStartedAt:                 assistantartifact.DisplayDate,
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
	FieldStartDate:                 assistantartifact.DisplayDate,
	FieldEndDate:                   assistantartifact.DisplayDate,
}

func Options(fields ...string) []toolpreview.Option {
	return []toolpreview.Option{
		toolpreview.Only(fields...),
		toolpreview.WithRefs(wfRefs),
		toolpreview.Types(wfTimes),
	}
}

func Result(
	action, kind, idParam string,
	id, workerID pulid.ID,
) *agentrecord.ToolExecutionResult {
	result := &agentrecord.ToolExecutionResult{
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
