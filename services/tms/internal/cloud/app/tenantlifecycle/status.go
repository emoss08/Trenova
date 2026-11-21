package tenantlifecycle

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/supportaccess"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	DefaultPageSize = 100
	MaxPageSize     = 200
	maxFailures     = 20
)

type StatusPage struct {
	Tenants []*cloudtenant.Status `json:"tenants"`
	Next    string                `json:"next,omitempty"`
}

func (s *Service) List(ctx context.Context, after string, limit int) (*StatusPage, error) {
	if !s.shared {
		return nil, ErrNotShared
	}
	if after != "" && !cloudtenant.ValidInstanceID(after) {
		return nil, ErrCursorInvalid
	}
	if limit <= 0 {
		limit = DefaultPageSize
	}
	page, err := s.tenants.List(ctx, &cloudtenantports.ListRequest{
		After: after,
		Limit: min(limit, MaxPageSize),
	})
	if err != nil {
		return nil, err
	}
	statuses, err := s.statuses(ctx, page.Tenants)
	if err != nil {
		return nil, err
	}
	return &StatusPage{Tenants: statuses, Next: page.Next}, nil
}

func (s *Service) statusOf(
	ctx context.Context,
	t *cloudtenant.Tenant,
) (*cloudtenant.Status, error) {
	statuses, err := s.statuses(ctx, []*cloudtenant.Tenant{t})
	if err != nil {
		return nil, err
	}
	return statuses[0], nil
}

func (s *Service) statuses(
	ctx context.Context,
	tenants []*cloudtenant.Tenant,
) ([]*cloudtenant.Status, error) {
	requests := make([]cloudtenantports.AdministratorRequest, 0, len(tenants))
	for _, t := range tenants {
		if t.AdminUserID.IsNil() || t.State == cloudtenant.StateRemoved ||
			t.State == cloudtenant.StateRemoving {
			continue
		}
		requests = append(requests, cloudtenantports.AdministratorRequest{
			UserID:      t.AdminUserID,
			InviteToken: t.InviteSHA256,
		})
	}
	facts, err := s.administrators.Facts(ctx, requests)
	if err != nil {
		return nil, err
	}
	failures, err := s.failures(ctx, tenants)
	if err != nil {
		return nil, err
	}
	grants, err := s.liveGrants(ctx, tenants)
	if err != nil {
		return nil, err
	}
	out := make([]*cloudtenant.Status, 0, len(tenants))
	for _, t := range tenants {
		var admin *cloudtenant.Administrator
		if f, ok := facts[t.AdminUserID]; ok {
			admin = &cloudtenant.Administrator{SetUp: f.SetUp}
			if f.InviteLive {
				admin.InviteTokenSHA256 = f.InviteToken
			}
		}
		status := t.Status(admin)
		status.CommandFailures = failures[t.InstanceID]
		if held, ok := grants[t.OrganizationID]; ok {
			status.SupportGrants = &held
		}
		out = append(out, status)
	}
	return out, nil
}

func (s *Service) failures(
	ctx context.Context,
	tenants []*cloudtenant.Tenant,
) (map[string][]cloudtenant.CommandFailure, error) {
	ids := make([]string, 0, len(tenants))
	for _, t := range tenants {
		ids = append(ids, t.InstanceID)
	}
	failed, err := s.commands.Failures(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string][]cloudtenant.CommandFailure, len(failed))
	for _, cmd := range failed {
		if len(out[cmd.InstanceID]) >= maxFailures {
			continue
		}
		reason := cmd.Reason
		if reason == "" {
			reason = "the command could not be completed"
		}
		out[cmd.InstanceID] = append(out[cmd.InstanceID], cloudtenant.CommandFailure{
			CommandID: cmd.CommandID,
			Kind:      string(cmd.Kind),
			Reason:    reason,
		})
	}
	return out, nil
}

func reportsGrants(t *cloudtenant.Tenant) bool {
	return t.Made() && t.State != cloudtenant.StateRemoved && t.State != cloudtenant.StateRemoving
}

func (s *Service) liveGrants(
	ctx context.Context,
	tenants []*cloudtenant.Tenant,
) (map[pulid.ID][]cloudtenant.SupportGrant, error) {
	orgIDs := make([]pulid.ID, 0, len(tenants))
	out := make(map[pulid.ID][]cloudtenant.SupportGrant, len(tenants))
	for _, t := range tenants {
		if reportsGrants(t) {
			orgIDs = append(orgIDs, t.OrganizationID)
			out[t.OrganizationID] = []cloudtenant.SupportGrant{}
		}
	}
	if len(orgIDs) == 0 {
		return out, nil
	}
	live, err := s.grants.ListLive(ctx, &supportaccessports.ListLiveRequest{
		Now:             s.clock(),
		Limit:           len(orgIDs) * supportaccess.MaxLiveGrants,
		OrganizationIDs: orgIDs,
	})
	if err != nil {
		return nil, fmt.Errorf("read the tenants' live support grants: %w", err)
	}
	for _, l := range live {
		held, ok := out[l.Grant.OrganizationID]
		if !ok {
			continue
		}
		out[l.Grant.OrganizationID] = append(held, cloudtenant.SupportGrant{
			ID:               l.Grant.ID.String(),
			OrganizationID:   l.Grant.OrganizationID.String(),
			OrganizationName: l.OrganizationName,
			StaffEmail:       l.Grant.StaffEmail,
			Scope:            l.Grant.Scope.ControlPlane(),
			ExpiresAt:        time.Unix(l.Grant.ExpiresAt, 0).UTC(),
		})
	}
	return out, nil
}
