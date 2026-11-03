package invoicepacket

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/maypok86/otter/v2"
)

const (
	previewCacheBytes = 512 << 20
	pageCacheBytes    = 256 << 20
	indexCacheBytes   = 128 << 20
	previewCacheIdle  = 10 * time.Minute
)

type PreviewKey struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	ItemID pulid.ID
}

type PageChunkKey struct {
	Preview PreviewKey
	Build   string
	Chunk   int
}

type PageIndexKey struct {
	Preview PreviewKey
	Build   string
}

type PageIndexBuilder func(ctx context.Context) (*PageIndex, error)

type PreviewBuilder func(ctx context.Context) (*billingqueueports.PacketPreview, error)

type PageChunkRenderer func(ctx context.Context) ([]billingqueueports.PacketPage, error)

type PreviewCache interface {
	Load(
		ctx context.Context,
		key PreviewKey,
		build PreviewBuilder,
	) (*billingqueueports.PacketPreview, error)
	Put(key PreviewKey, preview *billingqueueports.PacketPreview)
	LoadPages(
		ctx context.Context,
		key PageChunkKey,
		render PageChunkRenderer,
	) ([]billingqueueports.PacketPage, error)
	LoadIndex(ctx context.Context, key PageIndexKey, build PageIndexBuilder) (*PageIndex, error)
}

type otterPreviews struct {
	packets *otter.Cache[PreviewKey, *billingqueueports.PacketPreview]
	pages   *otter.Cache[PageChunkKey, []billingqueueports.PacketPage]
	indexes *otter.Cache[PageIndexKey, *PageIndex]
}

func NewPreviewCache() PreviewCache {
	return &otterPreviews{
		packets: otter.Must(&otter.Options[PreviewKey, *billingqueueports.PacketPreview]{
			MaximumWeight: previewCacheBytes,
			Weigher: func(_ PreviewKey, preview *billingqueueports.PacketPreview) uint32 {
				return weight(len(preview.Content))
			},
			ExpiryCalculator: otter.ExpiryAccessing[PreviewKey, *billingqueueports.PacketPreview](
				previewCacheIdle,
			),
		}),
		pages: otter.Must(&otter.Options[PageChunkKey, []billingqueueports.PacketPage]{
			MaximumWeight: pageCacheBytes,
			Weigher: func(_ PageChunkKey, pages []billingqueueports.PacketPage) uint32 {
				size := 0
				for i := range pages {
					size += len(pages[i].Content)
				}
				return weight(size)
			},
			ExpiryCalculator: otter.ExpiryAccessing[PageChunkKey, []billingqueueports.PacketPage](
				previewCacheIdle,
			),
		}),
		indexes: otter.Must(&otter.Options[PageIndexKey, *PageIndex]{
			MaximumWeight: indexCacheBytes,
			Weigher: func(_ PageIndexKey, index *PageIndex) uint32 {
				return weight(index.size())
			},
			ExpiryCalculator: otter.ExpiryAccessing[PageIndexKey, *PageIndex](previewCacheIdle),
		}),
	}
}

func weight(size int) uint32 {
	return uint32(
		min(max(size, 1), MaxPacketBytes),
	) //nolint:gosec // clamped to MaxPacketBytes, which fits
}

func (s *otterPreviews) Load(
	ctx context.Context,
	key PreviewKey,
	build PreviewBuilder,
) (*billingqueueports.PacketPreview, error) {
	return s.packets.Get(
		ctx,
		key,
		otter.LoaderFunc[PreviewKey, *billingqueueports.PacketPreview](
			func(ctx context.Context, _ PreviewKey) (*billingqueueports.PacketPreview, error) {
				return build(ctx)
			},
		),
	)
}

func (s *otterPreviews) Put(key PreviewKey, preview *billingqueueports.PacketPreview) {
	s.packets.Set(key, preview)
}

func (s *otterPreviews) LoadPages(
	ctx context.Context,
	key PageChunkKey,
	render PageChunkRenderer,
) ([]billingqueueports.PacketPage, error) {
	return s.pages.Get(
		ctx,
		key,
		otter.LoaderFunc[PageChunkKey, []billingqueueports.PacketPage](
			func(ctx context.Context, _ PageChunkKey) ([]billingqueueports.PacketPage, error) {
				return render(ctx)
			},
		),
	)
}

func (s *otterPreviews) LoadIndex(
	ctx context.Context,
	key PageIndexKey,
	build PageIndexBuilder,
) (*PageIndex, error) {
	return s.indexes.Get(
		ctx,
		key,
		otter.LoaderFunc[PageIndexKey, *PageIndex](
			func(ctx context.Context, _ PageIndexKey) (*PageIndex, error) {
				return build(ctx)
			},
		),
	)
}
