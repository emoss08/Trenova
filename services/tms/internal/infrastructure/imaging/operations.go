package imaging

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingpb"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/intutils"
)

const openEnded = 0xFFFFFFFF

var errStreamShape = errors.New("the imaging service sent a malformed stream")

func u32(value int) uint32 {
	return intutils.SafeIntToUint32(value)
}

func (c *Client) Probe(
	ctx context.Context,
	req *kernelports.ImagingProbeRequest,
) (*kernelports.ImagingProbe, error) {
	var out *kernelports.ImagingProbe
	err := c.call(
		ctx,
		c.cfg.Deadlines.GetProbe(),
		req.Priority,
		func(ctx context.Context, rpc imagingpb.ImagingServiceClient, _ *bool) error {
			stream, err := rpc.Probe(ctx)
			if err != nil {
				return err
			}
			if err = c.upload(ctx, req.Source, func() error {
				return stream.Send(&imagingpb.ProbeRequest{Part: &imagingpb.ProbeRequest_Header{
					Header: &imagingpb.ProbeHeader{
						Document:    document(req.Source),
						TextProfile: req.TextProfile,
					},
				}})
			}, func(chunk []byte) error {
				return stream.Send(
					&imagingpb.ProbeRequest{Part: &imagingpb.ProbeRequest_Chunk{Chunk: chunk}},
				)
			}, func() error { return nil }); err != nil {
				return err
			}
			resp, err := stream.CloseAndRecv()
			if err != nil {
				return err
			}
			out = probeFrom(resp)
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return out, nil
}

func probeFrom(resp *imagingpb.ProbeResponse) *kernelports.ImagingProbe {
	kind := kernelports.ImagingKindPDF
	if resp.GetKind() == imagingpb.DocumentKind_DOCUMENT_KIND_IMAGE {
		kind = kernelports.ImagingKindImage
	}
	pages := make([]kernelports.ImagingPageInfo, 0, len(resp.GetPages()))
	for _, page := range resp.GetPages() {
		pages = append(pages, kernelports.ImagingPageInfo{
			Index:        int(page.GetIndex()),
			WidthPoints:  float64(page.GetWidthPoints()),
			HeightPoints: float64(page.GetHeightPoints()),
			Rotation:     int(page.GetRotation()),
			HasText:      page.GetHasText(),
			CharCount:    int(page.GetCharCount()),
		})
	}
	return &kernelports.ImagingProbe{
		Kind:        kind,
		PageCount:   int(resp.GetPageCount()),
		Pages:       pages,
		Encrypted:   resp.GetEncrypted(),
		PDFVersion:  resp.GetPdfVersion(),
		Producer:    resp.GetProducer(),
		ImageFormat: resp.GetImageFormat(),
		Orientation: int(resp.GetOrientation()),
	}
}

type chunkSource func() (*imagingpb.ArtifactChunk, error)

type artifactBody struct {
	next    chunkSource
	pending []byte
	done    bool
	err     error
}

func (b *artifactBody) Read(p []byte) (int, error) {
	for len(b.pending) == 0 {
		if b.err != nil {
			return 0, b.err
		}
		if b.done {
			return 0, io.EOF
		}
		chunk, err := b.next()
		if err != nil {
			b.err = err
			return 0, err
		}
		b.pending = chunk.GetData()
		b.done = chunk.GetLast()
	}
	n := copy(p, b.pending)
	b.pending = b.pending[n:]
	return n, nil
}

func (b *artifactBody) drain() error {
	_, err := io.Copy(io.Discard, b)
	return err
}

func deliver(
	header *imagingpb.ArtifactHeader,
	next chunkSource,
	yield func(*kernelports.ImagingArtifact) error,
) error {
	body := &artifactBody{next: next}
	size, err := intutils.SafeUint64ToInt64(header.GetSize())
	if err != nil {
		return errStreamShape
	}
	if err = yield(&kernelports.ImagingArtifact{
		Index:       int(header.GetIndex()),
		ContentType: header.GetContentType(),
		Width:       int(header.GetWidthPx()),
		Height:      int(header.GetHeightPx()),
		Size:        size,
		Body:        body,
	}); err != nil {
		return err
	}
	return body.drain()
}

type chunked interface {
	GetChunk() *imagingpb.ArtifactChunk
}

func chunksFrom[R chunked](recv func() (R, error)) chunkSource {
	return func() (*imagingpb.ArtifactChunk, error) {
		msg, err := recv()
		if err != nil {
			return nil, err
		}
		if chunk := msg.GetChunk(); chunk != nil {
			return chunk, nil
		}
		return nil, errStreamShape
	}
}

func renderHeader(req *kernelports.ImagingRenderRequest) *imagingpb.RenderHeader {
	header := &imagingpb.RenderHeader{
		Document: document(req.Source),
		Pages:    toUint32s(req.Pages),
		Format:   formatOf(req.Format),
		Quality:  u32(req.Quality),
		Lossless: req.Lossless,
		MaxPages: u32(req.MaxPages),
	}
	if req.DPI > 0 {
		header.Size = &imagingpb.RenderSize{Size: &imagingpb.RenderSize_Dpi{Dpi: u32(req.DPI)}}
		return header
	}
	header.Size = &imagingpb.RenderSize{Size: &imagingpb.RenderSize_Fit{Fit: &imagingpb.FitBox{
		MaxWidth:  u32(req.FitWidth),
		MaxHeight: u32(req.FitHeight),
	}}}
	return header
}

func (c *Client) Render(
	ctx context.Context,
	req *kernelports.ImagingRenderRequest,
	yield func(*kernelports.ImagingArtifact) error,
) (*kernelports.ImagingRenderSummary, error) {
	header := renderHeader(req)
	var summary *kernelports.ImagingRenderSummary
	err := c.call(
		ctx,
		c.cfg.Deadlines.GetRender(),
		req.Priority,
		func(ctx context.Context, rpc imagingpb.ImagingServiceClient, progressed *bool) error {
			stream, err := rpc.Render(ctx)
			if err != nil {
				return err
			}
			if err = c.upload(ctx, req.Source, func() error {
				return stream.Send(
					&imagingpb.RenderRequest{Part: &imagingpb.RenderRequest_Header{Header: header}},
				)
			}, func(chunk []byte) error {
				return stream.Send(
					&imagingpb.RenderRequest{Part: &imagingpb.RenderRequest_Chunk{Chunk: chunk}},
				)
			}, stream.CloseSend); err != nil {
				return err
			}
			next := chunksFrom(stream.Recv)
			for summary == nil {
				msg, recvErr := stream.Recv()
				if recvErr != nil {
					return recvErr
				}
				if summary, err = renderPart(msg, next, progressed, yield); err != nil {
					return err
				}
			}
			return nil
		},
	)
	if err != nil {
		return nil, err
	}
	return summary, nil
}

func renderPart(
	msg *imagingpb.RenderResponse,
	next chunkSource,
	progressed *bool,
	yield func(*kernelports.ImagingArtifact) error,
) (*kernelports.ImagingRenderSummary, error) {
	if page := msg.GetPage(); page != nil {
		*progressed = true
		return nil, deliver(page, next, yield)
	}
	if failed := msg.GetPageError(); failed != nil {
		*progressed = true
		return nil, yield(&kernelports.ImagingArtifact{
			Index: int(failed.GetPageIndex()),
			Err:   pageError(failed),
		})
	}
	if done := msg.GetSummary(); done != nil {
		return &kernelports.ImagingRenderSummary{
			PageCount: int(done.GetPageCount()),
			Rendered:  int(done.GetRendered()),
		}, nil
	}
	return nil, errStreamShape
}

func (c *Client) upload(
	ctx context.Context,
	source kernelports.ImagingSource,
	sendHeader func() error,
	sendChunk func([]byte) error,
	closeSend func() error,
) error {
	if err := sendHeader(); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	buffer := make([]byte, c.cfg.GetChunkBytes())
	if err := sendSource(ctx, source, c.cfg.GetChunkBytes(), buffer, sendChunk); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return closeSend()
}

func (c *Client) InspectCapture(
	ctx context.Context,
	req *kernelports.ImagingCaptureRequest,
	yield func(*kernelports.ImagingCaptureResult) error,
) (*kernelports.ImagingCaptureSummary, error) {
	var summary *kernelports.ImagingCaptureSummary
	err := c.call(
		ctx,
		c.cfg.Deadlines.GetInspectCapture(),
		req.Priority,
		func(ctx context.Context, rpc imagingpb.ImagingServiceClient, progressed *bool) error {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			stream, err := rpc.InspectCapture(ctx)
			if err != nil {
				return err
			}

			sent := make(chan error, 1)
			go func() {
				sent <- c.sendCapture(ctx, stream, req)
			}()

			for {
				msg, recvErr := stream.Recv()
				if recvErr != nil {
					cancel()
					if sendErr := <-sent; sendErr != nil && !errors.Is(sendErr, io.EOF) &&
						!errors.Is(sendErr, context.Canceled) {
						return sendErr
					}
					return recvErr
				}
				if page := msg.GetPage(); page != nil {
					*progressed = true
					if err = yield(captureFrom(page)); err != nil {
						return err
					}
					continue
				}
				done := msg.GetSummary()
				if done == nil {
					return errStreamShape
				}
				summary = &kernelports.ImagingCaptureSummary{
					Inspected: int(done.GetInspected()),
					Failed:    int(done.GetFailed()),
				}
				if sendErr := <-sent; sendErr != nil && !errors.Is(sendErr, io.EOF) {
					return sendErr
				}
				return nil
			}
		},
	)
	if err != nil {
		return nil, err
	}
	return summary, nil
}

func (c *Client) sendCapture(
	ctx context.Context,
	stream imagingpb.ImagingService_InspectCaptureClient,
	req *kernelports.ImagingCaptureRequest,
) error {
	if err := stream.Send(&imagingpb.InspectCaptureRequest{
		Part: &imagingpb.InspectCaptureRequest_Header{
			Header: &imagingpb.InspectCaptureHeader{PageCount: u32(req.PageCount)},
		},
	}); err != nil {
		return err
	}
	buffer := make([]byte, c.cfg.GetChunkBytes())
	for page, err := range req.Pages() {
		if err != nil {
			return err
		}
		if err = stream.Send(&imagingpb.InspectCaptureRequest{
			Part: &imagingpb.InspectCaptureRequest_Page{
				Page: &imagingpb.CapturePageStart{
					Index:    u32(page.Index),
					Document: document(page.Source),
				},
			},
		}); err != nil {
			return err
		}
		if err = sendSource(
			ctx,
			page.Source,
			c.cfg.GetChunkBytes(),
			buffer,
			func(chunk []byte) error {
				return stream.Send(&imagingpb.InspectCaptureRequest{
					Part: &imagingpb.InspectCaptureRequest_Chunk{Chunk: chunk},
				})
			},
		); err != nil {
			return err
		}
	}
	return stream.CloseSend()
}

func captureFrom(page *imagingpb.CapturePageResult) *kernelports.ImagingCaptureResult {
	codes := make([]kernelports.ImagingBarcode, 0, len(page.GetCodes()))
	for _, code := range page.GetCodes() {
		codes = append(
			codes,
			kernelports.ImagingBarcode{Format: code.GetFormat(), Text: code.GetText()},
		)
	}
	return &kernelports.ImagingCaptureResult{
		Index:                int(page.GetIndex()),
		Width:                int(page.GetWidthPx()),
		Height:               int(page.GetHeightPx()),
		Thumbnail:            page.GetThumbnail(),
		ThumbnailContentType: page.GetThumbnailContentType(),
		InkCoverage:          page.GetInkCoverage(),
		Codes:                codes,
		Err:                  pageError(page.GetError()),
	}
}

func assembleHeader(req *kernelports.ImagingAssembleRequest) *imagingpb.AssembleHeader {
	header := &imagingpb.AssembleHeader{SourceCount: u32(len(req.Sources))}
	for _, source := range req.Sources {
		header.TotalSize += sizeOf(source)
	}
	for _, output := range req.Outputs {
		ranges := make([]*imagingpb.PageRange, 0, len(output))
		for _, r := range output {
			last := uint32(openEnded)
			if r.Last != openEnded {
				last = u32(r.Last)
			}
			ranges = append(ranges, &imagingpb.PageRange{
				Source:   u32(r.Source),
				First:    u32(r.First),
				Last:     last,
				Rotation: u32(((r.Rotation % 360) + 360) % 360),
			})
		}
		header.Outputs = append(header.Outputs, &imagingpb.AssembleOutput{Pages: ranges})
	}
	return header
}

func (c *Client) Assemble(
	ctx context.Context,
	req *kernelports.ImagingAssembleRequest,
	yield func(*kernelports.ImagingArtifact) error,
) ([]int, error) {
	header := assembleHeader(req)
	var counts []int
	err := c.call(
		ctx,
		c.cfg.Deadlines.GetAssemble(),
		req.Priority,
		func(ctx context.Context, rpc imagingpb.ImagingServiceClient, progressed *bool) error {
			stream, err := rpc.Assemble(ctx)
			if err != nil {
				return err
			}
			if err = c.sendSources(ctx, stream, header, req.Sources); err != nil {
				if errors.Is(err, io.EOF) {
					return closeStream(stream)
				}
				return err
			}
			next := chunksFrom(stream.Recv)
			for {
				msg, recvErr := stream.Recv()
				if recvErr != nil {
					return recvErr
				}
				if output := msg.GetOutput(); output != nil {
					*progressed = true
					if err = deliver(output, next, yield); err != nil {
						return err
					}
					continue
				}
				done := msg.GetSummary()
				if done == nil {
					return errStreamShape
				}
				for _, n := range done.GetOutputPageCounts() {
					counts = append(counts, int(n))
				}
				return nil
			}
		},
	)
	if err != nil {
		return nil, err
	}
	return counts, nil
}

func (c *Client) sendSources(
	ctx context.Context,
	stream imagingpb.ImagingService_AssembleClient,
	header *imagingpb.AssembleHeader,
	sources []kernelports.ImagingSource,
) error {
	if err := stream.Send(&imagingpb.AssembleRequest{
		Part: &imagingpb.AssembleRequest_Header{Header: header},
	}); err != nil {
		return err
	}
	buffer := make([]byte, c.cfg.GetChunkBytes())
	for index, source := range sources {
		if err := stream.Send(&imagingpb.AssembleRequest{Part: &imagingpb.AssembleRequest_Source{
			Source: &imagingpb.AssembleSourceStart{Index: u32(index), Document: document(source)},
		}}); err != nil {
			return err
		}
		if err := sendSource(ctx, source, c.cfg.GetChunkBytes(), buffer, func(chunk []byte) error {
			return stream.Send(
				&imagingpb.AssembleRequest{Part: &imagingpb.AssembleRequest_Chunk{Chunk: chunk}},
			)
		}); err != nil {
			return err
		}
	}
	return stream.CloseSend()
}

type receiver[T any] interface {
	Recv() (T, error)
	CloseSend() error
}

func closeStream[T any](stream receiver[T]) error {
	_ = stream.CloseSend()
	for {
		if _, err := stream.Recv(); err != nil {
			return err
		}
	}
}

func (c *Client) NormalizeImage(
	ctx context.Context,
	req *kernelports.ImagingNormalizeRequest,
	yield func(*kernelports.ImagingArtifact) error,
) (*kernelports.ImagingNormalizeSummary, error) {
	if req.Format == kernelports.ImagingWebP {
		return nil, fmt.Errorf(
			"%w: images are normalized to PNG or JPEG",
			kernelports.ErrImagingUnsupported,
		)
	}
	header := &imagingpb.NormalizeImageHeader{
		Document:  document(req.Source),
		Format:    formatOf(req.Format),
		Quality:   u32(req.Quality),
		MaxWidth:  u32(req.MaxWidth),
		MaxHeight: u32(req.MaxHeight),
	}

	var summary *kernelports.ImagingNormalizeSummary
	err := c.call(
		ctx,
		c.cfg.Deadlines.GetNormalizeImage(),
		req.Priority,
		func(ctx context.Context, rpc imagingpb.ImagingServiceClient, progressed *bool) error {
			stream, err := rpc.NormalizeImage(ctx)
			if err != nil {
				return err
			}
			if err = c.upload(ctx, req.Source, func() error {
				return stream.Send(&imagingpb.NormalizeImageRequest{
					Part: &imagingpb.NormalizeImageRequest_Header{Header: header},
				})
			}, func(chunk []byte) error {
				return stream.Send(&imagingpb.NormalizeImageRequest{
					Part: &imagingpb.NormalizeImageRequest_Chunk{Chunk: chunk},
				})
			}, stream.CloseSend); err != nil {
				return err
			}
			next := chunksFrom(stream.Recv)
			for {
				msg, recvErr := stream.Recv()
				if recvErr != nil {
					return recvErr
				}
				if image := msg.GetImage(); image != nil {
					*progressed = true
					if err = deliver(image, next, yield); err != nil {
						return err
					}
					continue
				}
				done := msg.GetSummary()
				if done == nil {
					return errStreamShape
				}
				summary = &kernelports.ImagingNormalizeSummary{
					SourceFormat:       done.GetSourceFormat(),
					SourceWidth:        int(done.GetSourceWidth()),
					SourceHeight:       int(done.GetSourceHeight()),
					OrientationApplied: int(done.GetOrientationApplied()),
				}
				return nil
			}
		},
	)
	if err != nil {
		return nil, err
	}
	return summary, nil
}
