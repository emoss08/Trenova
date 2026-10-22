package workerleaveresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
)

func (d *Deps) requireLeaveTeam(
	ctx context.Context,
	operation permission.Operation,
	workerID pulid.ID,
) error {
	_, _, err := d.RequireTeamScope(
		ctx,
		permission.ResourceWorkerLeave,
		operation,
		worker.ApprovalScopeTimeOff,
		workerID,
	)
	return err
}

func (d *Deps) requireCaseTeam(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	operation permission.Operation,
	caseID pulid.ID,
) error {
	workerID, err := d.WorkerLeaveService.CaseWorker(ctx, base.TenantInfo(authCtx), caseID)
	if err != nil {
		return err
	}
	return d.requireLeaveTeam(ctx, operation, workerID)
}

func (d *Deps) requireEntryTeam(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	operation permission.Operation,
	entryID pulid.ID,
) error {
	workerID, err := d.WorkerLeaveService.EntryWorker(ctx, base.TenantInfo(authCtx), entryID)
	if err != nil {
		return err
	}
	return d.requireLeaveTeam(ctx, operation, workerID)
}
