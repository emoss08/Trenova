package watchtowersources

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentplan"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/domain/insight"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/stopeta"
	"github.com/emoss08/trenova/internal/modules/operations/domain/weatheralert"
	"github.com/emoss08/trenova/internal/modules/operations/ports/checkcallports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/stopetaports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrierintelevent"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/telematics"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/telematicsports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/productguide"
	"github.com/emoss08/trenova/shared/pulid"
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
	pathInbox           = "/inbox"
)

const (
	// A paper with a week or less on it stops being a reminder and starts
	// being a problem; one already expired is a driver who cannot roll.
	credentialWarningDays = int64(7)
	// A move this close to its start has run out of room to plan around.
	coverageCriticalHours   = int64(4)
	maxStopVisitTitleLength = 200
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
func DescribeInsight(entity *insight.Insight) watchtowerports.WatchtowerItemInput {
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

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SourceKind:  watchtower.SourceInsight,
		SourceID:    entity.ID.String(),
		Severity:    severity,
		Title:       entity.Headline,
		Summary:     stringutils.FirstNonEmpty(entity.Recommendation, entity.Narrative),
		SubjectType: agent.SubjectInsight,
		SubjectID:   entity.ID,
		EventKind:   agenttrigger.EventInsightDetected,
		Path:        path,
		OccurredAt:  entity.DetectedAt,
	}
}

// DescribeProposal is a proposal waiting on a person. The agent's name is
// passed in because the proposal only carries the run.
func DescribeProposal(
	entity *agent.AgentProposal,
	agentName string,
) watchtowerports.WatchtowerItemInput {
	who := agentName
	if who == "" {
		who = "An agent"
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SourceKind: watchtower.SourceAgentProposal,
		SourceID:   entity.ID.String(),
		Severity:   watchtower.SeverityInfo,
		Title: who + " wants to " + stringutils.HumanizeSnakeCase(
			agenttoolports.ReadingOrder(entity.ToolName),
		),
		Summary: entity.Rationale,
		Path:    pathDecisions + "?run=" + entity.RunID.String(),
		Facts: []watchtower.Fact{
			fact("Agent", who),
			fact(
				"Change",
				stringutils.HumanizeSnakeCase(agenttoolports.ReadingOrder(entity.ToolName)),
			),
		},
		OccurredAt: entity.CreatedAt,
	}
}

// DescribePlan is a plan waiting on a person, decided as one.
func DescribePlan(
	entity *agentplan.AgentPlan,
	agentName string,
) watchtowerports.WatchtowerItemInput {
	who := agentName
	if who == "" {
		who = "An agent"
	}
	title := entity.Title
	if strings.TrimSpace(title) == "" {
		title = strconv.Itoa(entity.StepCount) + " changes"
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SourceKind: watchtower.SourceAgentPlan,
		SourceID:   entity.ID.String(),
		Severity:   watchtower.SeverityInfo,
		Title:      who + " proposes a plan: " + title,
		Summary:    entity.Summary,
		Path:       pathDecisions + "?run=" + entity.RunID.String(),
		Facts: []watchtower.Fact{
			fact("Agent", who),
			fact("Steps", strconv.Itoa(entity.StepCount)),
		},
		OccurredAt: entity.CreatedAt,
	}
}

// DescribeFailedRun is a run that ended in error, so the person who relies
// on the agent learns it did not do its work.
func DescribeFailedRun(
	entity *agent.AgentRun,
	agentName string,
) watchtowerports.WatchtowerItemInput {
	who := agentName
	if who == "" {
		who = "An agent"
	}
	occurredAt := entity.StartedAt
	if entity.CompletedAt != nil {
		occurredAt = *entity.CompletedAt
	}

	trail := make([]watchtower.TrailEvent, 0, 2)
	if entity.StartedAt > 0 {
		trail = append(trail, watchtower.TrailEvent{At: entity.StartedAt, Text: who + " started"})
	}
	if reason := strings.TrimSpace(entity.ErrorMessage); reason != "" {
		trail = append(trail, watchtower.TrailEvent{At: occurredAt, Text: reason})
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SourceKind:  watchtower.SourceAgentRunFailed,
		SourceID:    entity.ID.String(),
		Severity:    watchtower.SeverityWarning,
		Title:       who + " could not finish a run",
		Summary:     stringutils.FirstNonEmpty(entity.ErrorMessage, entity.Summary),
		SubjectType: entity.SubjectType,
		SubjectID:   entity.SubjectID,
		Path:        pathAgentControl + "?tab=activity&view=runs&run=" + entity.ID.String(),
		Facts: []watchtower.Fact{
			fact("Agent", who),
			fact("Error", entity.ErrorMessage),
			fact("Run", entity.ID.String()),
		},
		Trail:      trail,
		OccurredAt: occurredAt,
	}
}

// DescribeException is what an agent raised because it could not do its
// work the way it was asked.
func DescribeException(entity *agentexception.AgentException) watchtowerports.WatchtowerItemInput {
	severity := watchtower.SeverityWarning
	if entity.Severity == agentexception.SeverityCritical {
		severity = watchtower.SeverityCritical
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SourceKind:  watchtower.SourceAgentException,
		SourceID:    entity.ID.String(),
		Severity:    severity,
		Title:       "Agent exception: " + stringutils.HumanizeSnakeCase(string(entity.Category)),
		Summary:     entity.AttemptSummary,
		SubjectType: entity.SubjectType,
		SubjectID:   entity.SubjectID,
		Path:        pathAgentControl + "?tab=activity&view=exceptions&exception=" + entity.ID.String(),
		Facts: []watchtower.Fact{
			fact("Category", stringutils.HumanizeSnakeCase(string(entity.Category))),
		},
		OccurredAt: entity.CreatedAt,
	}
}

// DescribeServiceFailure is a late or missed stop.
func DescribeServiceFailure(
	entity *servicefailure.ServiceFailure,
) watchtowerports.WatchtowerItemInput {
	severity := watchtower.SeverityWarning
	if entity.Type == servicefailure.TypeMissedDelivery ||
		entity.Type == servicefailure.TypeMissedPickup {
		severity = watchtower.SeverityCritical
	}
	summary := fmt.Sprintf(
		"%s, %d minutes late",
		stringutils.HumanizeSnakeCase(string(entity.Type)),
		entity.LateMinutes,
	)
	if entity.Notes != "" {
		summary += ". " + entity.Notes
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SourceKind:  watchtower.SourceServiceFailure,
		SourceID:    entity.ID.String(),
		Severity:    severity,
		Title:       "Service failure " + entity.Number,
		Summary:     summary,
		SubjectType: agent.SubjectShipment,
		SubjectID:   entity.ShipmentID,
		EventKind:   agenttrigger.EventServiceFailureDetected,
		Path:        panelPath(pathServiceFailures, entity.ID.String()),
		Record:      resourceRecord(permission.ResourceServiceFailure, entity.Number),
		Facts: []watchtower.Fact{
			fact("Failure", stringutils.HumanizeSnakeCase(string(entity.Type))),
			fact("Late by", minutesPhrase(int64(entity.LateMinutes))),
			factAt("Detected", entity.DetectedAt),
		},
		OccurredAt: entity.DetectedAt,
	}
}

// CarrierIntelEventOnTower reports whether a change belongs on the feed.
// An informational or low finding is part of the carrier's record rather
// than something a person has to act on, and a closed event is over.
func CarrierIntelEventOnTower(entity *carrierintelevent.CarrierIntelEvent) bool {
	if entity == nil || entity.Status.IsClosed() {
		return false
	}

	return entity.Severity != carrierintel.SeverityInfo &&
		entity.Severity != carrierintel.SeverityLow
}

// DescribeCarrierIntelEvent is a change on a carrier's authority, insurance
// or safety record.
func DescribeCarrierIntelEvent(
	entity *carrierintelevent.CarrierIntelEvent,
) watchtowerports.WatchtowerItemInput {
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

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SourceKind:  watchtower.SourceCarrierIntelEvent,
		SourceID:    entity.ID.String(),
		Severity:    severity,
		Title:       name + ": " + stringutils.HumanizeSnakeCase(string(entity.Category)),
		Summary:     entity.Summary,
		SubjectType: agent.SubjectCarrierIntelEvent,
		SubjectID:   entity.ID,
		EventKind:   agenttrigger.EventCarrierIntelEventOpened,
		Path:        pathCarrierMonitor + "?tab=events&event=" + entity.ID.String(),
		Record:      resourceRecord(permission.ResourceCarrier, name),
		Facts: []watchtower.Fact{
			fact("DOT", entity.DOTNumber),
			fact("Change", stringutils.HumanizeSnakeCase(string(entity.Category))),
			factAt("Detected", entity.DetectedAt),
		},
		OccurredAt: entity.DetectedAt,
	}
}

// HOSViolationSourceID keys a violation, whose record has no id of its own.
func HOSViolationSourceID(entity *telematics.WorkerHOSViolation) string {
	return entity.WorkerID.String() + ":" + entity.ViolationType + ":" + strconv.FormatInt(
		entity.ViolationStartAt,
		10,
	)
}

// DescribeHOSViolation is a driver over their hours.
func DescribeHOSViolation(
	entity *telematics.WorkerHOSViolation,
) watchtowerports.WatchtowerItemInput {
	who := "A driver"
	if entity.Worker != nil {
		who = strings.TrimSpace(entity.Worker.FirstName + " " + entity.Worker.LastName)
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SourceKind: watchtower.SourceHOSViolation,
		SourceID:   HOSViolationSourceID(entity),
		Severity:   watchtower.SeverityWarning,
		Title:      who + ": " + stringutils.HumanizeSnakeCase(entity.ViolationType),
		Summary:    entity.Description,
		// Nothing is woken by an hours violation, so the item carries its
		// driver for a conversation to start from rather than an event to
		// hand off.
		SubjectType: agent.SubjectWorker,
		SubjectID:   entity.WorkerID,
		Path:        panelPath(pathWorkers, entity.WorkerID.String()),
		Record:      resourceRecord(permission.ResourceWorker, who),
		Facts: []watchtower.Fact{
			fact("Violation", stringutils.HumanizeSnakeCase(entity.ViolationType)),
			factAt("Started", entity.ViolationStartAt),
		},
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
func DescribeWeatherAlert(entity *weatheralert.WeatherAlert) watchtowerports.WatchtowerItemInput {
	severity, _ := WeatherSeverity(entity)
	occurredAt := entity.CreatedAt
	if entity.Effective != nil && *entity.Effective > 0 {
		occurredAt = *entity.Effective
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SourceKind: watchtower.SourceWeatherAlert,
		SourceID:   entity.ID.String(),
		Severity:   severity,
		Title:      stringutils.FirstNonEmpty(entity.Headline, entity.Event),
		Summary:    entity.AreaDesc,
		Path:       pathDispatchConsole,
		Facts: []watchtower.Fact{
			fact("Event", entity.Event),
			fact("Severity", entity.Severity),
			factAtPtr("Ends", entity.Ends),
		},
		Deadline:   deadlineAtPtr(entity.Expires, "Alert expires"),
		OccurredAt: occurredAt,
	}
}

// DescribeQuarantinedFile is an EDI file that could not be processed.
func DescribeQuarantinedFile(
	entity *ediinboundfile.EDIInboundFile,
) watchtowerports.WatchtowerItemInput {
	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SourceKind: watchtower.SourceEDIInboundQuarantined,
		SourceID:   entity.ID.String(),
		Severity:   watchtower.SeverityWarning,
		Title: "EDI file quarantined: " + stringutils.FirstNonEmpty(
			entity.FileName,
			entity.ID.String(),
		),
		Summary:     entity.FailureReason,
		SubjectType: agent.SubjectEDIInboundFile,
		SubjectID:   entity.ID,
		EventKind:   agenttrigger.EventEDIFileQuarantined,
		Path:        panelPath(pathEDIInbound, entity.ID.String()),
		Record:      kindRecord(permission.KindEDIInboundFile, entity.FileName),
		Facts: []watchtower.Fact{
			fact("Reason", entity.FailureReason),
			factAt("Received", firstNonZero(entity.ReceivedAt, entity.CreatedAt)),
		},
		OccurredAt: firstNonZero(entity.ReceivedAt, entity.CreatedAt),
	}
}

// DescribeBillingException is a billing queue item that cannot be invoiced
// as it stands.
func DescribeBillingException(
	entity *billingqueue.BillingQueueItem,
) watchtowerports.WatchtowerItemInput {
	reason := "Needs review"
	if entity.ExceptionReasonCode != nil {
		reason = stringutils.HumanizeSnakeCase(string(*entity.ExceptionReasonCode))
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SourceKind:  watchtower.SourceBillingException,
		SourceID:    entity.ID.String(),
		Severity:    watchtower.SeverityWarning,
		Title:       "Billing exception: " + reason,
		Summary:     entity.ExceptionNotes,
		SubjectType: agent.SubjectBillingQueueItem,
		SubjectID:   entity.ID,
		EventKind:   agenttrigger.EventBillingQueueItemException,
		Path:        panelPath(pathBillingQueue, entity.ID.String()),
		Facts: []watchtower.Fact{
			fact("Reason", reason),
		},
		OccurredAt: entity.UpdatedAt,
	}
}

// DescribeDetentionOccurrence is a clock running, or run out, at a stop.
func DescribeDetentionOccurrence(
	entity *detention.DetentionOccurrence,
) watchtowerports.WatchtowerItemInput {
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

	accruing := entity.BillableMinutes > 0
	dueLabel := "Free time ends"
	if accruing {
		dueLabel = "Detention is accruing"
	}
	facts := []watchtower.Fact{
		factAtPtr("Arrived", entity.ArrivedAt),
		fact("Free time", minutesPhrase(int64(entity.FreeMinutesGranted))),
	}
	impact := make([]string, 0, 2)
	if !entity.BillableAmount.IsZero() {
		accrued := entity.BillableAmount.StringFixed(2) + " " + entity.Currency
		facts = append(facts, fact("Accrued", accrued))
		impact = append(impact, accrued+" so far")
	}
	if entity.CustomerName != "" {
		impact = append(impact, entity.CustomerName)
	}
	trail := make([]watchtower.TrailEvent, 0, 3)
	if entity.ArrivedAt != nil {
		trail = append(
			trail,
			watchtower.TrailEvent{At: *entity.ArrivedAt, Text: "Arrived at " + where},
		)
	}
	if accruing {
		trail = append(
			trail,
			watchtower.TrailEvent{At: entity.FreeTimeExpiresAt, Text: "Free time ended"},
		)
	}
	if entity.DepartedAt == nil {
		trail = append(trail, watchtower.TrailEvent{Text: "Still checked in"})
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SourceKind:  watchtower.SourceDetentionOccurrence,
		SourceID:    entity.ID.String(),
		Severity:    severity,
		Title:       title,
		Summary:     summary,
		SubjectType: agent.SubjectDetentionOccurrence,
		SubjectID:   entity.ID,
		EventKind:   agenttrigger.EventDetentionOccurrenceOpened,
		Path:        pathDetentionDesk + "?occurrence=" + entity.ID.String(),
		Record:      shipmentRecord(entity.ShipmentProNumber),
		Facts:       facts,
		Impact:      impact,
		Trail:       trail,
		Deadline:    deadlineAt(entity.FreeTimeExpiresAt, dueLabel),
		OccurredAt:  entity.ClockStartAt,
	}
}

// ExpiringPaper is one credential coming due, in the words the desk uses
// for it.
type ExpiringPaper struct {
	Name      string
	DaysLeft  int64
	ExpiresAt int64
}

// ExpiringCredentials is a driver's papers coming due, gathered per driver
// rather than per paper: three certificates expiring in the same week is
// one conversation and one renewal packet, not three.
type ExpiringCredentials struct {
	TenantInfo pagination.TenantInfo
	WorkerID   pulid.ID
	WorkerName string
	Papers     []ExpiringPaper
	OccurredAt int64
}

// DescribeExpiringCredentials is a driver who will stop being able to drive
// unless something is renewed.
func DescribeExpiringCredentials(entity ExpiringCredentials) watchtowerports.WatchtowerItemInput {
	severity := watchtower.SeverityInfo
	soonest := int64(0)
	parts := make([]string, 0, len(entity.Papers))
	for i, paper := range entity.Papers {
		if i == 0 || paper.DaysLeft < soonest {
			soonest = paper.DaysLeft
		}
		parts = append(parts, paper.Name+" "+expiryPhrase(paper.DaysLeft))
	}
	switch {
	case soonest < 0:
		severity = watchtower.SeverityCritical
	case soonest <= credentialWarningDays:
		severity = watchtower.SeverityWarning
	}

	var soonestPaper *ExpiringPaper
	facts := make([]watchtower.Fact, 0, len(entity.Papers))
	for i := range entity.Papers {
		paper := &entity.Papers[i]
		if paper.ExpiresAt > 0 {
			facts = append(facts, factAt(paper.Name, paper.ExpiresAt))
		} else {
			facts = append(facts, fact(paper.Name, expiryPhrase(paper.DaysLeft)))
		}
		if paper.ExpiresAt > 0 &&
			(soonestPaper == nil || paper.ExpiresAt < soonestPaper.ExpiresAt) {
			soonestPaper = paper
		}
	}
	var deadline *watchtower.Deadline
	if soonestPaper != nil {
		deadline = deadlineAt(soonestPaper.ExpiresAt, soonestPaper.Name+" expires")
	}

	who := entity.WorkerName
	if who == "" {
		who = "A driver"
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  entity.TenantInfo,
		SourceKind:  watchtower.SourceWorkerCredential,
		SourceID:    entity.WorkerID.String(),
		Severity:    severity,
		Title:       who + ": " + expiryTitle(soonest, len(entity.Papers)),
		Summary:     strings.Join(parts, ", "),
		SubjectType: agent.SubjectWorker,
		SubjectID:   entity.WorkerID,
		EventKind:   agenttrigger.EventWorkerCredentialExpiring,
		Path:        panelPath(pathWorkers, entity.WorkerID.String()),
		Record:      resourceRecord(permission.ResourceWorker, who),
		Facts:       facts,
		Deadline:    deadline,
		OccurredAt:  entity.OccurredAt,
	}
}

func expiryPhrase(daysLeft int64) string {
	switch {
	case daysLeft < 0:
		return "expired " + strconv.FormatInt(-daysLeft, 10) + "d ago"
	case daysLeft == 0:
		return "expires today"
	case daysLeft == 1:
		return "expires tomorrow"
	default:
		return "expires in " + strconv.FormatInt(daysLeft, 10) + "d"
	}
}

func expiryTitle(soonest int64, count int) string {
	noun := "credential"
	if count > 1 {
		noun = strconv.Itoa(count) + " credentials"
	}
	if soonest < 0 {
		return noun + " expired"
	}

	return noun + " expiring"
}

// UncoveredMove is a move the planner could not put anybody on, close
// enough to its start that somebody has to decide.
type UncoveredMove struct {
	TenantInfo pagination.TenantInfo
	MoveID     pulid.ID
	ProNumber  string
	Reason     string
	StartsAt   int64
	HoursOut   int64
}

// DescribeMoveCoverageRisk is a load about to leave with nobody on it.
func DescribeMoveCoverageRisk(entity UncoveredMove) watchtowerports.WatchtowerItemInput {
	severity := watchtower.SeverityWarning
	if entity.HoursOut <= coverageCriticalHours {
		severity = watchtower.SeverityCritical
	}

	title := "Move uncovered"
	if entity.ProNumber != "" {
		title = "Move uncovered on " + entity.ProNumber
	}
	summary := entity.Reason
	if summary == "" {
		summary = "No driver or carrier is on this move."
	}
	if entity.HoursOut > 0 {
		summary += " Starts in " + strconv.FormatInt(entity.HoursOut, 10) + "h."
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  entity.TenantInfo,
		SourceKind:  watchtower.SourceMoveCoverage,
		SourceID:    entity.MoveID.String(),
		Severity:    severity,
		Title:       title,
		Summary:     summary,
		SubjectType: agent.SubjectShipmentMove,
		SubjectID:   entity.MoveID,
		EventKind:   agenttrigger.EventShipmentMoveCoverageAtRisk,
		Path:        pathDispatchConsole,
		Record:      resourceRecord(permission.ResourceShipmentMove, entity.ProNumber),
		Facts: []watchtower.Fact{
			factAt("Starts", entity.StartsAt),
			fact("Reason", entity.Reason),
		},
		Deadline:   deadlineAt(entity.StartsAt, "Move starts"),
		OccurredAt: entity.StartsAt,
	}
}

// DescribeInboundMessage is a piece of mail that is waiting on a person.
//
// Only a message in a waiting state reaches the tower. A message the desk
// answered by itself is work that happened, not work owed, and putting it on
// the feed would bury the ones that are actually waiting.
func DescribeInboundMessage(
	entity *inboundmessage.InboundMessage,
) watchtowerports.WatchtowerItemInput {
	severity := watchtower.SeverityWarning
	if entity.Status == inboundmessage.StatusQuarantined {
		severity = watchtower.SeverityCritical
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: entity.OrganizationID,
			BuID:  entity.BusinessUnitID,
		},
		SourceKind:  watchtower.SourceInboundMessage,
		SourceID:    entity.ID.String(),
		Severity:    severity,
		Title:       inboundMessageTitle(entity),
		Summary:     inboundMessageSummary(entity),
		SubjectType: agent.SubjectInboundMessage,
		SubjectID:   entity.ID,
		EventKind:   agenttrigger.EventInboundMessageClassified,
		Path:        pathInbox + "?message=" + entity.ID.String(),
		Record: resourceRecord(
			permission.ResourceInboundMessage,
			stringutils.FirstNonEmpty(entity.Subject, "(no subject)"),
		),
		Facts: []watchtower.Fact{
			fact("From", entity.FromAddress),
			fact("Kind", string(entity.Classification)),
			factAt("Received", firstNonZero(entity.ReceivedAt, entity.CreatedAt)),
		},
		OccurredAt: firstNonZero(entity.ReceivedAt, entity.CreatedAt),
	}
}

// inboundMessageTitle leads with what the message is, because that is what
// decides who picks it up. An unclassified one says so rather than guessing.
func inboundMessageTitle(entity *inboundmessage.InboundMessage) string {
	subject := stringutils.FirstNonEmpty(entity.Subject, "(no subject)")
	if entity.Classification == "" {
		return "Message needs review: " + subject
	}

	return string(entity.Classification) + " needs review: " + subject
}

// inboundMessageSummary says who it is from and what, if anything, it was
// matched to — with the reason, because a match nobody can check is a match
// nobody will trust.
func inboundMessageSummary(entity *inboundmessage.InboundMessage) string {
	parts := make([]string, 0, 3)
	if entity.FromAddress != "" {
		parts = append(parts, "From "+entity.FromAddress)
	}
	if entity.MatchReason != "" {
		parts = append(parts, entity.MatchReason)
	}
	if entity.FailureText != "" {
		parts = append(parts, entity.FailureText)
	}
	if entity.ReviewNote != "" {
		parts = append(parts, entity.ReviewNote)
	}

	return strings.Join(parts, " · ")
}

func TelematicsStopVisitSourceID(event *telematics.TelematicsEvent) string {
	target := "unlocated"
	switch {
	case event.StopID.IsNotNil():
		target = event.StopID.String()
	case event.LocationID.IsNotNil():
		target = event.LocationID.String()
	}

	return event.ShipmentMoveID.String() + ":" + target + ":" + string(event.StopVisit)
}

func DescribeTelematicsStopVisit(
	review *telematicsports.StopReview,
) watchtowerports.WatchtowerItemInput {
	event := review.Event

	visit := "Arrival"
	if event.StopVisit == shipment.VisitDeparture {
		visit = "Departure"
	}
	title := visit + " not recorded"
	if place := strings.TrimSpace(event.AddressName); place != "" {
		title = visit + " at " + place + " not recorded"
	}

	path := pathDispatchConsole
	if review.ShipmentID.IsNotNil() {
		if recordPath, ok := productguide.RecordPath("shipment", review.ShipmentID.String()); ok {
			path = recordPath
		}
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo: pagination.TenantInfo{
			OrgID: event.OrganizationID,
			BuID:  event.BusinessUnitID,
		},
		SourceKind:  watchtower.SourceTelematicsStopVisit,
		SourceID:    TelematicsStopVisitSourceID(event),
		Severity:    watchtower.SeverityWarning,
		Title:       stringutils.TruncateRunes(title, maxStopVisitTitleLength),
		Summary:     event.StopOutcomeReason,
		SubjectType: agent.SubjectShipmentMove,
		SubjectID:   event.ShipmentMoveID,
		Path:        path,
		Facts: []watchtower.Fact{
			fact("Place", event.AddressName),
			factAt("Seen", event.OccurredAt),
		},
		OccurredAt: event.OccurredAt,
	}
}

func DescribeNoCheckCall(
	tenant pagination.TenantInfo,
	move *checkcallports.OverdueMove,
) watchtowerports.WatchtowerItemInput {
	window := int64(move.IntervalHours) * 60 * 60
	severity := watchtower.SeverityWarning
	if window > 0 && move.OverdueBy >= window {
		severity = watchtower.SeverityCritical
	}

	title := "No check call"
	if move.ProNumber != "" {
		title = "No check call on " + move.ProNumber
	}

	carrier := stringutils.FirstNonEmpty(move.CarrierName, "The carrier")
	hours := strconv.FormatInt(int64(move.IntervalHours), 10)
	summary := carrier + " has not checked in within the " + hours + "h window for this load."
	if severity == watchtower.SeverityCritical {
		summary = carrier + " has been silent for more than twice the " + hours +
			"h check call window on this load."
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenant,
		SourceKind:  watchtower.SourceNoCheckCall,
		SourceID:    move.MoveID.String(),
		Severity:    severity,
		Title:       title,
		Summary:     summary,
		SubjectType: agent.SubjectShipmentMove,
		SubjectID:   move.MoveID,
		Path:        shipmentPath(move.ShipmentID),
		Record:      shipmentRecord(move.ProNumber),
		Facts: []watchtower.Fact{
			fact("Carrier", move.CarrierName),
			factAt("Last contact", move.LastContactAt),
			fact("Window", hours+"h"),
		},
		Trail: []watchtower.TrailEvent{
			{At: move.LastContactAt, Text: "Last heard from " + carrier},
			{At: move.LastContactAt + window, Text: "Check call window closed"},
		},
		OccurredAt: move.LastContactAt + window,
		OwnerID:    move.OwnerID,
	}
}

func DescribeEtaRisk(
	tenant pagination.TenantInfo,
	stop *stopetaports.AttentionStop,
) watchtowerports.WatchtowerItemInput {
	row := stop.ETA
	late := row.Verdict == stopeta.VerdictLate
	severity := watchtower.SeverityWarning
	if late {
		severity = watchtower.SeverityCritical
	}

	place := etaStopPlace(stop)
	title := "ETA at risk"
	summary := "Projected to reach " + place + " with less than " +
		strconv.FormatInt(row.ThresholdMinutes, 10) + " minutes to spare."
	if late {
		title = "Projected late"
		summary = "Projected to reach " + place + " after its window closes."
	}
	if stop.ProNumber != "" {
		title += " on " + stop.ProNumber
	}
	if row.InputStale {
		summary += " The projection is working from old information."
	}

	windowLabel := "Delivery window closes"
	if stop.StopType == shipment.StopTypePickup {
		windowLabel = "Pickup window closes"
	}
	facts := []watchtower.Fact{fact("Stop", place), factAtPtr("Window closes", row.WindowEnd)}
	if row.EstimatedArrival != nil {
		facts = append(facts, factAt("New ETA", *row.EstimatedArrival))
	}
	if row.MilesRemaining > 0 {
		facts = append(
			facts,
			fact("Miles to go", strconv.FormatFloat(row.MilesRemaining, 'f', 0, 64)),
		)
	}
	trail := make([]watchtower.TrailEvent, 0, 2)
	if row.InputAt != nil {
		trail = append(
			trail,
			watchtower.TrailEvent{At: *row.InputAt, Text: "Last position or check call"},
		)
	}
	if reason := strings.TrimSpace(row.Reason); reason != "" {
		trail = append(trail, watchtower.TrailEvent{Text: reason})
	}

	return watchtowerports.WatchtowerItemInput{
		TenantInfo:  tenant,
		SourceKind:  watchtower.SourceEtaAtRisk,
		SourceID:    row.ShipmentID.String(),
		Severity:    severity,
		Title:       title,
		Summary:     summary,
		SubjectType: agent.SubjectShipment,
		SubjectID:   row.ShipmentID,
		Path:        shipmentPath(row.ShipmentID),
		Record:      shipmentRecord(stop.ProNumber),
		Facts:       facts,
		Trail:       trail,
		Deadline:    deadlineAtPtr(row.WindowEnd, windowLabel),
		OccurredAt:  etaRiskOccurredAt(row, late),
		OwnerID:     stop.OwnerID,
	}
}

func etaStopPlace(stop *stopetaports.AttentionStop) string {
	kind := "the " + strings.ToLower(stringutils.FirstNonEmpty(string(stop.StopType), "stop"))
	city := stop.City
	if city != "" && stop.StateCode != "" {
		city += ", " + stop.StateCode
	}
	switch {
	case stop.LocationName != "" && city != "":
		return kind + " at " + stop.LocationName + " in " + city
	case stop.LocationName != "":
		return kind + " at " + stop.LocationName
	case city != "":
		return kind + " in " + city
	default:
		return kind
	}
}

func etaRiskOccurredAt(row *stopeta.StopETA, late bool) int64 {
	if row.WindowEnd == nil {
		return row.ComputedAt
	}
	if late {
		return min(*row.WindowEnd, row.ComputedAt)
	}
	return min(*row.WindowEnd-row.ThresholdMinutes*60, row.ComputedAt)
}
