package agentquerytoolservice

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/ports/carrierintelports"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonschemautils"
	"github.com/emoss08/trenova/shared/sliceutils"
)

const (
	paramCarrierIntelOpenOnly = "openOnly"

	carrierIntelEventDefaultLimit = 25
	carrierIntelEventMaxLimit     = 50
	carrierIntelEventsEntity      = "carrier intelligence events"
)

var (
	carrierIntelEventStatuses = agenttoolschema.Source(
		"carrierIntel.eventStatus",
		[]carrierintel.EventStatus{
			carrierintel.EventStatusOpen,
			carrierintel.EventStatusAcknowledged,
			carrierintel.EventStatusResolved,
			carrierintel.EventStatusDismissed,
		},
	)
	carrierIntelSeverities = agenttoolschema.Source(
		"carrierIntel.severity",
		[]carrierintel.Severity{
			carrierintel.SeverityCritical,
			carrierintel.SeverityHigh,
			carrierintel.SeverityMedium,
			carrierintel.SeverityLow,
			carrierintel.SeverityInfo,
		},
	)
	carrierIntelCategories = agenttoolschema.Source(
		"carrierIntel.category",
		carrierintel.AllSections(),
	)
)

type carrierIntelEventRow struct {
	ID             string       `json:"id"`
	CarrierID      string       `json:"carrierId,omitempty"`
	Carrier        string       `json:"carrier,omitempty"`
	DOTNumber      string       `json:"dotNumber"`
	SubjectType    string       `json:"subjectType"`
	Category       string       `json:"category"`
	Severity       string       `json:"severity"`
	Action         string       `json:"action,omitempty"`
	Status         string       `json:"status"`
	Summary        string       `json:"summary"`
	DetectedAt     optionalDate `json:"detectedAt"`
	AcknowledgedAt optionalDate `json:"acknowledgedAt"`
	ResolvedAt     optionalDate `json:"resolvedAt"`
	Resolution     string       `json:"resolution,omitempty"`
}

type carrierIntelEventsResult struct {
	searchOutcome

	NextCursor string `json:"nextCursor,omitempty"`
}

type listCarrierIntelEventsTool struct {
	events carrierintelports.CarrierIntelEventReader
}

func newListCarrierIntelEventsTool(
	events carrierintelports.CarrierIntelEventReader,
) agentports.AgentQueryTool {
	return &listCarrierIntelEventsTool{events: events}
}

func (t *listCarrierIntelEventsTool) Name() string { return "list_carrier_intel_events" }

func (t *listCarrierIntelEventsTool) Description() string {
	return "List carrier intelligence events, newest first: changes found on carriers' " +
		"authority, insurance and safety records. Each row gives the carrier, the category, " +
		"the severity, what the rule did, the summary, when it was detected and whether " +
		"anyone acknowledged or resolved it. Filter by status (openOnly for the ones not " +
		"settled yet), severity, category such as Authority or Insurance, a carrier, or the " +
		"days it was detected. Pass a row's id to get_carrier_intel_event for its prior and " +
		"current values, or to acknowledge_carrier_intel_event or resolve_carrier_intel_event."
}

func (t *listCarrierIntelEventsTool) SearchTerms() []string {
	return []string{"carrier risk alerts", "authority or insurance problems"}
}

func (t *listCarrierIntelEventsTool) ParamSchema() map[string]any {
	return jsonschemautils.Object(map[string]any{
		paramStatus: jsonschemautils.DescribedArray(
			"Only events in these statuses. Open and Acknowledged ones are not settled yet.",
			agenttoolschema.Enum("A status.", carrierIntelEventStatuses),
			len(carrierIntelEventStatuses.Values),
		),
		paramCarrierIntelOpenOnly: jsonschemautils.Boolean(
			"Only events nobody has resolved or dismissed yet: Open and Acknowledged.",
		),
		paramSeverity: jsonschemautils.DescribedArray(
			"Only events of these severities.",
			agenttoolschema.Enum("A severity.", carrierIntelSeverities),
			len(carrierIntelSeverities.Values),
		),
		paramCategory: jsonschemautils.DescribedArray(
			"Only events about these parts of a carrier's record, such as Authority or "+
				"Insurance.",
			agenttoolschema.Enum("A category.", carrierIntelCategories),
			len(carrierIntelCategories.Values),
		),
		paramCarrierID: agenttoolschema.RecordIDText(
			permission.ResourceCarrier,
			"Only this carrier's events, by id from list_carriers or get_carrier.",
		),
		paramQuery: jsonschemautils.Text(
			"Words to find in the carrier's name, its DOT number or the event's summary.",
		),
		paramFromDate: agenttoolschema.Date("Only events detected on or after this day."),
		paramToDate:   agenttoolschema.Date("Only events detected on or before this day."),
		paramLimit: jsonschemautils.Integer(fmt.Sprintf(
			"How many events to return, at most %d. Defaults to %d.",
			carrierIntelEventMaxLimit, carrierIntelEventDefaultLimit,
		)),
		paramAfter: jsonschemautils.Text(
			"The nextCursor from a previous call of this tool, for the next page.",
		),
	})
}

func (t *listCarrierIntelEventsTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{resource: permission.ResourceCarrierIntelligence})
}

func (t *listCarrierIntelEventsTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	criteria := filtercatalog.NewCriteria(carrierIntelEventsEntity).At(clockFor(params))
	req, err := carrierIntelEventsRequest(params, criteria)
	if err != nil {
		return nil, err
	}

	page, err := t.events.ListEvents(ctx, req)
	if err != nil {
		return nil, err
	}

	rows := make([]carrierIntelEventRow, 0, len(page.Items))
	for _, event := range page.Items {
		if event != nil {
			rows = append(rows, carrierIntelEventRowFrom(event))
		}
	}

	result := &carrierIntelEventsResult{searchOutcome: searchResult(criteria, rows, len(rows))}
	result.HasMore = page.HasNextPage
	if result.NextCursor, err = page.NextCursor(); err != nil {
		return nil, fmt.Errorf("encode the next page's cursor: %w", err)
	}

	return result, nil
}

func carrierIntelEventsRequest(
	params *agentports.QueryToolParams,
	criteria *filtercatalog.Criteria,
) (*carrierintelports.ListCarrierIntelEventsRequest, error) {
	statuses, err := optionalEnums(params.Params, paramStatus, carrierintel.EventStatus.IsValid)
	if err != nil {
		return nil, err
	}
	severities, err := optionalEnums(params.Params, paramSeverity, carrierintel.Severity.IsValid)
	if err != nil {
		return nil, err
	}
	categories, err := optionalEnums(params.Params, paramCategory, carrierintel.Section.IsValid)
	if err != nil {
		return nil, err
	}
	carrierID, err := optionalID(params.Params, paramCarrierID)
	if err != nil {
		return nil, err
	}
	from, to, err := detectedWindow(params.Params, criteria.Clock)
	if err != nil {
		return nil, err
	}

	limit := min(
		max(optionalInt(params.Params, paramLimit, carrierIntelEventDefaultLimit), 1),
		carrierIntelEventMaxLimit,
	)
	cursor, err := pagination.NewCursorInfo(limit, optionalString(params.Params, paramAfter))
	if err != nil {
		return nil, fmt.Errorf(
			"parameter %q is not a cursor this tool returned: %w",
			paramAfter,
			err,
		)
	}
	cursor.IncludeTotalCount = false

	tenant := tenantOf(params)
	query := optionalString(params.Params, paramQuery)
	openOnly := optionalBool(params.Params, paramCarrierIntelOpenOnly)

	criteria.Text(query)
	criteria.Field("status", strings.Join(sliceutils.Strings(statuses), ", "))
	if openOnly {
		criteria.Field("status", "not yet resolved or dismissed")
	}
	criteria.Field("severity", strings.Join(sliceutils.Strings(severities), ", "))
	criteria.Field("category", strings.Join(sliceutils.Strings(categories), ", "))
	if carrierID.IsNotNil() {
		criteria.Field(labelCarrier, carrierID.String())
	}
	criteria.Field("detected on or after", optionalString(params.Params, paramFromDate))
	criteria.Field("detected on or before", optionalString(params.Params, paramToDate))

	return &carrierintelports.ListCarrierIntelEventsRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenant,
			Pagination: pagination.Info{Limit: limit},
			Query:      query,
		},
		Cursor:       cursor,
		Statuses:     statuses,
		Severities:   severities,
		Categories:   categories,
		CarrierID:    carrierID,
		OpenOnly:     openOnly,
		DetectedFrom: from,
		DetectedTo:   to,
	}, nil
}

func detectedWindow(params map[string]any, clk clock) (int64, int64, error) {
	from, err := readDay(params, paramFromDate, clk)
	if err != nil {
		return 0, 0, err
	}
	to, err := readDay(params, paramToDate, clk)
	if err != nil {
		return 0, 0, err
	}
	if to > 0 {
		to = endOfDay(clk, to)
	}
	if from > 0 && to > 0 && from > to {
		return 0, 0, errors.New("fromDate must be on or before toDate")
	}

	return from, to, nil
}

func carrierIntelEventRowFrom(event *carrierintel.CarrierIntelEvent) carrierIntelEventRow {
	return carrierIntelEventRow{
		ID:             event.ID.String(),
		CarrierID:      pulidString(event.CarrierID),
		Carrier:        event.SubjectName,
		DOTNumber:      event.DOTNumber,
		SubjectType:    string(event.SubjectType),
		Category:       string(event.Category),
		Severity:       string(event.Severity),
		Action:         string(event.Action),
		Status:         string(event.Status),
		Summary:        event.Summary,
		DetectedAt:     recordedDate(event.DetectedAt),
		AcknowledgedAt: pointerDateOr(event.AcknowledgedAt, absentNotAcknowledged),
		ResolvedAt:     pointerDateOr(event.ResolvedAt, absentNotResolved),
		Resolution:     string(event.Resolution),
	}
}
