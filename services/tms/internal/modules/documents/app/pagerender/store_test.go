package pagerender

import (
	"bytes"
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
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
		Generator: thumbnailservice.NewGenerator(imaging),
		Config:    &config.Config{},
		Logger:    zap.NewNop(),
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
