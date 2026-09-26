package agentsubjectservice

import (
	"context"
	"fmt"
	"math"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/bankreceipt"
	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/core/domain/insight"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/shipmenttracking"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxSubjectNotesChars = 12000
	maxInsightLabelChars = 120
)

type Params struct {
	fx.In

	Logger            *zap.Logger
	BillingQueue      serviceports.BillingQueueService
	Shipments         serviceports.ShipmentService
	Console           repositories.DispatchConsoleRepository         `optional:"true"`
	Content           serviceports.DocumentContentService            `optional:"true"`
	Insights          repositories.InsightRepository                 `optional:"true"`
	BankReceipts      serviceports.BankReceiptService                `optional:"true"`
	WorkItems         repositories.BankReceiptWorkItemRepository     `optional:"true"`
	Occurrences       repositories.DetentionOccurrenceRepository     `optional:"true"`
	Workers           repositories.WorkerRepository                  `optional:"true"`
	Credentials       repositories.WorkerCredentialRepository        `optional:"true"`
	CarrierIntel      repositories.CarrierIntelEventRepository       `optional:"true"`
	EDIFiles          repositories.EDIInboundFileRepository          `optional:"true"`
	Inbound           repositories.InboundMessageRepository          `optional:"true"`
	Reports           repositories.ReportDefinitionRepository        `optional:"true"`
	Dashboards        repositories.ReportDashboardRepository         `optional:"true"`
	Accounting        repositories.AccountingConnectionRepository    `optional:"true"`
	SyncRecords       repositories.AccountingSyncRecordRepository    `optional:"true"`
	AccountingInbound repositories.AccountingInboundChangeRepository `optional:"true"`
	AccountingDrift   repositories.AccountingDriftFindingRepository  `optional:"true"`
	Dispatch          repositories.DispatchControlRepository         `optional:"true"`
	Formulas          repositories.FormulaTemplateRepository         `optional:"true"`
}

// Service describes the record an agent run or a conversation is about, so
// a run woken by an event and a thread opened from a page both start with
// the same picture of their subject in front of the model.
type Service struct {
	content           serviceports.DocumentContentService
	billingQueue      serviceports.BillingQueueService
	shipments         serviceports.ShipmentService
	console           repositories.DispatchConsoleRepository
	insights          repositories.InsightRepository
	receipts          serviceports.BankReceiptService
	workItems         repositories.BankReceiptWorkItemRepository
	occurrences       repositories.DetentionOccurrenceRepository
	workers           repositories.WorkerRepository
	credentials       repositories.WorkerCredentialRepository
	carrierIntel      repositories.CarrierIntelEventRepository
	ediFiles          repositories.EDIInboundFileRepository
	inbound           repositories.InboundMessageRepository
	reports           repositories.ReportDefinitionRepository
	dashboards        repositories.ReportDashboardRepository
	accounting        repositories.AccountingConnectionRepository
	syncRecords       repositories.AccountingSyncRecordRepository
	accountingInbound repositories.AccountingInboundChangeRepository
	accountingDrift   repositories.AccountingDriftFindingRepository
	dispatch          repositories.DispatchControlRepository
	formulas          repositories.FormulaTemplateRepository
	logger            *zap.Logger
}

func New(p Params) serviceports.AgentSubjectDescriber {
	return &Service{
		content:           p.Content,
		billingQueue:      p.BillingQueue,
		shipments:         p.Shipments,
		console:           p.Console,
		insights:          p.Insights,
		receipts:          p.BankReceipts,
		workItems:         p.WorkItems,
		occurrences:       p.Occurrences,
		workers:           p.Workers,
		credentials:       p.Credentials,
		carrierIntel:      p.CarrierIntel,
		ediFiles:          p.EDIFiles,
		accountingInbound: p.AccountingInbound,
		accountingDrift:   p.AccountingDrift,
		reports:           p.Reports,
		dashboards:        p.Dashboards,
		accounting:        p.Accounting,
		syncRecords:       p.SyncRecords,
		inbound:           p.Inbound,
		dispatch:          p.Dispatch,
		formulas:          p.Formulas,
		logger:            p.Logger.Named("service.agentsubject"),
	}
}

func (s *Service) Describe(
	ctx context.Context,
	tenant pagination.TenantInfo,
	subjectType agent.SubjectType,
	subjectID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	switch subjectType {
	case agent.SubjectBillingQueueItem:
		return s.billingQueueItem(ctx, tenant, subjectID)
	case agent.SubjectShipmentMove:
		return s.shipmentMove(ctx, tenant, subjectID)
	case agent.SubjectShipment:
		return s.shipment(ctx, tenant, subjectID)
	case agent.SubjectDocument:
		return s.document(ctx, tenant, subjectID)
	case agent.SubjectInsight:
		return s.insight(ctx, tenant, subjectID)
	case agent.SubjectBankReceipt:
		return s.bankReceipt(ctx, tenant, subjectID)
	case agent.SubjectDetentionOccurrence:
		return s.detentionOccurrence(ctx, tenant, subjectID)
	case agent.SubjectWorker:
		return s.worker(ctx, tenant, subjectID)
	case agent.SubjectCarrierIntelEvent:
		return s.carrierIntelEvent(ctx, tenant, subjectID)
	case agent.SubjectEDIInboundFile:
		return s.ediInboundFile(ctx, tenant, subjectID)
	case agent.SubjectInboundMessage:
		return s.inboundMessage(ctx, tenant, subjectID)
	case agent.SubjectReport:
		return s.report(ctx, tenant, subjectID)
	case agent.SubjectDashboard:
		return s.dashboard(ctx, tenant, subjectID)
	case agent.SubjectAccountingConnection:
		return s.accountingConnection(ctx, tenant, subjectID)
	case agent.SubjectAccountingSyncRecord:
		return s.accountingSyncRecord(ctx, tenant, subjectID)
	case agent.SubjectAccountingInbound:
		return s.accountingInboundChange(ctx, tenant, subjectID)
	case agent.SubjectAccountingDrift:
		return s.accountingDriftFinding(ctx, tenant, subjectID)
	case agent.SubjectFormulaTemplate:
		return s.formulaTemplate(ctx, tenant, subjectID)
	case agent.SubjectOrganization, "":
		return nil, nil
	default:
		return &agentdefinition.RuntimeSubject{
			Type:  subjectType,
			ID:    subjectID.String(),
			Label: string(subjectType),
		}, nil
	}
}

func (s *Service) billingQueueItem(
	ctx context.Context,
	tenant pagination.TenantInfo,
	itemID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	item, err := s.billingQueue.GetByID(ctx, &repositories.GetBillingQueueItemByIDRequest{
		TenantInfo:            tenant,
		ItemID:                itemID,
		ExpandShipmentDetails: true,
	})
	if err != nil {
		return nil, fmt.Errorf("load billing queue item: %w", err)
	}

	sections := map[string]any{"billingQueueItem": item}
	if item.ShipmentID.IsNotNil() && s.shipments != nil {
		readiness, rErr := s.shipments.GetBillingReadiness(ctx, item.ShipmentID, tenant)
		if rErr != nil {
			s.logger.Warn("agent subject: billing readiness unavailable", zap.Error(rErr))
		} else {
			sections["billingReadiness"] = map[string]any{
				"validationFailures":  readiness.ValidationFailures,
				"missingRequirements": readiness.MissingRequirements,
				"warnings":            readiness.Warnings,
				"serviceFailures":     readiness.ServiceFailureContext,
			}
		}
	}
	notes := strings.TrimSpace(
		strings.Join([]string{item.ReviewNotes, item.ExceptionNotes, item.CancelReason}, "\n"),
	)
	if notes != "" {
		sections["notes"] = notes
	}

	label := "Billing queue item"
	if item.ShipmentID.IsNotNil() {
		label = "Billing queue item for shipment " + item.ShipmentID.String()
	}

	return &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectBillingQueueItem,
		ID:    itemID.String(),
		Label: label,
		Notes: marshalNotes(sections),
	}, nil
}

func (s *Service) shipmentMove(
	ctx context.Context,
	tenant pagination.TenantInfo,
	moveID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	if s.console == nil {
		return &agentdefinition.RuntimeSubject{
			Type:  agent.SubjectShipmentMove,
			ID:    moveID.String(),
			Label: "Shipment move",
		}, nil
	}

	moves, err := s.console.ListBoardMoves(ctx, &repositories.DispatchBoardFilter{
		TenantInfo: tenant,
		MoveIDs:    []pulid.ID{moveID},
		Limit:      1,
	})
	if err != nil {
		return nil, fmt.Errorf("load shipment move: %w", err)
	}

	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectShipmentMove,
		ID:    moveID.String(),
		Label: "Shipment move",
	}
	if len(moves) > 0 {
		subject.Label = "Shipment move for PRO " + moves[0].ProNumber
		subject.Notes = marshalNotes(moveNotes{
			BoardMove: moves[0],
			Coverage:  s.moveCoverage(ctx, tenant, moves[0]),
		})
	}

	return subject, nil
}

type moveNotes struct {
	*repositories.BoardMove

	Coverage *moveCoverage `json:"coverage,omitempty"`
}

type moveCoverage struct {
	WindowHours    int16 `json:"windowHours"`
	StartsAt       int64 `json:"startsAt"`
	StartsInWindow bool  `json:"startsInsideWindow"`
}

func (s *Service) moveCoverage(
	ctx context.Context,
	tenant pagination.TenantInfo,
	move *repositories.BoardMove,
) *moveCoverage {
	if s.dispatch == nil || move == nil || move.OriginWindowStart <= 0 {
		return nil
	}

	control, err := s.dispatch.GetByOrgID(ctx, repositories.GetDispatchControlRequest{
		TenantInfo: tenant,
	})
	if err != nil {
		s.logger.Warn("shipment move subject: coverage window unavailable", zap.Error(err))

		return nil
	}

	return &moveCoverage{
		WindowHours: control.CoverageWindowHours(),
		StartsAt:    move.OriginWindowStart,
		StartsInWindow: control.StartsInsideCoverageWindow(
			move.OriginWindowStart,
			timeutils.NowUnix(),
		),
	}
}

func marshalNotes(value any) string {
	encoded, err := sonic.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", value)
	}
	notes := string(encoded)
	if len(notes) > maxSubjectNotesChars {
		return notes[:maxSubjectNotesChars] + "\n…(truncated)"
	}

	return notes
}

// shipment describes a shipment the way the tracking tool does: the stops
// with their windows and lateness, and who is on each move. A run woken by a
// service failure or a new shipment starts with that in front of it rather
// than a bare id.
func (s *Service) shipment(
	ctx context.Context,
	tenant pagination.TenantInfo,
	shipmentID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectShipment,
		ID:    shipmentID.String(),
		Label: "Shipment",
	}
	if s.shipments == nil {
		return subject, nil
	}

	entity, err := s.shipments.Get(ctx, &repositories.GetShipmentByIDRequest{
		ID:         shipmentID,
		TenantInfo: tenant,
		ShipmentOptions: repositories.ShipmentOptions{
			ExpandShipmentDetails: true,
			IncludeCustomer:       true,
		},
	})
	if err != nil {
		return nil, fmt.Errorf("load shipment: %w", err)
	}

	assignments := make(map[pulid.ID]*repositories.BoardMove, len(entity.Moves))
	if s.console != nil && len(entity.Moves) > 0 {
		moveIDs := make([]pulid.ID, 0, len(entity.Moves))
		for _, move := range entity.Moves {
			if move != nil {
				moveIDs = append(moveIDs, move.ID)
			}
		}
		moves, listErr := s.console.ListBoardMoves(ctx, &repositories.DispatchBoardFilter{
			TenantInfo:     tenant,
			MoveIDs:        moveIDs,
			IncludeCovered: true,
			Limit:          len(moveIDs),
		})
		if listErr != nil {
			s.logger.Warn("shipment subject: assignments unavailable", zap.Error(listErr))
		}
		for _, move := range moves {
			if move != nil {
				assignments[move.MoveID] = move
			}
		}
	}

	// Times render in UTC with the zone spelled out; the run's own context
	// carries the organization's zone for anything the model has to say back.
	snapshot := shipmenttracking.Build(shipmenttracking.Input{
		Shipment:    entity,
		Assignments: assignments,
		Now:         timeutils.NowUnix(),
		Timezone:    "UTC",
	})
	subject.Label = "Shipment PRO " + entity.ProNumber
	subject.Notes = marshalNotes(snapshot)
	if entity.EntryMethod == shipment.EntryMethodEDI {
		subject.OutsideAuthored = agent.TaintSourceEDI
	}

	return subject, nil
}

// document describes an uploaded document by what intelligence read from
// it: the draft's status and confidence, every field with its confidence,
// the stops and what is missing. A run woken by document.extracted starts
// with the whole draft in front of it, and get_shipment_draft is there for
// a second look after a tool call has moved the conversation on.
func (s *Service) document(
	ctx context.Context,
	tenant pagination.TenantInfo,
	documentID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectDocument,
		ID:    documentID.String(),
		Label: "Document",
	}
	if s.content == nil {
		return subject, nil
	}

	draft, err := s.content.GetShipmentDraft(ctx, documentID, tenant)
	if err != nil {
		return nil, fmt.Errorf("load document draft: %w", err)
	}

	if draft.DocumentKind != "" {
		subject.Label = "Document (" + stringutils.HumanizeSnakeCase(draft.DocumentKind) + ")"
	}
	notes := map[string]any{
		"status":     draft.Status,
		"confidence": draft.Confidence,
		"draft":      draft.DraftData,
	}
	if draft.AttachedShipmentID != nil && draft.AttachedShipmentID.IsNotNil() {
		notes["attachedShipmentId"] = draft.AttachedShipmentID.String()
		notes["warning"] = "A shipment was already created from this document; do not create another."
	}
	if draft.FailureMessage != "" {
		notes["failureMessage"] = draft.FailureMessage
	}
	subject.Notes = marshalNotes(notes)

	return subject, nil
}

// insight describes a finding by what the detector measured and what it
// suggests, so a run woken by insight.detected starts with the numbers, the
// records behind them and the recommendation, and get_insight is there for
// the trend when the run wants it.
func (s *Service) insight(
	ctx context.Context,
	tenant pagination.TenantInfo,
	insightID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectInsight,
		ID:    insightID.String(),
		Label: "Insight",
	}
	if s.insights == nil {
		return subject, nil
	}

	found, err := s.insights.GetByID(ctx, repositories.GetInsightByIDRequest{
		ID:         insightID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("load insight: %w", err)
	}

	subject.Label = "Insight: " + stringutils.TruncateRunes(found.Headline, maxInsightLabelChars)
	notes := map[string]any{
		"category":       found.Category,
		"severity":       found.Severity,
		"status":         found.Status,
		"subject":        found.Subject,
		"headline":       found.Headline,
		"narrative":      found.Narrative,
		"recommendation": found.Recommendation,
		"metrics":        found.Metrics,
		"links":          found.Links,
		"windowStart":    found.WindowStart,
		"windowEnd":      found.WindowEnd,
		"detectedAt":     found.DetectedAt,
	}
	if found.Status != insight.StatusActive {
		notes["warning"] = "This finding is no longer active; do not act on it as if it were."
	}
	subject.Notes = marshalNotes(notes)

	return subject, nil
}

// bankReceipt describes a receipt the way the reconciliation page does: the
// money, the bank's reference and memo, why it was not matched, the scored
// candidate payments and the queue entry. A run woken by
// bank_receipt.exception starts with all of that in front of it, and
// get_bank_receipt is there for a second look once a tool call has moved
// the conversation on.
func (s *Service) bankReceipt(
	ctx context.Context,
	tenant pagination.TenantInfo,
	receiptID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectBankReceipt,
		ID:    receiptID.String(),
		Label: "Bank receipt",
	}
	if s.receipts == nil {
		return subject, nil
	}

	req := &serviceports.GetBankReceiptRequest{ReceiptID: receiptID, TenantInfo: tenant}
	receipt, err := s.receipts.Get(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("load bank receipt: %w", err)
	}

	amount := money.DecimalFromMinor(receipt.AmountMinor).StringFixed(2)
	subject.Label = "Bank receipt " + amount
	if receipt.ReferenceNumber != "" {
		subject.Label += " ref " + receipt.ReferenceNumber
	}

	notes := map[string]any{
		"status":          receipt.Status,
		"amount":          amount,
		"receiptDate":     receipt.ReceiptDate,
		"referenceNumber": receipt.ReferenceNumber,
		"memo":            receipt.Memo,
		"exceptionReason": receipt.ExceptionReason,
	}
	if receipt.Status == bankreceipt.StatusMatched {
		notes["warning"] = "This receipt is already matched; do not match or post a payment for it again."
	} else {
		suggestions, sErr := s.receipts.SuggestMatches(ctx, req)
		if sErr != nil {
			s.logger.Warn("bank receipt subject: suggestions unavailable", zap.Error(sErr))
		} else {
			candidates := make([]map[string]any, 0, len(suggestions))
			for _, suggestion := range suggestions {
				if suggestion == nil {
					continue
				}
				candidates = append(candidates, map[string]any{
					"customerPaymentId": suggestion.CustomerPaymentID.String(),
					"customerId":        suggestion.CustomerID.String(),
					"referenceNumber":   suggestion.ReferenceNumber,
					"amount": money.DecimalFromMinor(suggestion.AmountMinor).
						StringFixed(2),
					"score":  suggestion.Score,
					"reason": suggestion.Reason,
				})
			}
			notes["candidatePayments"] = candidates
		}
	}

	if s.workItems != nil {
		item, iErr := s.workItems.GetActiveByReceiptID(ctx, tenant, receiptID)
		switch {
		case iErr != nil && !errortypes.IsNotFoundError(iErr):
			s.logger.Warn("bank receipt subject: work item unavailable", zap.Error(iErr))
		case item != nil:
			notes["workItem"] = map[string]any{
				"id":               item.ID.String(),
				"status":           item.Status,
				"assignedToUserId": item.AssignedToUserID.String(),
			}
		}
	}
	subject.Notes = marshalNotes(notes)

	return subject, nil
}

// detentionOccurrence describes one clock at one stop: the times that
// decide the charge, what has already gone to the customer, and whether
// anything is holding the notice back. A run woken by
// detention.occurrence_opened or detention.notice_due starts with the whole
// countdown in front of it rather than an id.
func (s *Service) detentionOccurrence(
	ctx context.Context,
	tenant pagination.TenantInfo,
	occurrenceID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectDetentionOccurrence,
		ID:    occurrenceID.String(),
		Label: "Detention occurrence",
	}
	if s.occurrences == nil {
		return subject, nil
	}

	occurrence, err := s.occurrences.GetByID(ctx, &repositories.GetDetentionOccurrenceByIDRequest{
		OccurrenceID:   occurrenceID,
		TenantInfo:     tenant,
		IncludeNotices: true,
	})
	if err != nil {
		return nil, fmt.Errorf("load detention occurrence: %w", err)
	}

	where := stringutils.FirstNonEmpty(occurrence.LocationName, "a stop")
	subject.Label = "Detention at " + where
	if occurrence.ShipmentProNumber != "" {
		subject.Label += " on " + occurrence.ShipmentProNumber
	}

	notes := map[string]any{
		"status":             occurrence.Status,
		"isOpen":             occurrence.IsOpen,
		"customerName":       occurrence.CustomerName,
		"shipmentId":         occurrence.ShipmentID.String(),
		"clockStartAt":       occurrence.ClockStartAt,
		"clockStopAt":        occurrence.ClockStopAt,
		"freeTimeExpiresAt":  occurrence.FreeTimeExpiresAt,
		"noticeDueAt":        occurrence.NoticeDueAt,
		"noticeDeadlineAt":   occurrence.NoticeDeadlineAt,
		"noticeSentAt":       occurrence.NoticeSentAt,
		"notificationStatus": occurrence.NotificationStatus,
		"billableMinutes":    occurrence.BillableMinutes,
		"billableAmount":     occurrence.BillableAmount.StringFixed(2),
		"currency":           occurrence.Currency,
		"requiresApproval":   occurrence.RequiresApproval,
		"suppressedByGate":   occurrence.SuppressedByGate,
	}
	switch {
	case !occurrence.IsOpen:
		notes["warning"] = "This clock has stopped; do not act as if it were still running."
	case occurrence.IsFrozen():
		notes["warning"] = "This occurrence is frozen and its figures will not change."
	case occurrence.NoticeSentAt != nil:
		notes["warning"] = "A notice has already gone to the customer; do not send a second one."
	}
	subject.Notes = marshalNotes(notes)

	return subject, nil
}

// worker describes a driver by what would stop them driving: the papers on
// file with their expiry, nearest first. A run woken by
// worker_credential.expiring starts with every credential in front of it,
// because one driver with three papers due is one renewal packet.
func (s *Service) worker(
	ctx context.Context,
	tenant pagination.TenantInfo,
	workerID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectWorker,
		ID:    workerID.String(),
		Label: "Driver",
	}
	if s.workers == nil {
		return subject, nil
	}

	entity, err := s.workers.GetByID(ctx, repositories.GetWorkerByIDRequest{
		ID:             workerID,
		TenantInfo:     tenant,
		IncludeProfile: true,
	})
	if err != nil {
		return nil, fmt.Errorf("load worker: %w", err)
	}

	subject.Label = "Driver " + strings.TrimSpace(entity.FirstName+" "+entity.LastName)
	notes := map[string]any{
		"status":     entity.Status,
		"type":       entity.Type,
		"firstName":  entity.FirstName,
		"lastName":   entity.LastName,
		"driverType": entity.DriverType,
	}
	if entity.Profile != nil {
		// The licence number is Confidential, and nothing Confidential is
		// sent to a model, whoever is asking: its expiry is what the work
		// turns on.
		notes["licenceExpiry"] = entity.Profile.LicenseExpiry
		notes["endorsement"] = entity.Profile.Endorsement
		notes["physicalDueDate"] = entity.Profile.PhysicalDueDate
		notes["mvrDueDate"] = entity.Profile.MVRDueDate
	}
	if s.credentials != nil {
		notes["credentials"] = s.workerCredentials(ctx, tenant, workerID)
	}
	subject.Notes = marshalNotes(notes)

	return subject, nil
}

// workerCredentials lists the driver's papers nearest expiry first, so the
// model reads the urgent one before it runs out of context.
func (s *Service) workerCredentials(
	ctx context.Context,
	tenant pagination.TenantInfo,
	workerID pulid.ID,
) []map[string]any {
	credentials, err := s.credentials.ListForWorker(ctx, &repositories.ListWorkerCredentialsRequest{
		TenantInfo:  tenant,
		WorkerID:    workerID,
		IncludeType: true,
	})
	if err != nil {
		s.logger.Warn("worker subject: credentials unavailable", zap.Error(err))
		return nil
	}

	now := timeutils.NowUnix()
	rows := make([]map[string]any, 0, len(credentials))
	for _, credential := range credentials {
		if credential == nil {
			continue
		}
		row := map[string]any{
			"id":     credential.ID.String(),
			"status": credential.Status,
			"number": credential.Number,
		}
		if credential.CredentialType != nil {
			row["type"] = credential.CredentialType.Name
			row["required"] = credential.CredentialType.IsRequired
		}
		if credential.ExpiresAt != nil {
			row["expiresAt"] = *credential.ExpiresAt
			row["daysUntilExpiry"] = worker.DaysUntil(*credential.ExpiresAt, now)
		}
		rows = append(rows, row)
	}
	slices.SortFunc(rows, func(a, b map[string]any) int {
		return credentialUrgency(a) - credentialUrgency(b)
	})

	return rows
}

// credentialUrgency orders a credential by how soon it expires; one with no
// expiry sorts last, because nothing about it is due.
func credentialUrgency(row map[string]any) int {
	days, ok := row["daysUntilExpiry"].(int64)
	if !ok {
		return math.MaxInt32
	}

	return int(days)
}

// carrierIntelEvent describes what monitoring found about a carrier and
// what it does to that carrier's eligibility, so a run woken by
// carrier_intel.event_opened knows whether the finding blocks a tender
// before it proposes anything.
func (s *Service) carrierIntelEvent(
	ctx context.Context,
	tenant pagination.TenantInfo,
	eventID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectCarrierIntelEvent,
		ID:    eventID.String(),
		Label: "Carrier finding",
	}
	if s.carrierIntel == nil {
		return subject, nil
	}

	events, err := s.carrierIntel.GetByIDs(ctx, tenant, []pulid.ID{eventID})
	if err != nil {
		return nil, fmt.Errorf("load carrier intel event: %w", err)
	}
	if len(events) == 0 || events[0] == nil {
		return subject, nil
	}

	event := events[0]
	name := stringutils.FirstNonEmpty(event.SubjectName, "DOT "+event.DOTNumber)
	subject.Label = name + ": " + stringutils.HumanizeSnakeCase(string(event.Category))

	notes := map[string]any{
		"status":      event.Status,
		"severity":    event.Severity,
		"source":      event.Source,
		"category":    event.Category,
		"ruleCode":    event.RuleCode,
		"action":      event.Action,
		"summary":     event.Summary,
		"subjectName": event.SubjectName,
		"dotNumber":   event.DOTNumber,
		"detectedAt":  event.DetectedAt,
	}
	if definition, ok := carrierintel.RuleByCode(event.RuleCode); ok {
		notes["gateRelevant"] = definition.GateRelevant
	}
	if event.CarrierID.IsNotNil() {
		notes["carrierId"] = event.CarrierID.String()
	}
	if event.Status.IsClosed() {
		notes["warning"] = "This finding is already closed; do not act on it as if it were open."
	}
	subject.Notes = marshalNotes(notes)

	return subject, nil
}

// ediInboundFile describes a file that could not be turned into shipments
// or updates: why it stopped and what it was meant to be, so a run woken by
// edi.file_quarantined can say whether a person has to look at it.
func (s *Service) ediInboundFile(
	ctx context.Context,
	tenant pagination.TenantInfo,
	fileID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectEDIInboundFile,
		ID:    fileID.String(),
		Label: "EDI inbound file",
	}
	if s.ediFiles == nil {
		return subject, nil
	}

	file, err := s.ediFiles.GetInboundFileByID(ctx, repositories.GetEDIInboundFileByIDRequest{
		ID:              fileID,
		TenantInfo:      tenant,
		IncludeMessages: true,
	})
	if err != nil {
		return nil, fmt.Errorf("load edi inbound file: %w", err)
	}

	subject.Label = "EDI file " + stringutils.FirstNonEmpty(file.FileName, file.ID.String())
	notes := map[string]any{
		"status":            file.Status,
		"fileName":          file.FileName,
		"failureReason":     file.FailureReason,
		"receivedAt":        file.ReceivedAt,
		"processedAt":       file.ProcessedAt,
		"transactionsFound": len(file.Messages),
	}
	if file.EDIPartnerID.IsNotNil() {
		notes["ediPartnerId"] = file.EDIPartnerID.String()
	}
	if file.Status != edi.InboundFileStatusQuarantined {
		notes["warning"] = "This file is not held back in quarantine; do not act as if it " +
			"were. Any load tenders it carried are listed by list_edi_transfers for this file."
	}
	subject.Notes = marshalNotes(notes)

	return subject, nil
}

// inboundMessage describes a piece of mail that arrived on a monitored
// address: what it was read as, what it was matched to and why, and what its
// attachments turned out to be — so a run woken by inbound_message.classified
// can answer it without opening the mailbox itself.
//
// The body is deliberately included as a quoted value rather than as
// instruction. It is whatever a sender chose to write, so it is evidence about
// the message, never direction to the run reading it.
func (s *Service) inboundMessage(
	ctx context.Context,
	tenant pagination.TenantInfo,
	messageID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectInboundMessage,
		ID:    messageID.String(),
		Label: "Inbound message",
	}
	if s.inbound == nil {
		return subject, nil
	}

	message, err := s.inbound.GetByID(ctx, repositories.GetInboundMessageByIDRequest{
		ID:                 messageID,
		TenantInfo:         tenant,
		IncludeAttachments: true,
	})
	if err != nil {
		return nil, fmt.Errorf("load inbound message: %w", err)
	}

	subject.Label = "Message: " + stringutils.FirstNonEmpty(message.Subject, "(no subject)")
	notes := map[string]any{
		"status":         message.Status,
		"classification": message.Classification,
		"confidence":     message.Confidence,
		"fromAddress":    message.FromAddress,
		"subject":        message.Subject,
		"receivedAt":     message.ReceivedAt,
		"body":           message.TextBody,
		"needsReview":    message.NeedsReview(),
	}
	if message.MatchReason != "" {
		notes["matchReason"] = message.MatchReason
	}
	if message.MatchedShipmentID.IsNotNil() {
		notes["matchedShipmentId"] = message.MatchedShipmentID.String()
	}
	if message.MatchedCustomerID.IsNotNil() {
		notes["matchedCustomerId"] = message.MatchedCustomerID.String()
	}
	if message.MatchedCarrierID.IsNotNil() {
		notes["matchedCarrierId"] = message.MatchedCarrierID.String()
	}
	if message.FailureText != "" {
		notes["failure"] = message.FailureText
	}
	if len(message.Attachments) > 0 {
		notes["attachments"] = describeAttachments(message.Attachments)
	}
	subject.Notes = marshalNotes(notes)

	return subject, nil
}

func (s *Service) report(
	ctx context.Context,
	tenant pagination.TenantInfo,
	definitionID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectReport,
		ID:    definitionID.String(),
		Label: "Report",
	}
	if s.reports == nil {
		return subject, nil
	}

	found, err := s.reports.GetByID(ctx, &repositories.GetReportDefinitionRequest{
		TenantInfo:   tenant,
		DefinitionID: definitionID,
	})
	if err != nil {
		return nil, fmt.Errorf("load report: %w", err)
	}

	subject.Label = "Report: " + found.Name
	notes := map[string]any{
		"name":        found.Name,
		"description": found.Description,
		"category":    found.Category,
		"kind":        found.Kind,
		"status":      found.Status,
		"visibility":  found.Visibility,
		"lastRunAt":   found.LastRunAt,
	}
	if len(found.Diagnostics) > 0 {
		notes["diagnostics"] = found.Diagnostics
	}
	subject.Notes = marshalNotes(notes)

	return subject, nil
}

func (s *Service) dashboard(
	ctx context.Context,
	tenant pagination.TenantInfo,
	dashboardID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectDashboard,
		ID:    dashboardID.String(),
		Label: "Dashboard",
	}
	if s.dashboards == nil {
		return subject, nil
	}

	found, err := s.dashboards.GetByID(ctx, &repositories.GetReportDashboardRequest{
		TenantInfo:  tenant,
		DashboardID: dashboardID,
	})
	if err != nil {
		return nil, fmt.Errorf("load dashboard: %w", err)
	}

	tiles := 0
	if found.Layout != nil {
		tiles = len(found.Layout.Tiles)
	}
	subject.Label = "Dashboard: " + found.Name
	subject.Notes = marshalNotes(map[string]any{
		"name":        found.Name,
		"description": found.Description,
		"category":    found.Category,
		"visibility":  found.Visibility,
		"tiles":       tiles,
	})

	return subject, nil
}

func (s *Service) formulaTemplate(
	ctx context.Context,
	tenant pagination.TenantInfo,
	templateID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectFormulaTemplate,
		ID:    templateID.String(),
		Label: "Formula template",
	}
	if s.formulas == nil {
		return subject, nil
	}

	found, err := s.formulas.GetByID(ctx, repositories.GetFormulaTemplateByIDRequest{
		TemplateID: templateID,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, fmt.Errorf("load formula template: %w", err)
	}

	subject.Label = "Formula template: " + found.Name
	notes := map[string]any{
		"name":                found.Name,
		"description":         found.Description,
		"type":                found.Type,
		"status":              found.Status,
		"schemaId":            found.SchemaID,
		"savedExpression":     found.Expression,
		"variableDefinitions": found.VariableDefinitions,
		"roundingMode":        found.RoundingMode,
		"roundingPrecision":   found.RoundingPrecision,
		"version":             found.CurrentVersionNumber,
	}
	if found.MinCharge.Valid {
		notes["minCharge"] = found.MinCharge.Decimal.String()
	}
	if found.MaxCharge.Valid {
		notes["maxCharge"] = found.MaxCharge.Decimal.String()
	}
	subject.Notes = marshalNotes(notes)

	return subject, nil
}

// describeAttachments says what each file turned out to be and, where it did
// not, why. A file that was refused or could not be read is worth more to the
// run than its absence would be: it is the reason the message may be
// incomplete.
func describeAttachments(
	attachments []*inboundmessage.InboundAttachment,
) []map[string]any {
	described := make([]map[string]any, 0, len(attachments))
	for _, attachment := range attachments {
		entry := map[string]any{
			"fileName": attachment.FileName,
			"kind":     attachment.Kind,
		}
		if attachment.DocumentID.IsNotNil() {
			entry["documentId"] = attachment.DocumentID.String()
		}
		if attachment.FailureText != "" {
			entry["failure"] = attachment.FailureText
		}
		described = append(described, entry)
	}

	return described
}

func (s *Service) accountingConnection(
	ctx context.Context,
	tenant pagination.TenantInfo,
	connectionID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectAccountingConnection,
		ID:    connectionID.String(),
		Label: "Accounting connection",
	}
	if s.accounting == nil {
		return subject, nil
	}

	found, err := s.accounting.GetByID(ctx, repositories.GetAccountingConnectionByIDRequest{
		TenantInfo: tenant,
		ID:         connectionID,
	})
	if err != nil {
		return nil, fmt.Errorf("load accounting connection: %w", err)
	}

	subject.Label = "Accounting connection: " + found.ExternalCompanyName
	subject.Notes = marshalNotes(map[string]any{
		"system":              found.IntegrationType,
		"company":             found.ExternalCompanyName,
		"status":              found.Status,
		"lastSuccessAt":       found.LastSuccessAt,
		"lastCheckedAt":       found.LastCheckedAt,
		"consecutiveFailures": found.ConsecutiveFailures,
		"lastErrorCategory":   found.LastErrorCategory,
		"lastError":           found.AgentErrorSummary(),
		"reconnectBy":         found.RefreshTokenAbsoluteExpiresAt,
	})

	return subject, nil
}

func (s *Service) accountingSyncRecord(
	ctx context.Context,
	tenant pagination.TenantInfo,
	recordID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectAccountingSyncRecord,
		ID:    recordID.String(),
		Label: "Accounting sync record",
	}
	if s.syncRecords == nil {
		return subject, nil
	}

	found, err := s.syncRecords.GetByID(ctx, repositories.GetAccountingSyncRecordRequest{
		TenantInfo: tenant,
		ID:         recordID,
	})
	if err != nil {
		return nil, fmt.Errorf("load accounting sync record: %w", err)
	}

	subject.Label = "Accounting sync record: " + string(found.ObjectType) + " " + found.ObjectNumber
	subject.Notes = marshalNotes(map[string]any{
		"documentType":  found.ObjectType,
		"documentId":    found.ObjectID,
		"number":        found.ObjectNumber,
		"operation":     found.Operation,
		"status":        found.Status,
		"attempts":      found.AttemptCount,
		"errorCategory": found.ErrorCategory,
		"resolution":    found.Resolution,
		"documentDate":  found.DocumentDate,
		"queuedAt":      found.QueuedAt,
		"nextAttemptAt": found.NextAttemptAt,
	})

	return subject, nil
}

func (s *Service) accountingInboundChange(
	ctx context.Context,
	tenant pagination.TenantInfo,
	changeID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectAccountingInbound,
		ID:    changeID.String(),
		Label: "Payment recorded in the books",
	}
	if s.accountingInbound == nil {
		return subject, nil
	}

	found, err := s.accountingInbound.GetByID(ctx, repositories.GetAccountingInboundChangeRequest{
		TenantInfo: tenant,
		ID:         changeID,
	})
	if err != nil {
		return nil, fmt.Errorf("load accounting inbound change: %w", err)
	}

	subject.Label = "Payment recorded in the books: " + found.PartyName + " " + found.ExternalNumber
	subject.Notes = marshalNotes(map[string]any{
		"kind":        found.Kind,
		"status":      found.Status,
		"reason":      found.Reason,
		"resolution":  found.Resolution,
		"amountMinor": found.AmountMinor,
		"currency":    found.CurrencyCode,
		"paidOn":      found.TxnDate,
		"party":       found.PartyName,
		"lines":       found.Document.Lines,
	})

	return subject, nil
}

func (s *Service) accountingDriftFinding(
	ctx context.Context,
	tenant pagination.TenantInfo,
	findingID pulid.ID,
) (*agentdefinition.RuntimeSubject, error) {
	subject := &agentdefinition.RuntimeSubject{
		Type:  agent.SubjectAccountingDrift,
		ID:    findingID.String(),
		Label: "Books differ from Trenova",
	}
	if s.accountingDrift == nil {
		return subject, nil
	}

	found, err := s.accountingDrift.GetByID(ctx, repositories.GetAccountingDriftFindingRequest{
		TenantInfo: tenant,
		ID:         findingID,
	})
	if err != nil {
		return nil, fmt.Errorf("load accounting drift finding: %w", err)
	}

	subject.Label = "Books differ from Trenova: " + found.ObjectNumber + " " + found.PartyName
	subject.Notes = marshalNotes(map[string]any{
		"kind":               found.Kind,
		"status":             found.Status,
		"objectType":         found.ObjectType,
		"objectId":           found.ObjectID,
		"trenovaMinor":       found.TrenovaMinor,
		"providerMinor":      found.ProviderMinor,
		"differenceMinor":    found.DifferenceMinor,
		"currency":           found.CurrencyCode,
		"trenovaState":       found.TrenovaState,
		"providerState":      found.ProviderState,
		"providerModifiedAt": found.ProviderModifiedAt,
		"providerModifiedBy": found.ProviderModifiedBy,
		"directions":         found.Directions(),
		"detail":             found.Detail,
	})

	return subject, nil
}
