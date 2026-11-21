package tenantlifecycle

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
)

type RestoreRequest struct {
	CommandID    string
	ExpectSHA256 string
	Access       *tenantarchive.Access
}

func (s *Service) StartRestore(
	ctx context.Context,
	instanceID string,
	req *RestoreRequest,
) (*tenantcommand.View, error) {
	if !s.shared {
		return nil, ErrNotShared
	}
	if !tenantarchive.ValidCommandID(req.CommandID) {
		return nil, ErrCommandIDInvalid
	}
	if !tenantarchive.ValidSHA256(req.ExpectSHA256) {
		return nil, ErrExpectInvalid
	}
	if err := req.Access.CheckRead(instanceID); err != nil {
		return nil, err
	}
	if err := s.requireServerBucket(ctx, &req.Access.Location); err != nil {
		return nil, err
	}
	var (
		cmd    *tenantcommand.Command
		tenant *cloudtenant.Tenant
	)
	err := s.inTenantTx(ctx, func(ctx context.Context) error {
		t, err := s.lockedTenant(ctx, instanceID)
		if err != nil {
			return err
		}
		tenant = t
		if held, gerr := s.commands.Get(ctx, req.CommandID); gerr == nil {
			if held.InstanceID != instanceID || held.Kind != tenantcommand.KindRestore {
				return ErrCommandConflict
			}
			cmd = held
			return nil
		}
		if !t.Made() ||
			(t.State != cloudtenant.StateSuspended && t.State != cloudtenant.StateRestoring) {
			return ErrTenantState
		}
		if err = s.busy(ctx, instanceID, req.CommandID); err != nil {
			return err
		}
		if cmd, err = s.createCommand(ctx, &newCommand{
			id:       req.CommandID,
			instance: instanceID,
			kind:     tenantcommand.KindRestore,
			expect:   req.ExpectSHA256,
			access:   req.Access,
		}); err != nil {
			return err
		}
		return s.markRestoring(ctx, t, cmd)
	})
	if err != nil {
		return nil, err
	}
	s.startCommand(ctx, cmd, tenant)
	return cmd.View(), nil
}

func (s *Service) markRestoring(
	ctx context.Context,
	t *cloudtenant.Tenant,
	cmd *tenantcommand.Command,
) error {
	scope := managedscope.Organization(tenantOf(t))
	if err := s.features.SetLifecycle(ctx, scope, managedfeatures.LifecycleSuspended); err != nil {
		return fmt.Errorf("suspend the tenant before its restore: %w", err)
	}
	now := s.clock()
	t.State = cloudtenant.StateRestoring
	t.LastRestoreCommandID = cmd.CommandID
	t.LastRestoreState = cloudtenant.RestoreRunning
	t.LastRestoreAt = now
	t.LastRestoreReason = ""
	t.LastRestoreNote = ""
	t.UpdatedAt = now
	if err := s.tenants.Save(ctx, t); err != nil {
		return err
	}
	return s.tenants.Record(ctx, &cloudtenant.Event{
		InstanceID: t.InstanceID,
		Kind:       cloudtenant.EventRestoring,
		Revision:   t.Desired.Revision,
		OccurredAt: now,
	})
}

type RestoreResult struct {
	CommandID string
	SHA256    string
	Rows      int64
	Documents int
	Note      string
}

func (s *Service) FinishRestore(ctx context.Context, result *RestoreResult) error {
	var t *cloudtenant.Tenant
	finished := false
	err := s.inTenantTx(ctx, func(ctx context.Context) error {
		cmd, err := s.commands.Get(ctx, result.CommandID)
		if err != nil {
			return err
		}
		if cmd.State.Settled() {
			return nil
		}
		if t, err = s.lockedTenant(ctx, cmd.InstanceID); err != nil {
			return err
		}
		now := s.clock()
		received, refusal := receivedOf(&t.Desired, now)
		if refusal != nil {
			return fmt.Errorf(
				"the tenant's held entry cannot be applied after its restore: %s",
				refusal.Reason,
			)
		}
		scope := managedscope.Organization(tenantOf(t))
		if err = s.features.SetFeatures(ctx, scope, received); err != nil {
			return fmt.Errorf("set the restored tenant's plan: %w", err)
		}
		if err = s.features.SetLifecycle(
			ctx,
			scope,
			managedfeatures.LifecycleSuspended,
		); err != nil {
			return fmt.Errorf("hold the restored tenant suspended: %w", err)
		}
		state := cloudtenant.StateSuspended
		t.State = state
		t.LastRestoreState = cloudtenant.RestoreDone
		t.LastRestoreAt = now
		t.LastRestoreReason = ""
		t.LastRestoreNote = result.Note
		t.UpdatedAt = now
		if err = s.tenants.Save(ctx, t); err != nil {
			return err
		}
		cmd.SHA256 = result.SHA256
		cmd.Progress.Phase = tenantcommand.PhaseFinished
		if err = s.settle(ctx, cmd, tenantcommand.StateDone, nil); err != nil {
			return err
		}
		finished = true
		return s.tenants.Record(ctx, &cloudtenant.Event{
			InstanceID: t.InstanceID,
			Kind:       cloudtenant.EventRestored,
			Revision:   t.Desired.Revision,
			Detail: map[string]any{
				"rows": result.Rows, "documents": result.Documents, "note": result.Note,
			},
			OccurredAt: now,
		})
	})
	if err != nil {
		return err
	}
	if finished {
		s.auditLifecycle(
			ctx,
			t,
			"organization.restored",
			"The Trenova control plane restored the organization from an archive; it stays suspended until the control plane resumes it",
		)
	}
	return nil
}

const (
	RestoreUntouched  = "Nothing was changed: "
	RestoreIncomplete = "The tenant is incomplete: "
)

func (s *Service) FailRestore(
	ctx context.Context,
	commandID string,
	refusal *tenantarchive.RefusalError,
	touched bool,
) error {
	return s.inTenantTx(ctx, func(ctx context.Context) error {
		cmd, err := s.commands.Get(ctx, commandID)
		if err != nil {
			return err
		}
		if cmd.State.Settled() {
			return nil
		}
		t, err := s.lockedTenant(ctx, cmd.InstanceID)
		if err != nil {
			return err
		}
		scope := managedscope.Organization(tenantOf(t))
		if err = s.features.SetLifecycle(
			ctx,
			scope,
			managedfeatures.LifecycleSuspended,
		); err != nil {
			return fmt.Errorf("keep the tenant suspended after a failed restore: %w", err)
		}
		now := s.clock()
		prefix := RestoreUntouched
		t.State = cloudtenant.StateSuspended
		if touched {
			prefix, t.State = RestoreIncomplete, cloudtenant.StateRestoring
		}
		t.LastRestoreState = cloudtenant.RestoreFailed
		t.LastRestoreAt = now
		t.LastRestoreReason = cloudtenant.Invalid(t.Desired.Revision, prefix+refusal.Reason).Reason
		t.LastRestoreNote = ""
		t.UpdatedAt = now
		if err = s.tenants.Save(ctx, t); err != nil {
			return err
		}
		return s.settle(ctx, cmd, tenantcommand.StateFailed, refusal)
	})
}

func (s *Service) Restoring(ctx context.Context, commandID string) (*cloudtenant.Tenant, error) {
	cmd, err := s.commands.Get(ctx, commandID)
	if err != nil {
		return nil, err
	}
	t, err := s.tenants.Get(ctx, cmd.InstanceID)
	if err != nil {
		return nil, err
	}
	if cmd.State.Settled() || t.State != cloudtenant.StateRestoring ||
		t.LastRestoreCommandID != commandID {
		return nil, ErrTenantState
	}
	return t, nil
}
