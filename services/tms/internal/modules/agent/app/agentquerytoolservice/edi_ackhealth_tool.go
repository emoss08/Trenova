package agentquerytoolservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttoolschema"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/edi/app/ediservice"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	paramOverdueAfterSeconds = "overdueAfterSeconds"
	paramWindowHours         = "windowHours"

	defaultAckHealthWindowHours = 24
	maxAckHealthWindowHours     = 30 * 24
	minOverdueAfterSeconds      = int64(60)
	maxOverdueAfterSeconds      = int64(30 * 24 * 60 * 60)
	ackHealthOverdueRows        = 25

	ackHealthFollowUp = "Open one with edi_get_message to read why; edi_retry_message " +
		"sends a dead-lettered or failed one again."
)

var ackedTransactionSets = agenttoolschema.Source("edi.ackedTransactionSet", []string{
	string(edi.TransactionSet204),
	string(edi.TransactionSet210),
	string(edi.TransactionSet214),
	string(edi.TransactionSet990),
	string(edi.TransactionSet997),
	string(edi.TransactionSet999),
})

var ackCountOrder = [...]edi.MessageAcknowledgmentStatus{
	edi.MessageAcknowledgmentStatusPending,
	edi.MessageAcknowledgmentStatusAccepted,
	edi.MessageAcknowledgmentStatusAcceptedWithErrors,
	edi.MessageAcknowledgmentStatusRejected,
	edi.MessageAcknowledgmentStatusFailed,
	edi.MessageAcknowledgmentStatusNotExpected,
}

type ediAckHealthReader interface {
	GetEDIAckHealth(
		ctx context.Context,
		req *ediservice.GetEDIAckHealthRequest,
	) (*ediservice.EDIAckHealth, error)
}

func provideGetEDIAckHealthTool(ediSvc *ediservice.Service) agentports.AgentQueryTool {
	return newGetEDIAckHealthTool(ediSvc)
}

type getEDIAckHealthTool struct {
	health ediAckHealthReader
}

func newGetEDIAckHealthTool(health ediAckHealthReader) agentports.AgentQueryTool {
	return &getEDIAckHealthTool{health: health}
}

func (t *getEDIAckHealthTool) Name() string { return "edi_get_ack_health" }

func (t *getEDIAckHealthTool) SearchTerms() []string {
	return []string{
		"overdue acknowledgment", "missing 997", "missing 999", "unacknowledged",
		"acks overdue", "ack health", "not acknowledged yet", "keeping up with acks",
	}
}

func (t *getEDIAckHealthTool) Description() string {
	return fmt.Sprintf(
		"How partners are acknowledging what was sent: ack status counts overall and per "+
			"partner, overdue messages by partner oldest first, and recent dead letters. One call answers for one partner too: read its row in ackCountsByPartner "+
			"rather than listing its messages, and lead with its answer: acknowledged counts every "+
			"997 or 999 that came back, a rejection included, and awaiting those still pending, "+
			"both since countsSince. A message is overdue "+
			"once its 997 or 999 has been pending %s unless you ask, the age the EDI overview "+
			"uses. Use it for \"what hasn't been acked\" or \"is a partner keeping up\"; "+
			"edi_list_messages lists messages by filter. Each message has its id for "+
			"edi_get_message or edi_retry_message.",
		timeutils.FormatLongDurationMs(ediservice.DefaultOverdueAckAgeSeconds*1000),
	)
}

func (t *getEDIAckHealthTool) ParamSchema() map[string]any {
	return objectSchema(map[string]any{
		paramPartnerID: agenttoolschema.KindID("Only this partner, by id from "+
			"edi_list_partners or a row of this tool.", permission.KindEDIPartner),
		paramTransactionSet: enumParam("Only messages of this transaction set that were "+
			"sent, such as 214 for \"has acme acked our 214s\". 997 or 999 asks about the "+
			"acknowledgments themselves, so it counts every set they answer.",
			ackedTransactionSets),
		paramOverdueAfterSeconds: intParam(fmt.Sprintf(
			"How long an acknowledgment may be pending before it is overdue, in seconds: "+
				"%d unless you ask, between %d and %d.",
			ediservice.DefaultOverdueAckAgeSeconds, minOverdueAfterSeconds, maxOverdueAfterSeconds,
		)),
		paramWindowHours: intParam(fmt.Sprintf(
			"How many hours back the counts by status reach: %d unless you ask, at most %d. "+
				"Overdue messages are counted whenever they were sent.",
			defaultAckHealthWindowHours, maxAckHealthWindowHours,
		)),
	})
}

func (t *getEDIAckHealthTool) Policy() agentports.ToolPolicy {
	return readPolicy(t.Name(), readSpec{
		resource: permission.ResourceEDI,
		reads:    agent.ExternalReadAlways,
		source:   agent.TaintSourceEDI,
		rationale: "Reads how partners have acknowledged what was sent, with the delivery " +
			"errors their systems returned; nothing changes and nothing is sent.",
	})
}

type ackCountRow struct {
	Status string `json:"status"`
	Count  int    `json:"count"`
}

type partnerAckCountsRow struct {
	PartnerID    string        `json:"partnerId"`
	Partner      string        `json:"partner"`
	PartnerCode  string        `json:"partnerCode,omitempty"`
	Acknowledged int           `json:"acknowledged"`
	Awaiting     int           `json:"awaiting"`
	Answer       string        `json:"answer"`
	AckCounts    []ackCountRow `json:"ackCounts"`
}

type overdueAckRow struct {
	ID             string       `json:"id"`
	TransactionSet string       `json:"transactionSet"`
	ShipmentID     string       `json:"shipmentId,omitempty"`
	InvoiceID      string       `json:"invoiceId,omitempty"`
	ControlNumber  string       `json:"transactionControlNumber,omitempty"`
	DeliveryStatus string       `json:"deliveryStatus,omitempty"`
	GeneratedAt    optionalDate `json:"generatedAt"`
	SentAt         optionalDate `json:"sentAt"`
	Waiting        string       `json:"waiting"`
}

type overduePartnerRow struct {
	PartnerID         string          `json:"partnerId"`
	Partner           string          `json:"partner"`
	PartnerCode       string          `json:"partnerCode,omitempty"`
	OverdueCount      int             `json:"overdueCount"`
	OldestGeneratedAt optionalDate    `json:"oldestGeneratedAt"`
	OldestWaiting     string          `json:"oldestWaiting"`
	Messages          []overdueAckRow `json:"messages"`
	NotListed         int             `json:"notListed,omitempty"`
}

type ediAckHealthView struct {
	Title               string                `json:"title"`
	PartnerID           string                `json:"partnerId,omitempty"`
	CountsSince         optionalDate          `json:"countsSince"`
	OverdueAfterSeconds int64                 `json:"overdueAfterSeconds"`
	OverdueAfter        string                `json:"overdueAfter"`
	Acknowledged        int                   `json:"acknowledged"`
	Awaiting            int                   `json:"awaiting"`
	AckCounts           []ackCountRow         `json:"ackCounts"`
	AckCountsByPartner  []partnerAckCountsRow `json:"ackCountsByPartner"`
	OverdueCount        int                   `json:"overdueCount"`
	OverdueByPartner    []overduePartnerRow   `json:"overdueByPartner"`
	OverdueNotListed    int                   `json:"overdueNotListed,omitempty"`
	RecentDeadLettered  []ediMessageListRow   `json:"recentDeadLettered"`
	FollowUp            string                `json:"followUp"`
	Note                string                `json:"note,omitempty"`

	outside []agent.RecordRef
}

func (v *ediAckHealthView) TaintedRecords() []agent.RecordRef { return v.outside }

func (t *getEDIAckHealthTool) Query(
	ctx context.Context,
	params *agentports.QueryToolParams,
) (any, error) {
	if err := guardQuery(params); err != nil {
		return nil, err
	}

	partnerID, err := optionalID(params.Params, paramPartnerID)
	if err != nil {
		return nil, err
	}
	set, err := validEnum(params.Params, paramTransactionSet, ackedTransactionSets)
	if err != nil {
		return nil, err
	}
	acknowledgment := edi.TransactionSet(set).IsAcknowledgment()
	if acknowledgment {
		set = ""
	}
	overdueAge, err := optionalUnixSeconds(params.Params, paramOverdueAfterSeconds)
	if err != nil {
		return nil, err
	}
	if _, given := params.Params[paramOverdueAfterSeconds]; given &&
		(overdueAge < minOverdueAfterSeconds || overdueAge > maxOverdueAfterSeconds) {
		return nil, fmt.Errorf("parameter %q must be between %d and %d seconds",
			paramOverdueAfterSeconds, minOverdueAfterSeconds, maxOverdueAfterSeconds)
	}
	windowHours := defaultAckHealthWindowHours
	if _, given := params.Params[paramWindowHours]; given {
		windowHours = optionalInt(params.Params, paramWindowHours, 0)
		if windowHours < 1 || windowHours > maxAckHealthWindowHours {
			return nil, fmt.Errorf("parameter %q must be between 1 and %d hours",
				paramWindowHours, maxAckHealthWindowHours)
		}
	}

	clk := clockFor(params)
	now := clk.Instant()
	health, err := t.health.GetEDIAckHealth(ctx, &ediservice.GetEDIAckHealthRequest{
		TenantInfo:     tenantOf(params),
		PartnerID:      partnerID,
		TransactionSet: edi.TransactionSet(set),
		Since:          now - int64(windowHours)*60*60,
		OverdueAckAge:  overdueAge,
		OverdueLimit:   ackHealthOverdueRows,
	})
	if err != nil {
		return nil, err
	}

	view := &ediAckHealthView{
		Title:               "EDI acknowledgment health",
		PartnerID:           pulidString(partnerID),
		CountsSince:         recordedDate(health.Since),
		OverdueAfterSeconds: health.OverdueAckAge,
		OverdueAfter:        timeutils.FormatLongDurationMs(health.OverdueAckAge * 1000),
		Acknowledged:        acknowledged(health.AckStatusCounts),
		Awaiting:            health.AckStatusCounts[edi.MessageAcknowledgmentStatusPending],
		AckCounts:           ackCounts(health.AckStatusCounts),
		AckCountsByPartner:  partnerAckCounts(health.AckCountsByPartner),
		OverdueCount:        health.OverdueAckCount,
		OverdueByPartner:    make([]overduePartnerRow, 0, len(health.OverdueByPartner)),
		OverdueNotListed:    health.OverdueOmitted,
		RecentDeadLettered:  make([]ediMessageListRow, 0, len(health.RecentDeadLettered)),
		FollowUp:            ackHealthFollowUp,
	}
	for _, group := range health.OverdueByPartner {
		if group == nil {
			continue
		}
		view.OverdueByPartner = append(view.OverdueByPartner, view.overdueGroup(group, now))
	}
	for _, message := range health.RecentDeadLettered {
		if message == nil {
			continue
		}
		view.RecentDeadLettered = append(view.RecentDeadLettered, ediMessageListRowFrom(message))
		view.outside = append(view.outside, ediMessageRef(message))
	}
	if acknowledgment {
		view.Note = "997s and 999s are the acknowledgments themselves, so these counts cover " +
			"every transaction set they answer."
	}
	if view.OverdueCount == 0 {
		view.Note = strings.TrimSpace(view.Note + " " + fmt.Sprintf(
			"Nothing is overdue: no outbound message has waited more than %s for its "+
				"acknowledgment.", view.OverdueAfter))
	}

	return view, nil
}

func (v *ediAckHealthView) overdueGroup(
	group *ediservice.EDIOverduePartner,
	now int64,
) overduePartnerRow {
	row := overduePartnerRow{
		PartnerID:         group.PartnerID.String(),
		Partner:           group.PartnerName,
		PartnerCode:       group.PartnerCode,
		OverdueCount:      group.OverdueCount,
		OldestGeneratedAt: recordedDate(group.OldestGeneratedAt),
		OldestWaiting:     timeutils.FormatLongDurationMs((now - group.OldestGeneratedAt) * 1000),
		Messages:          make([]overdueAckRow, 0, len(group.Messages)),
		NotListed:         max(group.OverdueCount-len(group.Messages), 0),
	}
	for _, message := range group.Messages {
		if message == nil {
			continue
		}
		row.Messages = append(row.Messages, overdueAckRow{
			ID:             message.ID.String(),
			TransactionSet: string(message.TransactionSet),
			ShipmentID:     pulidString(message.ShipmentID),
			InvoiceID:      pulidString(message.InvoiceID),
			ControlNumber:  message.TransactionControlNumber,
			DeliveryStatus: string(message.DeliveryStatus),
			GeneratedAt:    recordedDate(message.GeneratedAt),
			SentAt:         pointerDateOr(message.DeliverySentAt, absentNotSent),
			Waiting:        timeutils.FormatLongDurationMs((now - message.GeneratedAt) * 1000),
		})
		v.outside = append(v.outside, ediMessageRef(message))
	}

	return row
}

func ackCounts(counts map[edi.MessageAcknowledgmentStatus]int) []ackCountRow {
	rows := make([]ackCountRow, 0, len(ackCountOrder))
	for _, status := range ackCountOrder {
		rows = append(rows, ackCountRow{Status: string(status), Count: counts[status]})
	}

	return rows
}

func acknowledged(counts map[edi.MessageAcknowledgmentStatus]int) int {
	total := 0
	for status, count := range counts {
		if status.IsReceived() {
			total += count
		}
	}

	return total
}

func ackAnswer(counts map[edi.MessageAcknowledgmentStatus]int) string {
	received := acknowledged(counts)
	awaiting := counts[edi.MessageAcknowledgmentStatusPending]
	failed := counts[edi.MessageAcknowledgmentStatusFailed]
	expected := received + awaiting + failed
	if expected == 0 {
		return "nothing sent in this window expects an acknowledgment"
	}

	outcomes := make([]string, 0, 3)
	for _, outcome := range []struct {
		status edi.MessageAcknowledgmentStatus
		word   string
	}{
		{edi.MessageAcknowledgmentStatusAccepted, "accepted"},
		{edi.MessageAcknowledgmentStatusAcceptedWithErrors, "accepted with errors"},
		{edi.MessageAcknowledgmentStatusRejected, "rejected"},
	} {
		if count := counts[outcome.status]; count > 0 {
			outcomes = append(outcomes, fmt.Sprintf("%d %s", count, outcome.word))
		}
	}
	answer := fmt.Sprintf("%d of %d came back with a 997 or 999", received, expected)
	if len(outcomes) > 0 {
		answer += " (" + strings.Join(outcomes, ", ") + ")"
	}
	if awaiting > 0 {
		answer += fmt.Sprintf("; %d still awaiting one", awaiting)
	}
	if failed > 0 {
		answer += fmt.Sprintf("; %d marked Failed", failed)
	}

	return answer
}

func partnerAckCounts(groups []*ediservice.EDIPartnerAckCounts) []partnerAckCountsRow {
	rows := make([]partnerAckCountsRow, 0, len(groups))
	for _, group := range groups {
		if group == nil {
			continue
		}
		row := partnerAckCountsRow{
			PartnerID:    group.PartnerID.String(),
			Partner:      group.PartnerName,
			PartnerCode:  group.PartnerCode,
			Acknowledged: acknowledged(group.Counts),
			Awaiting:     group.Counts[edi.MessageAcknowledgmentStatusPending],
			Answer:       ackAnswer(group.Counts),
			AckCounts:    make([]ackCountRow, 0, len(group.Counts)),
		}
		for _, status := range ackCountOrder {
			if count := group.Counts[status]; count > 0 {
				row.AckCounts = append(
					row.AckCounts,
					ackCountRow{Status: string(status), Count: count},
				)
			}
		}
		rows = append(rows, row)
	}

	return rows
}

func ediMessageRef(message *edi.EDIMessage) agent.RecordRef {
	return agent.RecordRef{EntityType: ediMessageEntity, ID: message.ID.String()}
}
