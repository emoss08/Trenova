package agentqualityjobs

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/temporaljobs/schedule"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	sweepCatchupWindow = time.Hour

	reconcileEvery       = 15 * time.Minute
	reconcileDescription = "Keep one nightly quality sweep schedule behind every organization " +
		"that wants one"

	reconcilePageSize   = 200
	scheduleCallTimeout = 10 * time.Second
)

type QualitySchedulesParams struct {
	fx.In

	Client   client.Client
	Controls repositories.AgentQualityControlRepository
	Logger   *zap.Logger
}

type QualitySchedules struct {
	client   client.Client
	controls repositories.AgentQualityControlRepository
	l        *zap.Logger
}

var (
	_ serviceports.AgentQualityScheduler = (*QualitySchedules)(nil)
	_ serviceports.AgentSuiteStarter     = (*QualitySchedules)(nil)
)

func NewQualitySchedules(p QualitySchedulesParams) *QualitySchedules {
	return &QualitySchedules{
		client:   p.Client,
		controls: p.Controls,
		l:        p.Logger.Named("agent-quality-schedules"),
	}
}

func Wanted(target *repositories.QualityScheduleTarget) bool {
	if !target.HasActiveCases {
		return false
	}

	return target.Control == nil || target.Control.Enabled
}

func sweepControl(target *repositories.QualityScheduleTarget) *agentquality.Control {
	if target.Control != nil {
		return target.Control
	}

	return agentquality.DefaultControl(target.OrganizationID, target.BusinessUnitID)
}

func SweepScheduleOptions(target *repositories.QualityScheduleTarget) client.ScheduleOptions {
	control := sweepControl(target)
	timezone := control.EffectiveTimezone(target.OrganizationTimezone)

	return client.ScheduleOptions{
		ID: agentquality.SweepScheduleID(target.OrganizationID),
		Spec: client.ScheduleSpec{
			CronExpressions: []string{"0 " + strconv.Itoa(control.RunHourLocal) + " * * *"},
			TimeZoneName:    timezone,
		},
		Overlap:       enums.SCHEDULE_OVERLAP_POLICY_SKIP,
		CatchupWindow: sweepCatchupWindow,
		Note:          "Nightly agent quality sweep for " + target.OrganizationName,
		Memo:          map[string]any{schedule.ManagedByMemoKey: qualityScheduleOwner},
		Action: &client.ScheduleWorkflowAction{
			ID:        SweepWorkflowID(target.OrganizationID),
			Workflow:  AgentQualitySweepWorkflowName,
			TaskQueue: temporaltype.TaskQueueAgentHeavy.String(),
			Args: []any{&SweepPayload{
				OrganizationID: target.OrganizationID,
				BusinessUnitID: target.BusinessUnitID,
			}},
			Priority: evaluationPriority(target.OrganizationID),
		},
	}
}

func (s *QualitySchedules) Sync(ctx context.Context, tenant pagination.TenantInfo) {
	callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), scheduleCallTimeout)
	defer cancel()

	target, err := s.controls.ScheduleTarget(callCtx, tenant)
	if err == nil {
		err = s.sync(callCtx, target)
	}
	if err != nil {
		s.l.Warn("an organization's quality sweep schedule could not be updated; "+
			"the next reconcile will",
			zap.String("organization", tenant.OrgID.String()),
			zap.Error(err),
		)
	}
}

func (s *QualitySchedules) sync(
	ctx context.Context,
	target *repositories.QualityScheduleTarget,
) error {
	id := agentquality.SweepScheduleID(target.OrganizationID)
	if !Wanted(target) {
		return s.remove(ctx, id)
	}

	options := SweepScheduleOptions(target)
	_, err := s.client.ScheduleClient().Create(ctx, options)
	if err == nil {
		return nil
	}
	if !scheduleExists(err) {
		return fmt.Errorf("create schedule %s: %w", id, err)
	}

	return s.client.ScheduleClient().GetHandle(ctx, id).Update(ctx, client.ScheduleUpdateOptions{
		DoUpdate: func(input client.ScheduleUpdateInput) (*client.ScheduleUpdate, error) {
			desired := input.Description.Schedule
			desired.Spec = &options.Spec
			desired.Action = options.Action
			if desired.Policy == nil {
				desired.Policy = &client.SchedulePolicies{}
			}
			desired.Policy.Overlap = options.Overlap
			desired.Policy.CatchupWindow = options.CatchupWindow
			if desired.State == nil {
				desired.State = &client.ScheduleState{}
			}
			desired.State.Paused = false
			desired.State.Note = options.Note

			return &client.ScheduleUpdate{Schedule: &desired}, nil
		},
	})
}

func (s *QualitySchedules) remove(ctx context.Context, id string) error {
	err := s.client.ScheduleClient().GetHandle(ctx, id).Delete(ctx)
	var notFound *serviceerror.NotFound
	if err != nil && !errors.As(err, &notFound) {
		return fmt.Errorf("delete schedule %s: %w", id, err)
	}

	return nil
}

func scheduleExists(err error) bool {
	if errors.Is(err, temporal.ErrScheduleAlreadyRunning) {
		return true
	}

	var exists *serviceerror.AlreadyExists

	return errors.As(err, &exists)
}

func (s *QualitySchedules) Reconcile(ctx context.Context) (*ReconcileResult, error) {
	result := &ReconcileResult{}
	wanted := make(map[string]struct{})

	after := pulid.Nil
	for {
		page, err := s.controls.ListScheduleTargets(
			ctx,
			repositories.ListQualityScheduleTargetsRequest{
				AfterOrganizationID: after,
				Limit:               reconcilePageSize,
			},
		)
		if err != nil {
			return result, fmt.Errorf("list organizations to schedule: %w", err)
		}

		for _, target := range page {
			if Wanted(target) {
				wanted[agentquality.SweepScheduleID(target.OrganizationID)] = struct{}{}
			}
			if err = s.sync(ctx, target); err != nil {
				result.Failed++
				s.l.Warn("an organization's quality sweep schedule could not be reconciled",
					zap.String("organization", target.OrganizationID.String()),
					zap.Error(err),
				)

				continue
			}
			result.Synced++
		}

		if len(page) < reconcilePageSize {
			break
		}
		after = page[len(page)-1].OrganizationID
	}

	iter, err := s.client.ScheduleClient().List(ctx, client.ScheduleListOptions{PageSize: 100})
	if err != nil {
		return result, fmt.Errorf("list schedules: %w", err)
	}
	for iter.HasNext() {
		entry, nextErr := iter.Next()
		if nextErr != nil {
			return result, fmt.Errorf("iterate schedules: %w", nextErr)
		}
		if !strings.HasPrefix(entry.ID, agentquality.SweepScheduleIDPrefix()) {
			continue
		}
		if _, keep := wanted[entry.ID]; keep {
			continue
		}
		if err = s.remove(ctx, entry.ID); err != nil {
			result.Failed++
			s.l.Warn("an orphaned quality sweep schedule could not be removed",
				zap.String("schedule", entry.ID),
				zap.Error(err),
			)

			continue
		}
		result.Removed++
	}

	return result, nil
}

func (s *QualitySchedules) StartSuiteRun(
	ctx context.Context,
	start *serviceports.AgentSuiteRunStart,
) (string, error) {
	workflowID := agentquality.SuiteRunWorkflowID(start.SuiteRunID)
	if _, err := s.client.ExecuteWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    workflowID,
		TaskQueue:             temporaltype.TaskQueueAgentHeavy.String(),
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		Priority:              evaluationPriority(start.TenantInfo.OrgID),
	}, AgentSuiteRunWorkflowName, &SuiteRunPayload{
		OrganizationID:    start.TenantInfo.OrgID,
		BusinessUnitID:    start.TenantInfo.BuID,
		UserID:            start.UserID,
		SuiteRunID:        start.SuiteRunID,
		AgentDefinitionID: start.AgentDefinitionID,
		SampleSeed:        start.SampleSeed,
		Settings:          start.Settings,
		DayStart:          start.DayStart,
		MonthStart:        start.MonthStart,
	}); err != nil {
		return "", fmt.Errorf("start suite run workflow: %w", err)
	}

	return workflowID, nil
}

type ScheduleProvider struct{}

func NewScheduleProvider() *ScheduleProvider {
	return &ScheduleProvider{}
}

func (p *ScheduleProvider) GetSchedules() []*schedule.Schedule {
	return []*schedule.Schedule{
		{
			ID:            ReconcileQualitySchedulesScheduleID,
			Description:   reconcileDescription,
			Spec:          schedule.Every(reconcileEvery),
			Workflow:      ReconcileQualitySchedulesWorkflow,
			TaskQueue:     temporaltype.TaskQueueAgentHeavy.String(),
			OverlapPolicy: enums.SCHEDULE_OVERLAP_POLICY_SKIP,
			Memo: map[string]any{
				"purpose": ReconcileQualitySchedulesScheduleID,
			},
		},
	}
}
