package tenantlifecycle

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	reasonImportSource  = "an entry that imports the tenant must carry importSource: the archive's location, credentials and archive key"
	reasonImportPlace   = "the import's location is not the one its command names"
	reasonImportRunning = "another import of the tenant is running; its command settles before a new one starts"
	reasonRestoring     = "the tenant is being restored and cannot be removed until its restore settles"
)

func (s *Service) beginImport(
	ctx context.Context,
	t *cloudtenant.Tenant,
	in *applyInput,
	command *cloudtenant.Command,
	out *applied,
) error {
	refuse := func(reason string) error {
		return s.refuse(ctx, t, cloudtenant.Invalid(in.desired.Revision, reason), in.now)
	}
	source := in.source
	if source == nil {
		return refuse(reasonImportSource)
	}
	if err := source.CheckRead(t.InstanceID); err != nil {
		return refuse(err.Error())
	}
	if source.Location.String() != command.Location {
		return refuse(reasonImportPlace)
	}
	if err := s.requireServerBucket(ctx, &source.Location); err != nil {
		if refusal, ok := tenantarchive.RefusalOf(err); ok {
			return refuse(refusal.Reason)
		}
		return err
	}
	if err := s.busy(ctx, t.InstanceID, command.ID); errors.Is(err, ErrTenantBusy) {
		return refuse(reasonImportRunning)
	} else if err != nil {
		return err
	}
	cmd, err := s.createCommand(ctx, &newCommand{
		id:       command.ID,
		instance: t.InstanceID,
		kind:     tenantcommand.KindImport,
		expect:   command.ExpectSHA256,
		access:   source,
	})
	if errors.Is(err, ErrCommandConflict) {
		return refuse(err.Error())
	}
	if err != nil {
		return err
	}
	if cmd.State == tenantcommand.StateFailed {
		return nil
	}
	t.Desired = *in.desired
	t.State = cloudtenant.StateApplying
	t.Refuse(nil)
	if err = s.tenants.Save(ctx, t); err != nil {
		return err
	}
	out.importCmd = cmd
	return nil
}

type ImportResult struct {
	CommandID        string
	Tenant           dbscope.Tenant
	Owner            pulid.ID
	SHA256           string
	Rows             int64
	Documents        int
	RecoveryReplaced bool
}

func (s *Service) FinishImport(ctx context.Context, result *ImportResult) error {
	out := new(applied)
	err := s.inTenantTx(ctx, func(ctx context.Context) error {
		cmd, err := s.commands.Get(ctx, result.CommandID)
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
		now := s.clock()
		received, refusal := receivedOf(&t.Desired, now)
		if refusal != nil {
			return fmt.Errorf(
				"the tenant's held entry cannot be applied after its import: %s",
				refusal.Reason,
			)
		}
		t.OrganizationID = result.Tenant.OrganizationID
		t.BusinessUnitID = result.Tenant.BusinessUnitID
		t.AdminUserID = result.Owner
		t.ImportedAt = now
		out.tenant, out.previous = t, t.State
		if err = s.converge(ctx, t, &applyInput{
			instanceID: t.InstanceID,
			desired:    &t.Desired,
			digest:     t.DesiredDigest,
			received:   received,
			now:        now,
		}, out); err != nil {
			return err
		}
		cmd.SHA256 = result.SHA256
		cmd.Progress.Phase = tenantcommand.PhaseFinished
		if err = s.settle(ctx, cmd, tenantcommand.StateDone, nil); err != nil {
			return err
		}
		return s.tenants.Record(ctx, &cloudtenant.Event{
			InstanceID: t.InstanceID,
			Kind:       cloudtenant.EventImported,
			Revision:   t.Desired.Revision,
			Detail: map[string]any{
				"rows":             result.Rows,
				"documents":        result.Documents,
				"recoveryReplaced": result.RecoveryReplaced,
			},
			OccurredAt: now,
		})
	})
	if err != nil {
		return err
	}
	if out.tenant != nil && out.tenant.Made() {
		s.auditLifecycle(ctx, out.tenant, "organization.imported",
			"The Trenova control plane moved the organization onto this server from an archive")
		if out.startSessions {
			s.startSessions(ctx, out.tenant)
		}
	}
	return nil
}

func (s *Service) FailImport(
	ctx context.Context,
	commandID string,
	refusal *tenantarchive.RefusalError,
) error {
	return s.inTenantTx(ctx, func(ctx context.Context) error {
		cmd, err := s.commands.Get(ctx, commandID)
		if err != nil {
			return err
		}
		if cmd.State.Settled() {
			return nil
		}
		if err = s.settle(ctx, cmd, tenantcommand.StateFailed, refusal); err != nil {
			return err
		}
		return nil
	})
}
