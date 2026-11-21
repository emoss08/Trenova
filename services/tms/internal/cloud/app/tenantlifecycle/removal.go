package tenantlifecycle

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/jobs/tenantjobs/tenantjobscontract"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/zap"
)

const (
	removeReason = "mark one shared-server tenant as being removed, or removed, in the server's bookkeeping"
	sessionsPage = 500
)

var (
	ErrNotRemovable = errors.New(
		"the tenant is removed only after its final archive is recorded and while the control plane's entry says removed",
	)
	ErrNotRemoving = errors.New("the tenant is not being removed")
)

func (s *Service) Remove(ctx context.Context, instanceID string) (*cloudtenant.Status, error) {
	if !s.shared {
		return nil, ErrNotShared
	}
	out := new(applied)
	err := s.db.Transact(
		dbscope.WithSystem(ctx, removeReason),
		kernelports.TxOptions{LockTimeout: applyLockTimeout},
		func(ctx context.Context) error {
			if err := s.tenants.Lock(ctx, instanceID); err != nil {
				return err
			}
			t, err := s.tenants.Get(ctx, instanceID)
			if err != nil {
				return err
			}
			out.tenant, out.previous = t, t.State
			switch {
			case t.State == cloudtenant.StateRemoved:
				return nil
			case t.State == cloudtenant.StateRemoving:
				out.startRemoval = true
				return nil
			case !t.Made() || !t.HasFinalArchive() ||
				t.Desired.Lifecycle != cloudtenant.LifecycleRemoved:
				return ErrNotRemovable
			}
			return s.markRemoving(ctx, t, &t.Desired, s.clock(), out)
		},
	)
	if err != nil {
		return nil, err
	}
	s.afterApply(ctx, out)
	return s.statusOf(ctx, out.tenant)
}

func (s *Service) FinishRemoval(
	ctx context.Context,
	instanceID string,
	detail map[string]any,
) error {
	return s.db.Transact(
		dbscope.WithSystem(ctx, removeReason),
		kernelports.TxOptions{LockTimeout: applyLockTimeout},
		func(ctx context.Context) error {
			if err := s.tenants.Lock(ctx, instanceID); err != nil {
				return err
			}
			t, err := s.tenants.Get(ctx, instanceID)
			if err != nil {
				return err
			}
			if t.State == cloudtenant.StateRemoved {
				return nil
			}
			if t.State != cloudtenant.StateRemoving {
				return ErrNotRemoving
			}
			now := s.clock()
			t.State = cloudtenant.StateRemoved
			t.AppliedRevision = t.Desired.Revision
			t.RemovedAt = now
			t.UpdatedAt = now
			t.InviteSHA256 = ""
			if err = s.tenants.Save(ctx, t); err != nil {
				return err
			}
			return s.tenants.Record(ctx, &cloudtenant.Event{
				InstanceID: t.InstanceID,
				Kind:       cloudtenant.EventRemoved,
				Revision:   t.Desired.Revision,
				Detail:     detail,
				OccurredAt: now,
			})
		},
	)
}

func (s *Service) Removing(ctx context.Context, instanceID string) (*cloudtenant.Tenant, error) {
	t, err := s.tenants.Get(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if t.State != cloudtenant.StateRemoving && t.State != cloudtenant.StateRemoved {
		return nil, ErrNotRemoving
	}
	return t, nil
}

func (s *Service) startRemoval(ctx context.Context, t *cloudtenant.Tenant) {
	s.start(ctx, &startRequest{
		id:       tenantjobscontract.RemovalWorkflowID(t.InstanceID),
		workflow: tenantjobscontract.RemovalWorkflowName,
		reuse:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
		args:     payloadOf(t),
		summary:  "Remove a shared-server tenant's data and documents",
	})
}

func (s *Service) startSessions(ctx context.Context, t *cloudtenant.Tenant) {
	s.start(ctx, &startRequest{
		id:       tenantjobscontract.SessionsWorkflowID(t.InstanceID, t.AppliedRevision),
		workflow: tenantjobscontract.SessionsWorkflowName,
		reuse:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE_FAILED_ONLY,
		args:     payloadOf(t),
		summary:  "Sign a suspended shared-server tenant's people out",
	})
}

type startRequest struct {
	id       string
	workflow string
	reuse    enums.WorkflowIdReusePolicy
	args     any
	summary  string
}

func (s *Service) start(ctx context.Context, req *startRequest) {
	_, err := s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    req.id,
		TaskQueue:             temporaltype.TaskQueueSystem.String(),
		WorkflowIDReusePolicy: req.reuse,
		StaticSummary:         req.summary,
	}, req.workflow, req.args)
	if err == nil || temporal.IsWorkflowExecutionAlreadyStartedError(err) {
		return
	}
	s.l.Error(
		"cannot start a shared-server tenant's workflow; the next entry or removal call starts it again",
		zap.String("workflowId", req.id),
		zap.Error(err),
	)
}

func payloadOf(t *cloudtenant.Tenant) *tenantjobscontract.TenantPayload {
	return &tenantjobscontract.TenantPayload{
		InstanceID:     t.InstanceID,
		OrganizationID: t.OrganizationID,
		BusinessUnitID: t.BusinessUnitID,
		Revision:       t.Desired.Revision,
	}
}

type RevocationResult struct {
	Members  int `json:"members"`
	Sessions int `json:"sessions"`
	Moved    int `json:"moved"`
}

type RevocationRequest struct {
	Tenant   dbscope.Tenant
	Reassign bool
}

func (s *Service) RevokeSessions(
	ctx context.Context,
	req *RevocationRequest,
	heartbeat func(),
) (*RevocationResult, error) {
	out := new(RevocationResult)
	var cursor pulid.ID
	for {
		page, err := s.members.Page(ctx, req.Tenant, cursor, sessionsPage)
		if err != nil {
			return out, err
		}
		for _, m := range page.Members {
			out.Members++
			revoked, rerr := s.revokeMember(ctx, req, m)
			if rerr != nil {
				return out, rerr
			}
			out.Sessions += revoked.sessions
			if revoked.moved {
				out.Moved++
			}
		}
		if heartbeat != nil {
			heartbeat()
		}
		if page.Next.IsNil() {
			return out, nil
		}
		cursor = page.Next
	}
}

type memberRevocation struct {
	sessions int
	moved    bool
}

func (s *Service) revokeMember(
	ctx context.Context,
	req *RevocationRequest,
	m cloudtenantports.Member,
) (memberRevocation, error) {
	var out memberRevocation
	if m.OtherMemberships == 0 {
		sessions, err := s.sessions.ListForUser(ctx, m.UserID)
		if err != nil {
			return out, fmt.Errorf("list a member's sessions: %w", err)
		}
		if err = s.sessions.DeleteAllForUser(ctx, m.UserID); err != nil {
			return out, fmt.Errorf("end a member's sessions: %w", err)
		}
		out.sessions = len(sessions)
		return out, nil
	}
	sessions, err := s.sessions.ListForUser(ctx, m.UserID)
	if err != nil {
		return out, fmt.Errorf("list a member's sessions: %w", err)
	}
	for _, sess := range sessions {
		if sess.OrganizationID != req.Tenant.OrganizationID {
			continue
		}
		if err = s.sessions.Delete(ctx, sess.ID); err != nil {
			return out, fmt.Errorf("end a member's session: %w", err)
		}
		out.sessions++
	}
	if req.Reassign {
		moved, merr := s.members.Reassign(ctx, req.Tenant, m.UserID)
		if merr != nil {
			return out, merr
		}
		out.moved = moved
	}
	return out, nil
}
