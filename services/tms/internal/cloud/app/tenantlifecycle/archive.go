package tenantlifecycle

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/google/uuid"
)

const (
	nightlyPage    = 100
	maxNightlyPage = 500
)

type ArchiveRequest struct {
	CommandID string
	Purpose   tenantarchive.Purpose
	Access    *tenantarchive.Access
}

func (s *Service) StartArchive(
	ctx context.Context,
	instanceID string,
	req *ArchiveRequest,
) (*tenantcommand.View, error) {
	if !s.shared {
		return nil, ErrNotShared
	}
	if !tenantarchive.ValidCommandID(req.CommandID) {
		return nil, ErrCommandIDInvalid
	}
	if !req.Purpose.Commanded() {
		return nil, ErrPurposeInvalid
	}
	if err := req.Access.CheckWrite(instanceID, req.Purpose); err != nil {
		return nil, err
	}
	if req.Access.Location.Prefix != tenantarchive.CommandPrefix(instanceID, req.CommandID) {
		return nil, tenantarchive.ErrLocationInvalid
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
			if held.InstanceID != instanceID || held.Kind != tenantcommand.KindArchive ||
				held.Purpose != string(req.Purpose) {
				return ErrCommandConflict
			}
			cmd = held
			return nil
		}
		if err = archivable(t, req.Purpose); err != nil {
			return err
		}
		if err = s.busy(ctx, instanceID, req.CommandID); err != nil {
			return err
		}
		cmd, err = s.createCommand(ctx, &newCommand{
			id:       req.CommandID,
			instance: instanceID,
			kind:     tenantcommand.KindArchive,
			purpose:  req.Purpose,
			access:   req.Access,
		})
		return err
	})
	if err != nil {
		return nil, err
	}
	s.startCommand(ctx, cmd, tenant)
	return cmd.View(), nil
}

func archivable(t *cloudtenant.Tenant, purpose tenantarchive.Purpose) error {
	if !t.Made() {
		return ErrTenantState
	}
	switch t.State {
	case cloudtenant.StateSuspended:
		return nil
	case cloudtenant.StateActive:
		if purpose == tenantarchive.PurposeExport {
			return nil
		}
		return ErrTenantState
	case cloudtenant.StateApplying, cloudtenant.StateRestoring, cloudtenant.StateRemoving,
		cloudtenant.StateRemoved:
		return ErrTenantState
	default:
		return ErrTenantState
	}
}

type ArchiveResult struct {
	CommandID string
	Location  string
	SHA256    string
	SizeBytes int64
	TakenAt   time.Time
}

func (s *Service) FinishArchive(ctx context.Context, result *ArchiveResult) error {
	return s.inTenantTx(ctx, func(ctx context.Context) error {
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
		cmd.SHA256, cmd.SizeBytes, cmd.TakenAt = result.SHA256, result.SizeBytes, result.TakenAt.Unix()
		cmd.Progress.Phase = tenantcommand.PhaseFinished
		if err = s.settle(ctx, cmd, tenantcommand.StateDone, nil); err != nil {
			return err
		}
		if tenantarchive.Purpose(cmd.Purpose) == tenantarchive.PurposeNightly {
			return nil
		}
		now := s.clock()
		t.LastArchiveCommandID = cmd.CommandID
		t.LastArchivePurpose = cmd.Purpose
		t.LastArchiveLocation = result.Location
		t.LastArchiveSizeBytes = result.SizeBytes
		t.LastArchiveSHA256 = result.SHA256
		t.LastArchiveTakenAt = result.TakenAt.Unix()
		t.UpdatedAt = now
		if err = s.tenants.Save(ctx, t); err != nil {
			return err
		}
		return s.tenants.Record(ctx, &cloudtenant.Event{
			InstanceID: t.InstanceID,
			Kind:       cloudtenant.EventArchived,
			Revision:   t.Desired.Revision,
			Detail:     map[string]any{"purpose": cmd.Purpose, "sizeBytes": result.SizeBytes},
			OccurredAt: now,
		})
	})
}

func (s *Service) FailCommand(
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
		return s.settle(ctx, cmd, tenantcommand.StateFailed, refusal)
	})
}

func (s *Service) SetNightlyGrant(
	ctx context.Context,
	access *tenantarchive.Access,
	expiresAt time.Time,
) error {
	if !s.shared {
		return ErrNotShared
	}
	if expiresAt.IsZero() {
		expiresAt = time.Unix(s.clock(), 0).AddDate(grantYears, 0, 0)
	}
	if expiresAt.Unix() <= s.clock() {
		return ErrGrantExpiry
	}
	location := access.Location
	location.Prefix = tenantarchive.TenantPrefix(nightlyProbe)
	probe := *access
	probe.Location = location
	if err := probe.CheckWrite(nightlyProbe, tenantarchive.PurposeNightly); err != nil {
		return err
	}
	sealed, err := s.sealAccess(ctx, grantResource+tenantcommand.NightlyGrant, access)
	if err != nil {
		return err
	}
	return s.commands.SetGrant(ctx, &tenantcommand.Grant{
		Name:         tenantcommand.NightlyGrant,
		Location:     "s3://" + location.Bucket,
		SealedAccess: sealed,
		ExpiresAt:    expiresAt.Unix(),
		UpdatedAt:    s.clock(),
	})
}

func (s *Service) NightlyAccess(ctx context.Context) (*tenantarchive.Access, error) {
	grant, err := s.commands.Grant(ctx, tenantcommand.NightlyGrant)
	if err != nil {
		return nil, err
	}
	if grant.ExpiresAt <= s.clock() {
		return nil, cloudtenantports.ErrGrantMissing
	}
	return s.openAccess(ctx, grantResource+tenantcommand.NightlyGrant, grant.SealedAccess)
}

type NightlyTarget struct {
	InstanceID     string
	OrganizationID pulid.ID
	BusinessUnitID pulid.ID
	AdminUserID    pulid.ID
	RetentionDays  int64
}

func (t *NightlyTarget) Tenant() dbscope.Tenant {
	return dbscope.Tenant{OrganizationID: t.OrganizationID, BusinessUnitID: t.BusinessUnitID}
}

type NightlyPage struct {
	Targets []NightlyTarget
	Next    string
}

func (s *Service) NightlyTargets(
	ctx context.Context,
	after string,
	limit int,
) (*NightlyPage, error) {
	if limit <= 0 {
		limit = nightlyPage
	}
	page, err := s.tenants.List(
		ctx,
		&cloudtenantports.ListRequest{After: after, Limit: min(limit, maxNightlyPage)},
	)
	if err != nil {
		return nil, err
	}
	out := &NightlyPage{Next: page.Next, Targets: make([]NightlyTarget, 0, len(page.Tenants))}
	for _, t := range page.Tenants {
		if !t.Made() || t.State != cloudtenant.StateActive {
			continue
		}
		out.Targets = append(out.Targets, NightlyTarget{
			InstanceID:     t.InstanceID,
			OrganizationID: t.OrganizationID,
			BusinessUnitID: t.BusinessUnitID,
			AdminUserID:    t.AdminUserID,
			RetentionDays:  t.Desired.Plan.BackupRetentionDays,
		})
	}
	return out, nil
}

func (s *Service) NightlyTarget(ctx context.Context, instanceID string) (*NightlyTarget, error) {
	t, err := s.tenants.Get(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if !t.Made() || t.State != cloudtenant.StateActive {
		return nil, ErrTenantState
	}
	return &NightlyTarget{
		InstanceID:     t.InstanceID,
		OrganizationID: t.OrganizationID,
		BusinessUnitID: t.BusinessUnitID,
		AdminUserID:    t.AdminUserID,
		RetentionDays:  t.Desired.Plan.BackupRetentionDays,
	}, nil
}

func (s *Service) TenantFor(ctx context.Context, instanceID string) (*cloudtenant.Tenant, error) {
	return s.tenants.Get(ctx, instanceID)
}

const grantYears = 10

const nightlyProbe = "00000000-0000-4000-8000-000000000000"

var nightlyNamespace = uuid.MustParse("6f6b8a52-3a1e-4f0c-9d2b-0c7b3e1f9a10")

func NightlyCommandID(instanceID string, day time.Time) string {
	return uuid.NewSHA1(nightlyNamespace, []byte(instanceID+"/"+day.UTC().Format(time.DateOnly))).
		String()
}

func (s *Service) BeginNightly(
	ctx context.Context,
	target *NightlyTarget,
	day time.Time,
) (*tenantcommand.Command, *tenantarchive.Access, error) {
	grant, err := s.NightlyAccess(ctx)
	if err != nil {
		return nil, nil, err
	}
	grant.Location.Prefix = tenantarchive.NightlyPrefix(target.InstanceID, day)
	if err = grant.CheckWrite(target.InstanceID, tenantarchive.PurposeNightly); err != nil {
		return nil, nil, err
	}
	cmd, err := s.createCommand(ctx, &newCommand{
		id:       NightlyCommandID(target.InstanceID, day),
		instance: target.InstanceID,
		kind:     tenantcommand.KindArchive,
		purpose:  tenantarchive.PurposeNightly,
		access:   grant,
	})
	if err != nil {
		return nil, nil, err
	}
	if cmd.State.Settled() {
		return cmd, nil, nil
	}
	if err = s.BeginCommand(ctx, cmd.CommandID); err != nil {
		return nil, nil, err
	}
	return cmd, grant, nil
}
