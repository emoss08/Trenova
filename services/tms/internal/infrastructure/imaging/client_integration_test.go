//go:build integration

package imaging_test

import (
	"bytes"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"io"
	"iter"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/imagingtest"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	_ "golang.org/x/image/webp"
)

func TestProbeAgainstTheWorker(t *testing.T) {
	client := imagingtest.Client(t)

	probe, err := client.Probe(t.Context(), &kernelports.ImagingProbeRequest{
		Source: kernelports.ImagingBytes(
			"application/pdf",
			imagingtest.Fixture(t, "rate-confirmation.pdf"),
		),
		TextProfile: true,
	})
	require.NoError(t, err)
	assert.Equal(t, kernelports.ImagingKindPDF, probe.Kind)
	assert.Equal(t, 2, probe.PageCount)
	assert.True(t, probe.Pages[0].HasText)

	encrypted, err := client.Probe(t.Context(), &kernelports.ImagingProbeRequest{
		Source: kernelports.ImagingBytes(
			"application/pdf",
			imagingtest.Fixture(t, "hostile-encrypted.pdf"),
		),
	})
	require.NoError(t, err)
	assert.True(t, encrypted.Encrypted)

	_, err = client.Probe(t.Context(), &kernelports.ImagingProbeRequest{
		Source: kernelports.ImagingBytes(
			"application/pdf",
			imagingtest.Fixture(t, "hostile-garbage.pdf"),
		),
	})
	require.ErrorIs(t, err, kernelports.ErrImagingInvalidDocument)
}

func TestRenderAgainstTheWorker(t *testing.T) {
	client := imagingtest.Client(t)

	var sizes [][2]int
	summary, err := client.Render(t.Context(), &kernelports.ImagingRenderRequest{
		Source: kernelports.ImagingBytes(
			"application/pdf",
			imagingtest.Fixture(t, "rate-confirmation.pdf"),
		),
		FitWidth:  1400,
		FitHeight: 1800,
		Format:    kernelports.ImagingWebP,
	}, func(artifact *kernelports.ImagingArtifact) error {
		require.Nil(t, artifact.Err)
		data, readErr := io.ReadAll(artifact.Body)
		require.NoError(t, readErr)
		decoded, _, decodeErr := image.Decode(bytes.NewReader(data))
		require.NoError(t, decodeErr)
		sizes = append(sizes, [2]int{decoded.Bounds().Dx(), decoded.Bounds().Dy()})
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Rendered)
	assert.Equal(t, [][2]int{{1391, 1800}, {1391, 1800}}, sizes)
}

func TestInspectCaptureAgainstTheWorker(t *testing.T) {
	client := imagingtest.Client(t)

	pages := [][]byte{
		imagingtest.Fixture(t, "capture-cover-2qr.pdf"),
		imagingtest.Fixture(t, "capture-blank.pdf"),
		nil,
	}
	results := map[int]*kernelports.ImagingCaptureResult{}
	summary, err := client.InspectCapture(t.Context(), &kernelports.ImagingCaptureRequest{
		PageCount: len(pages),
		Pages: func() iter.Seq2[kernelports.ImagingCapturePage, error] {
			return func(yield func(kernelports.ImagingCapturePage, error) bool) {
				for i, page := range pages {
					if !yield(kernelports.ImagingCapturePage{
						Index:  i,
						Source: kernelports.ImagingBytes("application/pdf", page),
					}, nil) {
						return
					}
				}
			}
		},
	}, func(result *kernelports.ImagingCaptureResult) error {
		results[result.Index] = result
		return nil
	})
	require.NoError(t, err)
	assert.Equal(t, 2, summary.Inspected)
	assert.Equal(t, 1, summary.Failed)
	assert.Len(t, results[0].Codes, 2)
	assert.Less(t, results[1].InkCoverage, 0.005)
	require.NotNil(t, results[2].Err)
}

func TestAssembleAgainstTheWorker(t *testing.T) {
	client := imagingtest.Client(t)
	native := kernelports.ImagingBytes("application/pdf", imagingtest.Fixture(t, "native-10p.pdf"))

	merged, pages, err := imagingkit.Merge(t.Context(), client, []imagingkit.Part{
		{Source: native},
		{
			Source: kernelports.ImagingBytes(
				"application/pdf",
				imagingtest.Fixture(t, "native-1p.pdf"),
			),
			Rotation: 90,
		},
	}, 64<<20, kernelports.ImagingBatch)
	require.NoError(t, err)
	assert.Equal(t, 11, pages)
	count, err := imagingkit.PageCount(
		t.Context(),
		client,
		kernelports.ImagingBytes("application/pdf", merged),
		kernelports.ImagingBatch,
	)
	require.NoError(t, err)
	assert.Equal(t, 11, count)

	split, err := imagingkit.SplitPages(
		t.Context(),
		client,
		native,
		10,
		64<<20,
		kernelports.ImagingBatch,
	)
	require.NoError(t, err)
	assert.Len(t, split, 10)

	ranges, err := imagingkit.ExtractRanges(
		t.Context(),
		client,
		native,
		[]imagingkit.Range{
			{First: 0, Last: 2},
			{First: 3, Last: 9},
		},
		64<<20,
		kernelports.ImagingBatch,
	)
	require.NoError(t, err)
	require.Len(t, ranges, 2)
	second, err := imagingkit.PageCount(
		t.Context(),
		client,
		kernelports.ImagingBytes("application/pdf", ranges[1]),
		kernelports.ImagingBatch,
	)
	require.NoError(t, err)
	assert.Equal(t, 7, second)
}

func TestNormalizeImageAgainstTheWorker(t *testing.T) {
	client := imagingtest.Client(t)

	var png []byte
	summary, err := client.NormalizeImage(t.Context(), &kernelports.ImagingNormalizeRequest{
		Source: kernelports.ImagingBytes("image/webp", imagingtest.Fixture(t, "photo.webp")),
		Format: kernelports.ImagingPNG,
	}, func(artifact *kernelports.ImagingArtifact) error {
		data, collectErr := imagingkit.Collect(artifact, 16<<20)
		png = data
		return collectErr
	})
	require.NoError(t, err)
	assert.Equal(t, "webp", summary.SourceFormat)
	decoded, format, err := image.Decode(bytes.NewReader(png))
	require.NoError(t, err)
	assert.Equal(t, "png", format)
	assert.Equal(t, 320, decoded.Bounds().Dx())
}
