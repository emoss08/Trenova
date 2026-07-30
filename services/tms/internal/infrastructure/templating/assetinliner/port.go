package assetinliner

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/imageutils"
	"go.uber.org/zap"
)

var _ services.AssetInliner = (*Inliner)(nil)

// InlineAssets adapts Inline to the core port.
//
// The port exists so a core service can ask for images to be inlined without
// importing the HTTP client, the storage client, or the WebP transcoder that
// doing it safely requires.
func (i *Inliner) InlineAssets(
	ctx context.Context,
	req *services.AssetInlineRequest,
) (*services.AssetInlineResult, error) {
	if req == nil {
		return &services.AssetInlineResult{}, nil
	}

	result, err := i.Inline(ctx, &Request{HTML: req.HTML, Field: req.Field})
	if err != nil {
		return nil, err
	}

	return &services.AssetInlineResult{
		HTML:    result.HTML,
		Diags:   result.Diags,
		Inlined: result.Inlined,
		Bytes:   result.Bytes,
	}, nil
}

// ResolveImageDataURI resolves a single image reference for a context builder.
//
// It goes through the same resolve path as document markup, so an
// organization-supplied logo URL is fetched with the SSRF-safe client and
// re-checked at dial time. A failure is logged and reported as an empty string
// rather than an error: the invoice still has to reach the customer.
func (i *Inliner) ResolveImageDataURI(ctx context.Context, src string) (string, error) {
	if strings.TrimSpace(src) == "" {
		return "", nil
	}

	img, err := i.resolve(ctx, src)
	if err != nil {
		i.l.Debug("could not resolve image for a template context", zap.Error(err))
		return "", nil
	}

	if int64(len(img.Data)) > i.limits.MaxAssetBytes {
		i.l.Debug("image for a template context exceeded the per-asset budget",
			zap.Int("bytes", len(img.Data)))
		return "", nil
	}

	return imageutils.ToDataURI(img), nil
}
