package assetinliner

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/imageutils"
)

func (i *Inliner) transcoder(ctx context.Context) imageutils.WebPTranscoder {
	if i.imaging == nil {
		return nil
	}
	return func(body []byte) ([]byte, error) {
		var png []byte
		_, err := i.imaging.NormalizeImage(ctx, &kernelports.ImagingNormalizeRequest{
			Source:   kernelports.ImagingBytes("image/webp", body),
			Format:   kernelports.ImagingPNG,
			Priority: kernelports.ImagingInteractive,
		}, func(artifact *kernelports.ImagingArtifact) error {
			data, collectErr := imagingkit.Collect(artifact, i.limits.MaxAssetBytes*4)
			png = data
			return collectErr
		})
		if err != nil {
			return nil, fmt.Errorf("transcode webp: %w", err)
		}
		return png, nil
	}
}
