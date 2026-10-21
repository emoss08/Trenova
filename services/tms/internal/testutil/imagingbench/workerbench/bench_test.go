//go:build imagingbench

package workerbench_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"iter"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/documents/app/storedobject"
	"github.com/emoss08/trenova/internal/modules/documents/app/thumbnailservice"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/imagingbench"
	"github.com/emoss08/trenova/internal/testutil/imagingtest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	iterations       = 15
	throughputWindow = 10 * time.Second
	rssChildEnv      = "TRENOVA_IMAGING_RSS_CHILD"
	workerCPUsEnv    = "TRENOVA_IMAGING_WORKER_CPUS"
	outputLimit      = 256 << 20
)

func TestWorker(t *testing.T) {
	corpus := imagingbench.Load(t)
	rec := imagingbench.NewRecorder()
	t.Cleanup(func() { rec.Flush(t) })

	client := imagingtest.Client(t)
	generator := thumbnailservice.NewGenerator(client)
	ctx := t.Context()

	if _, err := generator.Generate(
		ctx,
		kernelports.ImagingBytes("application/pdf", corpus.Read(t, "native-1p.pdf")),
		kernelports.ImagingBatch,
	); err != nil {
		t.Fatalf("warm the worker: %v", err)
	}

	t.Run("thumbnail", func(t *testing.T) {
		for _, name := range []string{
			"native-1p.pdf", "rate-confirmation.pdf", "scan-bol-1p-gray.pdf", "large-5mb.pdf",
			"photo-bol.jpg", "photo-bol.png",
		} {
			source := kernelports.ImagingBytes(contentTypeOf(name), corpus.Read(t, name))
			rec.AddLatency(imagingbench.MeasureLatency(ctx, "thumbnail", name, iterations, "",
				func(ctx context.Context) (int, error) {
					_, err := generator.Generate(ctx, source, kernelports.ImagingBatch)
					return 1, err
				}))
		}
	})

	t.Run("page_view", func(t *testing.T) {
		for _, name := range []string{
			"native-1p.pdf", "rate-confirmation.pdf", "scan-bol-1p-gray.pdf", "scan-bol-20p-g4.pdf",
			"native-200p.pdf", "large-20mb.pdf",
		} {
			source := kernelports.ImagingBytes("application/pdf", corpus.Read(t, name))
			rec.AddLatency(imagingbench.MeasureLatency(ctx, "page_view", name, iterations, "",
				func(ctx context.Context) (int, error) {
					_, err := generator.RenderPage(ctx, source, 1, kernelports.ImagingInteractive)
					return 1, err
				}))
		}
	})

	t.Run("capture_inspection", func(t *testing.T) {
		entries := corpus.Kind("capture-page")
		sources := make([]kernelports.ImagingSource, 0, len(entries))
		for _, entry := range entries {
			source := kernelports.ImagingBytes("application/pdf", corpus.Read(t, entry.File))
			sources = append(sources, source)
			rec.AddLatency(
				imagingbench.MeasureLatency(ctx, "capture_inspection", entry.File, iterations, "",
					func(ctx context.Context) (int, error) {
						return inspect(ctx, client, []kernelports.ImagingSource{source})
					}),
			)
		}
		rec.AddLatency(imagingbench.MeasureLatency(
			ctx,
			"capture_inspection",
			strconv.Itoa(len(sources))+" pages in one batch",
			iterations,
			"pages",
			func(ctx context.Context) (int, error) {
				return inspect(ctx, client, sources)
			},
		))
	})

	t.Run("text_profile", func(t *testing.T) {
		for _, name := range []string{"native-10p.pdf", "native-200p.pdf", "scan-bol-5p-gray.pdf", "rate-confirmation.pdf"} {
			source := kernelports.ImagingBytes("application/pdf", corpus.Read(t, name))
			rec.AddLatency(
				imagingbench.MeasureLatency(ctx, "text_profile", name, iterations, "pages",
					func(ctx context.Context) (int, error) {
						probe, err := client.Probe(ctx, &kernelports.ImagingProbeRequest{
							Source:      source,
							TextProfile: true,
							Priority:    kernelports.ImagingBatch,
						})
						if err != nil {
							return 0, err
						}
						return len(probe.Pages), nil
					}),
			)
		}
	})

	t.Run("assemble", func(t *testing.T) {
		native50 := kernelports.ImagingBytes("application/pdf", corpus.Read(t, "native-50p.pdf"))
		pages, err := imagingkit.SplitPages(ctx, client, native50, 50, outputLimit, kernelports.ImagingBatch)
		if err != nil {
			t.Fatalf("split native-50p for the merge input: %v", err)
		}
		mergeInput := make([]imagingkit.Part, 0, len(pages))
		for _, page := range pages {
			mergeInput = append(mergeInput, imagingkit.Part{
				Source: kernelports.ImagingBytes("application/pdf", page),
			})
		}
		mixed := []imagingkit.Part{
			{Source: kernelports.ImagingBytes("application/pdf", corpus.Read(t, "native-10p.pdf"))},
			{Source: kernelports.ImagingBytes("application/pdf", corpus.Read(t, "scan-bol-5p-gray.pdf"))},
			{Source: kernelports.ImagingBytes("application/pdf", corpus.Read(t, "rate-confirmation.pdf"))},
		}
		rotated := []imagingkit.Part{{
			Source:   kernelports.ImagingBytes("application/pdf", corpus.Read(t, "scan-bol-1p-gray.pdf")),
			Rotation: 90,
		}}

		rec.AddLatency(
			imagingbench.MeasureLatency(ctx, "merge", "50 one-page PDFs", iterations, "pages",
				func(ctx context.Context) (int, error) {
					_, n, mergeErr := imagingkit.Merge(ctx, client, mergeInput, outputLimit, kernelports.ImagingBatch)
					return n, mergeErr
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
					_, _, mergeErr := imagingkit.Merge(ctx, client, mixed, outputLimit, kernelports.ImagingBatch)
					return 1, mergeErr
				},
			),
		)
		rec.AddLatency(
			imagingbench.MeasureLatency(ctx, "rotate", "scan-bol-1p-gray.pdf", iterations, "",
				func(ctx context.Context) (int, error) {
					_, _, mergeErr := imagingkit.Merge(ctx, client, rotated, outputLimit, kernelports.ImagingBatch)
					return 1, mergeErr
				}),
		)
		for _, name := range []string{"native-50p.pdf", "scan-bol-20p-g4.pdf", "native-200p.pdf"} {
			source := kernelports.ImagingBytes("application/pdf", corpus.Read(t, name))
			pageCount := corpus.Find(t, name).Pages
			rec.AddLatency(imagingbench.MeasureLatency(ctx, "split", name, iterations, "pages",
				func(ctx context.Context) (int, error) {
					count, countErr := imagingkit.PageCount(ctx, client, source, kernelports.ImagingBatch)
					if countErr != nil {
						return 0, countErr
					}
					if pageCount > 0 && count != pageCount {
						return 0, fmt.Errorf("%s has %d pages, the worker counted %d", name, pageCount, count)
					}
					out, splitErr := imagingkit.SplitPages(
						ctx, client, source, count, outputLimit, kernelports.ImagingBatch,
					)
					return len(out), splitErr
				}))
		}
		photo := kernelports.ImagingBytes("image/jpeg", corpus.Read(t, "photo-bol.jpg"))
		rec.AddLatency(
			imagingbench.MeasureLatency(ctx, "image_to_pdf", "photo-bol.jpg", iterations, "",
				func(ctx context.Context) (int, error) {
					_, convertErr := imagingkit.ImageToPDF(ctx, client, photo, outputLimit, kernelports.ImagingBatch)
					return 1, convertErr
				}),
		)
	})

	t.Run("throughput", func(t *testing.T) {
		cores := workerCPUs(t)
		page := kernelports.ImagingBytes("application/pdf", corpus.Read(t, "scan-bol-1p-gray.pdf"))
		capture := kernelports.ImagingBytes("application/pdf", corpus.Read(t, "capture-text-0.pdf"))
		for _, concurrency := range []int{1, 4, 8} {
			rec.AddThroughput(
				imagingbench.MeasureThroughput(ctx, "page_view", "scan-bol-1p-gray.pdf",
					concurrency, throughputWindow, cores, func(ctx context.Context) (int, error) {
						_, err := generator.RenderPage(ctx, page, 1, kernelports.ImagingInteractive)
						return 1, err
					}),
			)
			rec.AddThroughput(
				imagingbench.MeasureThroughput(ctx, "thumbnail", "scan-bol-1p-gray.pdf",
					concurrency, throughputWindow, cores, func(ctx context.Context) (int, error) {
						_, err := generator.Generate(ctx, page, kernelports.ImagingBatch)
						return 1, err
					}),
			)
			rec.AddThroughput(
				imagingbench.MeasureThroughput(ctx, "capture_inspection", "capture-text-0.pdf",
					concurrency, throughputWindow, cores, func(ctx context.Context) (int, error) {
						return inspect(ctx, client, []kernelports.ImagingSource{capture})
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
		t.Skip("run by TestWorker in a child process")
	}
	corpus := imagingbench.Load(t)
	ctx := t.Context()

	generator := thumbnailservice.NewGenerator(imagingtest.Client(t))
	if _, err := generator.Generate(
		ctx,
		kernelports.ImagingBytes("application/pdf", corpus.Read(t, "native-1p.pdf")),
		kernelports.ImagingBatch,
	); err != nil {
		t.Fatalf("warm the worker: %v", err)
	}
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
	if _, err = generator.Generate(
		ctx,
		kernelports.ImagingBytes("application/pdf", stored),
		kernelports.ImagingBatch,
	); err != nil {
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

func inspect(
	ctx context.Context,
	client kernelports.Imaging,
	sources []kernelports.ImagingSource,
) (int, error) {
	summary, err := client.InspectCapture(ctx, &kernelports.ImagingCaptureRequest{
		PageCount: len(sources),
		Pages: func() iter.Seq2[kernelports.ImagingCapturePage, error] {
			return func(yield func(kernelports.ImagingCapturePage, error) bool) {
				for i, source := range sources {
					if !yield(kernelports.ImagingCapturePage{Index: i, Source: source}, nil) {
						return
					}
				}
			}
		},
		Priority: kernelports.ImagingBatch,
	}, func(result *kernelports.ImagingCaptureResult) error {
		if result.Err != nil {
			return result.Err
		}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return summary.Inspected, nil
}

func workerCPUs(t *testing.T) int {
	t.Helper()

	value := os.Getenv(workerCPUsEnv)
	if value == "" {
		t.Fatalf("%s is not set; run this through `task imaging:bench`", workerCPUsEnv)
	}
	cpus, err := strconv.Atoi(value)
	if err != nil || cpus < 1 {
		t.Fatalf("%s=%q is not a CPU count", workerCPUsEnv, value)
	}
	return cpus
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
