package pagerender

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/imagingconfig"
	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingfake"
	"github.com/emoss08/trenova/internal/modules/documents/app/thumbnailservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/testutil/objectstoretest"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type getResult struct {
	data []byte
	err  error
}

func TestGet_OneWaiterCancellingDoesNotFailTheOthers(t *testing.T) {
	t.Parallel()

	view := []byte("RIFF....WEBPVP8 page one")
	started := make(chan struct{})
	release := make(chan struct{})
	var renders atomic.Int32
	imaging := imagingfake.New()
	imaging.RenderFunc = func(
		ctx context.Context,
		_ *kernelports.ImagingRenderRequest,
		yield func(*kernelports.ImagingArtifact) error,
	) (*kernelports.ImagingRenderSummary, error) {
		if renders.Add(1) == 1 {
			close(started)
		}
		select {
		case <-release:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
		if err := yield(&kernelports.ImagingArtifact{
			Index:       0,
			ContentType: ContentType,
			Size:        int64(len(view)),
			Body:        bytes.NewReader(view),
		}); err != nil {
			return nil, err
		}
		return &kernelports.ImagingRenderSummary{PageCount: 1, Rendered: 1}, nil
	}

	store := New(Params{
		Storage: objectstoretest.New(),
		Encryption: encryptionservice.NewWithKeyManager(
			encryptionservice.NewLocalKeyManager("pagerender-test-key-material"),
		),
		Generator:     thumbnailservice.NewGenerator(imaging),
		ImagingConfig: &imagingconfig.Config{},
		Logger:        zap.NewNop(),
	})
	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	doc := &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: tenant.OrgID,
		BusinessUnitID: tenant.BuID,
		ChecksumSHA256: "abc123",
	}
	source := func(context.Context) (kernelports.ImagingSource, error) {
		return kernelports.ImagingBytes("application/pdf", []byte("%PDF-1.7")), nil
	}
	get := func(ctx context.Context, into chan<- getResult) {
		data, err := store.Get(ctx, tenant, doc, 1, source)
		into <- getResult{data: data, err: err}
	}

	firstCtx, cancelFirst := context.WithCancel(t.Context())
	first := make(chan getResult, 1)
	go get(firstCtx, first)
	<-started
	second := make(chan getResult, 1)
	go get(t.Context(), second)
	time.Sleep(100 * time.Millisecond)

	cancelFirst()
	select {
	case got := <-first:
		require.ErrorIs(t, got.err, context.Canceled, "the caller that cancelled returns at once")
	case <-time.After(5 * time.Second):
		t.Fatal("the cancelled caller kept waiting for the render")
	}

	close(release)
	select {
	case got := <-second:
		require.NoError(t, got.err, "the other waiter still gets the page")
		assert.Equal(t, view, got.data)
	case <-time.After(5 * time.Second):
		t.Fatal("the other waiter never got the page")
	}
	assert.Equal(t, int32(1), renders.Load(), "the page was rendered once")

	stored, err := store.Get(t.Context(), tenant, doc, 1, source)
	require.NoError(t, err)
	assert.Equal(t, view, stored)
	assert.Equal(t, int32(1), renders.Load(), "and stored for the next caller")
}

type prerenderHarness struct {
	store    *Store
	storage  *objectstoretest.Store
	tenant   pagination.TenantInfo
	doc      *document.Document
	requests [][]int
	failAt   int
}

func newPrerenderHarness(t *testing.T, pageCount int) *prerenderHarness {
	t.Helper()

	h := &prerenderHarness{storage: objectstoretest.New()}
	imaging := imagingfake.New()
	imaging.RenderFunc = func(
		_ context.Context,
		req *kernelports.ImagingRenderRequest,
		yield func(*kernelports.ImagingArtifact) error,
	) (*kernelports.ImagingRenderSummary, error) {
		h.requests = append(h.requests, append([]int(nil), req.Pages...))
		pages := req.Pages
		if len(pages) == 0 {
			for page := range min(pageCount, req.MaxPages) {
				pages = append(pages, page)
			}
		}
		rendered := 0
		for _, page := range pages {
			if h.failAt > 0 && page+1 == h.failAt {
				return nil, errors.New("the worker went away mid-render")
			}
			time.Sleep(5 * time.Millisecond)
			view := fmt.Appendf(nil, "RIFF....WEBPVP8 page %d", page+1)
			if err := yield(&kernelports.ImagingArtifact{
				Index:       page,
				ContentType: ContentType,
				Size:        int64(len(view)),
				Body:        bytes.NewReader(view),
			}); err != nil {
				return nil, err
			}
			rendered++
		}
		return &kernelports.ImagingRenderSummary{PageCount: pageCount, Rendered: rendered}, nil
	}
	h.store = New(Params{
		Storage: h.storage,
		Encryption: encryptionservice.NewWithKeyManager(
			encryptionservice.NewLocalKeyManager("pagerender-test-key-material"),
		),
		Generator:     thumbnailservice.NewGenerator(imaging),
		ImagingConfig: &imagingconfig.Config{},
		Logger:        zap.NewNop(),
	})
	h.tenant = pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	h.doc = &document.Document{
		ID:             pulid.MustNew("doc_"),
		OrganizationID: h.tenant.OrgID,
		BusinessUnitID: h.tenant.BuID,
		ChecksumSHA256: "abc123",
		FileType:       "application/pdf",
	}
	return h
}

func (h *prerenderHarness) prerender(t *testing.T, maxPages int) (int, []int, error) {
	t.Helper()

	stored, progress, err := h.prerenderKnowing(t, maxPages, 0)
	counts := make([]int, 0, len(progress))
	for _, p := range progress {
		counts = append(counts, p.PagesStored)
	}
	return stored, counts, err
}

func (h *prerenderHarness) prerenderKnowing(
	t *testing.T,
	maxPages, pageCount int,
) (int, []Progress, error) {
	t.Helper()

	var progress []Progress
	stored, err := h.store.Prerender(t.Context(), &PrerenderRequest{
		TenantInfo: h.tenant,
		Document:   h.doc,
		Source:     kernelports.ImagingBytes(h.doc.FileType, []byte("%PDF-1.7")),
		MaxPages:   maxPages,
		PageCount:  pageCount,
		Progress:   func(p Progress) { progress = append(progress, p) },
	})
	return stored, progress, err
}

func TestPrerender_ReportsProgressAfterEveryStoredPage(t *testing.T) {
	t.Parallel()

	h := newPrerenderHarness(t, 4)

	stored, progress, err := h.prerender(t, 20)

	require.NoError(t, err)
	assert.Equal(t, 4, stored)
	assert.Equal(t, []int{1, 2, 3, 4, 4}, progress, "one per page, then the finished count")
}

func TestPrerender_ARetryRendersOnlyThePagesNotYetStored(t *testing.T) {
	t.Parallel()

	h := newPrerenderHarness(t, 5)
	h.failAt = 3
	stored, _, err := h.prerender(t, 5)
	require.Error(t, err)
	require.Equal(t, 2, stored)
	firstPage, ok := h.storage.Object(Key(h.doc, 1))
	require.True(t, ok)

	h.failAt = 0
	h.requests = nil
	stored, progress, err := h.prerender(t, 5)

	require.NoError(t, err)
	assert.Equal(t, 5, stored)
	assert.Equal(t, [][]int{{2, 3, 4}}, h.requests, "only pages 3 to 5 are rendered")
	assert.Equal(t, []int{2, 3, 4, 5, 5}, progress)
	again, _ := h.storage.Object(Key(h.doc, 1))
	assert.Equal(t, firstPage, again, "a stored page is not written again")
	for page := 1; page <= 5; page++ {
		_, ok := h.storage.Object(Key(h.doc, page))
		assert.True(t, ok, "page %d is stored", page)
	}

	h.requests = nil
	stored, _, err = h.prerender(t, 5)
	require.NoError(t, err)
	assert.Equal(t, 5, stored)
	assert.Empty(t, h.requests, "a finished document is not rendered again")
}

func TestPrerender_AFinishedRunReportsThePageCount(t *testing.T) {
	t.Parallel()

	h := newPrerenderHarness(t, 3)

	_, progress, err := h.prerenderKnowing(t, 20, 0)

	require.NoError(t, err)
	require.NotEmpty(t, progress)
	assert.Equal(t, Progress{PagesStored: 3, PageCount: 3}, progress[len(progress)-1])
}

func TestPrerender_AFullyStoredPDFWithAKnownPageCountIsNotSentAgain(t *testing.T) {
	t.Parallel()

	h := newPrerenderHarness(t, 3)
	_, _, err := h.prerenderKnowing(t, 20, 0)
	require.NoError(t, err)
	h.requests = nil

	stored, _, err := h.prerenderKnowing(t, 20, 3)

	require.NoError(t, err)
	assert.Equal(t, 3, stored)
	assert.Empty(t, h.requests, "the worker is not asked again")
}

func TestPrerender_AStoredImageIsDone(t *testing.T) {
	t.Parallel()

	h := newPrerenderHarness(t, 1)
	h.doc.FileType = "image/png"
	stored, _, err := h.prerenderKnowing(t, 20, 0)
	require.NoError(t, err)
	require.Equal(t, 1, stored)
	h.requests = nil

	stored, _, err = h.prerenderKnowing(t, 20, 0)

	require.NoError(t, err)
	assert.Equal(t, 1, stored)
	assert.Empty(t, h.requests, "an image has one page, already stored")
}

func TestPrerender_AResumeTheWorkerRefusesAsOutOfRangeIsDone(t *testing.T) {
	t.Parallel()

	h := newPrerenderHarness(t, 2)
	_, _, err := h.prerenderKnowing(t, 20, 0)
	require.NoError(t, err)
	imaging := imagingfake.New()
	imaging.RenderFunc = func(
		context.Context,
		*kernelports.ImagingRenderRequest,
		func(*kernelports.ImagingArtifact) error,
	) (*kernelports.ImagingRenderSummary, error) {
		return nil, fmt.Errorf(
			"%w: page 3 is outside the document",
			kernelports.ErrImagingPageOutOfRange,
		)
	}
	h.store.generator = thumbnailservice.NewGenerator(imaging)

	stored, _, err := h.prerenderKnowing(t, 20, 0)

	require.NoError(t, err, "nothing past the stored pages is not a failure")
	assert.Equal(t, 2, stored)
}
