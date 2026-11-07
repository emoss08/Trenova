package auditservice

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/infrastructure/observability/metrics"
	"github.com/emoss08/trenova/internal/modules/agent/ports/realtimeports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

type fakeRepo struct {
	mu        sync.Mutex
	inserted  [][]*audit.Entry
	insertErr error
	listReq   *auditports.ListAuditEntriesRequest
}

func (r *fakeRepo) Insert(_ context.Context, entries []*audit.Entry) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.insertErr != nil {
		return r.insertErr
	}
	r.inserted = append(r.inserted, entries)
	return nil
}

func (r *fakeRepo) entries() []*audit.Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]*audit.Entry, 0)
	for _, batch := range r.inserted {
		out = append(out, batch...)
	}
	return out
}

func (r *fakeRepo) ListConnection(
	_ context.Context,
	req *auditports.ListAuditEntriesRequest,
) (*pagination.CursorListResult[*audit.Entry], error) {
	r.listReq = req
	return &pagination.CursorListResult[*audit.Entry]{}, nil
}

func (r *fakeRepo) GetByID(context.Context, *auditports.GetAuditEntryByIDOptions) (*audit.Entry, error) {
	return &audit.Entry{}, nil
}

func (r *fakeRepo) ExportPage(context.Context, *auditports.ExportPageRequest) ([]*audit.Entry, error) {
	return nil, nil
}

func (r *fakeRepo) DeleteExpired(context.Context, auditports.DeleteExpiredAuditEntriesRequest) (int64, error) {
	return 0, nil
}

func (r *fakeRepo) PartitionHorizon(context.Context) (int64, error) { return 0, nil }


func (r *fakeRepo) TenantsWithEntries(context.Context, int64, int64) ([]auditports.TenantKey, error) {
	return nil, nil
}

type noopRealtimeService struct{}

func (s *noopRealtimeService) PublishResourceInvalidation(
	context.Context,
	*realtimeports.PublishResourceInvalidationRequest,
) error {
	return nil
}

func newTestService(repo *fakeRepo, realtime realtimeports.RealtimeService) *service {
	return &service{
		repo:     repo,
		realtime: realtime,
		logger:   zap.NewNop(),
		redactor: NewRedactor("test-fingerprint-pepper-that-is-long-enough"),
		metrics:  &metrics.Registry{},
		agents:   newAgentCredits(nil, zap.NewNop()),
	}
}

func validLogActionParams() *auditports.LogActionParams {
	return &auditports.LogActionParams{
		Resource:       permission.ResourceShipment,
		ResourceID:     pulid.MustNew("shp_").String(),
		Operation:      permission.OpUpdate,
		UserID:         pulid.MustNew("usr_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		CurrentState:   map[string]any{"status": "Assigned"},
	}
}
