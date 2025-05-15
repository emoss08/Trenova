package docpreview

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/infrastructure/storage/minio"
	"github.com/emoss08/trenova/internal/pkg/config"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger          *logger.Logger
	Client          *minio.Client
	ConfigM         *config.Manager
	FileService     services.FileService
	ImageGenService services.ImageGenerationService
}

type service struct {
	l           *zerolog.Logger
	client      *minio.Client
	fileService services.FileService
	imageGen    services.ImageGenerationService
}

func NewService(p ServiceParams) services.PreviewService {
	l := p.Logger.With().
		Str("service", "docpreview").
		Logger()

	return &service{
		l:           &l,
		client:      p.Client,
		fileService: p.FileService,
		imageGen:    p.ImageGenService,
	}
}

// GeneratePreview generates a preview image for a document
func (s *service) GeneratePreview(ctx context.Context, req *services.GeneratePreviewRequest) (*services.GeneratePreviewResponse, error) {
	log := s.l.With().
		Str("operation", "GeneratePreview").
		Str("fileName", req.FileName).
		Logger()

	// Create temporary files for document and image
	tmpFilePath, tmpImagePath, err := s.createTempFiles(req.FileName, log)
	if err != nil {
		return nil, err
	}
	defer os.Remove(tmpFilePath)
	defer os.Remove(tmpImagePath)

	// Write document data to temp file
	if err = s.writeToTempFile(tmpFilePath, req.File, log); err != nil {
		return nil, err
	}

	// Convert document to image using the image generation service
	quality := 90
	convReq := &services.ConvertToImageRequest{
		FilePath:   tmpFilePath,
		OutputPath: tmpImagePath,
		Options: &services.ConversionOptions{
			Quality: &quality,
		},
	}

	_, err = s.imageGen.ConvertToImage(ctx, convReq)
	if err != nil {
		log.Error().Err(err).Msg("failed to convert document to image")
		return nil, err
	}

	// Read the image data
	imgData, err := os.ReadFile(tmpImagePath)
	if err != nil {
		log.Error().Err(err).Msg("failed to read preview image")
		return nil, err
	}

	// Save the preview image to storage
	previewFileName, err := s.savePreviewImage(ctx, req, imgData, log)
	if err != nil {
		return nil, err
	}

	// Generate presigned URL for the preview
	previewURL, err := s.fileService.GetFileURL(ctx, req.BucketName, previewFileName, time.Hour*24)
	if err != nil {
		log.Warn().Err(err).Msg("failed to generate presigned URL for preview")
	}

	return &services.GeneratePreviewResponse{
		PreviewPath: previewFileName,
		PreviewURL:  previewURL,
	}, nil
}

// createTempFiles creates temporary files for document and image processing
func (s *service) createTempFiles(fileName string, log zerolog.Logger) (string, string, error) {
	// Create a temporary file for the document
	tmpFile, err := os.CreateTemp("", "doc-*"+filepath.Ext(fileName))
	if err != nil {
		log.Error().Err(err).Msg("failed to create temporary document file")
		return "", "", err
	}
	tmpFilePath := tmpFile.Name()
	tmpFile.Close()

	// Create a temporary file for the preview image
	tmpImageFile, err := os.CreateTemp("", fmt.Sprintf("preview-*.%s", services.GetFileTypeFromExtension(filepath.Ext(fileName))))
	if err != nil {
		os.Remove(tmpFilePath)
		log.Error().Err(err).Msg("failed to create temporary preview image file")
		return "", "", err
	}
	tmpImagePath := tmpImageFile.Name()
	tmpImageFile.Close()

	return tmpFilePath, tmpImagePath, nil
}

// writeToTempFile writes document data to a temporary file
func (s *service) writeToTempFile(tmpFilePath string, data []byte, log zerolog.Logger) error {
	if err := os.WriteFile(tmpFilePath, data, 0o600); err != nil {
		log.Error().Err(err).Msg("failed to write document data to temp file")
		return err
	}
	return nil
}

// savePreviewImage saves the preview image to storage and returns the path
func (s *service) savePreviewImage(ctx context.Context, req *services.GeneratePreviewRequest, imgData []byte, log zerolog.Logger) (string, error) {
	// Generate a consistent preview path
	timestamp := time.Now().Format("20060102150405")
	safeResourceType := strings.ToLower(string(req.ResourceType))
	previewFileName := fmt.Sprintf("previews/%s/%s/%s_preview.%s",
		safeResourceType,
		req.ResourceID.String(),
		timestamp,
		services.GetFileTypeFromExtension(filepath.Ext(req.FileName)))

	// Create file request
	fileReq := &services.SaveFileRequest{
		OrgID:          req.OrgID.String(),
		BucketName:     req.BucketName,
		UserID:         req.UserID,
		FileName:       previewFileName,
		File:           imgData,
		FileExtension:  services.GetFileTypeFromExtension(filepath.Ext(req.FileName)),
		Classification: services.ClassificationPublic,
		Category:       services.CategoryOther,
		Tags: map[string]string{
			"resource_id":   req.ResourceID.String(),
			"resource_type": string(req.ResourceType),
			"preview_for":   req.FileName,
		},
	}

	// Upload the preview image
	_, err := s.fileService.SaveFile(ctx, fileReq)
	if err != nil {
		log.Error().Err(err).Msg("failed to save preview image")
		return "", oops.In("save_preview_image").
			Tags("service", "docpreview").
			With("file_name", req.FileName).
			With("bucket_name", req.BucketName).
			With("file_type", services.GetFileTypeFromExtension(filepath.Ext(req.FileName))).
			Time(time.Now()).
			Wrapf(err, "save preview image")
	}

	return previewFileName, nil
}

// GetPreviewURL retrieves a presigned URL for a preview image
func (s *service) GetPreviewURL(ctx context.Context, req *services.GetPreviewURLRequest) (string, error) {
	if req.PreviewPath == "" {
		return "", nil
	}

	return s.fileService.GetFileURL(ctx, req.BucketName, req.PreviewPath, req.ExpiryTime)
}

// DeletePreview deletes a preview image
func (s *service) DeletePreview(ctx context.Context, req *services.DeletePreviewRequest) error {
	if req.PreviewPath == "" {
		return nil
	}

	return s.fileService.DeleteFile(ctx, req.BucketName, req.PreviewPath)
}
