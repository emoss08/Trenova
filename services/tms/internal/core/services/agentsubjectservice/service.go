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
	"github.com/emoss08/trenova/internal/core/domain/insight"
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

	Logger       *zap.Logger
	BillingQueue serviceports.BillingQueueService
	Shipments    serviceports.ShipmentService
	Console      repositories.DispatchConsoleRepository     `optional:"true"`
	Content      serviceports.DocumentContentService        `optional:"true"`
	Insights     repositories.InsightRepository             `optional:"true"`
	BankReceipts serviceports.BankReceiptService            `optional:"true"`
	WorkItems    repositories.BankReceiptWorkItemRepository `optional:"true"`
	Occurrences  repositories.DetentionOccurrenceRepository `optional:"true"`
	Workers      repositories.WorkerRepository              `optional:"true"`
	Credentials  repositories.WorkerCredentialRepository    `optional:"true"`
	CarrierIntel repositories.CarrierIntelEventRepository   `optional:"true"`
	EDIFiles     repositories.EDIInboundFileRepository      `optional:"true"`
}

// Service describes the record an agent run or a conversation is about, so
// a run woken by an event and a thread opened from a page both start with
// the same picture of their subject in front of the model.
type Service struct {
	content      serviceports.DocumentContentService
	billingQueue serviceports.BillingQueueService
	shipments    serviceports.ShipmentService
	console      repositories.DispatchConsoleRepository
	insights     repositories.InsightRepository
	receipts     serviceports.BankReceiptService
	workItems    repositories.BankReceiptWorkItemRepository
	occurrences  repositories.DetentionOccurrenceRepository
	workers      repositories.WorkerRepository
	credentials  repositories.WorkerCredentialRepository
	carrierIntel repositories.CarrierIntelEventRepository
	ediFiles     repositories.EDIInboundFileRepository
	logger       *zap.Logger
}

func New(p Params) serviceports.AgentSubjectDescriber {
	return &Service{
		content:      p.Content,
		billingQueue: p.BillingQueue,
		shipments:    p.Shipments,
		console:      p.Console,
		insights:     p.Insights,
		receipts:     p.BankReceipts,
		workItems:    p.WorkItems,
		occurrences:  p.Occurrences,
		workers:      p.Workers,
		credentials:  p.Credentials,
		carrierIntel: p.CarrierIntel,
		ediFiles:     p.EDIFiles,
		logger:       p.Logger.Named("service.agentsubject"),
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
		subject.Notes = marshalNotes(moves[0])
	}

	return subject, nil
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
					"amount":            money.DecimalFromMinor(suggestion.AmountMinor).StringFixed(2),
					"score":             suggestion.Score,
					"reason":            suggestion.Reason,
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
		notes["licenceNumber"] = entity.Profile.LicenseNumber
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
		notes["warning"] = "This file is no longer quarantined; do not act as if it were held back."
	}
	subject.Notes = marshalNotes(notes)

	return subject, nil
}
