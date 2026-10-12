package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type TriageRuleStatsLoaderFactoryParams struct {
	fx.In

	Stats triageports.TriageRuleStats
}

type TriageRuleMonthMatchesLoaderFactory struct {
	stats triageports.TriageRuleStats
}

func NewTriageRuleMonthMatchesLoaderFactory(
	p TriageRuleStatsLoaderFactoryParams,
) *TriageRuleMonthMatchesLoaderFactory {
	return &TriageRuleMonthMatchesLoaderFactory{stats: p.Stats}
}

func newTriageRuleMonthMatchesLoader(
	f *TriageRuleMonthMatchesLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return newLoader(batchCountFunc(func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
		return f.stats.MatchesThisMonth(ctx, tenantInfo, ids)
	}))
}

type TriageSampleByMessageIDLoaderFactory struct {
	stats triageports.TriageRuleStats
}

func NewTriageSampleByMessageIDLoaderFactory(
	p TriageRuleStatsLoaderFactoryParams,
) *TriageSampleByMessageIDLoaderFactory {
	return &TriageSampleByMessageIDLoaderFactory{stats: p.Stats}
}

func newTriageSampleByMessageIDLoader(
	f *TriageSampleByMessageIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *inboundmessageports.Sample] {
	return newLoader(batchValueFunc(fetchValuesByIDFunc[*inboundmessageports.Sample](
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*inboundmessageports.Sample, error) {
			return f.stats.Samples(ctx, tenantInfo, ids)
		},
	)))
}

type InboundMailboxByIDLoaderFactory struct {
	stats triageports.TriageRuleStats
}

func NewInboundMailboxByIDLoaderFactory(
	p TriageRuleStatsLoaderFactoryParams,
) *InboundMailboxByIDLoaderFactory {
	return &InboundMailboxByIDLoaderFactory{stats: p.Stats}
}

func newInboundMailboxByIDLoader(
	f *InboundMailboxByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *inboundmessage.Mailbox] {
	return newLoader(batchValueFunc(fetchValuesByIDFunc[*inboundmessage.Mailbox](
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*inboundmessage.Mailbox, error) {
			return f.stats.MailboxesByIDs(ctx, tenantInfo, ids)
		},
	)))
}
