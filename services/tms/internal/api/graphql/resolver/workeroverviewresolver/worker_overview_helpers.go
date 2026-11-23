package workeroverviewresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workeroverviewservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/approvaldelegation"
	"github.com/emoss08/trenova/pkg/authctx"
)

func (r *Deps) OverviewSections(
	ctx context.Context,
	authCtx *authctx.AuthContext,
) workeroverviewservice.Sections {
	can := func(resource permission.Resource) bool {
		return r.HasPermission(ctx, authCtx, resource, permission.OpRead)
	}
	sections := workeroverviewservice.Sections{
		Credentials:    can(permission.ResourceWorkerCredential),
		Qualification:  can(permission.ResourceQualification),
		DrugAlcohol:    can(permission.ResourceWorkerDOTTest),
		Training:       can(permission.ResourceWorkerTraining),
		Safety:         can(permission.ResourceWorkerSafetyEvent),
		Discipline:     can(permission.ResourceWorkerDisciplinaryAction),
		Employment:     can(permission.ResourceWorkerEmploymentEvent),
		Checklists:     can(permission.ResourceWorkerChecklist),
		PTO:            can(permission.ResourceWorkerPTO),
		Leave:          can(permission.ResourceWorkerLeave),
		Reviews:        can(permission.ResourcePerformanceReview),
		Schedule:       can(permission.ResourceWorkerSchedule),
		HOS:            true,
		Tractor:        can(permission.ResourceTractor),
		Pay:            can(permission.ResourceDriverPayProfile),
		Benefits:       can(permission.ResourceBenefitPlan),
		Portal:         can(permission.ResourceDriverPortal),
		ProfileChanges: can(permission.ResourceProfileChangeRequest),
		Documents:      can(permission.ResourceDocument),
		AgentMemory:    can(permission.ResourceAssistant),
	}
	sections.ProfileChanges = sections.ProfileChanges && sections.Portal
	return sections
}

func (r *Deps) OverviewMemoryActor(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	sections workeroverviewservice.Sections,
) *agentmemoryports.DeskMemoryActor {
	actor := actorutil.FromAuthContext(authCtx)
	if !sections.AgentMemory || !actor.IsUser() || actor.UserID.IsNil() {
		return nil
	}
	return &agentmemoryports.DeskMemoryActor{
		Actor: actor,
		MayManageShared: r.HasPermission(
			ctx,
			authCtx,
			permission.ResourceAgentMemory,
			permission.OpManage,
		),
	}
}

// AttentionAccess is what the home tile may count for the reader. The PTO
// liability and the overdue training count each follow their own resource, and
// the permission engine refuses both when the plan leaves out their HR Suite
// feature. Leave is
// counted across their team when their leave grant reaches only their team.
func (r *Deps) AttentionAccess(
	ctx context.Context,
	authCtx *authctx.AuthContext,
) (workeroverviewservice.AttentionAccess, error) {
	access := workeroverviewservice.AttentionAccess{
		Reviews: r.HasPermission(
			ctx,
			authCtx,
			permission.ResourcePerformanceReview,
			permission.OpRead,
		),
		Leave: r.HasPermission(ctx, authCtx, permission.ResourceWorkerLeave, permission.OpRead),
		PTO:   r.HasPermission(ctx, authCtx, permission.ResourceWorkerPTO, permission.OpRead),
		Training: r.HasPermission(
			ctx,
			authCtx,
			permission.ResourceWorkerTraining,
			permission.OpRead,
		),
	}
	if !access.Leave {
		return access, nil
	}
	_, managerIDs, err := r.RequireListTeamScope(
		ctx,
		permission.ResourceWorkerLeave,
		permission.OpRead,
		approvaldelegation.ApprovalScopeTimeOff,
	)
	if err != nil {
		return access, err
	}
	access.LeaveManagerIDs = managerIDs
	return access, nil
}
