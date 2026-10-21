package pagerender

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"strings"

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
	ContentType = "image/webp"
	variant     = "view-1400x1800-webp"
)

type Params struct {
	fx.In

	Storage    kernelports.Client
	Encryption *encryptionservice.Service
	Generator  *thumbnailservice.Generator
	Logger     *zap.Logger
}

type Store struct {
	storage    kernelports.Client
	encryption *encryptionservice.Service
	generator  *thumbnailservice.Generator
	group      singleflight.Group
	logger     *zap.Logger
}

func New(p Params) *Store {
	return &Store{
		storage:    p.Storage,
		encryption: p.Encryption,
		generator:  p.Generator,
		logger:     p.Logger.Named("documents.pagerender"),
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
		ctx, s.storage, s.encryption, key, storedobject.AAD(tenantInfo, key),
	)
	if err != nil {
		s.logger.Warn("a stored page render could not be read; rendering it again",
			zap.String("documentId", doc.ID.String()), zap.Int("page", page), zap.Error(err))
	}
	if found {
		return stored, nil
	}

	rendered, err, _ := s.group.Do(key, func() (any, error) {
		src, srcErr := source(ctx)
		if srcErr != nil {
			return nil, srcErr
		}
		data, renderErr := s.generator.RenderPage(ctx, src, page, kernelports.ImagingInteractive)
		if renderErr != nil {
			return nil, renderErr
		}
		if storeErr := s.write(ctx, tenantInfo, key, data); storeErr != nil {
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
	if err != nil {
		return nil, err
	}
	data, ok := rendered.([]byte)
	if !ok {
		return nil, errors.New("the page render has an unexpected type")
	}
	return data, nil
}

func (s *Store) Prerender(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	doc *document.Document,
	source kernelports.ImagingSource,
	maxPages int,
) (int, error) {
	stored := 0
	_, err := s.generator.RenderPages(ctx, source, maxPages, kernelports.ImagingBatch,
		func(page int, data []byte) error {
			if writeErr := s.write(ctx, tenantInfo, Key(doc, page), data); writeErr != nil {
				return writeErr
			}
			stored++
			return nil
		})
	if err != nil {
		return stored, err
	}
	return stored, nil
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
	if s.encryption == nil {
		return errors.New("document encryption is not configured")
	}
	envelope, err := s.encryption.EncryptBytesWithAAD(data, storedobject.AAD(tenantInfo, key))
	if err != nil {
		return fmt.Errorf("encrypt the page render: %w", err)
	}
	if _, err = s.storage.Upload(ctx, &kernelports.UploadParams{
		Key:         key,
		ContentType: "application/octet-stream",
		Size:        int64(len(envelope)),
		Body:        bytes.NewReader([]byte(envelope)),
	}); err != nil {
		return fmt.Errorf("store the page render: %w", err)
	}
	return nil
}
