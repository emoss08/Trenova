//go:build imagingbench

package documentextractionjobs

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/testutil/imagingbench"
	"github.com/emoss08/trenova/internal/testutil/pdftest"
	"go.uber.org/zap"
)

func TestLegacyOCRBaseline(t *testing.T) {
	corpus := imagingbench.Load(t)
	rec := imagingbench.NewRecorder()
	t.Cleanup(func() { rec.Flush(t) })

	a := &Activities{
		logger: zap.NewNop(),
		cfg: &config.AIConfig{
			OCRCommand:             "tesseract",
			OCRLanguage:            "eng",
			OCRTimeout:             2 * time.Minute,
			EnableOCRPreprocessing: true,
			OCRPreprocessingMode:   "standard",
			MaxOCRPages:            500,
		},
		pdfReader: pdftest.Reader(),
	}
	ctx := t.Context()

	for _, name := range []string{
		"scan-bol-1p-g4.pdf", "scan-bol-1p-gray.pdf", "scan-bol-5p-gray.pdf",
		"capture-text-0.pdf", "capture-text-1.pdf", "faxed-bol.pdf",
	} {
		data := corpus.Read(t, name)
		var recognised string
		var pages int
		latency := imagingbench.MeasureLatency(ctx, "legacy_ocr", name, 3, "pages",
			func(ctx context.Context) (int, error) {
				result, err := a.extractPDF(ctx, data, true)
				if err != nil {
					return 0, err
				}
				parts := make([]string, 0, len(result.Pages))
				for _, page := range result.Pages {
					parts = append(parts, page.Text)
				}
				recognised = strings.Join(parts, "\n")
				pages = len(result.Pages)
				return pages, nil
			})
		rec.AddLatency(latency)

		truth, ok := corpus.Truth(name)
		if !ok || pages == 0 {
			continue
		}
		rec.AddOCR(imagingbench.OCRAccuracy{
			Engine:        "tesseract-cli (legacy Go pipeline)",
			Input:         name,
			Pages:         pages,
			CER:           imagingbench.CER(truth, recognised),
			MeanPageMs:    latency.MeanMs / float64(pages),
			TruthChars:    len([]rune(truth)),
			RecognisedLen: len([]rune(recognised)),
		})
	}

	for _, name := range []string{"photo-bol.jpg", "photo-bol-exif6.jpg", "photo-bol.png"} {
		data := corpus.Read(t, name)
		ext := name[strings.LastIndexByte(name, '.'):]
		var recognised string
		latency := imagingbench.MeasureLatency(ctx, "legacy_ocr", name, 3, "pages",
			func(ctx context.Context) (int, error) {
				page, err := a.runOCRPage(ctx, data, ext, 1)
				if err != nil {
					return 0, err
				}
				recognised = page.Text
				return 1, nil
			})
		rec.AddLatency(latency)

		if truth, ok := corpus.Truth(name); ok && latency.Errors == 0 {
			rec.AddOCR(imagingbench.OCRAccuracy{
				Engine:        "tesseract-cli (legacy Go pipeline)",
				Input:         name,
				Pages:         1,
				CER:           imagingbench.CER(truth, recognised),
				MeanPageMs:    latency.MeanMs,
				TruthChars:    len([]rune(truth)),
				RecognisedLen: len([]rune(recognised)),
			})
		}
	}
}
