package workeremploymentservice

import (
	"context"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/equipment/ports/fleetcodeports"
	"github.com/emoss08/trenova/internal/modules/identity/app/ptoledgerservice"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/payassignment"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/driverpayports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/ptopolicyservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/pto"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerchecklistports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

type phase int

const (
	phaseBefore phase = iota
	phaseAtomic
	phaseAfter
)

type step struct {
	phase   phase
	effects []*employment.Effect
	run     func(ctx context.Context, plan *RecordPlan) ([]*employment.Effect, error)
}

type teamState struct {
	FleetCodeID pulid.ID `json:"fleetCodeId"`
	ManagerID   pulid.ID `json:"managerId"`
}

type teamChange struct {
	before teamState
	after  teamState
}

type RecordPlan struct {
	Request *RecordRequest
	Event   *employment.WorkerEmploymentEvent
	Stored  *worker.Worker
	Worker  *worker.Worker
	Changed bool
	team    *teamChange
	steps   []*step
}

func (p *RecordPlan) Effects() []*employment.Effect {
	out := make([]*employment.Effect, 0, len(p.steps))
	for _, st := range p.steps {
		out = append(out, st.effects...)
	}
	return out
}

func (p *RecordPlan) add(st *step) {
	if len(st.effects) > 0 {
		p.steps = append(p.steps, st)
	}
}

func (s *Service) Plan(ctx context.Context, req *RecordRequest) (*RecordPlan, error) {
	wrk, history, err := s.loadState(
		ctx,
		req.TenantInfo,
		req.WorkerID,
		req.Kind == employment.EmploymentEventTransferred,
	)
	if err != nil {
		return nil, err
	}

	event := &employment.WorkerEmploymentEvent{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		WorkerID:       req.WorkerID,
		Kind:           req.Kind,
		EffectiveAt:    req.EffectiveAt,
		Reason:         strings.TrimSpace(req.Reason),
		Notes:          strings.TrimSpace(req.Notes),
		DocumentID:     req.DocumentID,
		RecordedByID:   req.UserID,
		FromValues:     map[string]string{},
		ToValues:       map[string]string{},
	}

	multiErr := errortypes.NewMultiError()
	event.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	if err = req.Kind.CanRecord(employment.EmploymentStateOf(wrk, history)); err != nil {
		return nil, err
	}
	plan := &RecordPlan{Request: req, Event: event, Stored: wrk}
	if err = s.populateValues(ctx, plan); err != nil {
		return nil, err
	}
	if !event.DocumentID.IsNil() {
		if err = s.requireWorkerDocument(ctx, req.TenantInfo, wrk.ID, event.DocumentID); err != nil {
			return nil, err
		}
	}

	plan.Worker = projectedWorker(wrk)
	plan.Changed = event.Apply(plan.Worker)

	if err = s.planEffects(ctx, plan); err != nil {
		return nil, err
	}
	return plan, nil
}

func projectedWorker(wrk *worker.Worker) *worker.Worker {
	projected := *wrk
	if wrk.Profile != nil {
		profile := *wrk.Profile
		projected.Profile = &profile
	}
	return &projected
}

func (s *Service) planEffects(ctx context.Context, plan *RecordPlan) error {
	kind := plan.Request.Kind
	switch {
	case kind.TakesOffDuty():
		if plan.Stored.CanBeAssigned || plan.Stored.AvailableForDispatch {
			plan.add(&step{
				phase:   phaseAtomic,
				effects: []*employment.Effect{{Kind: employment.EffectDispatchRemoved}},
			})
		}
	case kind.PutsOnDuty():
		if !plan.Stored.CanBeAssigned || !plan.Stored.AvailableForDispatch {
			plan.add(&step{
				phase:   phaseAtomic,
				effects: []*employment.Effect{{Kind: employment.EffectDispatchRestored}},
			})
		}
	case kind == employment.EmploymentEventTerminated:
		if err := s.planTermination(ctx, plan); err != nil {
			return err
		}
	}
	return s.planFollowOns(ctx, plan)
}

func (s *Service) planTermination(ctx context.Context, plan *RecordPlan) error {
	req := plan.Request
	upcoming, err := s.upcomingTimeOff(ctx, req)
	if err != nil {
		return err
	}
	released := make([]ptoledgerservice.ReleasedUsage, 0, len(upcoming))
	for _, request := range upcoming {
		plan.add(s.timeOffStep(request))
		if request.Status == worker.PTOStatusApproved && request.BalanceAfterDays.Valid {
			released = append(released, ptoledgerservice.ReleasedUsage{
				PTOType: request.Type,
				Days:    request.Days,
			})
		}
	}

	settlement, err := s.ptoLedger.PreviewTerminationSettlement(
		ctx, req.TenantInfo, req.WorkerID, req.EffectiveAt, released,
	)
	if err != nil {
		return err
	}
	plan.add(s.settlementStep(settlement))

	assignment, err := s.openPolicyAssignment(ctx, req, settlement)
	if err != nil {
		return err
	}
	if assignment != nil {
		plan.add(s.endPolicyStep(assignment))
	}

	payAssignment, err := s.openPayAssignment(ctx, req)
	if err != nil {
		return err
	}
	if payAssignment != nil {
		plan.add(s.endPayStep(payAssignment))
	}

	status, err := s.portal.GetPortalStatus(ctx, req.TenantInfo, req.WorkerID)
	if err != nil {
		return err
	}
	if status.Linked || status.PendingInvitation != nil {
		plan.add(s.revokePortalStep())
	}
	return nil
}

func (s *Service) planFollowOns(ctx context.Context, plan *RecordPlan) error {
	req := plan.Request
	checklists, err := s.checklists.PlanForEvent(ctx, req.Kind, plan.Worker)
	if err != nil {
		return err
	}
	if len(checklists.Close) > 0 {
		plan.add(s.closeChecklistsStep(checklists))
	}
	if checklists.Template != nil {
		plan.add(s.startChecklistStep(checklists))
	}

	switch req.Kind { //nolint:exhaustive // only these kinds open required courses
	case employment.EmploymentEventHired,
		employment.EmploymentEventRehired,
		employment.EmploymentEventPromoted:
		courses, trainingErr := s.training.PlanAssignRequiredFor(ctx, req.TenantInfo, plan.Worker)
		if trainingErr != nil {
			return trainingErr
		}
		if len(courses) > 0 {
			plan.add(s.trainingStep(courses))
		}
	}

	if req.Kind == employment.EmploymentEventRehired {
		policy, policyErr := s.ptoPolicies.DefaultPolicy(ctx, req.TenantInfo)
		if policyErr != nil {
			return policyErr
		}
		if policy != nil {
			plan.add(s.defaultPolicyStep(policy))
		}
	}
	return nil
}

func (s *Service) upcomingTimeOff(
	ctx context.Context,
	req *RecordRequest,
) ([]*worker.WorkerPTO, error) {
	out := make([]*worker.WorkerPTO, 0)
	for _, status := range []worker.PTOStatus{worker.PTOStatusApproved, worker.PTOStatusRequested} {
		result, err := s.pto.List(ctx, &workerptoports.ListPTORequest{
			Filter: &pagination.QueryOptions{
				TenantInfo: req.TenantInfo,
				Pagination: pagination.Info{Limit: upcomingPTOPageSize},
			},
			Status:        status.String(),
			StartDateFrom: req.EffectiveAt,
			WorkerID:      req.WorkerID,
		})
		if err != nil {
			return nil, err
		}
		out = append(out, result.Items...)
	}
	return out, nil
}

func (s *Service) openPolicyAssignment(
	ctx context.Context,
	req *RecordRequest,
	settlement *ptoledgerservice.TerminationPlan,
) (*pto.WorkerPTOPolicyAssignment, error) {
	if settlement.Resolved != nil && settlement.Resolved.Assignment.IsOpen() {
		assignment := settlement.Resolved.Assignment
		if assignment.Policy == nil {
			assignment.Policy = settlement.Resolved.Policy
		}
		return assignment, nil
	}
	return s.ptoPolicies.ActiveAssignment(ctx, req.TenantInfo, req.WorkerID)
}

func (s *Service) openPayAssignment(
	ctx context.Context,
	req *RecordRequest,
) (*payassignment.WorkerPayAssignment, error) {
	assignment, err := s.payAssignRepo.GetEffectiveForWorker(
		ctx,
		driverpayports.GetWorkerPayAssignmentRequest{
			TenantInfo: req.TenantInfo,
			WorkerID:   req.WorkerID,
			AsOf:       req.EffectiveAt,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}
		return nil, err
	}
	if assignment == nil || assignment.EffectiveTo != nil {
		return nil, nil
	}
	return assignment, nil
}

func (s *Service) timeOffStep(request *worker.WorkerPTO) *step {
	days := request.Days
	ptoType := request.Type
	start, end := request.StartDate, request.EndDate
	effect := &employment.Effect{
		Kind:      employment.EffectTimeOffCancelled,
		PTOType:   &ptoType,
		Days:      &days,
		StartDate: &start,
		EndDate:   &end,
	}
	return &step{
		phase:   phaseBefore,
		effects: []*employment.Effect{effect},
		run: func(ctx context.Context, plan *RecordPlan) ([]*employment.Effect, error) {
			_, err := s.pto.Cancel(ctx, &workerptoports.UpdatePTOStatusRequest{
				ID:              request.ID,
				TenantInfo:      plan.Request.TenantInfo,
				Status:          worker.PTOStatusCancelled,
				UserID:          plan.Request.UserID,
				Reason:          terminationCancelReason,
				ExpectedVersion: request.Version,
			})
			return []*employment.Effect{effect}, err
		},
	}
}

func (s *Service) settlementStep(settlement *ptoledgerservice.TerminationPlan) *step {
	effects := make([]*employment.Effect, 0, len(settlement.Settlements))
	for _, planned := range settlement.Settlements {
		days := planned.Days
		ptoType := planned.PTOType
		kind := employment.EffectPTOForfeited
		if planned.EntryType == pto.PTOLedgerEntryPayout {
			kind = employment.EffectPTOPaidOut
		}
		effects = append(effects, &employment.Effect{Kind: kind, PTOType: &ptoType, Days: &days})
	}
	return &step{
		phase:   phaseAtomic,
		effects: effects,
		run: func(ctx context.Context, plan *RecordPlan) ([]*employment.Effect, error) {
			req := plan.Request
			settled, err := s.ptoLedger.SettleOnTermination(
				ctx, req.TenantInfo, req.WorkerID, req.EffectiveAt, settlement,
				ptoledgerservice.UserActor(req.UserID),
			)
			if err != nil {
				return nil, err
			}
			return settledEffects(effects, settled), nil
		},
	}
}

func settledEffects(
	planned []*employment.Effect,
	settled *ptoledgerservice.TerminationSettlement,
) []*employment.Effect {
	out := make([]*employment.Effect, 0, len(planned))
	for _, effect := range planned {
		for _, entry := range settled.Entries {
			if entry.PTOType != *effect.PTOType {
				continue
			}
			days := entry.AmountDays.Neg()
			effect.Days = &days
			out = append(out, effect)
			break
		}
	}
	return out
}

func (s *Service) endPolicyStep(assignment *pto.WorkerPTOPolicyAssignment) *step {
	effect := &employment.Effect{Kind: employment.EffectPTOPolicyEnded}
	if assignment.Policy != nil {
		effect.Name = new(assignment.Policy.Name)
	}
	return &step{
		phase:   phaseAtomic,
		effects: []*employment.Effect{effect},
		run: func(ctx context.Context, plan *RecordPlan) ([]*employment.Effect, error) {
			req := plan.Request
			effectiveTo := req.EffectiveAt
			if effectiveTo <= assignment.EffectiveFrom {
				effectiveTo = assignment.EffectiveFrom + 1
			}
			_, err := s.ptoPolicies.EndAssignment(ctx, &ptopolicyservice.EndAssignmentRequest{
				TenantInfo:   req.TenantInfo,
				AssignmentID: assignment.ID,
				EffectiveTo:  effectiveTo,
				Version:      assignment.Version,
				UserID:       req.UserID,
			})
			if err != nil {
				return nil, err
			}
			return []*employment.Effect{effect}, nil
		},
	}
}

func (s *Service) endPayStep(assignment *payassignment.WorkerPayAssignment) *step {
	effect := &employment.Effect{Kind: employment.EffectPayAssignmentEnded}
	if assignment.PayProfile != nil {
		effect.Name = new(assignment.PayProfile.Name)
	}
	return &step{
		phase:   phaseAtomic,
		effects: []*employment.Effect{effect},
		run: func(ctx context.Context, plan *RecordPlan) ([]*employment.Effect, error) {
			req := plan.Request
			endDate := req.EffectiveAt
			if endDate <= assignment.EffectiveFrom {
				endDate = assignment.EffectiveFrom + 1
			}
			if _, err := s.driverPay.EndAssignment(
				ctx, req.TenantInfo, assignment.ID, endDate, s.actor(req.TenantInfo, req.UserID),
			); err != nil {
				return nil, err
			}
			return []*employment.Effect{effect}, nil
		},
	}
}

func (s *Service) revokePortalStep() *step {
	effect := &employment.Effect{Kind: employment.EffectPortalAccessRevoked}
	return &step{
		phase:   phaseAfter,
		effects: []*employment.Effect{effect},
		run: func(ctx context.Context, plan *RecordPlan) ([]*employment.Effect, error) {
			req := plan.Request
			err := s.portal.RevokeAccess(
				ctx, req.TenantInfo, req.WorkerID, s.actor(req.TenantInfo, req.UserID),
			)
			return []*employment.Effect{effect}, err
		},
	}
}

func (s *Service) closeChecklistsStep(checklists *workerchecklistports.EventPlan) *step {
	effect := &employment.Effect{
		Kind:  employment.EffectChecklistClosed,
		Count: new(len(checklists.Close)),
	}
	if len(checklists.Close) == 1 {
		effect.Name = new(checklists.Close[0].Name)
	}
	return &step{
		phase:   phaseAfter,
		effects: []*employment.Effect{effect},
		run: func(ctx context.Context, plan *RecordPlan) ([]*employment.Effect, error) {
			closed, err := s.checklists.CloseForEvent(
				ctx,
				plan.Event,
				checklists,
				plan.Request.UserID,
			)
			effect.Count = new(closed)
			return []*employment.Effect{effect}, err
		},
	}
}

func (s *Service) startChecklistStep(checklists *workerchecklistports.EventPlan) *step {
	effect := &employment.Effect{
		Kind: employment.EffectChecklistStarted,
		Name: new(checklists.Template.Name),
	}
	return &step{
		phase:   phaseAfter,
		effects: []*employment.Effect{effect},
		run: func(ctx context.Context, plan *RecordPlan) ([]*employment.Effect, error) {
			_, err := s.checklists.SpawnForEvent(ctx, plan.Event, checklists, plan.Request.UserID)
			return []*employment.Effect{effect}, err
		},
	}
}

func (s *Service) trainingStep(courses []*workertraining.TrainingCourse) *step {
	effect := &employment.Effect{
		Kind:  employment.EffectTrainingAssigned,
		Count: new(len(courses)),
	}
	return &step{
		phase:   phaseAfter,
		effects: []*employment.Effect{effect},
		run: func(ctx context.Context, plan *RecordPlan) ([]*employment.Effect, error) {
			req := plan.Request
			records := s.training.AssignPlanned(
				ctx,
				req.TenantInfo,
				req.WorkerID,
				courses,
				req.UserID,
			)
			effect.Count = new(len(records))
			if len(records) < len(courses) {
				return []*employment.Effect{effect}, errTrainingIncomplete
			}
			return []*employment.Effect{effect}, nil
		},
	}
}

func (s *Service) defaultPolicyStep(policy *pto.PTOPolicy) *step {
	effect := &employment.Effect{
		Kind: employment.EffectDefaultPTOPolicyApplied,
		Name: new(policy.Name),
	}
	return &step{
		phase:   phaseAfter,
		effects: []*employment.Effect{effect},
		run: func(ctx context.Context, plan *RecordPlan) ([]*employment.Effect, error) {
			req := plan.Request
			err := s.ptoPolicies.AssignAtHire(ctx, &ptopolicyservice.HireAssignment{
				TenantInfo:    req.TenantInfo,
				PolicyID:      policy.ID,
				WorkerID:      req.WorkerID,
				EffectiveFrom: req.EffectiveAt,
				UserID:        req.UserID,
			})
			return []*employment.Effect{effect}, err
		},
	}
}

func (s *Service) populateValues(ctx context.Context, plan *RecordPlan) error {
	req, wrk, event := plan.Request, plan.Stored, plan.Event
	switch req.Kind {
	case employment.EmploymentEventHired, employment.EmploymentEventRehired:
		if wrk.Profile != nil && wrk.Profile.HireDate > 0 {
			event.FromValues[employment.EmploymentValueHireDate] = strconv.FormatInt(
				wrk.Profile.HireDate,
				10,
			)
		}
		event.ToValues[employment.EmploymentValueHireDate] = strconv.FormatInt(req.EffectiveAt, 10)
		event.FromValues[employment.EmploymentValueStatus] = wrk.Status.String()
		event.ToValues[employment.EmploymentValueStatus] = "Active"
	case employment.EmploymentEventTerminated:
		event.FromValues[employment.EmploymentValueStatus] = wrk.Status.String()
		event.ToValues[employment.EmploymentValueStatus] = "Inactive"
		event.ToValues[employment.EmploymentValueTerminationDate] = strconv.FormatInt(
			req.EffectiveAt,
			10,
		)
	case employment.EmploymentEventSuspended:
		event.FromValues[employment.EmploymentValueCanBeAssigned] = strconv.FormatBool(
			wrk.CanBeAssigned,
		)
		event.ToValues[employment.EmploymentValueCanBeAssigned] = "false"
	case employment.EmploymentEventLeaveStarted:
		if req.LeaveType == nil || !req.LeaveType.IsValid() {
			return errortypes.NewValidationError(
				"leaveType",
				errortypes.ErrRequired,
				"Choose the kind of leave",
			)
		}
		event.FromValues[employment.EmploymentValueCanBeAssigned] = strconv.FormatBool(
			wrk.CanBeAssigned,
		)
		event.ToValues[employment.EmploymentValueCanBeAssigned] = "false"
		event.ToValues[employment.EmploymentValueLeaveType] = req.LeaveType.String()
	case employment.EmploymentEventReinstated:
		event.FromValues[employment.EmploymentValueCanBeAssigned] = strconv.FormatBool(
			wrk.CanBeAssigned,
		)
		event.ToValues[employment.EmploymentValueCanBeAssigned] = "true"
	case employment.EmploymentEventLeaveEnded:
		event.FromValues[employment.EmploymentValueCanBeAssigned] = strconv.FormatBool(
			wrk.CanBeAssigned,
		)
		event.ToValues[employment.EmploymentValueCanBeAssigned] = "true"
		if wrk.LeaveType != "" {
			event.FromValues[employment.EmploymentValueLeaveType] = wrk.LeaveType.String()
		}
	case employment.EmploymentEventTransferred:
		return s.populateTransfer(ctx, plan)
	case employment.EmploymentEventPromoted:
		return populatePromotion(req, wrk, event)
	case employment.EmploymentEventRateChanged:
		rate := strings.TrimSpace(req.Rate)
		if rate == "" {
			return errortypes.NewValidationError(
				"rate",
				errortypes.ErrRequired,
				"Enter the new rate",
			)
		}
		event.ToValues[employment.EmploymentValueRate] = rate
		if unit := strings.TrimSpace(req.RateUnit); unit != "" {
			event.ToValues[employment.EmploymentValueRateUnit] = unit
		}
	case employment.EmploymentEventProbationEnded:
	}
	return nil
}

func (s *Service) populateTransfer(ctx context.Context, plan *RecordPlan) error {
	req, wrk, event := plan.Request, plan.Stored, plan.Event
	if req.FleetCodeID == nil && req.ManagerID == nil {
		return errortypes.NewValidationError(
			"fleetCodeId",
			errortypes.ErrRequired,
			"A transfer needs a new fleet code or manager",
		)
	}
	fleetChanges := req.FleetCodeID != nil && *req.FleetCodeID != wrk.FleetCodeID
	managerChanges := req.ManagerID != nil && *req.ManagerID != wrk.ManagerID
	if !fleetChanges && !managerChanges {
		if req.FleetCodeID != nil {
			return errortypes.NewValidationError(
				"fleetCodeId",
				errortypes.ErrInvalid,
				"The worker is already in that fleet",
			)
		}
		return errortypes.NewValidationError(
			"managerId",
			errortypes.ErrInvalid,
			"The worker already reports to that manager",
		)
	}

	if fleetChanges {
		if err := s.populateFleet(ctx, plan); err != nil {
			return err
		}
	}
	if managerChanges {
		if err := s.populateManager(ctx, plan); err != nil {
			return err
		}
	}

	change := &teamChange{
		before: teamState{FleetCodeID: wrk.FleetCodeID, ManagerID: wrk.ManagerID},
		after:  teamState{FleetCodeID: wrk.FleetCodeID, ManagerID: wrk.ManagerID},
	}
	if fleetChanges {
		change.after.FleetCodeID = pulid.ID(event.ToValues[employment.EmploymentValueFleetCodeID])
	}
	if managerChanges {
		change.after.ManagerID = pulid.ID(event.ToValues[employment.EmploymentValueManagerID])
	}
	plan.team = change
	return nil
}

func (s *Service) populateFleet(ctx context.Context, plan *RecordPlan) error {
	req, wrk, event := plan.Request, plan.Stored, plan.Event
	if !wrk.FleetCodeID.IsNil() {
		event.FromValues[employment.EmploymentValueFleetCodeID] = wrk.FleetCodeID.String()
		if wrk.FleetCode != nil {
			event.FromValues[employment.EmploymentValueFleetCode] = wrk.FleetCode.Code
		}
	}
	if req.FleetCodeID.IsNil() {
		event.ToValues[employment.EmploymentValueFleetCodeID] = ""
		return nil
	}
	tenantInfo := req.TenantInfo
	fleet, err := s.fleetCodeRepo.GetByID(ctx, fleetcodeports.GetFleetCodeByIDRequest{
		ID:         *req.FleetCodeID,
		TenantInfo: &tenantInfo,
	})
	if err != nil {
		return err
	}
	event.ToValues[employment.EmploymentValueFleetCodeID] = fleet.ID.String()
	event.ToValues[employment.EmploymentValueFleetCode] = fleet.Code
	return nil
}

func (s *Service) populateManager(ctx context.Context, plan *RecordPlan) error {
	req, wrk, event := plan.Request, plan.Stored, plan.Event
	if !wrk.ManagerID.IsNil() {
		event.FromValues[employment.EmploymentValueManagerID] = wrk.ManagerID.String()
		if wrk.Manager != nil {
			event.FromValues[employment.EmploymentValueManager] = wrk.Manager.Name
		}
	}
	if req.ManagerID.IsNil() {
		event.ToValues[employment.EmploymentValueManagerID] = ""
		return nil
	}
	manager, err := s.users.GetTenantMember(ctx, tenantports.GetTenantMemberRequest{
		UserID:     *req.ManagerID,
		TenantInfo: req.TenantInfo,
		Now:        timeutils.NowUnix(),
	})
	if err != nil && !errortypes.IsNotFoundError(err) {
		return err
	}
	if err != nil || !manager.IsActive() {
		return errortypes.NewValidationError(
			"managerId",
			errortypes.ErrInvalid,
			"Choose an active person in this organization",
		)
	}
	event.ToValues[employment.EmploymentValueManagerID] = manager.ID.String()
	event.ToValues[employment.EmploymentValueManager] = manager.Name
	return nil
}

func populatePromotion(
	req *RecordRequest,
	wrk *worker.Worker,
	event *employment.WorkerEmploymentEvent,
) error {
	if req.DriverType == nil && req.WorkerType == nil {
		return errortypes.NewValidationError(
			"driverType",
			errortypes.ErrRequired,
			"A promotion needs a new driver type or worker type",
		)
	}
	changed := false
	if req.DriverType != nil {
		if !req.DriverType.IsValid() {
			return errortypes.NewValidationError(
				"driverType",
				errortypes.ErrInvalid,
				"Unknown driver type",
			)
		}
		if *req.DriverType != wrk.DriverType {
			event.FromValues[employment.EmploymentValueDriverType] = wrk.DriverType.String()
			event.ToValues[employment.EmploymentValueDriverType] = req.DriverType.String()
			changed = true
		}
	}
	if req.WorkerType != nil {
		if !req.WorkerType.IsValid() {
			return errortypes.NewValidationError(
				"workerType",
				errortypes.ErrInvalid,
				"Unknown worker type",
			)
		}
		if *req.WorkerType != wrk.Type {
			event.FromValues[employment.EmploymentValueWorkerType] = wrk.Type.String()
			event.ToValues[employment.EmploymentValueWorkerType] = req.WorkerType.String()
			changed = true
		}
	}
	if !changed {
		return errortypes.NewValidationError(
			"driverType",
			errortypes.ErrInvalid,
			"Nothing changes — the worker already has that type",
		)
	}
	return nil
}
