package file

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"github.com/samber/lo"
	"github.com/trenova-app/transport/internal/core/ports/services"
	"github.com/trenova-app/transport/internal/infrastructure/storage/minio"
	"github.com/trenova-app/transport/internal/pkg/config"
	"github.com/trenova-app/transport/internal/pkg/logger"
	"github.com/trenova-app/transport/internal/pkg/utils/sortutils"
	"go.uber.org/fx"
)

const (
	defaultRegion     = "us-east-1"
	defaultExpiration = 24 * time.Hour
	maxFileSize       = 100 * 1024 * 1024 // 100MB
)

type Config struct {
	MaxFileSize            int64
	AllowedFileTypes       map[services.FileType][]string
	DefaultRegion          string
	ClassificationPolicies map[services.FileClassification]services.ClassificationPolicy
}

type ServiceParams struct {
	fx.In

	Client  *minio.Client
	Logger  *logger.Logger
	ConfigM *config.Manager
}

type service struct {
	client   *minio.Client
	l        *zerolog.Logger
	cfg      *Config
	endpoint string
}

func NewService(p ServiceParams) services.FileService {
	log := p.Logger.With().
		Str("service", "file").
		Logger()

	cfg := &Config{
		MaxFileSize: maxFileSize,
		AllowedFileTypes: map[services.FileType][]string{
			services.ImageFile: {".jpg", ".jpeg", ".png", ".gif", ".webp"},
			services.DocFile:   {".doc", ".docx", ".xls", ".xlsx", ".csv", ".ppt", ".pptx"},
			services.PDFFile:   {".pdf"},
		},
		DefaultRegion: defaultRegion,
		ClassificationPolicies: map[services.FileClassification]services.ClassificationPolicy{
			services.ClassificationPublic: {
				RetentionPeriod:    30 * 24 * time.Hour, // 30 days
				RequiresEncryption: false,
				AllowedCategories:  []services.FileCategory{services.CategoryBranding, services.CategoryProfile},
				MaxFileSize:        5 * 1024 * 1024, // 5MB
				RequireVersioning:  false,
				MaxVersions:        1,
				VersionRetention:   "none",
			},
			services.ClassificationPrivate: {
				RetentionPeriod:    90 * 24 * time.Hour, // 90 days
				RequiresEncryption: true,
				AllowedCategories:  []services.FileCategory{services.CategoryShipment},
				MaxFileSize:        maxFileSize,
				RequireVersioning:  true,
				MaxVersions:        5,
				VersionRetention:   "latest-5",
			},
			services.ClassificationSensitive: {
				RetentionPeriod:    365 * 24 * time.Hour, // 1 year
				RequiresEncryption: true,
				AllowedCategories:  []services.FileCategory{services.CategoryWorker},
				MaxFileSize:        50 * 1024 * 1024, // 50MB
				RequireVersioning:  true,
				MaxVersions:        10,
				VersionRetention:   "all",
			},
			services.ClassificationRegulatory: {
				RetentionPeriod:    3 * 365 * 24 * time.Hour, // 3 years
				RequiresEncryption: true,
				AllowedCategories:  []services.FileCategory{services.CategoryRegulatory},
				MaxFileSize:        maxFileSize,
				RequireVersioning:  true,
				MaxVersions:        0, // Keep all versions
				VersionRetention:   "all",
			},
		},
	}

	return &service{
		client:   p.Client,
		endpoint: p.ConfigM.Minio().Endpoint,
		l:        &log,
		cfg:      cfg,
	}
}

func (s *service) SaveFileVersion(ctx context.Context, req *services.SaveFileRequest) (*services.SaveFileResponse, error) {
	policy := s.cfg.ClassificationPolicies[req.Classification]
	// Check if versioning is required
	if policy.RequireVersioning {
		// Ensure versioning is enabled on the bucket
		if err := s.ensureVersioningEnabled(ctx, req.OrgID); err != nil {
			return nil, eris.Wrap(err, "versioning enabled")
		}
	}

	// continue with the save file logic
	resp, err := s.SaveFile(ctx, req)
	if err != nil {
		return nil, eris.Wrap(err, "save file")
	}

	// if versioning is enabled, manage versions
	if policy.RequireVersioning && policy.MaxVersions > 0 {
		if err = s.manageVersions(ctx, req.OrgID, resp.Key, policy.MaxVersions); err != nil {
			s.l.Warn().Err(err).Msg("failed to manage versions")
		}
	}

	return resp, nil
}

func (s *service) GetFileVersion(ctx context.Context, bucketName, objectName string) ([]services.VersionInfo, error) {
	opts := minio.ListObjectsOptions{
		Prefix:       objectName,
		Recursive:    true,
		WithVersions: true,
	}

	var versions []services.VersionInfo
	for obj := range s.client.ListObjects(ctx, bucketName, opts) {
		if obj.Err != nil {
			return nil, eris.Wrap(obj.Err, "list object versions")
		}

		objInfo, err := s.client.StatObject(ctx, bucketName, obj.Key, minio.StatObjectOptions{
			VersionID: obj.VersionID,
		})
		if err != nil {
			s.l.Warn().Err(err).
				Str("bucket", bucketName).
				Str("object", obj.Key).
				Str("versionId", obj.VersionID).
				Msg("failed to get version metadata")
			continue
		}

		versions = append(versions, services.VersionInfo{
			VersionID:      obj.VersionID,
			LastModified:   obj.LastModified,
			CreatedBy:      objInfo.Metadata.Get("user_id"),
			Comment:        objInfo.Metadata.Get("version_comment"),
			Size:           obj.Size,
			Checksum:       objInfo.Metadata.Get("checksum"),
			Metadata:       objInfo.Metadata,
			IsLatest:       obj.IsLatest,
			Classification: objInfo.Metadata.Get("classification"),
		})
	}

	return versions, nil
}

func (s *service) checkObjectExists(ctx context.Context, bucketName, objectName string) (bool, error) {
	_, err := s.client.StatObject(ctx, bucketName, objectName, minio.StatObjectOptions{})
	if err != nil {
		return false, eris.Wrap(err, "check object exists")
	}

	return true, nil
}

func (s *service) GetFileByBucketName(ctx context.Context, bucketName, objectName string) (*minio.Object, error) {
	// Check if the object exists
	exists, err := s.checkObjectExists(ctx, bucketName, objectName)
	if err != nil {
		return nil, eris.Wrap(err, "check object exists")
	}

	if !exists {
		return nil, eris.New("object not found")
	}

	obj, err := s.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{})
	if err != nil {
		return nil, eris.Wrap(err, "get object")
	}

	// ! Do not defer obj.Close() here since we need to return the object
	// ! the caller is responsible for closing the object

	return obj, nil
}

func (s *service) GetSpecificVersion(ctx context.Context, bucketName, objectName, versionID string) ([]byte, *services.VersionInfo, error) {
	obj, err := s.client.GetObject(ctx, bucketName, objectName, minio.GetObjectOptions{
		VersionID: versionID,
	})
	if err != nil {
		return nil, nil, eris.Wrap(err, "get object version")
	}
	defer obj.Close()

	data, err := io.ReadAll(obj)
	if err != nil {
		return nil, nil, eris.Wrap(err, "read object data")
	}

	objInfo, err := obj.Stat()
	if err != nil {
		return nil, nil, eris.Wrap(err, "get object info")
	}

	versionInfo := &services.VersionInfo{
		VersionID:      versionID,
		LastModified:   objInfo.LastModified,
		CreatedBy:      objInfo.Metadata.Get("user_id"),
		Comment:        objInfo.Metadata.Get("version_comment"),
		Size:           objInfo.Size,
		Checksum:       objInfo.Metadata.Get("checksum"),
		Metadata:       objInfo.Metadata,
		Classification: objInfo.Metadata.Get("classification"),
	}

	return data, versionInfo, nil
}

func (s *service) RestoreVersion(ctx context.Context, req *services.SaveFileRequest, versionID string) (*services.SaveFileResponse, error) {
	// Get the specified version
	data, versionInfo, err := s.GetSpecificVersion(ctx, req.OrgID, req.FileName, versionID)
	if err != nil {
		return nil, eris.Wrap(err, "get version to restore")
	}

	restoreReq := *req
	restoreReq.File = data
	restoreReq.Metadata = versionInfo.Metadata
	restoreReq.VersionComment = fmt.Sprintf("Restored version from %s", versionID)

	// save as a new version
	return s.SaveFileVersion(ctx, &restoreReq)
}

func (s *service) ensureVersioningEnabled(ctx context.Context, bucketName string) error {
	// First, create bucket if it doesn't exist
	if err := s.createOrgBucket(ctx, bucketName); err != nil {
		return err
	}

	// Enable versioning
	err := s.client.EnableVersioning(ctx, bucketName)
	if err != nil {
		return eris.Wrap(err, "enable versioning")
	}

	return nil
}

func (s *service) manageVersions(ctx context.Context, bucketName, objectName string, maxVersions int) error {
	versions, err := s.GetFileVersion(ctx, bucketName, objectName)
	if err != nil {
		return err
	}

	// If we have more version than allowed, delete the oldest ones
	if len(versions) > maxVersions {
		sortutils.Sort(versions, func(a, b services.VersionInfo) bool {
			return a.LastModified.Before(b.LastModified)
		})

		// Delete oldest version that exceed the limit
		for i := 0; i < len(versions)-maxVersions; i++ {
			err = s.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{
				VersionID: versions[i].VersionID,
			})
			if err != nil {
				s.l.Warn().Err(err).
					Str("bucket", bucketName).
					Str("object", objectName).
					Str("versionId", versions[i].VersionID).
					Msg("failed to delete version")
			}
		}
	}

	return nil
}

func (s *service) validateClassification(req *services.SaveFileRequest) error {
	policy, exists := s.cfg.ClassificationPolicies[req.Classification]
	if !exists {
		s.l.Error().Str("classification", string(req.Classification)).Msg("classification policy not found")
		return eris.Errorf("invalid classification: %s", req.Classification)
	}

	if !lo.Contains(policy.AllowedCategories, req.Category) {
		s.l.Error().Str("category", req.Category.String()).Str("classification", string(req.Classification)).Msg("category not allowed for classification")
		return eris.Errorf("category %s is not allowed for classification %s", req.Category, req.Classification)
	}

	if int64(len(req.File)) > policy.MaxFileSize {
		return eris.Wrapf(services.ErrFileSizeExceedsMaxSize, "max size for %s classification is %d bytes", req.Classification, policy.MaxFileSize)
	}

	return nil
}

func (s *service) ValidateFile(filename string, size int64, fileType services.FileType) error {
	if size > s.cfg.MaxFileSize {
		return services.ErrFileSizeExceedsMaxSize
	}

	ext := strings.ToLower(filepath.Ext(filename))
	allowedExts, ok := s.cfg.AllowedFileTypes[fileType]
	if !ok || !lo.Contains(allowedExts, ext) {
		return services.ErrFileExtensionNotAllowed
	}

	return nil
}

func (s *service) generateObjectPath(req *services.SaveFileRequest) string {
	// Create a structured path based on classification and category
	timestamp := time.Now().Format("2006/01/02")
	return fmt.Sprintf("%s/%s/%s/%s", req.Classification, req.Category, timestamp, req.FileName)
}

func (s *service) createOrgBucket(ctx context.Context, bucketName string) error {
	exists, err := s.client.BucketExists(ctx, bucketName)
	if err != nil {
		s.l.Error().Err(err).Msg("check if bucket exists")
	}

	if !exists {
		err = s.client.MakeBucket(ctx, bucketName, minio.MakeBucketOptions{
			Region: s.cfg.DefaultRegion,
		})
		if err != nil {
			s.l.Error().Err(err).Str("bucket", bucketName).Msg("error creating bucket")
			return eris.Wrap(err, "create bucket")
		}
	}

	// Set the bucket policy programmatically
	policy := s.generateBucketPolicy(bucketName)
	if err = s.client.SetBucketPolicy(ctx, bucketName, policy); err != nil {
		s.l.Error().Err(err).Msg("set bucket policy")
		return eris.Wrap(err, "set bucket policy")
	}

	return nil
}

func (s *service) SaveFile(ctx context.Context, req *services.SaveFileRequest) (*services.SaveFileResponse, error) {
	// Validate classification and category
	if err := s.validateClassification(req); err != nil {
		return nil, err
	}

	// Validate file type and size
	if err := s.ValidateFile(req.FileName, int64(len(req.File)), req.FileType); err != nil {
		return nil, err
	}

	// Ensure bucket exists
	if err := s.createOrgBucket(ctx, req.BucketName); err != nil {
		return nil, eris.Wrap(err, "create org bucket")
	}

	// Generate file hash
	hash := sha256.New()
	if _, err := io.Copy(hash, bytes.NewReader(req.File)); err != nil {
		return nil, eris.Wrap(err, "calculate file hash")
	}
	checksum := hex.EncodeToString(hash.Sum(nil))

	// Prepare metadata
	metadata := http.Header{
		"organization_id": []string{req.OrgID},
		"user_id":         []string{req.UserID},
		"file_type":       []string{string(req.FileType)},
		"classification":  []string{string(req.Classification)},
		"category":        []string{string(req.Category)},
		"content_type":    []string{http.DetectContentType(req.File)},
		"checksum":        []string{checksum},
	}

	// Add custom metadata and tags
	for k, v := range req.Metadata {
		metadata[k] = v
	}
	for k, v := range req.Tags {
		metadata["tag_"+k] = []string{v}
	}

	// Generate object path
	objectName := s.generateObjectPath(req)

	// Save file
	ui, err := s.client.PutObject(
		ctx,
		req.BucketName,
		objectName,
		bytes.NewBuffer(req.File),
		int64(len(req.File)),
		minio.PutObjectOptions{
			ContentType: metadata.Get("content_type"),
			UserMetadata: map[string]string{
				"organization_id": req.OrgID,
				"user_id":         req.UserID,
				"file_type":       string(req.FileType),
				"classification":  string(req.Classification),
				"category":        string(req.Category),
				"content_type":    http.DetectContentType(req.File),
				"checksum":        checksum,
			},
			UserTags:     req.Tags,
			AutoChecksum: minio.ChecksumSHA256,
		},
	)
	if err != nil {
		s.l.Error().Err(err).
			Str("bucket", req.OrgID).
			Str("object", objectName).
			Msg("failed to save file")
		return nil, eris.Wrap(err, "save file")
	}

	fileURL := fmt.Sprintf("http://%s/%s/%s", s.endpoint, req.BucketName, objectName)

	return &services.SaveFileResponse{
		Key:         objectName,
		Location:    fileURL,
		Etag:        ui.ETag,
		Checksum:    checksum,
		BucketName:  req.BucketName,
		Size:        ui.Size,
		Expiration:  ui.Expiration,
		ContentType: metadata.Get("content_type"),
		Metadata:    metadata,
	}, nil
}

func (s *service) GetFileURL(ctx context.Context, bucketName, objectName string, expiry time.Duration) (string, error) {
	url, err := s.client.PresignedGetObject(ctx, bucketName, objectName, expiry, nil)
	if err != nil {
		s.l.Error().Err(err).
			Str("bucket", bucketName).
			Str("object", objectName).
			Msg("failed to generate presigned URL")
		return "", eris.Wrap(err, "generate presigned URL")
	}

	return url.String(), nil
}

func (s *service) DeleteFile(ctx context.Context, bucketName, objectName string) error {
	err := s.client.RemoveObject(ctx, bucketName, objectName, minio.RemoveObjectOptions{})
	if err != nil {
		s.l.Error().Err(err).
			Str("bucket", bucketName).
			Str("object", objectName).
			Msg("failed to delete file")
		return eris.Wrap(err, "delete file")
	}

	return nil
}

func (s *service) ListFiles(ctx context.Context, bucketName, prefix, token string, pageSize int) ([]minio.ObjectInfo, string, error) {
	opts := minio.ListObjectsOptions{
		Prefix:    prefix,
		MaxKeys:   pageSize,
		Recursive: true,
	}

	if token != "" {
		opts.StartAfter = token
	}

	var objects []minio.ObjectInfo

	for object := range s.client.ListObjects(ctx, bucketName, opts) {
		if object.Err != nil {
			return nil, "", eris.Wrap(object.Err, "list objects")
		}
		objects = append(objects, object)
	}

	var nextToken string
	if len(objects) == pageSize {
		nextToken = objects[len(objects)-1].Key
	}

	return objects, nextToken, nil
}

func (s *service) generateBucketPolicy(bucketName string) string {
	return fmt.Sprintf(`{
        "Version": "2012-10-17",
        "Statement": [
            {
                "Effect": "Allow",
                "Principal": {"AWS": ["*"]},
                "Action": ["s3:GetBucketLocation", "s3:ListBucket"],
                "Resource": ["arn:aws:s3:::%s"]
            },
            {
                "Effect": "Allow",
                "Principal": {"AWS": ["*"]},
                "Action": ["s3:GetObject"],
                "Resource": ["arn:aws:s3:::%s/*"]
            }
        ]
    }`, bucketName, bucketName)
}
