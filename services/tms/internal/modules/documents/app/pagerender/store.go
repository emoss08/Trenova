package pagerender

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/documents/app/storedobject"
	"github.com/emoss08/trenova/internal/modules/documents/app/thumbnailservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

const (
	ContentType    = "image/webp"
	variant        = "view-1400x1800-webp"
	maxRenderBytes = 32 << 20
)

type Params struct {
	fx.In

	Storage    kernelports.Client
	Encryption *encryptionservice.Service
	Generator  *thumbnailservice.Generator
	Config     *config.Config
	Logger     *zap.Logger
}

type Store struct {
	storage       kernelports.Client
	encryption    *encryptionservice.Service
	generator     *thumbnailservice.Generator
	group         singleflight.Group
	renderTimeout time.Duration
	logger        *zap.Logger
}

func New(p Params) *Store {
	return &Store{
		storage:       p.Storage,
		encryption:    p.Encryption,
		generator:     p.Generator,
		renderTimeout: p.Config.Imaging.Deadlines.GetRender(),
		logger:        p.Logger.Named("documents.pagerender"),
	}
}

type Source func(ctx context.Context) (kernelports.ImagingSource, error)

func Prefix(doc *document.Document) string {
	return fmt.Sprintf("%s/renders/%s/", doc.OrganizationID, doc.ID)
}

func Key(doc *document.Document, page int) string {
	return fmt.Sprintf("%s%s/%s/p%05d.webp", Prefix(doc), version(doc), variant, page)
}

func version(doc *document.Document) string {
	if checksum := strings.TrimSpace(doc.ChecksumSHA256); checksum != "" {
		return checksum
	}
	return fmt.Sprintf("v%d-%d", doc.VersionNumber, doc.UpdatedAt)
}

func (s *Store) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	doc *document.Document,
	page int,
	source Source,
) ([]byte, error) {
	key := Key(doc, page)
	stored, found, err := storedobject.ReadOptional(
		ctx, s.storage, s.encryption, key, storedobject.AAD(tenantInfo, key), maxRenderBytes,
	)
	if err != nil {
		s.logger.Warn("a stored page render could not be read; rendering it again",
			zap.String("documentId", doc.ID.String()), zap.Int("page", page), zap.Error(err))
	}
	if found {
		return stored, nil
	}

	pending := s.group.DoChan(key, func() (any, error) {
		renderCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), s.renderTimeout)
		defer cancel()
		src, srcErr := source(renderCtx)
		if srcErr != nil {
			return nil, srcErr
		}
		data, renderErr := s.generator.RenderPage(
			renderCtx,
			src,
			page,
			kernelports.ImagingInteractive,
		)
		if renderErr != nil {
			return nil, renderErr
		}
		if storeErr := s.write(renderCtx, tenantInfo, key, data); storeErr != nil {
			s.logger.Warn(
				"a page render could not be stored",
				zap.String(
					"documentId",
					doc.ID.String(),
				),
				zap.Int("page", page),
				zap.Error(storeErr),
			)
		}
		return data, nil
	})
	var result singleflight.Result
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case result = <-pending:
	}
	if result.Err != nil {
		return nil, result.Err
	}
	data, ok := result.Val.([]byte)
	if !ok {
		return nil, errors.New("the page render has an unexpected type")
	}
	return data, nil
}

type Progress struct {
	PagesStored int `json:"pagesStored"`
	PageCount   int `json:"pageCount,omitempty"`
}

type PrerenderRequest struct {
	TenantInfo pagination.TenantInfo
	Document   *document.Document
	Source     kernelports.ImagingSource
	MaxPages   int
	PageCount  int
	Progress   func(Progress)
}

func (s *Store) Prerender(ctx context.Context, req *PrerenderRequest) (int, error) {
	doc := req.Document
	report := func(progress Progress) {
		if req.Progress != nil {
			req.Progress(progress)
		}
	}
	limit := req.MaxPages
	if req.PageCount > 0 {
		limit = min(limit, req.PageCount)
	}
	if strings.HasPrefix(doc.FileType, "image/") {
		limit = min(limit, 1)
	}
	stored := s.storedPrefix(ctx, doc, limit)
	if stored >= limit {
		return stored, nil
	}
	if stored > 0 {
		report(Progress{PagesStored: stored, PageCount: req.PageCount})
	}
	resumed := stored
	pageCount, err := s.generator.RenderPagesFrom(
		ctx,
		req.Source,
		stored+1,
		limit,
		kernelports.ImagingBatch,
		func(page int, data []byte) error {
			if page <= resumed {
				return nil
			}
			if writeErr := s.write(ctx, req.TenantInfo, Key(doc, page), data); writeErr != nil {
				return writeErr
			}
			stored++
			report(Progress{PagesStored: stored, PageCount: req.PageCount})
			return nil
		},
	)
	if resumed > 0 && errors.Is(err, thumbnailservice.ErrPageOutOfRange) {
		return stored, nil
	}
	if err != nil {
		return stored, err
	}
	report(Progress{PagesStored: stored, PageCount: pageCount})
	return stored, nil
}

func (s *Store) storedPrefix(ctx context.Context, doc *document.Document, maxPages int) int {
	for page := 1; page <= maxPages; page++ {
		exists, err := s.storage.Exists(ctx, Key(doc, page))
		if err != nil {
			s.logger.Warn("could not check for a stored page render; rendering from it",
				zap.String("documentId", doc.ID.String()), zap.Int("page", page), zap.Error(err))
			return page - 1
		}
		if !exists {
			return page - 1
		}
	}
	return maxPages
}

func (s *Store) Delete(ctx context.Context, doc *document.Document) error {
	deleter, ok := s.storage.(kernelports.PrefixDeleter)
	if !ok {
		return nil
	}
	if _, err := deleter.DeletePrefix(ctx, Prefix(doc)); err != nil {
		return fmt.Errorf("delete the page renders of document %s: %w", doc.ID, err)
	}
	return nil
}

func (s *Store) write(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	key string,
	data []byte,
) error {
	if _, err := storedobject.Write(ctx, s.storage, s.encryption, &storedobject.WriteRequest{
		Key:         key,
		ContentType: ContentType,
		AAD:         storedobject.AAD(tenantInfo, key),
		Body:        bytes.NewReader(data),
		Size:        int64(len(data)),
	}); err != nil {
		return fmt.Errorf("store the page render: %w", err)
	}
	return nil
}
