package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	defaultTimeOffLimit = 25
	maxTimeOffLimit     = 50
	// defaultTimeOffWindowDays is how far ahead an unqualified question reaches.
	// "Who is off?" means this week and next, not the whole year of accrued
	// leave, and a model handed a year of rows summarises the wrong ones.
	defaultTimeOffWindowDays = 14
	maxTimeOffWindowDays     = 365
)

// ptoLister is the read slice of the PTO service.
type ptoLister interface {
	List(
		ctx context.Context,
		req *repositories.ListPTORequest,
	) (*pagination.CursorListResult[*worker.WorkerPTO], error)
}

// timeOffRow is one request in the words a dispatcher covering a board would
// use. The stored entity carries the ledger balance, the search vector and
// three user relations, none of which answer who is out on Thursday.
type timeOffRow struct {
	ID         string `json:"id"`
	WorkerID   string `json:"workerId"`
	WorkerName string `json:"workerName,omitempty"`
	Status     string `json:"status"`
	Type       string `json:"type"`
	StartDate  int64  `json:"startDate"`
	EndDate    int64  `json:"endDate"`
	Days       string `json:"days"`
	Reason     string `json:"reason,omitempty"`
	// Decision carries the reason attached to a rejection or a cancellation,
	// so "why was this turned down" is answerable without a second lookup.
	Decision string `json:"decision,omitempty"`
}

type listTimeOffTool struct {
	pto ptoLister
}

func newListTimeOffTool(pto ptoLister) serviceports.AgentQueryTool {
	return &listTimeOffTool{pto: pto}
}

func (t *listTimeOffTool) Name() string { return "list_time_off" }

func (t *listTimeOffTool) Description() string {
	return "List worker (driver) time-off requests — who is out, who is asking to " +
		"be, and who was turned down. This is the tool for any question about " +
		"availability by date, and the one that yields the request ids the " +
		"approve, reject and cancel tools need. Filter by status to find what is " +
		"still awaiting a decision."
}

func (t *listTimeOffTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"status": map[string]any{
				"type": "string",
				"enum": []string{
					string(worker.PTOStatusRequested),
					string(worker.PTOStatusApproved),
					string(worker.PTOStatusRejected),
					string(worker.PTOStatusCancelled),
				},
				"description": "Requested means still awaiting a decision. Omit for " +
					"every status.",
			},
			"type": map[string]any{
				"type": "string",
				"enum": []string{
					string(worker.PTOTypePersonal),
					string(worker.PTOTypeVacation),
					string(worker.PTOTypeSick),
					string(worker.PTOTypeHoliday),
					string(worker.PTOTypeBereavement),
					string(worker.PTOTypeMaternity),
					string(worker.PTOTypePaternity),
				},
				"description": "The kind of leave. Omit for every kind.",
			},
			"workerId": map[string]any{
				"type": "string",
				"description": "Narrow to one worker, by id from list_workers or " +
					"search_worker. A name is not an id.",
			},
			"withinDays": map[string]any{
				"type": "integer",
				"description": fmt.Sprintf(
					"How far ahead to look from today, in days. Defaults to %d, at "+
						"most %d. Ignored when startingFrom or startingBefore is given.",
					defaultTimeOffWindowDays, maxTimeOffWindowDays,
				),
			},
			"startingFrom": map[string]any{
				"type":        "integer",
				"description": "Only leave starting on or after this Unix second.",
			},
			"startingBefore": map[string]any{
				"type":        "integer",
				"description": "Only leave starting on or before this Unix second.",
			},
			"limit": map[string]any{
				"type":        "integer",
				"description": fmt.Sprintf("How many to return, at most %d.", maxTimeOffLimit),
			},
		},
		"additionalProperties": false,
	}
}

func (t *listTimeOffTool) PermissionResource() permission.Resource {
	return permission.ResourceWorkerPTO
}

func (t *listTimeOffTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	limit := optionalInt(params.Params, "limit", defaultTimeOffLimit)
	if limit <= 0 || limit > maxTimeOffLimit {
		limit = defaultTimeOffLimit
	}

	request := &repositories.ListPTORequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: pagination.TenantInfo{
				OrgID:  params.OrganizationID,
				BuID:   params.BusinessUnitID,
				UserID: params.Actor.UserID,
			},
			Pagination: pagination.Info{Limit: limit},
		},
		IncludeWorker: true,
	}

	criteria := newSearchCriteria("time-off requests")

	status, err := timeOffEnum(params.Params, "status", worker.PTOStatusFromString)
	if err != nil {
		return nil, err
	}
	request.Status = status
	criteria.field("status", status)

	leaveType, err := timeOffEnum(params.Params, "type", worker.PTOTypeFromString)
	if err != nil {
		return nil, err
	}
	request.Type = leaveType
	criteria.field("type", leaveType)

	if raw := optionalString(params.Params, "workerId"); raw != "" {
		workerID, wErr := requirePulid(params.Params, "workerId")
		if wErr != nil {
			return nil, wErr
		}
		request.WorkerID = workerID
		criteria.field("worker", workerID.String())
	}

	applyTimeOffWindow(params.Params, request, criteria)

	result, err := t.pto.List(ctx, request)
	if err != nil {
		return nil, err
	}

	rows := make([]timeOffRow, 0, len(result.Items))
	for _, pto := range result.Items {
		rows = append(rows, toTimeOffRow(pto))
	}

	return criteria.result(rows, len(rows)), nil
}

// applyTimeOffWindow resolves the date window server-side.
//
// An explicit pair wins when the caller gives one. Otherwise the window runs
// from now to withinDays out, which is what an unqualified "who is off" means
// and what keeps the answer from being last year's vacations.
func applyTimeOffWindow(
	params map[string]any,
	request *repositories.ListPTORequest,
	criteria *searchCriteria,
) {
	from := int64(optionalInt(params, "startingFrom", 0))
	before := int64(optionalInt(params, "startingBefore", 0))

	if from > 0 || before > 0 {
		request.StartDateFrom = from
		request.StartDateTo = before
		if from > 0 {
			criteria.field("starting on or after", timeutils.FormatUnixDate(from))
		}
		if before > 0 {
			criteria.field("starting on or before", timeutils.FormatUnixDate(before))
		}

		return
	}

	horizon := optionalInt(params, "withinDays", defaultTimeOffWindowDays)
	if horizon <= 0 || horizon > maxTimeOffWindowDays {
		horizon = defaultTimeOffWindowDays
	}

	now := timeutils.NowUnix()
	request.StartDateFrom = now
	request.StartDateTo = now + int64(horizon)*secondsPerDay
	criteria.field("starting within", fmt.Sprintf("%d days", horizon))
}

// timeOffEnum refuses a value outside the set rather than passing it through.
// A status of "Pending" would match nothing, and an empty list reads to a model
// as "nobody has asked for time off" — the exact failure this catalog exists to
// stop.
func timeOffEnum[T ~string](
	params map[string]any,
	key string,
	parse func(string) (T, error),
) (string, error) {
	raw := optionalString(params, key)
	if raw == "" {
		return "", nil
	}

	value, err := parse(raw)
	if err != nil {
		return "", fmt.Errorf("parameter %q does not accept %q: %w", key, raw, err)
	}

	return string(value), nil
}

func toTimeOffRow(pto *worker.WorkerPTO) timeOffRow {
	row := timeOffRow{
		ID:         pto.ID.String(),
		WorkerID:   pto.WorkerID.String(),
		WorkerName: workerName(pto.Worker),
		Status:     string(pto.Status),
		Type:       string(pto.Type),
		StartDate:  pto.StartDate,
		EndDate:    pto.EndDate,
		Days:       pto.Days.String(),
		Reason:     pto.Reason,
	}

	switch {
	case strings.TrimSpace(pto.RejectionReason) != "":
		row.Decision = pto.RejectionReason
	case strings.TrimSpace(pto.CancellationReason) != "":
		row.Decision = pto.CancellationReason
	}

	return row
}
