package agentquerytoolservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/toolschema"
)

type getWorkerTool struct {
	repo   workerports.WorkerRepository
	access fieldAccess
}

func newGetWorkerTool(
	repo workerports.WorkerRepository,
	permissions permissionports.PermissionEngine,
) agentports.AgentQueryTool {
	return &getWorkerTool{repo: repo, access: newFieldAccess(permissions)}
}

func (t *getWorkerTool) Name() string { return "worker_get" }

func (t *getWorkerTool) Description() string {
	return "Retrieve one worker (driver) by id, including their profile and current state. " +
		"Use worker_list with a query first when you only have a name."
}

func (t *getWorkerTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workerId": agenttoolschema.OfResource(map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyDescription: "The worker's id, from worker_list, " +
					"the page you are on, or this run's subject.",
			}, permission.ResourceWorker),
		},
		"required":             []string{"workerId"},
		"additionalProperties": false,
	}
}

func (t *getWorkerTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceWorker,
	})
}

func (t *getWorkerTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	workerID, err := requirePulid(params.Params, "workerId")
	if err != nil {
		return nil, err
	}

	entity, err := t.repo.GetByID(ctx, workerports.GetWorkerByIDRequest{
		ID: workerID,
		TenantInfo: pagination.TenantInfo{
			OrgID:  params.OrganizationID,
			BuID:   params.BusinessUnitID,
			UserID: params.Actor.UserID,
		},
		IncludeProfile: true,
		IncludeState:   true,
	})
	if err != nil {
		return nil, err
	}

	ceiling := t.access.ceiling(ctx, params, permission.ResourceWorker)

	return workerDetailFrom(entity, t.access, ceiling), nil
}

// workerDetailRow is one worker as a model may see them: the roster row, the
// qualification dates, and the personal fields the person's role reaches.
// The stored entity used to go back whole — date of birth, licence and TWIC
// numbers, drug and alcohol status, home address, emergency contacts — under
// a plain worker:read grant.
type workerDetailRow struct {
	workerRow

	ExternalID   string       `json:"externalId,omitempty"`
	State        string       `json:"state,omitempty"`
	PostalCode   string       `json:"postalCode,omitempty"`
	Email        string       `json:"email,omitempty"`
	PhoneNumber  string       `json:"phoneNumber,omitempty"`
	Emergency    string       `json:"emergencyContact,omitempty"`
	HireDate     optionalDate `json:"hireDate"`
	Termination  optionalDate `json:"terminationDate"`
	CDLRestrict  string       `json:"cdlRestrictions,omitempty"`
	TWICExpiry   optionalDate `json:"twicExpiry"`
	PhysicalDue  optionalDate `json:"physicalDueDate"`
	MVRDue       optionalDate `json:"mvrDueDate"`
	LastMVRCheck optionalDate `json:"lastMvrCheck"`
	TrainingDue  optionalDate `json:"nextTrainingDue"`
	SafetyRating string       `json:"safetyRating,omitempty"`
	Disqualified string       `json:"disqualificationReason,omitempty"`
	ELDExempt    bool         `json:"eldExempt"`
	ShortHaul    bool         `json:"shortHaulExempt"`
	AvailableNow bool         `json:"availableForDispatch"`
	LeaveType    string       `json:"leaveType,omitempty"`
	// Withheld names the fields this person's access does not reach, so a
	// model reads their absence as withheld rather than as not on file.
	Withheld []string `json:"withheldByAccess,omitempty"`
}

// workerDetailFrom projects a worker for the model under a sensitivity
// ceiling. Fields the ceiling does not reach are left out and named in
// Withheld; Confidential fields are never included and never named.
func workerDetailFrom(
	entity *worker.Worker,
	access fieldAccess,
	ceiling permission.FieldSensitivity,
) workerDetailRow {
	row := workerDetailRow{workerRow: toWorkerRow(entity)}
	row.ExternalID = entity.ExternalID
	row.AvailableNow = entity.AvailableForDispatch
	row.LeaveType = string(entity.LeaveType)

	show := func(field string) bool {
		if access.visible(permission.ResourceWorker, field, ceiling) {
			return true
		}
		if access.registry.GetFieldSensitivity(
			permission.ResourceWorker.String(),
			field,
		) != permission.SensitivityConfidential {
			row.Withheld = append(row.Withheld, field)
		}

		return false
	}

	if !show("city") {
		row.City = ""
	}
	if show("postalCode") {
		row.PostalCode = entity.PostalCode
	}
	if entity.State != nil && show("stateId") {
		row.State = entity.State.Abbreviation
	}
	if show("email") {
		row.Email = entity.Email
	}
	if show("phoneNumber") {
		row.PhoneNumber = entity.PhoneNumber
	}
	if show("emergencyContactName") && show("emergencyContactPhone") {
		row.Emergency = strings.TrimSpace(
			entity.EmergencyContactName + " " + entity.EmergencyContactPhone,
		)
	}

	profile := entity.Profile
	if profile == nil {
		return row
	}
	if show("hireDate") {
		row.HireDate = recordedDate(profile.HireDate)
	}
	if show("terminationDate") {
		row.Termination = expectedDate(pointerSeconds(profile.TerminationDate), "still employed")
	}
	if show("disqualificationReason") {
		row.Disqualified = profile.DisqualificationReason
	}
	row.CDLRestrict = profile.CDLRestrictions
	row.TWICExpiry = pointerDate(profile.TWICExpiry)
	row.PhysicalDue = pointerDate(profile.PhysicalDueDate)
	row.MVRDue = pointerDate(profile.MVRDueDate)
	row.LastMVRCheck = recordedDate(profile.LastMVRCheck)
	row.TrainingDue = pointerDate(profile.NextTrainingDue)
	row.SafetyRating = string(profile.SafetyRating)
	row.ELDExempt = profile.ELDExempt
	row.ShortHaul = profile.ShortHaulExempt

	return row
}

// workerName is how a person names a driver. Every tool that returns a worker
// row needs it, and a driver whose record carries only one of the two names
// should still read as a name rather than a stray space.
func workerName(w *worker.Worker) string {
	if w == nil {
		return ""
	}

	return strings.TrimSpace(w.FirstName + " " + w.LastName)
}
