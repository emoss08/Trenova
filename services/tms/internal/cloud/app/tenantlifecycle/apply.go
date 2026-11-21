package tenantlifecycle

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantarchive"
	"github.com/emoss08/trenova/internal/cloud/domain/tenantcommand"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/instancebootstrapports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedfeatures"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	applyReason     = "apply the control plane's entry for one shared-server tenant: make its business unit, organization and first administrator before any tenant scope exists, then set its plan"
	loginSlugLength = 60

	reasonIdentityFixed = "organizationName and the first administrator's name and email cannot change after the tenant is made"
	reasonEmailTaken    = "the first administrator's email address cannot be used on this server"
	reasonNoArchive     = "the server holds no final archive of the tenant, so it does not remove it; take an archive of purpose final first"
)

var errEmailTaken = errors.New("the first administrator's email address is taken on this server")

type applied struct {
	tenant        *cloudtenant.Tenant
	created       *tenantports.BootstrapTenantResult
	previous      cloudtenant.State
	startSessions bool
	startRemoval  bool
	importCmd     *tenantcommand.Command
}

func (s *Service) Apply(
	ctx context.Context,
	instanceID string,
	desired *cloudtenant.Desired,
	source *tenantarchive.Access,
) (*cloudtenant.Status, error) {
	if !s.shared {
		return nil, ErrNotShared
	}
	if err := desired.CheckAddress(instanceID); err != nil {
		return nil, err
	}
	digest, err := desired.Digest()
	if err != nil {
		return nil, err
	}
	now := s.clock()
	received, refusal := receivedOf(desired, now)
	if refusal == nil {
		refusal = desired.Refusal()
	}

	out := new(applied)
	err = s.db.Transact(
		dbscope.WithSystem(ctx, applyReason),
		kernelports.TxOptions{LockTimeout: applyLockTimeout},
		func(ctx context.Context) error {
			return s.apply(ctx, &applyInput{
				instanceID: instanceID,
				desired:    desired,
				digest:     digest,
				received:   received,
				refusal:    refusal,
				now:        now,
				source:     source,
			}, out)
		},
	)
	if errors.Is(err, errEmailTaken) {
		out, err = s.refuseApart(ctx, instanceID, desired, digest, now)
	}
	if err != nil {
		return nil, err
	}

	s.afterApply(ctx, out)
	return s.statusOf(ctx, out.tenant)
}

type applyInput struct {
	instanceID string
	desired    *cloudtenant.Desired
	digest     string
	received   *managedfeatures.Received
	refusal    *cloudtenant.Refusal
	now        int64
	source     *tenantarchive.Access
}

func (s *Service) apply(ctx context.Context, in *applyInput, out *applied) error {
	if err := s.tenants.Lock(ctx, in.instanceID); err != nil {
		return err
	}
	held, err := s.tenants.Get(ctx, in.instanceID)
	switch {
	case errors.Is(err, cloudtenantports.ErrNotFound):
		held = nil
	case err != nil:
		return err
	}
	if s.unchanged(held, in, out) {
		return nil
	}

	t := held
	if t == nil {
		t = &cloudtenant.Tenant{
			InstanceID: in.instanceID,
			State:      cloudtenant.StateApplying,
			Desired:    *in.desired,
			CreatedAt:  in.now,
		}
	}
	out.tenant, out.previous = t, t.State
	t.DesiredRevision, t.DesiredDigest, t.UpdatedAt = in.desired.Revision, in.digest, in.now

	refusal := in.refusal
	if refusal == nil && t.Made() && !in.desired.SameIdentity(&t.Desired) {
		refusal = cloudtenant.Invalid(in.desired.Revision, reasonIdentityFixed)
	}
	if refusal != nil {
		return s.refuse(ctx, t, refusal, in.now)
	}

	if t.Restoring() {
		return s.holdWhileRestoring(ctx, t, in)
	}

	if in.desired.Lifecycle == cloudtenant.LifecycleRemoved {
		return s.beginRemoval(ctx, t, in, out)
	}

	if !t.Made() {
		if command := in.desired.ImportCommand(); command != nil {
			return s.beginImport(ctx, t, in, command, out)
		}
		taken, terr := s.administrators.EmailTaken(ctx, in.desired.FirstAdministrator.Email)
		if terr != nil {
			return terr
		}
		if taken {
			return errEmailTaken
		}
		if err = s.create(ctx, t, in, out); err != nil {
			return err
		}
	}

	return s.converge(ctx, t, in, out)
}

func (s *Service) unchanged(held *cloudtenant.Tenant, in *applyInput, out *applied) bool {
	if held == nil {
		return false
	}
	out.tenant, out.previous = held, held.State
	switch {
	case held.State == cloudtenant.StateRemoved:
		return true
	case held.State == cloudtenant.StateRemoving:
		out.startRemoval = true
		return true
	case in.desired.Revision < held.DesiredRevision:
		return true
	case in.desired.Revision == held.DesiredRevision && in.digest == held.DesiredDigest:
		return true
	default:
		return false
	}
}

func (s *Service) refuse(
	ctx context.Context,
	t *cloudtenant.Tenant,
	refusal *cloudtenant.Refusal,
	now int64,
) error {
	t.Refuse(refusal)
	if err := s.tenants.Save(ctx, t); err != nil {
		return err
	}
	return s.tenants.Record(ctx, &cloudtenant.Event{
		InstanceID: t.InstanceID,
		Kind:       cloudtenant.EventRefused,
		Revision:   refusal.Revision,
		Detail:     map[string]any{"code": refusal.Code, "reason": refusal.Reason},
		OccurredAt: now,
	})
}

func (s *Service) refuseApart(
	ctx context.Context,
	instanceID string,
	desired *cloudtenant.Desired,
	digest string,
	now int64,
) (*applied, error) {
	out := new(applied)
	err := s.db.Transact(
		dbscope.WithSystem(ctx, applyReason),
		kernelports.TxOptions{LockTimeout: applyLockTimeout},
		func(ctx context.Context) error {
			if err := s.tenants.Lock(ctx, instanceID); err != nil {
				return err
			}
			t, err := s.tenants.Get(ctx, instanceID)
			switch {
			case errors.Is(err, cloudtenantports.ErrNotFound):
				t = &cloudtenant.Tenant{
					InstanceID: instanceID,
					State:      cloudtenant.StateApplying,
					Desired:    *desired,
					CreatedAt:  now,
				}
			case err != nil:
				return err
			}
			if t.Made() {
				out.tenant = t
				return nil
			}
			out.tenant, out.previous = t, t.State
			t.DesiredRevision, t.DesiredDigest, t.UpdatedAt = desired.Revision, digest, now
			return s.refuse(ctx, t, cloudtenant.Invalid(desired.Revision, reasonEmailTaken), now)
		},
	)
	return out, err
}

func (s *Service) create(
	ctx context.Context,
	t *cloudtenant.Tenant,
	in *applyInput,
	out *applied,
) error {
	inputs := (&instancebootstrap.Inputs{
		OrganizationName: in.desired.OrganizationName,
		AdminName:        in.desired.FirstAdministrator.Name,
		AdminEmail:       in.desired.FirstAdministrator.Email,
	}).Normalize()
	owner := inputs.Administrator()
	hashed, err := owner.GenerateUnusablePassword()
	if err != nil {
		return fmt.Errorf("hash an unusable administrator password: %w", err)
	}
	owner.Password = hashed

	result, err := s.bootstrap.Bootstrap(ctx, &tenantports.BootstrapTenantRequest{
		BusinessUnitName:  inputs.OrganizationName,
		Organization:      inputs.Organization(),
		LoginSlugBase:     stringutils.SlugifyASCII(inputs.OrganizationName, loginSlugLength),
		StateAbbreviation: inputs.State,
		Owner:             owner,
		UsernameBase:      tenant.UsernameBase(inputs.AdminEmail),
		Now:               in.now,
	})
	if err != nil {
		if dberror.IsUniqueConstraintViolation(err) {
			return errEmailTaken
		}
		return err
	}

	t.OrganizationID = result.Organization.ID
	t.BusinessUnitID = result.BusinessUnit.ID
	t.AdminUserID = result.Owner.ID
	t.Desired = *in.desired
	out.created = result
	return s.tenants.Record(ctx, &cloudtenant.Event{
		InstanceID: t.InstanceID,
		Kind:       cloudtenant.EventCreated,
		Revision:   in.desired.Revision,
		OccurredAt: in.now,
	})
}

func (s *Service) converge(
	ctx context.Context,
	t *cloudtenant.Tenant,
	in *applyInput,
	out *applied,
) error {
	scope := managedscope.Organization(tenantOf(t))
	if err := s.features.SetFeatures(ctx, scope, in.received); err != nil {
		return fmt.Errorf("set the tenant's plan: %w", err)
	}
	lifecycle := managedfeatures.LifecycleActive
	state := cloudtenant.StateActive
	if in.desired.Lifecycle == cloudtenant.LifecycleSuspended {
		lifecycle, state = managedfeatures.LifecycleSuspended, cloudtenant.StateSuspended
	}
	if err := s.features.SetLifecycle(ctx, scope, lifecycle); err != nil {
		return fmt.Errorf("set the tenant's lifecycle: %w", err)
	}
	if err := s.installInvite(ctx, t, in); err != nil {
		return err
	}

	t.Desired = *in.desired
	t.State = state
	t.AppliedRevision = in.desired.Revision
	t.Refuse(nil)
	if err := s.tenants.Save(ctx, t); err != nil {
		return err
	}

	kind := cloudtenant.EventApplied
	switch {
	case state == cloudtenant.StateSuspended && out.previous != cloudtenant.StateSuspended:
		kind = cloudtenant.EventSuspended
		out.startSessions = true
	case state == cloudtenant.StateActive && out.previous == cloudtenant.StateSuspended:
		kind = cloudtenant.EventResumed
	}
	return s.tenants.Record(ctx, &cloudtenant.Event{
		InstanceID: t.InstanceID,
		Kind:       kind,
		Revision:   in.desired.Revision,
		Detail: map[string]any{
			"features": len(in.desired.Features),
			"addons":   len(in.desired.Addons),
		},
		OccurredAt: in.now,
	})
}

func (s *Service) installInvite(ctx context.Context, t *cloudtenant.Tenant, in *applyInput) error {
	invite := in.desired.FirstAdministrator.Invite
	if invite == nil || t.ImportedAt > 0 || invite.TokenSHA256 == t.InviteSHA256 ||
		invite.ExpiresAt.Unix() <= in.now {
		return nil
	}
	install := &instancebootstrapports.FirstAdministratorInvite{
		TokenSHA256: invite.TokenSHA256,
		ExpiresAt:   invite.ExpiresAt.Unix(),
	}
	if _, err := s.invites.InstallFor(ctx, t.AdminUserID, install); err != nil {
		return fmt.Errorf("install the first administrator's invite: %w", err)
	}
	t.InviteSHA256 = invite.TokenSHA256
	return nil
}

func (s *Service) beginRemoval(
	ctx context.Context,
	t *cloudtenant.Tenant,
	in *applyInput,
	out *applied,
) error {
	if !t.Made() {
		t.Desired = *in.desired
		t.State = cloudtenant.StateRemoved
		t.AppliedRevision = in.desired.Revision
		t.RemovedAt = in.now
		t.Refuse(nil)
		if err := s.tenants.Save(ctx, t); err != nil {
			return err
		}
		return s.tenants.Record(ctx, &cloudtenant.Event{
			InstanceID: t.InstanceID,
			Kind:       cloudtenant.EventRemoved,
			Revision:   in.desired.Revision,
			Detail:     map[string]any{"made": false},
			OccurredAt: in.now,
		})
	}
	if !t.HasFinalArchive() {
		return s.refuse(ctx, t, cloudtenant.Invalid(in.desired.Revision, reasonNoArchive), in.now)
	}
	return s.markRemoving(ctx, t, in.desired, in.now, out)
}

func (s *Service) markRemoving(
	ctx context.Context,
	t *cloudtenant.Tenant,
	desired *cloudtenant.Desired,
	now int64,
	out *applied,
) error {
	scope := managedscope.Organization(tenantOf(t))
	if err := s.features.SetLifecycle(ctx, scope, managedfeatures.LifecycleSuspended); err != nil {
		return fmt.Errorf("suspend the tenant before its removal: %w", err)
	}
	t.Desired = *desired
	t.State = cloudtenant.StateRemoving
	t.RemovalStartedAt = now
	t.Refuse(nil)
	if err := s.tenants.Save(ctx, t); err != nil {
		return err
	}
	out.tenant = t
	out.startRemoval = true
	return s.tenants.Record(ctx, &cloudtenant.Event{
		InstanceID: t.InstanceID,
		Kind:       cloudtenant.EventRemoving,
		Revision:   desired.Revision,
		OccurredAt: now,
	})
}

func (s *Service) afterApply(ctx context.Context, out *applied) {
	if out.tenant == nil {
		return
	}
	if out.created != nil {
		s.auditCreated(ctx, out.tenant, out.created)
	}
	switch {
	case out.tenant.State == cloudtenant.StateSuspended && out.previous != cloudtenant.StateSuspended &&
		out.previous != cloudtenant.StateApplying:
		s.auditLifecycle(ctx, out.tenant, "organization.suspended",
			"The Trenova control plane suspended the organization")
	case out.tenant.State == cloudtenant.StateActive && out.previous == cloudtenant.StateSuspended:
		s.auditLifecycle(ctx, out.tenant, "organization.resumed",
			"The Trenova control plane resumed the organization")
	}
	if out.startSessions {
		s.startSessions(ctx, out.tenant)
	}
	if out.startRemoval {
		s.startRemoval(ctx, out.tenant)
	}
	if out.importCmd != nil {
		s.startCommand(ctx, out.importCmd, nil)
	}
}

func (s *Service) holdWhileRestoring(
	ctx context.Context,
	t *cloudtenant.Tenant,
	in *applyInput,
) error {
	if in.desired.Lifecycle == cloudtenant.LifecycleRemoved {
		return s.refuse(ctx, t, cloudtenant.Invalid(in.desired.Revision, reasonRestoring), in.now)
	}
	return nil
}

func receivedOf(
	desired *cloudtenant.Desired,
	now int64,
) (*managedfeatures.Received, *cloudtenant.Refusal) {
	features := make([]platformcatalog.FeatureKey, 0, len(desired.Features))
	for _, f := range desired.Features {
		features = append(features, platformcatalog.FeatureKey(f))
	}
	endings := make([]managedfeatures.AddOnEnding, 0, len(desired.AddonEndings))
	for _, e := range desired.AddonEndings {
		endings = append(endings, managedfeatures.AddOnEnding{
			AddOn:  platformcatalog.AddOnKey(e.Addon),
			EndsAt: e.EndsAt.Unix(),
		})
	}
	received := &managedfeatures.Received{
		Features:        features,
		AddOnEndings:    &endings,
		FinalBackupDays: desired.Account.FinalBackupDays,
	}
	if url := strings.TrimSpace(desired.Account.PlanURL); url != "" {
		received.PlanURL = &url
	}
	if desired.Account.EndsAt != nil && !desired.Account.EndsAt.IsZero() {
		received.PlanEndsAt = desired.Account.EndsAt.Unix()
	}
	if err := received.Validate(now); err != nil {
		reason := err.Error()
		if refusal, ok := errortypes.RefusalOf(err); ok {
			reason = refusal.Error()
		}
		return received, cloudtenant.Invalid(desired.Revision, reason)
	}
	return received, nil
}

func tenantOf(t *cloudtenant.Tenant) dbscope.Tenant {
	return dbscope.Tenant{OrganizationID: t.OrganizationID, BusinessUnitID: t.BusinessUnitID}
}
