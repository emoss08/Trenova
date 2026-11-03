package invoicepacket

import (
	"context"
	"iter"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const (
	indexChunkSize        = 32
	indexImageWidth       = 425
	indexImageHeight      = 550
	indexImageDPI         = 100
	indexImageQuality     = 75
	thumbnailWidth        = 200
	thumbnailQuality      = 72
	maxIndexImageBytes    = 4 << 20
	unknownInk            = -1
	thumbnailFallbackType = "image/jpeg"
)

type PageIndex struct {
	Thumbnails []billingqueueports.PacketPage
	Ink        []float64
}

func (x *PageIndex) size() int {
	total := 0
	for i := range x.Thumbnails {
		total += len(x.Thumbnails[i].Content)
	}
	return total + len(x.Ink)*8
}

func (a *Assembler) IndexPages(ctx context.Context, packet []byte, pages int) (*PageIndex, error) {
	index := &PageIndex{
		Thumbnails: make([]billingqueueports.PacketPage, pages),
		Ink:        make([]float64, pages),
	}
	for i := range index.Ink {
		index.Ink[i] = unknownInk
	}
	for first := 1; first <= pages; first += indexChunkSize {
		last := min(first+indexChunkSize-1, pages)
		images, err := a.renderIndexImages(ctx, packet, first, last)
		if err != nil {
			return nil, err
		}
		if err = a.inspectIndexImages(ctx, images, index); err != nil {
			return nil, err
		}
	}

	return index, nil
}

type indexImage struct {
	page int
	data []byte
}

func (a *Assembler) renderIndexImages(
	ctx context.Context,
	packet []byte,
	first, last int,
) ([]indexImage, error) {
	indexes := make([]int, 0, last-first+1)
	for page := first; page <= last; page++ {
		indexes = append(indexes, page-1)
	}
	images := make([]indexImage, 0, len(indexes))
	_, err := a.imaging.Render(ctx, &kernelports.ImagingRenderRequest{
		Source:    kernelports.ImagingBytes(PDFContentType, packet),
		Pages:     indexes,
		FitWidth:  indexImageWidth,
		FitHeight: indexImageHeight,
		Format:    kernelports.ImagingJPEG,
		Quality:   indexImageQuality,
		Priority:  kernelports.ImagingInteractive,
	}, func(artifact *kernelports.ImagingArtifact) error {
		if artifact.Err != nil || artifact.Body == nil {
			return nil
		}
		data, collectErr := imagingkit.Collect(artifact, maxIndexImageBytes)
		if collectErr != nil {
			return collectErr
		}
		images = append(images, indexImage{page: artifact.Index + 1, data: data})
		return nil
	})
	if err != nil {
		return nil, err
	}

	return images, nil
}

func (a *Assembler) inspectIndexImages(
	ctx context.Context,
	images []indexImage,
	index *PageIndex,
) error {
	if len(images) == 0 {
		return nil
	}
	_, err := a.imaging.InspectCapture(ctx, &kernelports.ImagingCaptureRequest{
		PageCount: len(images),
		Pages: func() iter.Seq2[kernelports.ImagingCapturePage, error] {
			return func(yield func(kernelports.ImagingCapturePage, error) bool) {
				for _, image := range images {
					if !yield(kernelports.ImagingCapturePage{
						Index:  image.page - 1,
						Source: kernelports.ImagingBytes(jpegContentType, image.data),
					}, nil) {
						return
					}
				}
			}
		},
		Priority:         kernelports.ImagingInteractive,
		RenderDPI:        indexImageDPI,
		ThumbnailWidth:   thumbnailWidth,
		ThumbnailFormat:  kernelports.ImagingWebP,
		ThumbnailQuality: thumbnailQuality,
		SkipBarcodes:     true,
	}, func(result *kernelports.ImagingCaptureResult) error {
		if result.Err != nil || result.Index < 0 || result.Index >= len(index.Ink) {
			return nil
		}
		contentType := result.ThumbnailContentType
		if contentType == "" {
			contentType = thumbnailFallbackType
		}
		index.Ink[result.Index] = result.InkCoverage
		index.Thumbnails[result.Index] = billingqueueports.PacketPage{
			Content:     result.Thumbnail,
			ContentType: contentType,
		}
		return nil
	})

	return err
}
