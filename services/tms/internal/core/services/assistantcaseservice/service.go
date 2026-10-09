package assistantcaseservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentwait"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/internal/core/domain/documenttype"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/subjectaccess"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// CasesResource is the realtime resource a case's binding or snooze
	// moves under, addressed to the conversation's owner.
	CasesResource = "assistant_case"

	// minSnoozeSeconds keeps a snooze from ending before the Desk has
	// drawn it; maxSnoozeSeconds keeps a case from being put out of sight
	// for longer than any appointment is booked ahead.
	minSnoozeSeconds int64 = 60
	maxSnoozeSeconds int64 = 30 * 24 * 60 * 60

	defaultReplyHours = 72
	maxReplyHours     = 168
	secondsPerHour    = 3600
)

type checklistBuilder func(
	*Service,
	context.Context,
	pagination.TenantInfo,
	*deskcase.Record,
) (*deskcase.Checklist, error)

// checklistKinds is the checklist each kind of record has.
//
//nolint:exhaustive // only a shipment and an invoice have a checklist; a dispute has none
var checklistKinds = map[agent.SubjectType]deskcase.ChecklistKind{
	agent.SubjectShipment: deskcase.ChecklistReadyToBill,
	agent.SubjectInvoice:  deskcase.ChecklistReadyToClose,
}

//nolint:exhaustive // only a shipment and an invoice have a checklist; a dispute has none
var checklists = map[agent.SubjectType]checklistBuilder{
	agent.SubjectShipment: (*Service).readyToBill,
	agent.SubjectInvoice:  (*Service).readyToClose,
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	States        *States
	Conversations repositories.ConversationRepository
	Cases         repositories.AssistantCaseRepository
	Subjects      repositories.AgentSubjectRepository
	Shipments     serviceports.ShipmentService
	Waits         serviceports.AgentWaitService
	Checklists    repositories.CaseChecklistRepository
	Definitions   repositories.AgentDefinitionRepository
	Runtime       serviceports.AgentRuntime
	Permissions   serviceports.PermissionEngine
	Realtime      serviceports.RealtimeService `optional:"true"`
}

// Service makes a conversation a case about a shipment, invoice or dispute
// and keeps it: what stands between the record and what comes next, a
// snooze until a time, the next appointment or the ETA, and a wait on the
// customer's or a carrier's reply that picks the case back up.
type Service struct {
	*States

	conversations repositories.ConversationRepository
	subjects      repositories.AgentSubjectRepository
	shipments     serviceports.ShipmentService
	agentWaits    serviceports.AgentWaitService
	checklists    repositories.CaseChecklistRepository
	definitions   repositories.AgentDefinitionRepository
	runtime       serviceports.AgentRuntime
	permissions   serviceports.PermissionEngine
	realtime      serviceports.RealtimeService
}

//nolint:gocritic // fx passes params by value
func New(p Params) serviceports.AssistantCaseService {
	return &Service{
		States:        p.States,
		conversations: p.Conversations,
		subjects:      p.Subjects,
		shipments:     p.Shipments,
		agentWaits:    p.Waits,
		checklists:    p.Checklists,
		definitions:   p.Definitions,
		runtime:       p.Runtime,
		permissions:   p.Permissions,
		realtime:      p.Realtime,
	}
}

// Get is the case as its conversation shows it. The person must still be
// able to read the record: the checklist reads its paperwork and charges.
func (s *Service) Get(
	ctx context.Context,
	req *serviceports.CaseThreadRequest,
) (*deskcase.View, error) {
	thread, err := s.caseThread(ctx, req)
	if err != nil {
		return nil, err
	}
	if err = subjectaccess.AssertReadable(
		ctx, s.permissions, req.Actor, thread.SubjectType,
	); err != nil {
		return nil, err
	}

	summary, err := s.summarize(ctx, req.TenantInfo, thread)
	if err != nil {
		return nil, err
	}

	view := &deskcase.View{Summary: summary, Parties: []deskcase.Party{}}
	record := summary.Record
	if record.Gone() {
		return view, nil
	}

	view.Parties, err = s.cases.Parties(ctx, &repositories.ListCasePartiesRequest{
		TenantInfo: req.TenantInfo,
		CustomerID: record.CustomerID,
		CarrierIDs: record.CarrierIDs,
	})
	if err != nil {
		return nil, err
	}

	if build, ok := checklists[thread.SubjectType]; ok {
		view.Checklist, err = build(s, ctx, req.TenantInfo, record)
	}
	if err != nil {
		return nil, err
	}
	view.Abilities = s.abilities(ctx, req, thread, view.Checklist)

	return view, nil
}

func (s *Service) readyToBill(
	ctx context.Context,
	tenant pagination.TenantInfo,
	record *deskcase.Record,
) (*deskcase.Checklist, error) {
	shipmentID := record.ID
	facts, err := s.cases.ShipmentFacts(ctx, &repositories.GetShipmentCaseFactsRequest{
		TenantInfo: tenant,
		ShipmentID: shipmentID,
		PODCode:    documenttype.CodePOD,
	})
	if err != nil {
		return nil, err
	}
	onFile := facts.DocumentTypesOnFile
	if facts.Arrangement, err = s.arrangement(ctx, tenant, deskcase.ChecklistReadyToBill, record); err != nil {
		return nil, err
	}
	facts.DocumentTypesOnFile = onFile

	readiness, err := s.shipments.GetBillingReadiness(ctx, shipmentID, tenant)
	if err != nil {
		return nil, fmt.Errorf("read the shipment's billing readiness: %w", err)
	}
	facts.Requirements = make([]deskcase.Requirement, 0, len(readiness.Requirements))
	for i := range readiness.Requirements {
		requirement := &readiness.Requirements[i]
		facts.Requirements = append(facts.Requirements, deskcase.Requirement{
			Code:      requirement.DocumentTypeCode,
			Name:      requirement.DocumentTypeName,
			Satisfied: requirement.Satisfied,
		})
	}
	facts.Validations = make([]deskcase.Validation, 0, len(readiness.ValidationFailures))
	for _, failure := range readiness.ValidationFailures {
		facts.Validations = append(facts.Validations, deskcase.Validation{
			Code:    failure.Code,
			Message: failure.Message,
		})
	}

	return deskcase.ReadyToBill(facts), nil
}

func (s *Service) readyToClose(
	ctx context.Context,
	tenant pagination.TenantInfo,
	record *deskcase.Record,
) (*deskcase.Checklist, error) {
	facts, err := s.cases.InvoiceFacts(ctx, &repositories.GetInvoiceCaseFactsRequest{
		TenantInfo: tenant,
		InvoiceID:  record.ID,
	})
	if err != nil {
		return nil, err
	}
	facts.Now = s.now()
	if facts.Arrangement, err = s.arrangement(ctx, tenant, deskcase.ChecklistReadyToClose, record); err != nil {
		return nil, err
	}

	return deskcase.ReadyToClose(facts), nil
}

// arrangement is how the organization wants this record's checklist laid
// out (the bill-to customer's template, else the organization's, else the
// default) and people's ticks on the steps it added.
func (s *Service) arrangement(
	ctx context.Context,
	tenant pagination.TenantInfo,
	kind deskcase.ChecklistKind,
	record *deskcase.Record,
) (deskcase.Arrangement, error) {
	template, err := s.templateFor(ctx, tenant, kind, record)
	if err != nil {
		return deskcase.Arrangement{}, err
	}

	ticks, err := s.checklists.ListTicks(ctx, &repositories.ListCaseChecklistTicksRequest{
		TenantInfo:  tenant,
		SubjectType: string(record.Type),
		SubjectID:   record.ID,
	})
	if err != nil {
		return deskcase.Arrangement{}, err
	}

	out := deskcase.Arrangement{
		Template: template,
		Ticks:    make(map[deskcase.ItemKey]deskcase.TickFact, len(ticks)),
	}
	for _, tick := range ticks {
		out.Ticks[tick.ItemKey] = deskcase.TickFact{At: tick.TickedAt, By: tick.TickedByName}
	}

	return out, nil
}

func (s *Service) templateFor(
	ctx context.Context,
	tenant pagination.TenantInfo,
	kind deskcase.ChecklistKind,
	record *deskcase.Record,
) (deskcase.TemplateItems, error) {
	template, err := s.checklists.TemplateFor(ctx, &repositories.GetCaseChecklistTemplateForRequest{
		TenantInfo: tenant,
		Kind:       kind,
		CustomerID: record.BillToID,
	})
	if err != nil {
		return nil, err
	}
	if template == nil {
		return nil, nil
	}

	return template.Items, nil
}

// Tick ticks, or unticks, a step the organization added that a person
// ticks, on the record the case is about. The tick is the record's, so
// everyone working a case about it sees it.
func (s *Service) Tick(
	ctx context.Context,
	req *serviceports.TickCaseItemRequest,
) (*deskcase.View, error) {
	thread, err := s.caseThread(ctx, &req.CaseThreadRequest)
	if err != nil {
		return nil, err
	}
	if err = subjectaccess.AssertReadable(
		ctx, s.permissions, req.Actor, thread.SubjectType,
	); err != nil {
		return nil, err
	}

	kind, ok := checklistKinds[thread.SubjectType]
	if !ok {
		return nil, errortypes.NewBusinessError("A dispute's case has no checklist to tick")
	}
	summary, err := s.summarize(ctx, req.TenantInfo, thread)
	if err != nil {
		return nil, err
	}
	if summary.Record.Gone() {
		return nil, errortypes.NewBusinessError("The case's record is gone")
	}

	template, err := s.templateFor(ctx, req.TenantInfo, kind, summary.Record)
	if err != nil {
		return nil, err
	}
	if !deskcase.Tickable(kind, template, req.ItemKey) {
		return nil, errortypes.NewValidationError(
			"itemKey", errortypes.ErrInvalid,
			"Only a step your organization added for a person to tick can be ticked",
		)
	}

	if req.Ticked {
		err = s.checklists.Tick(ctx, &deskcase.ChecklistTick{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			SubjectType:    string(thread.SubjectType),
			SubjectID:      thread.SubjectID,
			ItemKey:        req.ItemKey,
			TickedByID:     req.Actor.UserID,
			TickedAt:       s.now(),
		})
	} else {
		err = s.checklists.Untick(ctx, &repositories.UntickCaseChecklistItemRequest{
			TenantInfo:  req.TenantInfo,
			SubjectType: string(thread.SubjectType),
			SubjectID:   thread.SubjectID,
			ItemKey:     req.ItemKey,
		})
	}
	if err != nil {
		return nil, err
	}
	s.announce(ctx, thread)

	return s.Get(ctx, &req.CaseThreadRequest)
}

// Bind makes the conversation a case about the record, or moves it to
// another. A snooze set against the record it was about goes with it.
func (s *Service) Bind(
	ctx context.Context,
	req *serviceports.BindCaseRequest,
) (*conversation.Thread, error) {
	if !deskcase.IsSubject(req.SubjectType) {
		return nil, errortypes.NewValidationError(
			"subjectType", errortypes.ErrInvalid,
			"A case is about a shipment, an invoice or an invoice dispute",
		)
	}
	if err := req.SubjectType.CheckID(req.SubjectID); err != nil {
		return nil, errortypes.NewValidationError("subjectId", errortypes.ErrInvalid, err.Error())
	}

	thread, err := s.ownThread(ctx, &req.CaseThreadRequest)
	if err != nil {
		return nil, err
	}
	if thread.Origin.PageBound() {
		return nil, errortypes.NewBusinessError(
			"This conversation belongs to the page it was opened on and cannot become a case",
		)
	}
	if err = subjectaccess.AssertReadable(
		ctx,
		s.permissions,
		req.Actor,
		req.SubjectType,
	); err != nil {
		return nil, err
	}

	exists, err := s.subjects.Exists(ctx, repositories.AgentSubjectExistsRequest{
		TenantInfo:  req.TenantInfo,
		SubjectType: req.SubjectType,
		SubjectID:   req.SubjectID,
	})
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errortypes.NewValidationError(
			"subjectId", errortypes.ErrNotFound,
			"There is no such {0}", req.SubjectType.Noun(),
		)
	}

	if thread.SubjectType == req.SubjectType && thread.SubjectID == req.SubjectID {
		return s.served(ctx, req.TenantInfo, thread)
	}
	thread.SubjectType = req.SubjectType
	thread.SubjectID = req.SubjectID
	thread.SetSnooze(deskcase.Snooze{})

	return s.save(ctx, req.TenantInfo, thread)
}

// Unbind makes the case an ordinary conversation again.
func (s *Service) Unbind(
	ctx context.Context,
	req *serviceports.CaseThreadRequest,
) (*conversation.Thread, error) {
	thread, err := s.caseThread(ctx, req)
	if err != nil {
		return nil, err
	}
	if thread.Origin.PageBound() {
		return nil, errortypes.NewBusinessError(
			"This conversation belongs to the page it was opened on and stays about its record",
		)
	}

	thread.SubjectType = ""
	thread.SubjectID = pulid.Nil
	thread.SetSnooze(deskcase.Snooze{})

	return s.save(ctx, req.TenantInfo, thread)
}

// Snooze puts the case out of the way until a time, the shipment's next
// appointment or its ETA. The appointment and the ETA are read again each
// time the case is, so the snooze follows them as they move.
func (s *Service) Snooze(
	ctx context.Context,
	req *serviceports.SnoozeCaseRequest,
) (*conversation.Thread, error) {
	thread, err := s.caseThread(ctx, &req.CaseThreadRequest)
	if err != nil {
		return nil, err
	}
	summary, err := s.summarize(ctx, req.TenantInfo, thread)
	if err != nil {
		return nil, err
	}
	if summary.State == deskcase.StateSettled {
		return nil, errortypes.NewBusinessError("A settled case has nothing left to come back to")
	}

	snooze, err := s.resolveSnooze(ctx, req, thread, summary.Record)
	if err != nil {
		return nil, err
	}
	thread.SetSnooze(snooze)

	return s.save(ctx, req.TenantInfo, thread)
}

func (s *Service) resolveSnooze(
	ctx context.Context,
	req *serviceports.SnoozeCaseRequest,
	thread *conversation.Thread,
	record *deskcase.Record,
) (deskcase.Snooze, error) {
	now := s.now()
	switch req.Anchor {
	case deskcase.SnoozeTime:
		if err := checkSnoozeTime(req.Until, now); err != nil {
			return deskcase.Snooze{}, err
		}
		until := req.Until
		return deskcase.Snooze{Until: &until, Anchor: deskcase.SnoozeTime}, nil
	case deskcase.SnoozeAppointment:
		if thread.SubjectType != agent.SubjectShipment {
			return deskcase.Snooze{}, onlyShipments()
		}
		if record == nil || record.NextStopID.IsNil() || record.NextAppointmentAt == nil {
			return deskcase.Snooze{}, errortypes.NewValidationError(
				"anchor", errortypes.ErrInvalidOperation,
				"The shipment has no stop ahead to snooze to",
			)
		}
		if err := checkSnoozeTime(*record.NextAppointmentAt, now); err != nil {
			return deskcase.Snooze{}, errortypes.NewValidationError(
				"anchor", errortypes.ErrInvalidOperation,
				"The next appointment has already begun; snooze to a time instead",
			)
		}
		return deskcase.Snooze{
			Until:  record.NextAppointmentAt,
			Anchor: deskcase.SnoozeAppointment,
			StopID: record.NextStopID,
		}, nil
	case deskcase.SnoozeETA:
		if thread.SubjectType != agent.SubjectShipment {
			return deskcase.Snooze{}, onlyShipments()
		}
		until := s.snoozedETAs(ctx, req.TenantInfo, []*conversation.Thread{{
			SubjectType:  thread.SubjectType,
			SubjectID:    thread.SubjectID,
			SnoozeAnchor: deskcase.SnoozeETA,
		}})[thread.SubjectID]
		if until == nil && req.Until > 0 {
			until = &req.Until
		}
		if until == nil {
			return deskcase.Snooze{}, errortypes.NewValidationError(
				"anchor", errortypes.ErrInvalidOperation,
				"The shipment has no ETA yet; snooze to a time instead",
			)
		}
		if err := checkSnoozeTime(*until, now); err != nil {
			return deskcase.Snooze{}, err
		}
		return deskcase.Snooze{Until: until, Anchor: deskcase.SnoozeETA}, nil
	default:
		return deskcase.Snooze{}, errortypes.NewValidationError(
			"anchor", errortypes.ErrInvalid,
			"Snooze until a time, the next appointment or the ETA",
		)
	}
}

func checkSnoozeTime(until, now int64) error {
	switch {
	case until < now+minSnoozeSeconds:
		return errortypes.NewValidationError(
			"until", errortypes.ErrInvalid, "Snooze until a time that has not come yet",
		)
	case until > now+maxSnoozeSeconds:
		return errortypes.NewValidationError(
			"until", errortypes.ErrInvalid, "A case can be snoozed for at most 30 days",
		)
	default:
		return nil
	}
}

func onlyShipments() error {
	return errortypes.NewValidationError(
		"anchor", errortypes.ErrInvalidOperation,
		"Only a case about a shipment can be snoozed to its appointment or ETA",
	)
}

// Wake brings a snoozed case back now.
func (s *Service) Wake(
	ctx context.Context,
	req *serviceports.CaseThreadRequest,
) (*conversation.Thread, error) {
	thread, err := s.caseThread(ctx, req)
	if err != nil {
		return nil, err
	}
	if !thread.Snooze().Set() {
		return s.served(ctx, req.TenantInfo, thread)
	}
	thread.SetSnooze(deskcase.Snooze{})

	return s.save(ctx, req.TenantInfo, thread)
}

// AwaitReply parks the case on the customer's or a carrier's reply. It is a
// wait like any the agent sets: the reply, matched to the party, ends it and
// starts a turn that picks the case back up, and it gives up after the hours
// given.
func (s *Service) AwaitReply(
	ctx context.Context,
	req *serviceports.AwaitCaseReplyRequest,
) (*agentwait.Wait, error) {
	thread, err := s.caseThread(ctx, &req.CaseThreadRequest)
	if err != nil {
		return nil, err
	}
	if err = subjectaccess.AssertReadable(
		ctx, s.permissions, req.Actor, thread.SubjectType,
	); err != nil {
		return nil, err
	}
	summary, err := s.summarize(ctx, req.TenantInfo, thread)
	if err != nil {
		return nil, err
	}
	if summary.State == deskcase.StateSettled {
		return nil, errortypes.NewBusinessError("A settled case is not waiting on anyone")
	}

	party, err := s.party(ctx, req, summary.Record)
	if err != nil {
		return nil, err
	}

	condition := agentwait.Condition{}
	if party.Kind == deskcase.WaitingOnCarrier {
		condition.CarrierID = party.ID
	} else {
		condition.CustomerID = party.ID
	}

	return s.agentWaits.Register(ctx, &serviceports.RegisterWaitRequest{
		Actor:        req.Actor,
		DefinitionID: thread.AgentDefinitionID,
		ThreadID:     thread.ID,
		Kind:         agentwait.KindReply,
		Condition:    condition,
		Description: fmt.Sprintf(
			"A reply from %s about %s %s",
			party.Name, thread.SubjectType.Noun(), summary.Record.Label,
		),
		Then:            "Read what they said and pick the case back up from there.",
		LifetimeSeconds: int64(replyHours(req.GiveUpAfterHours)) * secondsPerHour,
	})
}

func replyHours(requested int) int {
	switch {
	case requested <= 0:
		return defaultReplyHours
	case requested > maxReplyHours:
		return maxReplyHours
	default:
		return requested
	}
}

// party is the customer or carrier the person asked to wait on, which must
// be the record's own.
func (s *Service) party(
	ctx context.Context,
	req *serviceports.AwaitCaseReplyRequest,
	record *deskcase.Record,
) (*deskcase.Party, error) {
	if record.Gone() {
		return nil, errortypes.NewBusinessError("The case's record is gone")
	}

	var belongs bool
	switch req.Party {
	case deskcase.WaitingOnCustomer:
		belongs = req.PartyID.IsNotNil() && req.PartyID == record.CustomerID
	case deskcase.WaitingOnCarrier:
		belongs = slices.Contains(record.CarrierIDs, req.PartyID)
	case deskcase.WaitingOnReply, deskcase.WaitingOnEvent:
		return nil, errortypes.NewValidationError(
			"party", errortypes.ErrInvalid, "Wait on the customer or a carrier",
		)
	default:
		return nil, errortypes.NewValidationError(
			"party", errortypes.ErrInvalid, "Wait on the customer or a carrier",
		)
	}
	if !belongs {
		return nil, errortypes.NewValidationError(
			"partyId", errortypes.ErrInvalid,
			"That is not the customer or a carrier of this {0}", record.Type.Noun(),
		)
	}

	parties, err := s.cases.Parties(ctx, &repositories.ListCasePartiesRequest{
		TenantInfo: req.TenantInfo,
		CustomerID: record.CustomerID,
		CarrierIDs: record.CarrierIDs,
	})
	if err != nil {
		return nil, err
	}
	for i := range parties {
		if parties[i].Kind == req.Party && parties[i].ID == req.PartyID {
			return &parties[i], nil
		}
	}

	return nil, errortypes.NewValidationError(
		"partyId", errortypes.ErrNotFound, "That customer or carrier is gone",
	)
}

func (s *Service) ownThread(
	ctx context.Context,
	req *serviceports.CaseThreadRequest,
) (*conversation.Thread, error) {
	if req.Actor == nil {
		return nil, errortypes.NewAuthorizationError("A case belongs to a person")
	}

	return s.conversations.GetThread(ctx, repositories.GetThreadRequest{
		ID:         req.ThreadID,
		UserID:     req.Actor.UserID,
		TenantInfo: req.TenantInfo,
	})
}

func (s *Service) caseThread(
	ctx context.Context,
	req *serviceports.CaseThreadRequest,
) (*conversation.Thread, error) {
	thread, err := s.ownThread(ctx, req)
	if err != nil {
		return nil, err
	}
	if !thread.IsCase() {
		return nil, errortypes.NewNotFoundError(
			"This conversation is not a case about a shipment, invoice or dispute",
		)
	}

	return thread, nil
}

func (s *Service) save(
	ctx context.Context,
	tenant pagination.TenantInfo,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	updated, err := s.conversations.UpdateThread(ctx, thread)
	if err != nil {
		return nil, err
	}
	s.announce(ctx, updated)

	return s.served(ctx, tenant, updated)
}

func (s *Service) served(
	ctx context.Context,
	tenant pagination.TenantInfo,
	thread *conversation.Thread,
) (*conversation.Thread, error) {
	if err := s.Attach(ctx, tenant, thread.UserID, []*conversation.Thread{thread}); err != nil {
		return nil, err
	}

	return thread, nil
}

func (s *Service) announce(ctx context.Context, thread *conversation.Thread) {
	if s.realtime == nil {
		return
	}
	if err := s.realtime.PublishResourceInvalidation(
		context.WithoutCancel(ctx),
		&serviceports.PublishResourceInvalidationRequest{
			OrganizationID: thread.OrganizationID,
			BusinessUnitID: thread.BusinessUnitID,
			AudienceUserID: thread.UserID,
			Resource:       CasesResource,
			Action:         "updated",
			RecordID:       thread.ID,
			Entity:         map[string]string{"threadId": thread.ID.String()},
		},
	); err != nil {
		s.l.Debug("could not announce a change to a case",
			zap.String("thread", thread.ID.String()), zap.Error(err))
	}
}
