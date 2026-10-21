//go:build imagingbench

package gobaseline_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/captureimaging"
	"github.com/emoss08/trenova/internal/infrastructure/pdfassembly"
	"github.com/emoss08/trenova/internal/infrastructure/pdfrender/pdfreader"
	"github.com/emoss08/trenova/internal/modules/documents/app/storedobject"
	"github.com/emoss08/trenova/internal/modules/documents/app/thumbnailservice"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/imagingbench"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	iterations       = 15
	throughputWindow = 10 * time.Second
	rssChildEnv      = "TRENOVA_IMAGING_RSS_CHILD"
)

func TestBaseline(t *testing.T) {
	corpus := imagingbench.Load(t)
	rec := imagingbench.NewRecorder()
	t.Cleanup(func() { rec.Flush(t) })

	reader := pdfreader.NewReader(nil, nil)
	t.Cleanup(func() { _ = reader.Close() })
	generator := thumbnailservice.NewGenerator(reader)
	inspector := captureimaging.New(reader)
	assembler := pdfassembly.New()
	images := pdfassembly.NewImageConverter()
	ctx := t.Context()

	warm, err := reader.Open(ctx, corpus.Read(t, "native-1p.pdf"))
	if err != nil {
		t.Fatalf("warm the PDF reader: %v", err)
	}
	_ = warm.Close()

	t.Run("thumbnail", func(t *testing.T) {
		for _, name := range []string{
			"native-1p.pdf", "rate-confirmation.pdf", "scan-bol-1p-gray.pdf", "large-5mb.pdf",
			"photo-bol.jpg", "photo-bol.png",
		} {
			data := corpus.Read(t, name)
			contentType := contentTypeOf(name)
			rec.AddLatency(imagingbench.MeasureLatency(ctx, "thumbnail", name, iterations, "",
				func(ctx context.Context) (int, error) {
					_, err := generator.Generate(ctx, bytes.NewReader(data), contentType)
					return 1, err
				}))
		}
	})

	t.Run("page_view", func(t *testing.T) {
		for _, name := range []string{
			"native-1p.pdf", "rate-confirmation.pdf", "scan-bol-1p-gray.pdf", "scan-bol-20p-g4.pdf",
			"native-200p.pdf", "large-20mb.pdf",
		} {
			data := corpus.Read(t, name)
			rec.AddLatency(imagingbench.MeasureLatency(ctx, "page_view", name, iterations, "",
				func(ctx context.Context) (int, error) {
					_, err := generator.RenderPage(ctx, bytes.NewReader(data), "application/pdf", 1)
					return 1, err
				}))
		}
	})

	t.Run("capture_inspection", func(t *testing.T) {
		for _, entry := range corpus.Kind("capture-page") {
			data := corpus.Read(t, entry.File)
			rec.AddLatency(
				imagingbench.MeasureLatency(ctx, "capture_inspection", entry.File, iterations, "",
					func(ctx context.Context) (int, error) {
						_, err := inspector.Inspect(ctx, data)
						return 1, err
					}),
			)
		}
	})

	t.Run("text_profile", func(t *testing.T) {
		for _, name := range []string{"native-10p.pdf", "native-200p.pdf", "scan-bol-5p-gray.pdf", "rate-confirmation.pdf"} {
			data := corpus.Read(t, name)
			rec.AddLatency(
				imagingbench.MeasureLatency(ctx, "text_profile", name, iterations, "pages",
					func(ctx context.Context) (int, error) {
						return textProfile(ctx, reader, data)
					}),
			)
		}
	})

	t.Run("assemble", func(t *testing.T) {
		pages, err := assembler.Split(ctx, corpus.Read(t, "native-50p.pdf"))
		if err != nil {
			t.Fatalf("split native-50p for the merge input: %v", err)
		}
		mergeInput := make([]captureports.CaptureAssemblyPage, 0, len(pages))
		for _, page := range pages {
			mergeInput = append(mergeInput, captureports.CaptureAssemblyPage{PDF: page})
		}
		mixed := []captureports.CaptureAssemblyPage{
			{PDF: corpus.Read(t, "native-10p.pdf")},
			{PDF: corpus.Read(t, "scan-bol-5p-gray.pdf")},
			{PDF: corpus.Read(t, "rate-confirmation.pdf")},
		}
		rotated := []captureports.CaptureAssemblyPage{
			{PDF: corpus.Read(t, "scan-bol-1p-gray.pdf"), Rotation: 90},
		}

		rec.AddLatency(
			imagingbench.MeasureLatency(ctx, "merge", "50 one-page PDFs", iterations, "pages",
				func(ctx context.Context) (int, error) {
					_, err := assembler.Assemble(ctx, mergeInput)
					return len(mergeInput), err
				}),
		)
		rec.AddLatency(
			imagingbench.MeasureLatency(
				ctx,
				"merge",
				"native-10p+scan-bol-5p+rate-confirmation",
				iterations,
				"",
				func(ctx context.Context) (int, error) {
					_, err := assembler.Assemble(ctx, mixed)
					return 1, err
				},
			),
		)
		rec.AddLatency(
			imagingbench.MeasureLatency(ctx, "rotate", "scan-bol-1p-gray.pdf", iterations, "",
				func(ctx context.Context) (int, error) {
					_, err := assembler.Assemble(ctx, rotated)
					return 1, err
				}),
		)
		for _, name := range []string{"native-50p.pdf", "scan-bol-20p-g4.pdf", "native-200p.pdf"} {
			data := corpus.Read(t, name)
			rec.AddLatency(imagingbench.MeasureLatency(ctx, "split", name, iterations, "pages",
				func(ctx context.Context) (int, error) {
					out, err := assembler.Split(ctx, data)
					return len(out), err
				}))
		}
		photo := corpus.Read(t, "photo-bol.jpg")
		rec.AddLatency(
			imagingbench.MeasureLatency(ctx, "image_to_pdf", "photo-bol.jpg", iterations, "",
				func(ctx context.Context) (int, error) {
					_, err := images.ImageToPDF(ctx, photo)
					return 1, err
				}),
		)
	})

	t.Run("throughput", func(t *testing.T) {
		cores := runtime.GOMAXPROCS(0)
		page := corpus.Read(t, "scan-bol-1p-gray.pdf")
		capture := corpus.Read(t, "capture-text-0.pdf")
		for _, concurrency := range []int{1, 4, 8} {
			rec.AddThroughput(
				imagingbench.MeasureThroughput(ctx, "page_view", "scan-bol-1p-gray.pdf",
					concurrency, throughputWindow, cores, func(ctx context.Context) (int, error) {
						_, err := generator.RenderPage(
							ctx,
							bytes.NewReader(page),
							"application/pdf",
							1,
						)
						return 1, err
					}),
			)
			rec.AddThroughput(
				imagingbench.MeasureThroughput(ctx, "thumbnail", "scan-bol-1p-gray.pdf",
					concurrency, throughputWindow, cores, func(ctx context.Context) (int, error) {
						_, err := generator.Generate(ctx, bytes.NewReader(page), "application/pdf")
						return 1, err
					}),
			)
			rec.AddThroughput(
				imagingbench.MeasureThroughput(ctx, "capture_inspection", "capture-text-0.pdf",
					concurrency, throughputWindow, cores, func(ctx context.Context) (int, error) {
						_, err := inspector.Inspect(ctx, capture)
						return 1, err
					}),
			)
		}
	})

	t.Run("peak_rss", func(t *testing.T) {
		for _, name := range []string{"large-5mb.pdf", "large-20mb.pdf", "large-50mb.pdf"} {
			rec.AddPeakRSS(runRSSChild(t, corpus, name))
		}
	})
}

func TestPeakRSSChild(t *testing.T) {
	name := os.Getenv(rssChildEnv)
	if name == "" {
		t.Skip("run by TestBaseline in a child process")
	}
	corpus := imagingbench.Load(t)
	ctx := t.Context()

	reader := pdfreader.NewReader(nil, nil)
	generator := thumbnailservice.NewGenerator(reader)
	warm, err := reader.Open(ctx, corpus.Read(t, "native-1p.pdf"))
	if err != nil {
		t.Fatalf("warm the PDF reader: %v", err)
	}
	_ = warm.Close()
	runtime.GC()
	baseline, err := imagingbench.CurrentRSSMB()
	if err != nil {
		t.Fatalf("read RSS: %v", err)
	}

	encryption := encryptionservice.NewWithKeyManager(
		encryptionservice.NewLocalKeyManager("imaging-bench-local-key-material"),
	)
	storage := &fileStorage{dir: t.TempDir()}
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	key := "documents/bench/" + name
	aad := storedobject.AAD(tenant, key)

	start := time.Now()
	file, err := os.Open(corpus.Path(name))
	if err != nil {
		t.Fatalf("open %s: %v", name, err)
	}
	plaintext, err := io.ReadAll(file)
	_ = file.Close()
	if err != nil {
		t.Fatalf("read upload: %v", err)
	}
	envelope, err := encryption.EncryptBytesWithAAD(plaintext, aad)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	plaintext = nil
	if _, err = storage.Upload(ctx, &kernelports.UploadParams{
		Key:         key,
		Body:        bytes.NewReader([]byte(envelope)),
		Size:        int64(len(envelope)),
		ContentType: "application/octet-stream",
	}); err != nil {
		t.Fatalf("store: %v", err)
	}
	envelope = ""

	stored, err := storedobject.Read(ctx, storage, encryption, key, aad)
	if err != nil {
		t.Fatalf("read stored document: %v", err)
	}
	if _, err = generator.Generate(ctx, bytes.NewReader(stored), "application/pdf"); err != nil {
		t.Fatalf("thumbnail: %v", err)
	}
	elapsed := time.Since(start)

	peak, err := imagingbench.PeakRSSMB()
	if err != nil {
		t.Fatalf("read peak RSS: %v", err)
	}
	info, _ := os.Stat(corpus.Path(name))
	out, _ := sonic.Marshal(imagingbench.PeakRSS{
		Input:      name,
		InputBytes: info.Size(),
		Pipeline:   "upload -> encrypt -> store -> download -> decrypt -> thumbnail",
		PeakRSSMB:  peak,
		BaselineMB: baseline,
		Seconds:    elapsed.Seconds(),
	})
	fmt.Printf("RSS-RESULT %s\n", out)
}

func runRSSChild(t *testing.T, corpus *imagingbench.Corpus, name string) imagingbench.PeakRSS {
	t.Helper()

	cmd := exec.CommandContext(t.Context(), os.Args[0], "-test.run=^TestPeakRSSChild$", "-test.v")
	cmd.Env = append(os.Environ(), rssChildEnv+"="+name, imagingbench.CorpusEnv+"="+corpus.Dir)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("RSS child for %s: %v\n%s", name, err, out)
	}
	marker := []byte("RSS-RESULT ")
	idx := bytes.Index(out, marker)
	if idx < 0 {
		t.Fatalf("RSS child for %s printed no result:\n%s", name, out)
	}
	line := out[idx+len(marker):]
	if end := bytes.IndexByte(line, '\n'); end >= 0 {
		line = line[:end]
	}
	var result imagingbench.PeakRSS
	if err = sonic.Unmarshal(line, &result); err != nil {
		t.Fatalf("parse RSS result for %s: %v", name, err)
	}

	return result
}

func textProfile(ctx context.Context, reader *pdfreader.Reader, data []byte) (int, error) {
	doc, err := reader.Open(ctx, data)
	if err != nil {
		return 0, err
	}
	defer doc.Close()

	native := make(map[int]bool, doc.PageCount())
	for page := range doc.PageCount() {
		text, textErr := doc.PageText(ctx, page)
		native[page+1] = textErr == nil && strings.TrimSpace(text) != ""
	}

	return len(native), nil
}

func contentTypeOf(name string) string {
	switch {
	case strings.HasSuffix(name, ".jpg"):
		return "image/jpeg"
	case strings.HasSuffix(name, ".png"):
		return "image/png"
	default:
		return "application/pdf"
	}
}

type fileStorage struct {
	kernelports.Client
	dir string
}

func (s *fileStorage) Upload(
	_ context.Context,
	params *kernelports.UploadParams,
) (*kernelports.FileInfo, error) {
	path := s.dir + "/" + strings.ReplaceAll(params.Key, "/", "_")
	file, err := os.Create(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	n, err := io.Copy(file, params.Body)
	if err != nil {
		return nil, err
	}

	return &kernelports.FileInfo{Key: params.Key, Size: n}, nil
}

func (s *fileStorage) Download(_ context.Context, key string) (*kernelports.DownloadResult, error) {
	file, err := os.Open(s.dir + "/" + strings.ReplaceAll(key, "/", "_"))
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}

	return &kernelports.DownloadResult{Body: file, Size: info.Size()}, nil
}
