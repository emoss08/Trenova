package agentquerytoolservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
)

type getWorkerTool struct {
	repo   repositories.WorkerRepository
	access fieldAccess
}

func newGetWorkerTool(
	repo repositories.WorkerRepository,
	permissions serviceports.PermissionEngine,
) serviceports.AgentQueryTool {
	return &getWorkerTool{repo: repo, access: newFieldAccess(permissions)}
}

func (t *getWorkerTool) Name() string { return "get_worker" }

func (t *getWorkerTool) Description() string {
	return "Retrieve one worker (driver) by id, including their profile and current state. " +
		"Use search_worker first when you only have a name."
}

func (t *getWorkerTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"workerId": map[string]any{
				"type":        "string",
				"description": "The worker's id",
			},
		},
		"required":             []string{"workerId"},
		"additionalProperties": false,
	}
}

func (t *getWorkerTool) PermissionResource() permission.Resource {
	return permission.ResourceWorker
}

func (t *getWorkerTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	workerID, err := requirePulid(params.Params, "workerId")
	if err != nil {
		return nil, err
	}

	entity, err := t.repo.GetByID(ctx, repositories.GetWorkerByIDRequest{
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
		if access.registry.GetFieldSensitivity(permission.ResourceWorker.String(), field) != permission.SensitivityConfidential {
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
		row.Emergency = strings.TrimSpace(entity.EmergencyContactName + " " + entity.EmergencyContactPhone)
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

type searchWorkerTool struct {
	repo repositories.WorkerRepository
}

func newSearchWorkerTool(repo repositories.WorkerRepository) serviceports.AgentQueryTool {
	return &searchWorkerTool{repo: repo}
}

func (t *searchWorkerTool) Name() string { return "search_worker" }

func (t *searchWorkerTool) Description() string {
	return "List workers (drivers), optionally narrowed by a name or code. " +
		"Call it with no query to see who is on the roster; pass a query only when " +
		"you already have a name. Returns matches with their ids, which get_worker " +
		"can then expand. To find drivers by a licence or medical card date, use " +
		"list_expiring_credentials instead — this tool does not filter on dates."
}

func (t *searchWorkerTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Optional name or code to match. Omit to list workers unfiltered.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": "How many results to return, at most 25",
			},
		},
		"additionalProperties": false,
	}
}

func (t *searchWorkerTool) PermissionResource() permission.Resource {
	return permission.ResourceWorker
}

func (t *searchWorkerTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	// The term is optional because the repository never needed it:
	// ApplyCursorFilters only applies a text search when Query is non-empty, so
	// an empty one already means "the most recent N". Requiring it here made
	// the tool reject a call the layer beneath it would have served.
	query := optionalString(params.Params, "query")

	limit := optionalInt(params.Params, "limit", defaultSearchLimit)
	if limit <= 0 || limit > maxSearchLimit {
		limit = defaultSearchLimit
	}

	criteria := filtercatalog.NewCriteria("workers").At(clockFor(params))
	criteria.Text(query)

	result, err := t.repo.List(ctx, &repositories.ListWorkersRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID:  params.OrganizationID,
				BuID:   params.BusinessUnitID,
				UserID: params.Actor.UserID,
			},
			Pagination: pagination.Info{Limit: limit},
			Query:      query,
		},
	})
	if err != nil {
		return nil, err
	}

	// The same projection the list tools use, rather than the stored entity.
	// Returning the entity cost six times the bytes for the same answer and
	// reported an unrecorded credential as a bare null.
	rows := make([]workerRow, 0, len(result.Items))
	for _, item := range result.Items {
		rows = append(rows, toWorkerRow(item))
	}

	return searchResult(criteria, rows, len(rows)), nil
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
