package photoservice

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"mime/multipart"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingkit"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/objectkey"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/personphoto"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	MaxUploadBytes = 20 << 20

	fileField        = "file"
	photoContentType = "image/jpeg"
	photoFileName    = "photo.jpg"
	maxOutputBytes   = 8 << 20
	sniffBytes       = 512

	signWindow     = 10 * time.Minute
	signedLifetime = 70 * time.Minute
	discardTimeout = 30 * time.Second
)

type variant struct {
	edge    int
	quality int
}

var (
	fullVariant      = variant{edge: 1024, quality: 85}
	thumbnailVariant = variant{edge: 256, quality: 80}
)

type Params struct {
	fx.In

	Logger        *zap.Logger
	Storage       kernelports.Client
	StorageConfig *storageconfig.Config
	Imaging       kernelports.Imaging
	Scanner       kernelports.UploadScanner
	Clock         timeutils.Clock
}

type Service struct {
	l          *zap.Logger
	storage    kernelports.Client
	storageCfg *storageconfig.Config
	imaging    kernelports.Imaging
	scanner    kernelports.UploadScanner
	clock      timeutils.Clock
}

var _ kernelports.Photos = (*Service)(nil)

//nolint:gocritic // fx passes params by value
func New(p Params) *Service {
	return &Service{
		l:          p.Logger.Named("service.photo"),
		storage:    p.Storage,
		storageCfg: p.StorageConfig,
		imaging:    p.Imaging,
		scanner:    p.Scanner,
		clock:      p.Clock,
	}
}

func (s *Service) Store(
	ctx context.Context,
	req *kernelports.StorePhotoRequest,
) (*kernelports.StoredPhoto, error) {
	if err := s.validate(req.File); err != nil {
		return nil, err
	}

	content, err := fileutils.ReadMultipart(req.File)
	if err != nil {
		return nil, fmt.Errorf("read the uploaded photo: %w", err)
	}
	contentType, err := s.sniff(content)
	if err != nil {
		return nil, err
	}

	if err = s.scanner.ScanUpload(ctx, &kernelports.UploadScanRequest{
		TenantInfo: req.TenantInfo,
		Field:      fileField,
		FileName:   req.File.Filename,
		Content:    content,
	}); err != nil {
		return nil, err
	}

	full, err := s.normalize(ctx, kernelports.ImagingBytes(contentType, content), fullVariant)
	if err != nil {
		return nil, err
	}
	thumbnail, err := s.normalize(
		ctx,
		kernelports.ImagingBytes(photoContentType, full),
		thumbnailVariant,
	)
	if err != nil {
		return nil, err
	}

	stored := &kernelports.StoredPhoto{
		Key:          newKey(req.Holder),
		ThumbnailKey: newKey(req.Holder),
	}
	if err = s.upload(ctx, req, stored.Key, full); err != nil {
		return nil, err
	}
	if err = s.upload(ctx, req, stored.ThumbnailKey, thumbnail); err != nil {
		s.Discard(ctx, &kernelports.PhotoKeys{Holder: req.Holder, Key: stored.Key})
		return nil, err
	}

	return stored, nil
}

func (s *Service) Sign(
	ctx context.Context,
	keys *kernelports.PhotoKeys,
) (*personphoto.URLs, error) {
	if keys == nil || keys.Key == "" {
		return nil, nil //nolint:nilnil // no photo is not an error
	}

	thumbnailKey := keys.ThumbnailKey
	if thumbnailKey == "" {
		thumbnailKey = keys.Key
	}

	signedAt := time.Unix(s.clock(), 0).UTC().Truncate(signWindow)
	url, err := s.sign(ctx, keys.Holder, keys.Key, signedAt)
	if err != nil || url == "" {
		return nil, err
	}
	thumbnailURL, err := s.sign(ctx, keys.Holder, thumbnailKey, signedAt)
	if err != nil {
		return nil, err
	}
	if thumbnailURL == "" {
		thumbnailURL = url
	}

	return &personphoto.URLs{URL: url, ThumbnailURL: thumbnailURL}, nil
}

func (s *Service) Discard(ctx context.Context, keys *kernelports.PhotoKeys) {
	if keys == nil {
		return
	}

	deleteCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), discardTimeout)
	defer cancel()

	for _, key := range []string{keys.Key, keys.ThumbnailKey} {
		if key == "" || fileutils.IsExternalURL(key) || !held(key, keys.Holder) {
			continue
		}
		err := s.storage.Delete(deleteCtx, key)
		if err != nil && !errors.Is(err, kernelports.ErrObjectNotFound) {
			s.l.Warn("failed to delete a photo object", zap.String("objectKey", key), zap.Error(err))
		}
	}
}

func newKey(holder kernelports.PhotoHolder) string {
	return objectkey.Organization.NewKey(
		holder.OrganizationID,
		holder.Folder+"/"+holder.ID.String(),
		photoFileName,
	)
}

func held(key string, holder kernelports.PhotoHolder) bool {
	if holder.ID.IsNil() || holder.Folder == "" || !objectkey.OwnedBy(key, holder.OrganizationID) {
		return false
	}
	return strings.HasPrefix(
		key,
		objectkey.Organization.Key(holder.OrganizationID, holder.Folder, holder.ID.String())+"/",
	)
}

func (s *Service) sign(
	ctx context.Context,
	holder kernelports.PhotoHolder,
	key string,
	signedAt time.Time,
) (string, error) {
	if fileutils.IsExternalURL(key) {
		return key, nil
	}
	if !held(key, holder) {
		s.l.Warn("refused to sign a photo its holder does not own",
			zap.String("holderID", holder.ID.String()),
			zap.String("objectKey", key),
		)
		return "", nil
	}

	url, err := s.storage.GetPresignedURL(ctx, &kernelports.PresignedURLParams{
		Key:      key,
		Expiry:   signedLifetime,
		SignedAt: signedAt,
	})
	if err != nil {
		return "", fmt.Errorf("sign the photo: %w", err)
	}
	return url, nil
}

func (s *Service) maxBytes() int64 {
	return min(s.storageCfg.GetMaxFileSize(), MaxUploadBytes)
}

func (s *Service) validate(file *multipart.FileHeader) error {
	me := errortypes.NewMultiError()
	if file == nil {
		me.Add(fileField, errortypes.ErrRequired, "A photo file is required")
		return me.Err()
	}
	if file.Size == 0 {
		me.Add(fileField, errortypes.ErrRequired, "The photo file cannot be empty")
	}
	if file.Size > s.maxBytes() {
		me.Add(
			fileField,
			errortypes.ErrInvalidLength,
			"The photo exceeds the maximum allowed size",
		)
	}
	if !fileutils.HasSupportedImageExtension(file.Filename) {
		me.Add(fileField, errortypes.ErrInvalidFormat, "Unsupported photo file extension")
	}
	return me.Err()
}

func (s *Service) sniff(content []byte) (string, error) {
	if int64(len(content)) > s.maxBytes() {
		return "", errortypes.NewValidationError(
			fileField,
			errortypes.ErrInvalidLength,
			"The photo exceeds the maximum allowed size",
		)
	}
	contentType := http.DetectContentType(content[:min(len(content), sniffBytes)])
	if !slices.Contains(fileutils.AllowedImageMIMETypes, contentType) {
		return "", errortypes.NewValidationError(
			fileField,
			errortypes.ErrInvalidFormat,
			"Only image files are allowed for photos",
		)
	}
	return contentType, nil
}

func (s *Service) normalize(
	ctx context.Context,
	source kernelports.ImagingSource,
	size variant,
) ([]byte, error) {
	rendered, err := imagingkit.Normalize(ctx, s.imaging, &kernelports.ImagingNormalizeRequest{
		Source:    source,
		Format:    kernelports.ImagingJPEG,
		Quality:   size.quality,
		MaxWidth:  size.edge,
		MaxHeight: size.edge,
		Priority:  kernelports.ImagingInteractive,
	}, maxOutputBytes)
	if err != nil {
		return nil, unreadable(err)
	}
	return rendered.Data, nil
}

func unreadable(err error) error {
	switch {
	case errors.Is(err, kernelports.ErrImagingPixelBudget),
		errors.Is(err, kernelports.ErrImagingTooLarge):
		return errortypes.NewValidationError(
			fileField,
			errortypes.ErrInvalid,
			"The photo is too large to process. Use a smaller picture.",
		)
	case errors.Is(err, kernelports.ErrImagingInvalidDocument),
		errors.Is(err, kernelports.ErrImagingEncrypted),
		errors.Is(err, kernelports.ErrImagingUnsupported):
		return errortypes.NewValidationError(
			fileField,
			errortypes.ErrInvalidFormat,
			"The photo could not be read as an image",
		)
	case errors.Is(err, kernelports.ErrImagingUnavailable),
		errors.Is(err, kernelports.ErrImagingTimeout):
		return errortypes.NewBusinessError(
			"Photos cannot be processed right now. Try again in a few minutes.",
		).WithInternal(err)
	default:
		return fmt.Errorf("normalize the photo: %w", err)
	}
}

func (s *Service) upload(
	ctx context.Context,
	req *kernelports.StorePhotoRequest,
	key string,
	content []byte,
) error {
	if _, err := s.storage.Upload(ctx, &kernelports.UploadParams{
		Key:         key,
		ContentType: photoContentType,
		Size:        int64(len(content)),
		Body:        bytes.NewReader(content),
		Metadata: map[string]string{
			"resource_type": req.ResourceType,
			"resource_id":   req.Holder.ID.String(),
		},
	}); err != nil {
		return fmt.Errorf("upload the photo: %w", err)
	}
	return nil
}
