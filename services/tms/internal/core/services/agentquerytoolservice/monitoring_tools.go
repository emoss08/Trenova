package agentquerytoolservice

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/detentionservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/timeutils"
)

type serviceFailureLister interface {
	List(
		ctx context.Context,
		req *repositories.ListServiceFailuresRequest,
	) (*pagination.ListResult[*servicefailure.ServiceFailure], error)
}

type reasonCodeSelector interface {
	SelectOptions(
		ctx context.Context,
		req *repositories.ServiceFailureReasonCodeSelectOptionsRequest,
	) (*pagination.ListResult[*servicefailure.ReasonCode], error)
}

type detentionDeskReader interface {
	ListDesk(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) ([]*detentionservice.DeskEntry, error)
}

type weatherReader interface {
	GetActiveAlerts(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
	) (*serviceports.WeatherAlertFeatureCollection, error)
}

var (
	serviceFailureStatuses = []string{"Open", "Reviewed", "Resolved", "Voided"}
	serviceFailureTypes    = []string{
		"LatePickup", "LateDelivery", "MissedPickup", "MissedDelivery", "AppointmentMissed", "Other",
	}
	serviceFailureSources = []string{"Detected", "Manual", "EDI", "Integration"}
	stopTypes             = []string{"Pickup", "Delivery", "SplitDelivery", "SplitPickup"}
	weatherSeverities     = []string{"Extreme", "Severe", "Moderate", "Minor", "Unknown"}
)

type serviceFailureRow struct {
	ID              string `json:"id"`
	Number          string `json:"number"`
	ShipmentID      string `json:"shipmentId"`
	MoveID          string `json:"moveId"`
	StopID          string `json:"stopId"`
	Type            string `json:"type"`
	Source          string `json:"source"`
	Status          string `json:"status"`
	StopType        string `json:"stopType"`
	ScheduledCutoff string `json:"scheduledCutoff"`
	ActualArrival   string `json:"actualArrival,omitempty"`
	LateMinutes     int64  `json:"lateMinutes"`
	ReasonCodeID    string `json:"reasonCodeId,omitempty"`
	ReasonCode      string `json:"reasonCode,omitempty"`
	Notes           string `json:"notes,omitempty"`
	DetectedAt      string `json:"detectedAt"`
	Version         int64  `json:"version"`
}

func newListServiceFailuresTool(failures serviceFailureLister) serviceports.AgentQueryTool {
	spec := listSpec{
		name:         "list_service_failures",
		entityPlural: "service failures",
		summary: "List service failures: the late and missed pickups and deliveries the " +
			"system has detected or a person recorded, with how late, at which stop, and " +
			"whether anyone has reviewed or resolved them. Filter on status Open for what " +
			"still needs attention, on detectedAt for a period, or give a shipmentId. " +
			"Resolving one takes its id and version, which are in the row.",
		resource: permission.ResourceServiceFailure,
		config:   querybuilder.GetFieldConfiguration((*servicefailure.ServiceFailure)(nil)),
		fields: []listField{
			{Name: "status", Kind: filterEnum, Values: serviceFailureStatuses, Sortable: true},
			{Name: "type", Kind: filterEnum, Values: serviceFailureTypes},
			{Name: "source", Kind: filterEnum, Values: serviceFailureSources},
			{Name: "stopType", Kind: filterEnum, Values: stopTypes},
			{Name: "lateMinutes", Kind: filterNumber, Sortable: true},
			{Name: "detectedAt", Kind: filterDate, Sortable: true},
			{Name: "shipmentId", Kind: filterText, Note: "an exact shipment id"},
		},
	}
	spec.fetchIn = func(
		ctx context.Context,
		opts *pagination.QueryOptions,
		clk clock,
	) ([]any, error) {
		result, err := failures.List(ctx, &repositories.ListServiceFailuresRequest{Filter: opts})
		if err != nil {
			return nil, err
		}

		return listRows(result.Items, func(item *servicefailure.ServiceFailure) any {
			return toServiceFailureRow(item, clk.timezone)
		}), nil
	}

	return newListTool(spec)
}

func toServiceFailureRow(item *servicefailure.ServiceFailure, timezone string) serviceFailureRow {
	row := serviceFailureRow{
		ID:              item.ID.String(),
		Number:          item.Number,
		ShipmentID:      item.ShipmentID.String(),
		MoveID:          item.ShipmentMoveID.String(),
		StopID:          item.StopID.String(),
		Type:            string(item.Type),
		Source:          string(item.Source),
		Status:          string(item.Status),
		StopType:        string(item.StopType),
		ScheduledCutoff: timeutils.FormatUnixDateTimeIn(item.ScheduledCutoff, timezone),
		LateMinutes:     item.LateMinutes,
		Notes:           item.Notes,
		DetectedAt:      timeutils.FormatUnixDateTimeIn(item.DetectedAt, timezone),
		Version:         item.Version,
	}
	if item.ActualArrival > 0 {
		row.ActualArrival = timeutils.FormatUnixDateTimeIn(item.ActualArrival, timezone)
	}
	if item.ReasonCodeID != nil && !item.ReasonCodeID.IsNil() {
		row.ReasonCodeID = item.ReasonCodeID.String()
	}
	if item.ReasonCode != nil {
		row.ReasonCode = item.ReasonCode.Code + " " + item.ReasonCode.Label
	}

	return row
}

type reasonCodeRow struct {
	ID          string `json:"id"`
	Code        string `json:"code"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category"`
	AppliesTo   string `json:"appliesTo"`
	DefaultNote string `json:"defaultNote,omitempty"`
}

type listReasonCodesTool struct {
	codes reasonCodeSelector
}

func newListServiceFailureReasonCodesTool(codes reasonCodeSelector) serviceports.AgentQueryTool {
	return &listReasonCodesTool{codes: codes}
}

func (t *listReasonCodesTool) Name() string { return "list_service_failure_reason_codes" }

func (t *listReasonCodesTool) Description() string {
	return "The reason codes a service failure can carry, such as Weather, Shipper " +
		"delay or Equipment. Call this before resolve_service_failure, which needs a " +
		"reason code id, and pick the code whose category matches what actually " +
		"happened; do not resolve with a code that merely sounds close."
}

func (t *listReasonCodesTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Optional text matched against the code and label.",
			},
			"appliesTo": map[string]any{
				"type":        "string",
				"enum":        []string{"Pickup", "Delivery"},
				"description": "Optional: only codes usable on this kind of stop.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *listReasonCodesTool) PermissionResource() permission.Resource {
	return permission.ResourceServiceFailureReasonCode
}

func (t *listReasonCodesTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	query := optionalString(params.Params, "query")
	appliesTo := optionalString(params.Params, "appliesTo")
	criteria := newSearchCriteria("reason codes").at(clockFor(params))
	criteria.text(query)
	criteria.field("applies to", appliesTo)

	result, err := t.codes.SelectOptions(
		ctx,
		&repositories.ServiceFailureReasonCodeSelectOptionsRequest{
			SelectQueryRequest: &pagination.SelectQueryRequest{
				TenantInfo: tenantOf(params),
				Pagination: pagination.Info{Limit: maxListLimit},
				Query:      query,
			},
			AppliesTo: servicefailure.ReasonCodeAppliesTo(appliesTo),
		},
	)
	if err != nil {
		return nil, err
	}

	rows := make([]reasonCodeRow, 0, len(result.Items))
	for _, code := range result.Items {
		if code == nil || !code.Active {
			continue
		}
		rows = append(rows, reasonCodeRow{
			ID:          code.ID.String(),
			Code:        code.Code,
			Label:       code.Label,
			Description: code.Description,
			Category:    string(code.Category),
			AppliesTo:   string(code.AppliesTo),
			DefaultNote: code.DefaultNote,
		})
	}

	return criteria.result(rows, len(rows)), nil
}

type detentionDeskRow struct {
	OccurrenceID          string `json:"occurrenceId"`
	ShipmentID            string `json:"shipmentId"`
	ProNumber             string `json:"proNumber,omitempty"`
	Customer              string `json:"customer,omitempty"`
	Location              string `json:"location,omitempty"`
	StopType              string `json:"stopType"`
	Status                string `json:"status"`
	NotificationStatus    string `json:"notificationStatus"`
	ArrivedAt             string `json:"arrivedAt,omitempty"`
	FreeTimeExpiresAt     string `json:"freeTimeExpiresAt"`
	MinutesUntilFreeEnds  int32  `json:"minutesUntilFreeEnds"`
	MinutesUntilNoticeDue *int32 `json:"minutesUntilNoticeDue,omitempty"`
	NoticeWindowOpen      bool   `json:"noticeWindowOpen"`
	NoticeSentAt          string `json:"noticeSentAt,omitempty"`
	BillableMinutes       int32  `json:"billableMinutes"`
	AmountAtRisk          string `json:"amountAtRisk"`
	Currency              string `json:"currency,omitempty"`
	RequiresApproval      bool   `json:"requiresApproval"`
	Urgency               string `json:"urgency,omitempty"`
}

type listDetentionDeskTool struct {
	desk detentionDeskReader
}

func newListDetentionDeskTool(desk detentionDeskReader) serviceports.AgentQueryTool {
	return &listDetentionDeskTool{desk: desk}
}

func (t *listDetentionDeskTool) Name() string { return "list_detention_desk" }

func (t *listDetentionDeskTool) Description() string {
	return "Every open detention occurrence: a truck sitting at a stop past its free " +
		"time, with the minutes until free time ends, whether the customer notice is " +
		"due or overdue, and the amount at risk. Urgency Lost means the notice window " +
		"closed without a notice and the charge may not be collectable; NoticeOverdue " +
		"and NoticeDueSoon say what to send. send_detention_notice sends it."
}

func (t *listDetentionDeskTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"urgency": map[string]any{
				"type":        "string",
				"enum":        []string{"Lost", "NoticeOverdue", "NoticeDueSoon"},
				"description": "Optional: only occurrences at this urgency.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *listDetentionDeskTool) PermissionResource() permission.Resource {
	return permission.ResourceDetentionPolicy
}

func (t *listDetentionDeskTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	urgency := optionalString(params.Params, "urgency")
	criteria := newSearchCriteria("open detention occurrences").at(clockFor(params))
	criteria.field("urgency", urgency)

	entries, err := t.desk.ListDesk(ctx, tenantOf(params))
	if err != nil {
		return nil, err
	}

	rows := make([]detentionDeskRow, 0, len(entries))
	for _, entry := range entries {
		if entry == nil || entry.Occurrence == nil {
			continue
		}
		if urgency != "" && !strings.EqualFold(entry.Urgency, urgency) {
			continue
		}
		rows = append(rows, toDetentionDeskRow(entry, params.Timezone))
	}

	return criteria.result(rows, len(rows)), nil
}

func toDetentionDeskRow(entry *detentionservice.DeskEntry, timezone string) detentionDeskRow {
	o := entry.Occurrence
	row := detentionDeskRow{
		OccurrenceID:          o.ID.String(),
		ShipmentID:            o.ShipmentID.String(),
		ProNumber:             o.ShipmentProNumber,
		Customer:              o.CustomerName,
		Location:              o.LocationName,
		StopType:              string(o.StopType),
		Status:                string(o.Status),
		NotificationStatus:    string(o.NotificationStatus),
		FreeTimeExpiresAt:     timeutils.FormatUnixDateTimeIn(o.FreeTimeExpiresAt, timezone),
		MinutesUntilFreeEnds:  entry.MinutesUntilFreeEnds,
		MinutesUntilNoticeDue: entry.MinutesUntilNoticeDue,
		NoticeWindowOpen:      entry.NoticeWindowOpen,
		BillableMinutes:       o.BillableMinutes,
		AmountAtRisk:          entry.AmountAtRisk.StringFixed(2),
		Currency:              o.Currency,
		RequiresApproval:      o.RequiresApproval,
		Urgency:               entry.Urgency,
	}
	if o.ArrivedAt != nil && *o.ArrivedAt > 0 {
		row.ArrivedAt = timeutils.FormatUnixDateTimeIn(*o.ArrivedAt, timezone)
	}
	if o.NoticeSentAt != nil && *o.NoticeSentAt > 0 {
		row.NoticeSentAt = timeutils.FormatUnixDateTimeIn(*o.NoticeSentAt, timezone)
	}

	return row
}

type weatherAlertRow struct {
	ID          string `json:"id"`
	Event       string `json:"event"`
	Severity    string `json:"severity,omitempty"`
	Urgency     string `json:"urgency,omitempty"`
	Category    string `json:"category"`
	Headline    string `json:"headline,omitempty"`
	Area        string `json:"area,omitempty"`
	Effective   string `json:"effective,omitempty"`
	Expires     string `json:"expires,omitempty"`
	Instruction string `json:"instruction,omitempty"`
}

type listWeatherAlertsTool struct {
	weather weatherReader
}

func newListWeatherAlertsTool(weather weatherReader) serviceports.AgentQueryTool {
	return &listWeatherAlertsTool{weather: weather}
}

func (t *listWeatherAlertsTool) Name() string { return "list_weather_alerts" }

func (t *listWeatherAlertsTool) Description() string {
	return "Active National Weather Service alerts: the event, its severity, the area " +
		"it covers and when it expires. By default only Severe and Extreme alerts are " +
		"returned. The alerts are not matched to lanes or trucks; compare the area " +
		"against a shipment's stops and the tractor's position yourself, and say " +
		"which alert you matched to which load."
}

func (t *listWeatherAlertsTool) ParamSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"severity": map[string]any{
				"type":        "string",
				"enum":        weatherSeverities,
				"description": "Return alerts at this severity and worse. Default Severe.",
			},
			"query": map[string]any{
				"type": "string",
				"description": "Optional text matched against the event, headline and area, " +
					"such as a state or county.",
			},
		},
		"additionalProperties": false,
	}
}

func (t *listWeatherAlertsTool) PermissionResource() permission.Resource {
	return permission.ResourceShipment
}

// severityRank orders NWS severities so "Severe and worse" is a comparison.
var severityRank = map[string]int{
	"Extreme": 4, "Severe": 3, "Moderate": 2, "Minor": 1, "Unknown": 0,
}

func (t *listWeatherAlertsTool) Query(
	ctx context.Context,
	params serviceports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	floor := optionalString(params.Params, "severity")
	if _, known := severityRank[floor]; !known {
		floor = "Severe"
	}
	query := strings.ToLower(optionalString(params.Params, "query"))

	criteria := newSearchCriteria("weather alerts").at(clockFor(params))
	criteria.field("severity at least", floor)
	criteria.text(query)

	collection, err := t.weather.GetActiveAlerts(ctx, tenantOf(params))
	if err != nil {
		return nil, err
	}

	rows := make([]weatherAlertRow, 0, len(collection.Features))
	for _, feature := range collection.Features {
		if feature == nil {
			continue
		}
		p := feature.Properties
		if severityRank[p.Severity] < severityRank[floor] {
			continue
		}
		if query != "" && !strings.Contains(
			strings.ToLower(p.Event+" "+p.Headline+" "+p.AreaDesc), query,
		) {
			continue
		}
		row := weatherAlertRow{
			ID:          p.ID.String(),
			Event:       p.Event,
			Severity:    p.Severity,
			Urgency:     p.Urgency,
			Category:    string(p.AlertCategory),
			Headline:    p.Headline,
			Area:        p.AreaDesc,
			Instruction: p.Instruction,
		}
		if p.Effective != nil {
			row.Effective = timeutils.FormatUnixDateTimeIn(*p.Effective, params.Timezone)
		}
		if p.Expires != nil {
			row.Expires = timeutils.FormatUnixDateTimeIn(*p.Expires, params.Timezone)
		}
		rows = append(rows, row)
	}

	return criteria.result(rows, len(rows)), nil
}
