package messagefileservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiprovider"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
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
	cache   lrucache.Cache[string, *agentports.MessageFileContent]
}

func New(p Params) agentports.MessageFileReader {
	return &Service{
		logger:  p.Logger.Named("message-files"),
		files:   p.Files,
		imaging: p.Imaging,
		cache: lrucache.NewExpirable[string, *agentports.MessageFileContent](
			cachedFiles,
			cachedFileTTL,
		),
	}
}

func (s *Service) ReadMessageFile(
	ctx context.Context,
	req *agentports.ReadMessageFileRequest,
) (*agentports.MessageFileContent, error) {
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

	content := &agentports.MessageFileContent{
		Data:        file.Data,
		ContentType: file.Document.FileType,
	}
	if aiprovider.MediaType(file.Document.FileType) == "application/pdf" {
		content.PageCount, content.Unreadable, err = s.pages(ctx, file.Data)
		if err != nil {
			return nil, err
		}
	}
	if len(content.Data) <= cachedFileBytes {
		s.cache.Add(key, content)
	}

	return content, nil
}

// pages counts a PDF's pages, and reports one that will not open: encrypted
// or damaged, which a model refuses as well. A worker that could not answer
// is an error, so the file is neither marked unreadable nor cached.
func (s *Service) pages(
	ctx context.Context,
	data []byte,
) (count int, unreadable bool, err error) {
	if s.imaging == nil {
		return 0, false, nil
	}

	probe, err := s.imaging.Probe(ctx, &kernelports.ImagingProbeRequest{
		Source:   kernelports.ImagingBytes("application/pdf", data),
		Priority: kernelports.ImagingInteractive,
	})
	if errors.Is(err, kernelports.ErrImagingInvalidDocument) ||
		errors.Is(err, kernelports.ErrImagingEncrypted) {
		s.logger.Debug("an attached PDF could not be opened", zap.Error(err))
		return 0, true, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("count the pages of an attached PDF: %w", err)
	}
	if probe.Encrypted {
		return 0, true, nil
	}

	return probe.PageCount, false, nil
}
