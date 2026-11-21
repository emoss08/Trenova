package tenantlifecycle

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/jobs/archivejobs/archivejobscontract"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"go.temporal.io/api/enums/v1"
)

const (
	commandReason = "record one shared-server tenant's archive, restore or import command in the server's bookkeeping"
	grantResource = "grant:"
)

var (
	ErrCommandIDInvalid = errors.New("commandId must be a lowercase UUID")
	ErrPurposeInvalid   = errors.New("purpose must be final, export or move")
	ErrExpectInvalid    = errors.New(
		"expectSha256 must be the archive manifest's lowercase hex SHA-256",
	)
	ErrCommandConflict = errors.New(
		"the command id is already used for another tenant or another kind of command",
	)
	ErrTenantBusy = errors.New(
		"another archive, restore or import of the tenant is running; send this command when it settles",
	)
	ErrTenantState = errors.New("the tenant's state does not allow this command")
	ErrGrantExpiry = errors.New("expiresAt must be in the future")
)

func (s *Service) sealAccess(
	ctx context.Context,
	resource string,
	access *tenantarchive.Access,
) (string, error) {
	body, err := sonic.ConfigStd.Marshal(access)
	if err != nil {
		return "", fmt.Errorf("encode the archive access: %w", err)
	}
	sealed, err := s.secrets.Seal(ctx, string(body), &kernelports.SecretScope{
		Purpose:    kernelports.SecretPurposeArchiveAccess,
		ResourceID: resource,
	})
	if err != nil {
		return "", fmt.Errorf("seal the archive access: %w", err)
	}
	return sealed, nil
}

func (s *Service) openAccess(
	ctx context.Context,
	resource, sealed string,
) (*tenantarchive.Access, error) {
	body, err := s.secrets.Open(ctx, sealed, &kernelports.SecretScope{
		Purpose:    kernelports.SecretPurposeArchiveAccess,
		ResourceID: resource,
	})
	if err != nil {
		return nil, fmt.Errorf("open the archive access: %w", err)
	}
	access := new(tenantarchive.Access)
	if err = sonic.ConfigStd.UnmarshalFromString(body, access); err != nil {
		return nil, fmt.Errorf("decode the archive access: %w", err)
	}
	return access, nil
}

type newCommand struct {
	id       string
	instance string
	kind     tenantcommand.Kind
	purpose  tenantarchive.Purpose
	expect   string
	access   *tenantarchive.Access
}

func (s *Service) createCommand(
	ctx context.Context,
	in *newCommand,
) (*tenantcommand.Command, error) {
	sealed, err := s.sealAccess(ctx, in.id, in.access)
	if err != nil {
		return nil, err
	}
	now := s.clock()
	cmd, _, err := s.commands.Create(ctx, &tenantcommand.Command{
		CommandID:    in.id,
		InstanceID:   in.instance,
		Kind:         in.kind,
		Purpose:      string(in.purpose),
		State:        tenantcommand.StateQueued,
		Location:     in.access.Location.String(),
		Prefix:       in.access.Location.Prefix,
		ExpectSHA256: in.expect,
		SealedAccess: sealed,
		Progress:     tenantcommand.Progress{Phase: tenantcommand.PhaseQueued},
		RequestedAt:  now,
		UpdatedAt:    now,
	})
	if err != nil {
		return nil, err
	}
	if cmd.InstanceID != in.instance || cmd.Kind != in.kind ||
		(in.kind == tenantcommand.KindArchive && cmd.Purpose != string(in.purpose)) {
		return nil, ErrCommandConflict
	}
	return cmd, nil
}

func (s *Service) busy(ctx context.Context, instanceID, except string) error {
	open, err := s.commands.Open(ctx, instanceID)
	if err != nil {
		return err
	}
	if slices.ContainsFunc(
		open,
		func(c *tenantcommand.Command) bool { return c.CommandID != except },
	) {
		return ErrTenantBusy
	}
	return nil
}

func (s *Service) startCommand(
	ctx context.Context,
	cmd *tenantcommand.Command,
	t *cloudtenant.Tenant,
) {
	if cmd == nil || cmd.State.Settled() {
		return
	}
	payload := &archivejobscontract.CommandPayload{
		CommandID:  cmd.CommandID,
		InstanceID: cmd.InstanceID,
	}
	if t != nil {
		payload.OrganizationID, payload.BusinessUnitID = t.OrganizationID, t.BusinessUnitID
	}
	s.start(ctx, &startRequest{
		id:       archivejobscontract.WorkflowID(cmd.Kind, cmd.CommandID),
		workflow: archivejobscontract.WorkflowName(cmd.Kind),
		reuse:    enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		args:     payload,
		summary:  "Run a shared-server tenant's " + string(cmd.Kind) + " command",
	})
}

func (s *Service) Command(
	ctx context.Context,
	instanceID, commandID string,
	kind tenantcommand.Kind,
) (*tenantcommand.View, error) {
	if !s.shared {
		return nil, ErrNotShared
	}
	if !tenantarchive.ValidCommandID(commandID) {
		return nil, ErrCommandIDInvalid
	}
	cmd, err := s.commands.Get(ctx, commandID)
	if err != nil {
		return nil, err
	}
	if cmd.InstanceID != instanceID || cmd.Kind != kind {
		return nil, cloudtenantports.ErrCommandNotFound
	}
	return cmd.View(), nil
}

func (s *Service) CommandAccess(
	ctx context.Context,
	commandID string,
) (*tenantcommand.Command, *tenantarchive.Access, error) {
	cmd, err := s.commands.Get(ctx, commandID)
	if err != nil {
		return nil, nil, err
	}
	if cmd.State.Settled() || cmd.SealedAccess == "" {
		return cmd, nil, nil
	}
	access, err := s.openAccess(ctx, cmd.CommandID, cmd.SealedAccess)
	if err != nil {
		return nil, nil, err
	}
	return cmd, access, nil
}

func (s *Service) BeginCommand(ctx context.Context, commandID string) error {
	cmd, err := s.commands.Get(ctx, commandID)
	if err != nil {
		return err
	}
	if cmd.State.Settled() || cmd.State == tenantcommand.StateRunning {
		return nil
	}
	now := s.clock()
	cmd.State = tenantcommand.StateRunning
	cmd.StartedAt = now
	cmd.UpdatedAt = now
	return s.commands.Save(ctx, cmd)
}

func (s *Service) ReportProgress(
	ctx context.Context,
	commandID string,
	progress *tenantcommand.Progress,
) error {
	return s.commands.Progress(ctx, commandID, progress)
}

func (s *Service) settle(
	ctx context.Context,
	cmd *tenantcommand.Command,
	state tenantcommand.State,
	refusal *tenantarchive.RefusalError,
) error {
	now := s.clock()
	cmd.State = state
	cmd.SealedAccess = ""
	cmd.FinishedAt = now
	cmd.UpdatedAt = now
	if refusal != nil {
		cmd.FailureCode = string(refusal.Code)
		cmd.Reason = refusal.Reason
	}
	return s.commands.Save(ctx, cmd)
}

func (s *Service) lockedTenant(
	ctx context.Context,
	instanceID string,
) (*cloudtenant.Tenant, error) {
	if err := s.tenants.Lock(ctx, instanceID); err != nil {
		return nil, err
	}
	return s.tenants.Get(ctx, instanceID)
}

func (s *Service) inTenantTx(
	ctx context.Context,
	fn func(ctx context.Context) error,
) error {
	return s.db.Transact(
		dbscope.WithSystem(ctx, commandReason),
		kernelports.TxOptions{LockTimeout: applyLockTimeout},
		fn,
	)
}

func Failure(err error) *tenantarchive.RefusalError {
	if refusal, ok := tenantarchive.RefusalOf(err); ok {
		return refusal
	}
	return tenantarchive.Refuse(
		tenantarchive.RefusalFailed,
		"the command could not be completed; the server's logs hold the cause",
	)
}
