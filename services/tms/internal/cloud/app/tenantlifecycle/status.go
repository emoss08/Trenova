package tenantlifecycle

import (
	"context"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
)

const (
	DefaultPageSize = 100
	MaxPageSize     = 200
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
	out := make([]*cloudtenant.Status, 0, len(tenants))
	for _, t := range tenants {
		var admin *cloudtenant.Administrator
		if f, ok := facts[t.AdminUserID]; ok {
			admin = &cloudtenant.Administrator{SetUp: f.SetUp}
			if f.InviteLive {
				admin.InviteTokenSHA256 = f.InviteToken
			}
		}
		out = append(out, t.Status(admin))
	}
	return out, nil
}
