package imagingkit

import (
	"context"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const ToLastPage = math.MaxUint32

var ErrNoOutput = errors.New("the imaging service returned no output")

type Part struct {
	Source   kernelports.ImagingSource
	Rotation int
}

type Rendered struct {
	Data        []byte
	ContentType string
	Width       int
	Height      int
}

func Collect(artifact *kernelports.ImagingArtifact, limit int64) ([]byte, error) {
	if artifact.Err != nil {
		return nil, artifact.Err
	}
	if artifact.Body == nil {
		return nil, ErrNoOutput
	}
	data, err := io.ReadAll(io.LimitReader(artifact.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf(
			"%w: the output is larger than %d bytes",
			kernelports.ErrImagingTooLarge,
			limit,
		)
	}
	return data, nil
}

func PageCount(
	ctx context.Context,
	img kernelports.Imaging,
	source kernelports.ImagingSource,
	priority kernelports.ImagingPriority,
) (int, error) {
	probe, err := img.Probe(
		ctx,
		&kernelports.ImagingProbeRequest{Source: source, Priority: priority},
	)
	if err != nil {
		return 0, err
	}
	if probe.Encrypted {
		return 0, kernelports.ErrImagingEncrypted
	}
	return probe.PageCount, nil
}

func Merge(
	ctx context.Context,
	img kernelports.Imaging,
	parts []Part,
	limit int64,
	priority kernelports.ImagingPriority,
) (data []byte, pages int, err error) {
	if len(parts) == 0 {
		return nil, 0, fmt.Errorf(
			"%w: there are no pages to assemble",
			kernelports.ErrImagingInvalidDocument,
		)
	}
	sources := make([]kernelports.ImagingSource, 0, len(parts))
	ranges := make([]kernelports.ImagingPageRange, 0, len(parts))
	for i, part := range parts {
		sources = append(sources, part.Source)
		ranges = append(ranges, kernelports.ImagingPageRange{
			Source:   i,
			First:    0,
			Last:     ToLastPage,
			Rotation: part.Rotation,
		})
	}

	var out []byte
	counts, err := img.Assemble(ctx, &kernelports.ImagingAssembleRequest{
		Sources:  sources,
		Outputs:  [][]kernelports.ImagingPageRange{ranges},
		Priority: priority,
	}, func(artifact *kernelports.ImagingArtifact) error {
		collected, collectErr := Collect(artifact, limit)
		out = collected
		return collectErr
	})
	if err != nil {
		return nil, 0, err
	}
	if out == nil || len(counts) != 1 {
		return nil, 0, ErrNoOutput
	}
	return out, counts[0], nil
}

func SplitPages(
	ctx context.Context,
	img kernelports.Imaging,
	source kernelports.ImagingSource,
	pageCount int,
	limit int64,
	priority kernelports.ImagingPriority,
) ([][]byte, error) {
	outputs := make([][]kernelports.ImagingPageRange, 0, pageCount)
	for page := range pageCount {
		outputs = append(
			outputs,
			[]kernelports.ImagingPageRange{{Source: 0, First: page, Last: page}},
		)
	}

	pages := make([][]byte, pageCount)
	_, err := img.Assemble(ctx, &kernelports.ImagingAssembleRequest{
		Sources:  []kernelports.ImagingSource{source},
		Outputs:  outputs,
		Priority: priority,
	}, func(artifact *kernelports.ImagingArtifact) error {
		if artifact.Index < 0 || artifact.Index >= pageCount {
			return ErrNoOutput
		}
		data, collectErr := Collect(artifact, limit)
		pages[artifact.Index] = data
		return collectErr
	})
	if err != nil {
		return nil, err
	}
	for _, page := range pages {
		if page == nil {
			return nil, ErrNoOutput
		}
	}
	return pages, nil
}

func ImageToPDF(
	ctx context.Context,
	img kernelports.Imaging,
	source kernelports.ImagingSource,
	limit int64,
	priority kernelports.ImagingPriority,
) ([]byte, error) {
	data, _, err := Merge(ctx, img, []Part{{Source: source}}, limit, priority)
	return data, err
}

func RenderPage(
	ctx context.Context,
	img kernelports.Imaging,
	req *kernelports.ImagingRenderRequest,
	limit int64,
) (*Rendered, error) {
	var out *Rendered
	_, err := img.Render(ctx, req, func(artifact *kernelports.ImagingArtifact) error {
		data, collectErr := Collect(artifact, limit)
		if collectErr != nil {
			return collectErr
		}
		out = &Rendered{
			Data:        data,
			ContentType: artifact.ContentType,
			Width:       artifact.Width,
			Height:      artifact.Height,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if out == nil {
		return nil, ErrNoOutput
	}
	return out, nil
}

type Range struct {
	First int
	Last  int
}

func ExtractRanges(
	ctx context.Context,
	img kernelports.Imaging,
	source kernelports.ImagingSource,
	ranges []Range,
	limit int64,
	priority kernelports.ImagingPriority,
) ([][]byte, error) {
	outputs := make([][]kernelports.ImagingPageRange, 0, len(ranges))
	for _, r := range ranges {
		outputs = append(
			outputs,
			[]kernelports.ImagingPageRange{{Source: 0, First: r.First, Last: r.Last}},
		)
	}

	files := make([][]byte, len(ranges))
	_, err := img.Assemble(ctx, &kernelports.ImagingAssembleRequest{
		Sources:  []kernelports.ImagingSource{source},
		Outputs:  outputs,
		Priority: priority,
	}, func(artifact *kernelports.ImagingArtifact) error {
		if artifact.Index < 0 || artifact.Index >= len(ranges) {
			return ErrNoOutput
		}
		data, collectErr := Collect(artifact, limit)
		files[artifact.Index] = data
		return collectErr
	})
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		if file == nil {
			return nil, ErrNoOutput
		}
	}
	return files, nil
}
