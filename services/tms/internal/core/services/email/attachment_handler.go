/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package email

import (
	"bytes"
	"context"
	"encoding/base64"
	"io"
	"path/filepath"
	"strconv"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/intutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rs/zerolog"
	"github.com/samber/oops"
	"go.uber.org/fx"
)

// AttachmentHandler handles email attachment operations
type AttachmentHandler interface {
	SaveAttachments(
		ctx context.Context,
		attachments []email.AttachmentMeta,
		orgID, userID pulid.ID,
	) ([]email.AttachmentMeta, error)
	GetAttachmentData(
		ctx context.Context,
		attachment *email.AttachmentMeta,
		orgID pulid.ID,
	) ([]byte, error)
	DeleteAttachments(ctx context.Context, attachments []email.AttachmentMeta, orgID pulid.ID) error
	ValidateAttachments(attachments []email.AttachmentMeta) error
}

type attachmentHandler struct {
	l           *zerolog.Logger
	fileService services.FileService
	orgRepo     repositories.OrganizationRepository
}

type AttachmentHandlerParams struct {
	fx.In

	Logger      *logger.Logger
	FileService services.FileService
	OrgRepo     repositories.OrganizationRepository
}

// NewAttachmentHandler creates a new attachment handler
func NewAttachmentHandler(p AttachmentHandlerParams) AttachmentHandler {
	log := p.Logger.With().
		Str("component", "attachment_handler").
		Logger()

	return &attachmentHandler{
		l:           &log,
		fileService: p.FileService,
		orgRepo:     p.OrgRepo,
	}
}

// SaveAttachments saves email attachments to MinIO storage
func (h *attachmentHandler) SaveAttachments(
	ctx context.Context,
	attachments []email.AttachmentMeta,
	orgID, userID pulid.ID,
) ([]email.AttachmentMeta, error) {
	if len(attachments) == 0 {
		return attachments, nil
	}

	log := h.l.With().
		Str("operation", "save_attachments").
		Str("org_id", orgID.String()).
		Int("attachment_count", len(attachments)).
		Logger()

	bucketName, err := h.getBucketName(ctx, orgID)
	if err != nil {
		return nil, err
	}

	savedAttachments, err := h.processAttachments(ctx, attachments, bucketName, orgID, userID, &log)
	if err != nil {
		return nil, err
	}

	log.Info().Int("saved_count", len(savedAttachments)).Msg("all attachments saved successfully")
	return savedAttachments, nil
}

// GetAttachmentData retrieves attachment data from MinIO storage
func (h *attachmentHandler) GetAttachmentData(
	ctx context.Context,
	attachment *email.AttachmentMeta,
	orgID pulid.ID,
) ([]byte, error) {
	log := h.l.With().
		Str("operation", "get_attachment_data").
		Str("filename", attachment.FileName).
		Str("org_id", orgID.String()).
		Logger()

	// Get organization bucket name
	bucketName, err := h.getBucketName(ctx, orgID)
	if err != nil {
		return nil, err
	}

	// Extract object name from URL
	objectName := h.extractObjectNameFromURL(attachment.URL)

	// Get and read file from MinIO
	data, err := h.readFileFromMinIO(ctx, bucketName, objectName, attachment.FileName, &log)
	if err != nil {
		return nil, err
	}

	log.Debug().
		Int("data_size", len(data)).
		Msg("attachment data retrieved successfully")

	return data, nil
}

// DeleteAttachments deletes email attachments from MinIO storage
func (h *attachmentHandler) DeleteAttachments(
	ctx context.Context,
	attachments []email.AttachmentMeta,
	orgID pulid.ID,
) error {
	log := h.l.With().
		Str("operation", "delete_attachments").
		Str("org_id", orgID.String()).
		Int("attachment_count", len(attachments)).
		Logger()

	if len(attachments) == 0 {
		return nil
	}

	// Get organization bucket name
	bucketName, err := h.getBucketName(ctx, orgID)
	if err != nil {
		return err
	}

	// Delete all attachments and collect errors
	deletionErrors := h.deleteAttachmentsFromBucket(ctx, attachments, bucketName, &log)

	// Handle deletion errors
	if len(deletionErrors) > 0 {
		log.Warn().
			Int("failed_deletions", len(deletionErrors)).
			Int("total_attachments", len(attachments)).
			Msg("some attachments failed to delete")

		return oops.In("attachment_handler").
			Tags("operation", "delete_files").
			Tags("failed_count", strconv.Itoa(len(deletionErrors))).
			Time(time.Now()).
			Errorf("failed to delete %d out of %d attachments", len(deletionErrors), len(attachments))
	}

	log.Info().Msg("all attachments deleted successfully")
	return nil
}

// ValidateAttachments validates attachment metadata
func (h *attachmentHandler) ValidateAttachments(attachments []email.AttachmentMeta) error {
	const maxAttachments = 10
	const maxTotalSize = 25 * 1024 * 1024 // 25MB total

	if len(attachments) > maxAttachments {
		return oops.In("attachment_handler").
			Tags("operation", "validate_attachments").
			Tags("attachment_count", strconv.Itoa(len(attachments))).
			Tags("max_allowed", strconv.Itoa(maxAttachments)).
			Time(time.Now()).
			Errorf("too many attachments: %d (max %d)", len(attachments), maxAttachments)
	}

	var totalSize int64
	for i, attachment := range attachments {
		if attachment.FileName == "" {
			return oops.In("attachment_handler").
				Tags("operation", "validate_attachments").
				Tags("attachment_index", strconv.Itoa(i)).
				Time(time.Now()).
				Errorf("attachment %d has empty filename", i)
		}

		if attachment.Size <= 0 {
			return oops.In("attachment_handler").
				Tags("operation", "validate_attachments").
				Tags("filename", attachment.FileName).
				Time(time.Now()).
				Errorf("attachment '%s' has invalid size: %d", attachment.FileName, attachment.Size)
		}

		totalSize += attachment.Size
		if totalSize > maxTotalSize {
			return oops.In("attachment_handler").
				Tags("operation", "validate_attachments").
				Tags("total_size", strconv.FormatInt(totalSize, 10)).
				Tags("max_size", strconv.FormatInt(maxTotalSize, 10)).
				Time(time.Now()).
				Errorf("total attachment size %d exceeds limit %d", totalSize, maxTotalSize)
		}

		// Validate content type
		ext := filepath.Ext(attachment.FileName)
		if !services.IsSupportedFileType(services.GetFileTypeFromExtension(ext)) {
			return oops.In("attachment_handler").
				Tags("operation", "validate_attachments").
				Tags("filename", attachment.FileName).
				Tags("extension", ext).
				Time(time.Now()).
				Errorf("unsupported file type: %s", ext)
		}
	}

	return nil
}

// getBucketName gets the organization bucket name
func (h *attachmentHandler) getBucketName(ctx context.Context, orgID pulid.ID) (string, error) {
	bucketName, err := h.orgRepo.GetOrganizationBucketName(ctx, orgID)
	if err != nil {
		h.l.Error().Err(err).Msg("failed to get organization bucket name")
		return "", oops.In("attachment_handler").
			Tags("operation", "get_bucket_name").
			Tags("org_id", orgID.String()).
			Time(time.Now()).
			Wrapf(err, "failed to get organization bucket name")
	}

	return bucketName, nil
}

// processAttachments processes all attachments for saving
func (h *attachmentHandler) processAttachments(
	ctx context.Context,
	attachments []email.AttachmentMeta,
	bucketName string,
	orgID, userID pulid.ID,
	log *zerolog.Logger,
) ([]email.AttachmentMeta, error) {
	savedAttachments := make([]email.AttachmentMeta, 0, len(attachments))

	for i, attachment := range attachments {
		savedAttachment, err := h.saveIndividualAttachment(
			ctx, attachment, i, bucketName, orgID, userID, log,
		)
		if err != nil {
			return nil, err
		}
		savedAttachments = append(savedAttachments, savedAttachment)
	}

	return savedAttachments, nil
}

// saveIndividualAttachment saves a single attachment to MinIO
func (h *attachmentHandler) saveIndividualAttachment(
	ctx context.Context,
	attachment email.AttachmentMeta,
	index int,
	bucketName string,
	orgID, userID pulid.ID,
	parentLog *zerolog.Logger,
) (email.AttachmentMeta, error) {
	attachmentLog := parentLog.With().
		Int("attachment_index", index).
		Str("filename", attachment.FileName).
		Logger()

	// Extract file data
	fileData, err := h.extractFileData(attachment.URL)
	if err != nil {
		attachmentLog.Error().Err(err).Msg("failed to extract file data")
		return email.AttachmentMeta{}, oops.In("attachment_handler").
			Tags("operation", "extract_file_data").
			Tags("filename", attachment.FileName).
			Time(time.Now()).
			Wrapf(err, "failed to extract file data")
	}

	// Prepare save request
	saveReq, err := h.prepareSaveRequest(attachment, fileData, bucketName, orgID, userID)
	if err != nil {
		return email.AttachmentMeta{}, err
	}

	// Validate and save file
	saveResp, err := h.validateAndSaveFile(
		ctx,
		saveReq,
		fileData,
		attachment.FileName,
		&attachmentLog,
	)
	if err != nil {
		return email.AttachmentMeta{}, err
	}

	// Create saved attachment metadata
	savedAttachment := email.AttachmentMeta{
		FileName:    attachment.FileName,
		ContentType: attachment.ContentType,
		Size:        saveResp.Size,
		URL:         saveResp.Location,
		ContentID:   attachment.ContentID,
	}

	attachmentLog.Info().
		Str("saved_url", saveResp.Location).
		Int64("size", saveResp.Size).
		Msg("attachment saved successfully")

	return savedAttachment, nil
}

// prepareSaveRequest prepares the save file request
func (h *attachmentHandler) prepareSaveRequest(
	attachment email.AttachmentMeta,
	fileData []byte,
	bucketName string,
	orgID, userID pulid.ID,
) (*services.SaveFileRequest, error) {
	ext := filepath.Ext(attachment.FileName)
	baseName := attachment.FileName[:len(attachment.FileName)-len(ext)]
	uniqueFileName := h.generateUniqueFileName(baseName, ext)

	return &services.SaveFileRequest{
		File:           fileData,
		FileName:       uniqueFileName,
		OrgID:          orgID.String(),
		UserID:         userID.String(),
		FileExtension:  services.GetFileTypeFromExtension(ext),
		Classification: services.ClassificationPrivate,
		Category:       services.CategoryOther,
		BucketName:     bucketName,
		Tags: map[string]string{
			"source":        "email_attachment",
			"original_name": attachment.FileName,
			"content_type":  attachment.ContentType,
		},
	}, nil
}

// validateAndSaveFile validates and saves the file to MinIO
func (h *attachmentHandler) validateAndSaveFile(
	ctx context.Context,
	saveReq *services.SaveFileRequest,
	fileData []byte,
	fileName string,
	log *zerolog.Logger,
) (*services.SaveFileResponse, error) {
	// Validate file
	if err := h.fileService.ValidateFile(int64(len(fileData)), saveReq.FileExtension); err != nil {
		log.Error().Err(err).Msg("file validation failed")
		return nil, oops.In("attachment_handler").
			Tags("operation", "validate_file").
			Tags("filename", fileName).
			Tags("file_size", strconv.Itoa(len(fileData))).
			Time(time.Now()).
			Wrapf(err, "file validation failed")
	}

	// Save file to MinIO
	saveResp, err := h.fileService.SaveFile(ctx, saveReq)
	if err != nil {
		log.Error().Err(err).Msg("failed to save attachment")
		return nil, oops.In("attachment_handler").
			Tags("operation", "save_file").
			Tags("filename", fileName).
			Time(time.Now()).
			Wrapf(err, "failed to save attachment")
	}

	return saveResp, nil
}

// extractFileData extracts file data from various sources (data URL, file path, etc.)
func (h *attachmentHandler) extractFileData(url string) ([]byte, error) {
	// Handle data URLs (base64 encoded)
	if len(url) > 11 && url[:5] == "data:" {
		// Find the comma that separates metadata from data
		commaIndex := bytes.IndexByte([]byte(url), ',')
		if commaIndex == -1 {
			return nil, oops.In("attachment_handler").
				Tags("operation", "parse_data_url").
				Time(time.Now()).
				Errorf("invalid data URL format")
		}

		// Decode base64 data
		encodedData := url[commaIndex+1:]
		data, err := base64.StdEncoding.DecodeString(encodedData)
		if err != nil {
			return nil, oops.In("attachment_handler").
				Tags("operation", "decode_base64").
				Time(time.Now()).
				Wrapf(err, "failed to decode base64 data")
		}

		return data, nil
	}

	// For now, we only support data URLs for attachments
	// In the future, we could support file paths or URLs
	return nil, oops.In("attachment_handler").
		Tags("operation", "extract_file_data").
		Tags("url_type", "unsupported").
		Time(time.Now()).
		Errorf("unsupported attachment URL format: %s", url[:intutils.Min(50, len(url))])
}

// generateUniqueFileName generates a unique filename with timestamp
func (h *attachmentHandler) generateUniqueFileName(baseName, ext string) string {
	timestamp := time.Now().Format("20060102_150405")
	return "email_attachments/" + baseName + "_" + timestamp + ext
}

// readFileFromMinIO reads file data from MinIO
func (h *attachmentHandler) readFileFromMinIO(
	ctx context.Context,
	bucketName, objectName, fileName string,
	log *zerolog.Logger,
) ([]byte, error) {
	// Get file from MinIO
	obj, err := h.fileService.GetFileByBucketName(ctx, bucketName, objectName)
	if err != nil {
		log.Error().Err(err).Msg("failed to get file from MinIO")
		return nil, oops.In("attachment_handler").
			Tags("operation", "get_file").
			Tags("bucket", bucketName).
			Tags("object", objectName).
			Time(time.Now()).
			Wrapf(err, "failed to get file from MinIO")
	}
	defer obj.Close()

	// Read file data
	data, err := io.ReadAll(obj)
	if err != nil {
		log.Error().Err(err).Msg("failed to read file data")
		return nil, oops.In("attachment_handler").
			Tags("operation", "read_file").
			Tags("filename", fileName).
			Time(time.Now()).
			Wrapf(err, "failed to read file data")
	}

	return data, nil
}

// deleteAttachmentsFromBucket deletes attachments from the specified bucket
func (h *attachmentHandler) deleteAttachmentsFromBucket(
	ctx context.Context,
	attachments []email.AttachmentMeta,
	bucketName string,
	log *zerolog.Logger,
) []error {
	var deletionErrors []error

	for _, attachment := range attachments {
		objectName := h.extractObjectNameFromURL(attachment.URL)

		if err := h.fileService.DeleteFile(ctx, bucketName, objectName); err != nil {
			log.Error().
				Err(err).
				Str("filename", attachment.FileName).
				Str("object", objectName).
				Msg("failed to delete attachment")

			deletionErrors = append(deletionErrors, err)
		} else {
			log.Debug().
				Str("filename", attachment.FileName).
				Msg("attachment deleted successfully")
		}
	}

	return deletionErrors
}

// extractObjectNameFromURL extracts the object name from a MinIO URL
func (h *attachmentHandler) extractObjectNameFromURL(url string) string {
	// URL format: http://endpoint/bucket/object_name
	// Find the last two '/' separators to get the object name
	parts := bytes.Split([]byte(url), []byte("/"))
	if len(parts) >= 2 {
		return string(parts[len(parts)-1])
	}

	return url // fallback to the full URL as object name
}
