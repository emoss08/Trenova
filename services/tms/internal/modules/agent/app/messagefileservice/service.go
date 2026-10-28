package messagefileservice

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/documentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/lrucache"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger *zap.Logger
	Files  documentports.ConversationFileReader
	// Imaging counts a PDF's pages from its bytes. Optional: without it a PDF's
	// pages are whatever reading the file counted.
	Imaging kernelports.Imaging `optional:"true"`
}

const (
	// A turn sends the same files on every model call of its tool loop, and a
	// retry sends them again, so a file read once is kept a while. A stored
	// file never changes under its document, only small files are kept, and
	// the cache is bounded both ways.
	cachedFiles     = 8
	cachedFileBytes = 8 << 20
	cachedFileTTL   = 10 * time.Minute
)

// Service reads a file attached to a conversation for a model that reads
// files itself. Only a conversation's files are read, through the documents
// service that decrypts them, as document intelligence reads an upload.
type Service struct {
	logger  *zap.Logger
	files   documentports.ConversationFileReader
	imaging kernelports.Imaging
	cache   lrucache.Cache[string, *assistantports.MessageFileContent]
}

func New(p Params) assistantports.MessageFileReader {
	return &Service{
		logger:  p.Logger.Named("message-files"),
		files:   p.Files,
		imaging: p.Imaging,
		cache: lrucache.NewExpirable[string, *assistantports.MessageFileContent](
			cachedFiles,
			cachedFileTTL,
		),
	}
}

func (s *Service) ReadMessageFile(
	ctx context.Context,
	req *assistantports.ReadMessageFileRequest,
) (*assistantports.MessageFileContent, error) {
	key := req.TenantInfo.OrgID.String() + "/" + req.TenantInfo.BuID.String() + "/" +
		req.DocumentID.String()
	if cached, ok := s.cache.Get(key); ok {
		return cached, nil
	}

	file, err := s.files.ReadConversationFile(ctx, documentports.GetDocumentByIDRequest{
		ID:         req.DocumentID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	content := &assistantports.MessageFileContent{
		Data:        file.Data,
		ContentType: file.Document.FileType,
	}
	counted := true
	if aiprovider.MediaType(file.Document.FileType) == "application/pdf" {
		content.PageCount, content.Unreadable, counted = s.pages(ctx, file.Data)
	}
	if counted && len(content.Data) <= cachedFileBytes {
		s.cache.Add(key, content)
	}

	return content, nil
}

// pages counts a PDF's pages, and reports one that will not open: encrypted
// or damaged, which a model refuses as well. The count is only metadata, so a
// worker that could not answer (unavailable, slow, or refusing the file's
// size) leaves it unknown rather than failing the read, and counted is false
// so the file is not cached without it.
func (s *Service) pages(
	ctx context.Context,
	data []byte,
) (count int, unreadable, counted bool) {
	if s.imaging == nil {
		return 0, false, true
	}

	probe, err := s.imaging.Probe(ctx, &kernelports.ImagingProbeRequest{
		Source:   kernelports.ImagingBytes("application/pdf", data),
		Priority: kernelports.ImagingInteractive,
	})
	if errors.Is(err, kernelports.ErrImagingInvalidDocument) ||
		errors.Is(err, kernelports.ErrImagingEncrypted) {
		s.logger.Debug("an attached PDF could not be opened", zap.Error(err))
		return 0, true, true
	}
	if err != nil {
		s.logger.Warn("the pages of an attached PDF could not be counted", zap.Error(err))
		return 0, false, false
	}
	if probe.Encrypted {
		return 0, true, true
	}

	return probe.PageCount, false, true
}
