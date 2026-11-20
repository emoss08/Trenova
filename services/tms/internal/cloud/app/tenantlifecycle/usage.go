package tenantlifecycle

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

const (
	MaxLookback     = 35 * 24 * time.Hour
	hourSeconds     = int64(time.Hour / time.Second)
	MaxUsageEntries = 1000
	refusedNotHeld  = "not_placed"
)

var (
	ErrCursorInvalid = errors.New("the cursor must be a tenant's instanceId from a previous page")
	ErrWindowInvalid = errors.New(
		"from and to must be RFC 3339 times on the hour, from not after to, from within 35 days and not after the current hour",
	)
	ErrTooManyEntries = fmt.Errorf("at most %d tenants' usage answers a request", MaxUsageEntries)
)

type ReadingsRequest struct {
	From  int64
	To    int64
	After string
	Limit int
}

type TenantReading struct {
	Meter    string `json:"meter"`
	Hour     string `json:"hour"`
	Quantity int64  `json:"quantity"`
}

type TenantReadings struct {
	InstanceID string          `json:"instanceId"`
	Readings   []TenantReading `json:"readings"`
}

type ReadingsPage struct {
	Tenants []TenantReadings `json:"tenants"`
	Next    string           `json:"next,omitempty"`
}

func (s *Service) Readings(ctx context.Context, req *ReadingsRequest) (*ReadingsPage, error) {
	if !s.shared {
		return nil, ErrNotShared
	}
	now := s.clock()
	current := now - now%hourSeconds
	if req.From%hourSeconds != 0 || req.To%hourSeconds != 0 || req.From > req.To ||
		req.From > current || req.From < current-int64(MaxLookback/time.Second) {
		return nil, ErrWindowInvalid
	}
	if req.After != "" && !cloudtenant.ValidInstanceID(req.After) {
		return nil, ErrCursorInvalid
	}
	limit := req.Limit
	if limit <= 0 {
		limit = DefaultPageSize
	}
	page, err := s.tenants.List(ctx, &cloudtenantports.ListRequest{
		After: req.After,
		Limit: min(limit, MaxPageSize),
	})
	if err != nil {
		return nil, err
	}

	metered := make([]*cloudtenant.Tenant, 0, len(page.Tenants))
	ids := make([]pulid.ID, 0, len(page.Tenants))
	for _, t := range page.Tenants {
		if !isMetered(t) {
			continue
		}
		metered = append(metered, t)
		ids = append(ids, t.OrganizationID)
	}
	byOrg, err := s.readings.ReadingsForOrganizations(ctx, ids, req.From)
	if err != nil {
		return nil, err
	}

	out := &ReadingsPage{Tenants: make([]TenantReadings, 0, len(metered)), Next: page.Next}
	for _, t := range metered {
		entry := TenantReadings{InstanceID: t.InstanceID, Readings: make([]TenantReading, 0)}
		for _, rd := range byOrg[t.OrganizationID] {
			if rd.Hour > req.To {
				continue
			}
			entry.Readings = append(entry.Readings, TenantReading{
				Meter:    string(rd.Meter),
				Hour:     time.Unix(rd.Hour, 0).UTC().Format(time.RFC3339),
				Quantity: rd.Quantity,
			})
		}
		out.Tenants = append(out.Tenants, entry)
	}
	return out, nil
}

func isMetered(t *cloudtenant.Tenant) bool {
	return t.Made() && (t.State == cloudtenant.StateActive || t.State == cloudtenant.StateSuspended)
}

type UsageWindow struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type TenantUsage struct {
	InstanceID string               `json:"instanceId"`
	Window     *UsageWindow         `json:"window"`
	Overage    bool                 `json:"overage"`
	Meters     []managedusage.Meter `json:"meters"`
}

type UsageRefused struct {
	InstanceID string `json:"instanceId"`
	Code       string `json:"code"`
	Reason     string `json:"reason,omitempty"`
}

type UsageAnswer struct {
	Applied int            `json:"applied"`
	Refused []UsageRefused `json:"refused"`
}

func (s *Service) SetUsage(ctx context.Context, entries []TenantUsage) (*UsageAnswer, error) {
	if !s.shared {
		return nil, ErrNotShared
	}
	if len(entries) > MaxUsageEntries {
		return nil, ErrTooManyEntries
	}
	out := &UsageAnswer{Refused: make([]UsageRefused, 0)}
	held, err := s.heldByInstance(ctx, entries)
	if err != nil {
		return nil, err
	}
	for i := range entries {
		entry := &entries[i]
		t, ok := held[entry.InstanceID]
		if !ok || !isMetered(t) {
			out.Refused = append(
				out.Refused,
				UsageRefused{InstanceID: entry.InstanceID, Code: refusedNotHeld},
			)
			continue
		}
		state := &managedusage.State{Overage: entry.Overage, Meters: entry.Meters}
		if state.Meters == nil {
			state.Meters = []managedusage.Meter{}
		}
		if entry.Window != nil {
			state.WindowStart, state.WindowEnd = entry.Window.Start.Unix(), entry.Window.End.Unix()
		}
		scope := managedscope.Organization(tenantOf(t))
		if err = s.usage.SetState(ctx, scope, state); err != nil {
			if refusal, isRefusal := errortypes.RefusalOf(err); isRefusal {
				out.Refused = append(out.Refused, UsageRefused{
					InstanceID: entry.InstanceID,
					Code:       cloudtenant.RefusalInvalid,
					Reason:     refusal.Error(),
				})
				continue
			}
			return nil, fmt.Errorf("record the usage state of a tenant: %w", err)
		}
		out.Applied++
	}
	if len(out.Refused) > 0 {
		s.l.Warn("usage answers for tenants this server cannot take",
			zap.Int("refused", len(out.Refused)),
		)
	}
	return out, nil
}

func (s *Service) heldByInstance(
	ctx context.Context,
	entries []TenantUsage,
) (map[string]*cloudtenant.Tenant, error) {
	ids := make([]string, 0, len(entries))
	for i := range entries {
		if cloudtenant.ValidInstanceID(entries[i].InstanceID) {
			ids = append(ids, entries[i].InstanceID)
		}
	}
	tenants, err := s.tenants.ByInstances(ctx, ids)
	if err != nil {
		return nil, err
	}
	out := make(map[string]*cloudtenant.Tenant, len(tenants))
	for _, t := range tenants {
		out[t.InstanceID] = t
	}
	return out, nil
}
