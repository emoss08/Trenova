package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/aiauditservice"
	"github.com/emoss08/trenova/internal/modules/agent/domain/aiaudit"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiauditports"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/audit"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type aiAuditLedger interface {
	ListByIDs(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) ([]*aiaudit.AIAuditEvent, error)
	ListCorrelatedAuditEntries(
		ctx context.Context,
		req *aiauditports.ListCorrelatedAuditEntriesRequest,
	) ([]*audit.Entry, error)
}

type AuditEntriesByAIAuditEventIDLoaderFactoryParams struct {
	fx.In

	Ledger aiauditports.AIAuditRepository
}

// AuditEntriesByAIAuditEventIDLoaderFactory finds, for a page of trail
// events, the audit log rows matched to each by record, principal and time,
// in two queries however long the page.
type AuditEntriesByAIAuditEventIDLoaderFactory struct {
	ledger aiAuditLedger
}

func NewAuditEntriesByAIAuditEventIDLoaderFactory(
	p AuditEntriesByAIAuditEventIDLoaderFactoryParams,
) *AuditEntriesByAIAuditEventIDLoaderFactory {
	return &AuditEntriesByAIAuditEventIDLoaderFactory{ledger: p.Ledger}
}

func newAuditEntriesByAIAuditEventIDLoader(f *AuditEntriesByAIAuditEventIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*audit.Entry] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *AuditEntriesByAIAuditEventIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*audit.Entry] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*audit.Entry, error) {
			events, err := f.ledger.ListByIDs(ctx, tenantInfo, ids)
			if err != nil {
				return nil, err
			}

			return aiauditservice.Correlate(ctx, f.ledger, tenantInfo, events)
		},
	)
}
