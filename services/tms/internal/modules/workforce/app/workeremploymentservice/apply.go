package workeremploymentservice

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/employment"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

var errTrainingIncomplete = errors.New("some required courses could not be assigned")

func (s *Service) Record(ctx context.Context, req *RecordRequest) (*RecordResult, error) {
	plan, err := s.Plan(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.apply(ctx, plan)
}

func (s *Service) apply(ctx context.Context, plan *RecordPlan) (*RecordResult, error) {
	req := plan.Request
	log := s.l.With(
		zap.String("operation", "Record"),
		zap.String("workerId", req.WorkerID.String()),
		zap.String("kind", req.Kind.String()),
	)

	carried := make([]*employment.Effect, 0, len(plan.steps))
	carried = append(carried, s.runBestEffort(ctx, plan, phaseBefore, log)...)

	var atomic []*employment.Effect
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		var txErr error
		atomic, txErr = s.applyAtomic(txCtx, plan)
		return txErr
	})
	if err != nil {
		log.Error("failed to record employment event", zap.Error(err))
		return nil, err
	}
	carried = append(carried, atomic...)
	carried = append(carried, s.runBestEffort(ctx, plan, phaseAfter, log)...)

	return &RecordResult{Event: plan.Event, Effects: carried}, nil
}

func (s *Service) applyAtomic(ctx context.Context, plan *RecordPlan) ([]*employment.Effect, error) {
	req := plan.Request
	if plan.Changed {
		if _, err := s.workers.UpdateFromEmployment(
			ctx,
			plan.Worker,
			s.actor(req.TenantInfo, req.UserID),
		); err != nil {
			return nil, fmt.Errorf("apply employment event to worker: %w", err)
		}
	}

	created, err := s.repo.Create(ctx, plan.Event)
	if err != nil {
		return nil, err
	}
	plan.Event = created

	carried := make([]*employment.Effect, 0, len(plan.steps))
	for _, st := range plan.steps {
		if st.phase != phaseAtomic {
			continue
		}
		if st.run == nil {
			carried = append(carried, st.effects...)
			continue
		}
		effects, runErr := st.run(ctx, plan)
		if runErr != nil {
			return nil, runErr
		}
		carried = append(carried, effects...)
	}

	if err = s.audit(
		ctx,
		created,
		nil,
		permission.OpCreate,
		req.UserID,
		"Employment event recorded",
	); err != nil {
		return nil, fmt.Errorf("audit employment event: %w", err)
	}
	s.publish(ctx, req.TenantInfo, created, permission.OpCreate, req.UserID)
	s.recordTeamChange(ctx, plan)
	return carried, nil
}

func (s *Service) runBestEffort(
	ctx context.Context,
	plan *RecordPlan,
	at phase,
	log *zap.Logger,
) []*employment.Effect {
	carried := make([]*employment.Effect, 0, len(plan.steps))
	for _, st := range plan.steps {
		if st.phase != at {
			continue
		}
		effects, err := st.run(ctx, plan)
		if err != nil {
			log.Warn("employment event effect failed",
				zap.String("effect", st.effects[0].Kind.String()),
				zap.Error(err))
			for _, effect := range st.effects {
				effect.Failed = true
			}
			effects = st.effects
		}
		carried = append(carried, effects...)
	}
	return carried
}

func (s *Service) recordTeamChange(ctx context.Context, plan *RecordPlan) {
	if plan.team == nil {
		return
	}
	req := plan.Request
	change := &permissionports.SecurityChange{
		Resource:       permission.ResourceWorker,
		ResourceID:     req.WorkerID.String(),
		Operation:      permission.OpUpdate,
		Action:         teamChangeAction,
		Actor:          s.actor(req.TenantInfo, req.UserID).AuditActor(),
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Before:         plan.team.before,
		After:          plan.team.after,
		Comment:        "Worker transferred",
		Metadata:       map[string]any{"employmentEventId": plan.Event.ID.String()},
	}
	kernelports.AfterCommit(ctx, func(committed context.Context) {
		s.security.RecordChange(committed, change)
	})
}

func (s *Service) RecordHired(ctx context.Context, wrk *worker.Worker, userID pulid.ID) error {
	if wrk == nil || wrk.Profile == nil || wrk.Profile.HireDate <= 0 {
		return nil
	}
	tenantInfo := pagination.TenantInfo{OrgID: wrk.OrganizationID, BuID: wrk.BusinessUnitID}
	log := s.l.With(
		zap.String("operation", "RecordHired"),
		zap.String("workerId", wrk.ID.String()),
	)
	event := &employment.WorkerEmploymentEvent{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		WorkerID:       wrk.ID,
		Kind:           employment.EmploymentEventHired,
		EffectiveAt:    wrk.Profile.HireDate,
		RecordedByID:   userID,
		FromValues:     map[string]string{},
		ToValues: map[string]string{
			employment.EmploymentValueHireDate: strconv.FormatInt(wrk.Profile.HireDate, 10),
		},
	}
	if !wrk.FleetCodeID.IsNil() {
		event.ToValues[employment.EmploymentValueFleetCodeID] = wrk.FleetCodeID.String()
	}
	plan := &RecordPlan{
		Request: &RecordRequest{
			TenantInfo:  tenantInfo,
			WorkerID:    wrk.ID,
			Kind:        employment.EmploymentEventHired,
			EffectiveAt: wrk.Profile.HireDate,
			UserID:      userID,
		},
		Event:  event,
		Stored: wrk,
		Worker: wrk,
	}
	if err := s.planFollowOns(ctx, plan); err != nil {
		log.Warn("failed to plan what the hire starts", zap.Error(err))
		plan.steps = nil
	}

	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		created, createErr := s.repo.Create(txCtx, event)
		if createErr != nil {
			return createErr
		}
		plan.Event = created
		if auditErr := s.audit(
			txCtx,
			created,
			nil,
			permission.OpCreate,
			userID,
			"Employment event recorded",
		); auditErr != nil {
			return fmt.Errorf("audit hire event: %w", auditErr)
		}
		s.publish(txCtx, tenantInfo, created, permission.OpCreate, userID)
		return nil
	})
	if err != nil {
		return err
	}
	s.runBestEffort(ctx, plan, phaseAfter, log)
	return nil
}
