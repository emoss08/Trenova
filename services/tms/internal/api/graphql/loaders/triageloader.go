package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type fetchCountsByKeyFunc func(context.Context, []string) (map[string]int, error)

func batchKeyCountFunc(fetch fetchCountsByKeyFunc) batchFetchFunc[int] {
	return func(ctx context.Context, keys []string) ([]int, []error) {
		values := make([]int, len(keys))
		errs := make([]error, len(keys))

		unique := make([]string, 0, len(keys))
		seen := make(map[string]struct{}, len(keys))
		for _, key := range keys {
			if _, ok := seen[key]; ok {
				continue
			}
			seen[key] = struct{}{}
			unique = append(unique, key)
		}

		found, err := fetch(ctx, unique)
		if err != nil {
			fillMissingErrors(errs, err)
			return values, errs
		}
		for idx, key := range keys {
			values[idx] = found[key]
		}

		return values, errs
	}
}

type inboundThreadCounter interface {
	CountByThreadKeys(
		ctx context.Context,
		req inboundmessageports.ThreadCountsRequest,
	) (map[string]int, error)
}

type InboundThreadCountLoaderFactoryParams struct {
	fx.In

	QueueRepo inboundmessageports.TriageQueueRepository
}

type InboundThreadCountLoaderFactory struct {
	threads inboundThreadCounter
}

func NewInboundThreadCountLoaderFactory(
	p InboundThreadCountLoaderFactoryParams,
) *InboundThreadCountLoaderFactory {
	return &InboundThreadCountLoaderFactory{threads: p.QueueRepo}
}

func newInboundThreadCountLoader(
	f *InboundThreadCountLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *InboundThreadCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[int] {
	return batchKeyCountFunc(func(ctx context.Context, keys []string) (map[string]int, error) {
		return f.threads.CountByThreadKeys(ctx, inboundmessageports.ThreadCountsRequest{
			TenantInfo: tenantInfo,
			ThreadKeys: keys,
		})
	})
}

type inboundActivityLister interface {
	ListByMessageIDs(
		ctx context.Context,
		req inboundmessageports.ListActivitiesRequest,
	) (map[pulid.ID][]*triage.Activity, error)
}

type InboundActivitiesByMessageIDLoaderFactoryParams struct {
	fx.In

	ActivityRepo inboundmessageports.TriageActivityRepository
}

type InboundActivitiesByMessageIDLoaderFactory struct {
	activities inboundActivityLister
}

func NewInboundActivitiesByMessageIDLoaderFactory(
	p InboundActivitiesByMessageIDLoaderFactoryParams,
) *InboundActivitiesByMessageIDLoaderFactory {
	return &InboundActivitiesByMessageIDLoaderFactory{activities: p.ActivityRepo}
}

func newInboundActivitiesByMessageIDLoader(
	f *InboundActivitiesByMessageIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*triage.Activity] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *InboundActivitiesByMessageIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*triage.Activity] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*triage.Activity, error) {
			return f.activities.ListByMessageIDs(ctx, inboundmessageports.ListActivitiesRequest{
				TenantInfo: tenantInfo,
				MessageIDs: ids,
				Limit:      inboundmessageports.MaxActivitiesPerRead,
			})
		},
	)
}

type documentTypeNamer interface {
	DocumentTypeNames(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		ids []pulid.ID,
	) (map[pulid.ID]string, error)
}

type DocumentTypeNameByIDLoaderFactoryParams struct {
	fx.In

	DocumentTypes inboundmessageports.InboundDocumentTypeFinder
}

type DocumentTypeNameByIDLoaderFactory struct {
	documentTypes documentTypeNamer
}

func NewDocumentTypeNameByIDLoaderFactory(
	p DocumentTypeNameByIDLoaderFactoryParams,
) *DocumentTypeNameByIDLoaderFactory {
	return &DocumentTypeNameByIDLoaderFactory{documentTypes: p.DocumentTypes}
}

func newDocumentTypeNameByIDLoader(
	f *DocumentTypeNameByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, string] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *DocumentTypeNameByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[string] {
	return batchValueFunc(func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]string, error) {
		return f.documentTypes.DocumentTypeNames(ctx, tenantInfo, ids)
	})
}

type triageViewCounter interface {
	WaitingCounts(
		ctx context.Context,
		req inboundmessageports.ViewWaitingCountsRequest,
	) (map[pulid.ID]int, error)
}

type TriageViewWaitingCountLoaderFactoryParams struct {
	fx.In

	ViewRepo inboundmessageports.TriageViewRepository
}

type TriageViewWaitingCountLoaderFactory struct {
	views triageViewCounter
}

func NewTriageViewWaitingCountLoaderFactory(
	p TriageViewWaitingCountLoaderFactoryParams,
) *TriageViewWaitingCountLoaderFactory {
	return &TriageViewWaitingCountLoaderFactory{views: p.ViewRepo}
}

func newTriageViewWaitingCountLoader(
	f *TriageViewWaitingCountLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, int] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *TriageViewWaitingCountLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[int] {
	return batchCountFunc(func(ctx context.Context, ids []pulid.ID) (map[pulid.ID]int, error) {
		return f.views.WaitingCounts(ctx, inboundmessageports.ViewWaitingCountsRequest{
			TenantInfo: tenantInfo,
			ViewIDs:    ids,
		})
	})
}
