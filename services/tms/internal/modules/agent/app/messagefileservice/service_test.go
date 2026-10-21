package messagefileservice

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingfake"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

type fakeFiles struct {
	mu    sync.Mutex
	files map[pulid.ID]*documentports.ConversationFile
	reads int
	seen  []documentports.GetDocumentByIDRequest
}

func (f *fakeFiles) ReadConversationFile(
	_ context.Context,
	req documentports.GetDocumentByIDRequest,
) (*documentports.ConversationFile, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.reads++
	f.seen = append(f.seen, req)
	file, ok := f.files[req.ID]
	if !ok {
		return nil, errors.New("not found")
	}

	return file, nil
}

func fakePDFs(pages int) kernelports.Imaging {
	fake := imagingfake.New()
	fake.ProbeFunc = func(ctx context.Context, req *kernelports.ImagingProbeRequest) (*kernelports.ImagingProbe, error) {
		data, err := imagingfake.Read(ctx, req.Source)
		if err != nil {
			return nil, err
		}
		if len(data) < 4 || string(data[:4]) != "%PDF" {
			return nil, kernelports.ErrImagingInvalidDocument
		}
		return &kernelports.ImagingProbe{Kind: kernelports.ImagingKindPDF, PageCount: pages}, nil
	}
	return fake
}

func newTestReader(files *fakeFiles, imaging kernelports.Imaging) agentports.MessageFileReader {
	return New(Params{Logger: zap.NewNop(), Files: files, Imaging: imaging})
}

func attached(contentType string, data []byte) *documentports.ConversationFile {
	return &documentports.ConversationFile{
		Document: &document.Document{FileType: contentType},
		Data:     data,
	}
}

func TestReadMessageFile_CountsAPDFsPagesFromItsBytes(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	pdf, locked, photo := pulid.MustNew("doc_"), pulid.MustNew("doc_"), pulid.MustNew("doc_")
	files := &fakeFiles{files: map[pulid.ID]*documentports.ConversationFile{
		pdf:    attached("application/pdf", []byte("%PDF-1.7 rate confirmation")),
		locked: attached("application/pdf", []byte("encrypted bytes")),
		photo:  attached("image/png", []byte("\x89PNG photo")),
	}}
	reader := newTestReader(files, fakePDFs(7))

	content, err := reader.ReadMessageFile(t.Context(), &agentports.ReadMessageFileRequest{
		TenantInfo: tenant, DocumentID: pdf,
	})
	require.NoError(t, err)
	assert.Equal(t, 7, content.PageCount)
	assert.False(t, content.Unreadable)
	assert.Equal(t, "application/pdf", content.ContentType)
	assert.Equal(t, tenant, files.seen[0].TenantInfo, "the file is read within the request's tenant")

	content, err = reader.ReadMessageFile(t.Context(), &agentports.ReadMessageFileRequest{
		TenantInfo: tenant, DocumentID: locked,
	})
	require.NoError(t, err)
	assert.True(t, content.Unreadable, "a PDF that will not open is marked, so it goes as its text")

	content, err = reader.ReadMessageFile(t.Context(), &agentports.ReadMessageFileRequest{
		TenantInfo: tenant, DocumentID: photo,
	})
	require.NoError(t, err)
	assert.Zero(t, content.PageCount)
	assert.False(t, content.Unreadable, "an image is never opened as a PDF")
}

func TestReadMessageFile_WithoutTheImagingServiceLeavesThePagesUnknown(t *testing.T) {
	t.Parallel()

	pdf := pulid.MustNew("doc_")
	files := &fakeFiles{files: map[pulid.ID]*documentports.ConversationFile{
		pdf: attached("application/pdf", []byte("%PDF-1.7")),
	}}

	content, err := newTestReader(files, nil).ReadMessageFile(t.Context(),
		&agentports.ReadMessageFileRequest{DocumentID: pdf})

	require.NoError(t, err)
	assert.Zero(t, content.PageCount)
	assert.False(t, content.Unreadable)
}

/*
A turn sends its files on every model call of its tool loop, and a retry sends
them again. A small file is read once and kept a while; a large one is read
each time rather than held in memory; and one tenant's copy is never another's.
*/
func TestReadMessageFile_KeepsASmallFileItHasJustRead(t *testing.T) {
	t.Parallel()

	tenant := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	other := pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")}
	small, large := pulid.MustNew("doc_"), pulid.MustNew("doc_")
	files := &fakeFiles{files: map[pulid.ID]*documentports.ConversationFile{
		small: attached("image/png", []byte("\x89PNG photo")),
		large: attached("image/png", make([]byte, cachedFileBytes+1)),
	}}
	reader := newTestReader(files, nil)
	read := func(tenant pagination.TenantInfo, id pulid.ID) {
		_, err := reader.ReadMessageFile(t.Context(), &agentports.ReadMessageFileRequest{
			TenantInfo: tenant, DocumentID: id,
		})
		require.NoError(t, err)
	}

	read(tenant, small)
	read(tenant, small)
	assert.Equal(t, 1, files.reads, "the second read is served from what was kept")

	read(other, small)
	assert.Equal(t, 2, files.reads, "another tenant's read goes to the store")

	read(tenant, large)
	read(tenant, large)
	assert.Equal(t, 4, files.reads, "a large file is not kept")
}
