package watchtowersources

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/servicefailure"
	"github.com/emoss08/trenova/internal/core/domain/telematics"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/domain/weatheralert"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/stringutils"
)

/*
One description per source, used twice: by the source's own write path the
moment a record opens, and by the nightly snapshot that corrects the feed
against the record. Writing it once is what keeps the live item and the
reconciled one the same item.

Every path here is an application route, checked again when the item is
stored.
*/

// Where each kind opens.
const (
	pathInsights        = "/insights"
	pathDecisions       = "/desk/decisions"
	pathAgentControl    = "/admin/agent-control"
	pathServiceFailures = "/shipment-management/service-failures"
	pathCarrierMonitor  = "/dispatch/carrier-monitoring"
	pathWorkers         = "/hr/workers"
	pathDispatchConsole = "/dispatch/console"
	pathEDIInbound      = "/edi/inbound-files"
	pathBillingQueue    = "/billing/queue"
	pathDetentionDesk   = "/detention/desk"
)

func panelPath(base string, id string) string {
	return base + "?panelType=edit&panelEntityId=" + id
}

// firstNonZero is for the timestamps a record keeps in more than one place:
// an EDI file's received time, or failing that when the row was written.
func firstNonZero(values ...int64) int64 {
	for _, value := range values {
		if value > 0 {
			return value
		}
	}

	return 0
}

// DescribeInsight is a finding as the tower shows it: the headline, the
// recommendation, and the finding's own severity.
func DescribeInsight(entity *insight.Insight) services.WatchtowerItemInput {
	severity := watchtower.SeverityInfo
	switch entity.Severity {
	case insight.SeverityCritical:
		severity = watchtower.SeverityCritical
	case insight.SeverityWarning:
		severity = watchtower.SeverityWarning
	}
	path := pathInsights + "?id=" + entity.ID.String()
	if len(entity.Links) > 0 && entity.Links[0].IsSafe() {
		path = entity.Links[0].Path
	}

	return services.WatchtowerItemInput{
		TenantInfo:  pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		SourceKind:  watchtower.SourceInsight,
		SourceID:    entity.ID.String(),
		Severity:    severity,
		Title:       entity.Headline,
		Summary:     stringutils.FirstNonEmpty(entity.Recommendation, entity.Narrative),
		SubjectType: agent.SubjectInsight,
		SubjectID:   entity.ID,
		EventKind:   agent.EventInsightDetected,
		Path:        path,
		OccurredAt:  entity.DetectedAt,
	}
}

// DescribeProposal is a proposal waiting on a person. The agent's name is
// passed in because the proposal only carries the run.
func DescribeProposal(entity *agent.AgentProposal, agentName string) services.WatchtowerItemInput {
	who := agentName
	if who == "" {
		who = "An agent"
	}

	return services.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		SourceKind: watchtower.SourceAgentProposal,
		SourceID:   entity.ID.String(),
		Severity:   watchtower.SeverityInfo,
		Title:      who + " wants to " + stringutils.HumanizeSnakeCase(entity.ToolName),
		Summary:    entity.Rationale,
		Path:       pathDecisions + "?run=" + entity.RunID.String(),
		OccurredAt: entity.CreatedAt,
	}
}

// DescribePlan is a plan waiting on a person, decided as one.
func DescribePlan(entity *agent.AgentPlan, agentName string) services.WatchtowerItemInput {
	who := agentName
	if who == "" {
		who = "An agent"
	}
	title := entity.Title
	if strings.TrimSpace(title) == "" {
		title = strconv.Itoa(entity.StepCount) + " changes"
	}

	return services.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		SourceKind: watchtower.SourceAgentPlan,
		SourceID:   entity.ID.String(),
		Severity:   watchtower.SeverityInfo,
		Title:      who + " proposes a plan: " + title,
		Summary:    entity.Summary,
		Path:       pathDecisions + "?run=" + entity.RunID.String(),
		OccurredAt: entity.CreatedAt,
	}
}

// DescribeFailedRun is a run that ended in error, so the person who relies
// on the agent learns it did not do its work.
func DescribeFailedRun(entity *agent.AgentRun, agentName string) services.WatchtowerItemInput {
	who := agentName
	if who == "" {
		who = "An agent"
	}
	occurredAt := entity.StartedAt
	if entity.CompletedAt != nil {
		occurredAt = *entity.CompletedAt
	}

	return services.WatchtowerItemInput{
		TenantInfo:  pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		SourceKind:  watchtower.SourceAgentRunFailed,
		SourceID:    entity.ID.String(),
		Severity:    watchtower.SeverityWarning,
		Title:       who + " could not finish a run",
		Summary:     stringutils.FirstNonEmpty(entity.ErrorMessage, entity.Summary),
		SubjectType: entity.SubjectType,
		SubjectID:   entity.SubjectID,
		Path:        pathAgentControl + "?tab=activity&view=runs&run=" + entity.ID.String(),
		OccurredAt:  occurredAt,
	}
}

// DescribeException is what an agent raised because it could not do its
// work the way it was asked.
func DescribeException(entity *agent.AgentException) services.WatchtowerItemInput {
	severity := watchtower.SeverityWarning
	if entity.Severity == agent.SeverityCritical {
		severity = watchtower.SeverityCritical
	}

	return services.WatchtowerItemInput{
		TenantInfo:  pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		SourceKind:  watchtower.SourceAgentException,
		SourceID:    entity.ID.String(),
		Severity:    severity,
		Title:       "Agent exception: " + stringutils.HumanizeSnakeCase(string(entity.Category)),
		Summary:     entity.AttemptSummary,
		SubjectType: entity.SubjectType,
		SubjectID:   entity.SubjectID,
		Path:        pathAgentControl + "?tab=activity&view=exceptions&exception=" + entity.ID.String(),
		OccurredAt:  entity.CreatedAt,
	}
}

// DescribeServiceFailure is a late or missed stop.
func DescribeServiceFailure(entity *servicefailure.ServiceFailure) services.WatchtowerItemInput {
	severity := watchtower.SeverityWarning
	if entity.Type == servicefailure.TypeMissedDelivery || entity.Type == servicefailure.TypeMissedPickup {
		severity = watchtower.SeverityCritical
	}
	summary := fmt.Sprintf("%s, %d minutes late", stringutils.HumanizeSnakeCase(string(entity.Type)), entity.LateMinutes)
	if entity.Notes != "" {
		summary += ". " + entity.Notes
	}

	return services.WatchtowerItemInput{
		TenantInfo:  pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		SourceKind:  watchtower.SourceServiceFailure,
		SourceID:    entity.ID.String(),
		Severity:    severity,
		Title:       "Service failure " + entity.Number,
		Summary:     summary,
		SubjectType: agent.SubjectShipment,
		SubjectID:   entity.ShipmentID,
		EventKind:   agent.EventServiceFailureDetected,
		Path:        panelPath(pathServiceFailures, entity.ID.String()),
		OccurredAt:  entity.DetectedAt,
	}
}

// CarrierIntelEventOnTower reports whether a change belongs on the feed.
// An informational or low finding is part of the carrier's record rather
// than something a person has to act on, and a closed event is over.
func CarrierIntelEventOnTower(entity *carrierintel.CarrierIntelEvent) bool {
	if entity == nil || entity.Status.IsClosed() {
		return false
	}

	return entity.Severity != carrierintel.SeverityInfo &&
		entity.Severity != carrierintel.SeverityLow
}

// DescribeCarrierIntelEvent is a change on a carrier's authority, insurance
// or safety record.
func DescribeCarrierIntelEvent(entity *carrierintel.CarrierIntelEvent) services.WatchtowerItemInput {
	severity := watchtower.SeverityInfo
	switch entity.Severity {
	case carrierintel.SeverityCritical:
		severity = watchtower.SeverityCritical
	case carrierintel.SeverityHigh, carrierintel.SeverityMedium:
		severity = watchtower.SeverityWarning
	}
	name := entity.SubjectName
	if name == "" {
		name = "DOT " + entity.DOTNumber
	}

	return services.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		SourceKind: watchtower.SourceCarrierIntelEvent,
		SourceID:   entity.ID.String(),
		Severity:   severity,
		Title:      name + ": " + stringutils.HumanizeSnakeCase(string(entity.Category)),
		Summary:    entity.Summary,
		Path:       pathCarrierMonitor + "?tab=events&event=" + entity.ID.String(),
		OccurredAt: entity.DetectedAt,
	}
}

// HOSViolationSourceID keys a violation, whose record has no id of its own.
func HOSViolationSourceID(entity *telematics.WorkerHOSViolation) string {
	return entity.WorkerID.String() + ":" + entity.ViolationType + ":" + strconv.FormatInt(entity.ViolationStartAt, 10)
}

// DescribeHOSViolation is a driver over their hours.
func DescribeHOSViolation(entity *telematics.WorkerHOSViolation) services.WatchtowerItemInput {
	who := "A driver"
	if entity.Worker != nil {
		who = strings.TrimSpace(entity.Worker.FirstName + " " + entity.Worker.LastName)
	}

	return services.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		SourceKind: watchtower.SourceHOSViolation,
		SourceID:   HOSViolationSourceID(entity),
		Severity:   watchtower.SeverityWarning,
		Title:      who + ": " + stringutils.HumanizeSnakeCase(entity.ViolationType),
		Summary:    entity.Description,
		// A worker is not yet a subject an agent runs on, so an hours
		// violation is read rather than handed off; the path opens the
		// driver.
		Path:       panelPath(pathWorkers, entity.WorkerID.String()),
		OccurredAt: entity.ViolationStartAt,
	}
}

// WeatherSeverity maps the weather service's wording to the tower's. Only
// a severe or extreme alert reaches the tower; the rest is the map's.
func WeatherSeverity(alert *weatheralert.WeatherAlert) (watchtower.Severity, bool) {
	switch strings.ToLower(alert.Severity) {
	case "extreme":
		return watchtower.SeverityCritical, true
	case "severe":
		return watchtower.SeverityWarning, true
	default:
		return "", false
	}
}

// DescribeWeatherAlert is a severe or extreme weather alert in an area the
// organization runs through.
func DescribeWeatherAlert(entity *weatheralert.WeatherAlert) services.WatchtowerItemInput {
	severity, _ := WeatherSeverity(entity)
	occurredAt := entity.CreatedAt
	if entity.Effective != nil && *entity.Effective > 0 {
		occurredAt = *entity.Effective
	}

	return services.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		SourceKind: watchtower.SourceWeatherAlert,
		SourceID:   entity.ID.String(),
		Severity:   severity,
		Title:      stringutils.FirstNonEmpty(entity.Headline, entity.Event),
		Summary:    entity.AreaDesc,
		Path:       pathDispatchConsole,
		OccurredAt: occurredAt,
	}
}

// DescribeQuarantinedFile is an EDI file that could not be processed.
func DescribeQuarantinedFile(entity *edi.EDIInboundFile) services.WatchtowerItemInput {
	return services.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		SourceKind: watchtower.SourceEDIInboundQuarantined,
		SourceID:   entity.ID.String(),
		Severity:   watchtower.SeverityWarning,
		Title:      "EDI file quarantined: " + stringutils.FirstNonEmpty(entity.FileName, entity.ID.String()),
		Summary:    entity.FailureReason,
		Path:       panelPath(pathEDIInbound, entity.ID.String()),
		OccurredAt: firstNonZero(entity.ReceivedAt, entity.CreatedAt),
	}
}

// DescribeBillingException is a billing queue item that cannot be invoiced
// as it stands.
func DescribeBillingException(entity *billingqueue.BillingQueueItem) services.WatchtowerItemInput {
	reason := "Needs review"
	if entity.ExceptionReasonCode != nil {
		reason = stringutils.HumanizeSnakeCase(string(*entity.ExceptionReasonCode))
	}

	return services.WatchtowerItemInput{
		TenantInfo:  pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		SourceKind:  watchtower.SourceBillingException,
		SourceID:    entity.ID.String(),
		Severity:    watchtower.SeverityWarning,
		Title:       "Billing exception: " + reason,
		Summary:     entity.ExceptionNotes,
		SubjectType: agent.SubjectBillingQueueItem,
		SubjectID:   entity.ID,
		EventKind:   agent.EventBillingQueueItemException,
		Path:        panelPath(pathBillingQueue, entity.ID.String()),
		OccurredAt:  entity.UpdatedAt,
	}
}

// DescribeDetentionOccurrence is a clock running, or run out, at a stop.
func DescribeDetentionOccurrence(entity *detention.DetentionOccurrence) services.WatchtowerItemInput {
	severity := watchtower.SeverityInfo
	if entity.BillableMinutes > 0 {
		severity = watchtower.SeverityWarning
	}
	where := entity.LocationName
	if where == "" {
		where = "a stop"
	}
	title := "Detention at " + where
	if entity.ShipmentProNumber != "" {
		title += " on " + entity.ShipmentProNumber
	}
	summary := fmt.Sprintf("%d billable minutes", entity.BillableMinutes)
	if entity.CustomerName != "" {
		summary += " for " + entity.CustomerName
	}
	if !entity.BillableAmount.IsZero() {
		summary += ", " + entity.BillableAmount.StringFixed(2) + " " + entity.Currency
	}

	return services.WatchtowerItemInput{
		TenantInfo:  pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID},
		SourceKind:  watchtower.SourceDetentionOccurrence,
		SourceID:    entity.ID.String(),
		Severity:    severity,
		Title:       title,
		Summary:     summary,
		SubjectType: agent.SubjectShipment,
		SubjectID:   entity.ShipmentID,
		Path:        pathDetentionDesk + "?occurrence=" + entity.ID.String(),
		OccurredAt:  entity.ClockStartAt,
	}
}
