package documentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
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
	return loadercore.NewLoader(
		loadercore.BatchCountFunc(
			func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
				return f.stats.MatchesThisMonth(ctx, tenantInfo, ids)
			},
		),
	)
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
	return loadercore.NewLoader(
		loadercore.BatchValueFunc(loadercore.FetchValuesByIDFunc[*inboundmessageports.Sample](
			func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*inboundmessageports.Sample, error) {
				return f.stats.Samples(ctx, tenantInfo, ids)
			},
		)),
	)
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
	return loadercore.NewLoader(
		loadercore.BatchValueFunc(loadercore.FetchValuesByIDFunc[*inboundmessage.Mailbox](
			func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]*inboundmessage.Mailbox, error) {
				return f.stats.MailboxesByIDs(ctx, tenantInfo, ids)
			},
		)),
	)
}

var (
	triageRuleMonthMatchesKey  = loadercore.NewKey(newTriageRuleMonthMatchesLoader)
	triageSampleByMessageIDKey = loadercore.NewKey(newTriageSampleByMessageIDLoader)
	inboundMailboxByIDKey      = loadercore.NewKey(newInboundMailboxByIDLoader)
)

func TriageRuleMonthMatches(ctx context.Context, key string) (int, error) {
	return loadercore.Load(ctx, triageRuleMonthMatchesKey, key)
}

func (f *TriageRuleMonthMatchesLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func TriageSampleByMessageID(ctx context.Context, key string) (*inboundmessageports.Sample, error) {
	return loadercore.Load(ctx, triageSampleByMessageIDKey, key)
}

func (f *TriageSampleByMessageIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}

func InboundMailboxByID(ctx context.Context, key string) (*inboundmessage.Mailbox, error) {
	return loadercore.Load(ctx, inboundMailboxByIDKey, key)
}

func (f *InboundMailboxByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
