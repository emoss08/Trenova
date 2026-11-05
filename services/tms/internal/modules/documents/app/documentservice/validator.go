package documentservice

import (
	"mime/multipart"
	"path/filepath"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/fileutils"
	"go.uber.org/fx"
)

var dangerousExtensions = []string{
	".exe", ".bat", ".cmd", ".sh", ".ps1", ".vbs", ".js", ".jar",
	".msi", ".dll", ".scr", ".com", ".pif", ".application", ".gadget",
	".msp", ".hta", ".cpl", ".msc", ".ws", ".wsf", ".wsc", ".wsh",
	".lnk", ".inf", ".reg", ".vb", ".vbe", ".jse", ".sct", ".pyc",
}

var contentTypesByExtension = map[string]string{
	".pdf":  "application/pdf",
	".jpg":  "image/jpeg",
	".jpeg": "image/jpeg",
	".png":  "image/png",
	".webp": "image/webp",
	".gif":  "image/gif",
	".tif":  "image/tiff",
	".tiff": "image/tiff",
	".heic": "image/heic",
	".heif": "image/heif",
	".doc":  "application/msword",
	".docx": "application/vnd.openxmlformats-officedocument.wordprocessingml.document",
	".xls":  "application/vnd.ms-excel",
	".xlsx": "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
	".txt":  "text/plain",
	".csv":  "text/csv",
}

type ValidatorParams struct {
	fx.In

	StorageConfig *storageconfig.Config
}

type Validator struct {
	maxFileSize      int64
	allowedMIMETypes []string
}

func NewValidator(p ValidatorParams) *Validator {
	cfg := p.StorageConfig
	return &Validator{
		maxFileSize:      cfg.GetMaxFileSize(),
		allowedMIMETypes: cfg.GetAllowedMIMETypes(),
	}
}

func (v *Validator) ValidateFile(file *multipart.FileHeader) *errortypes.MultiError {
	return v.validateFileMetadata(file.Filename, file.Size, file.Header.Get("Content-Type"))
}

func (v *Validator) ValidateUploadMetadata(
	filename string,
	size int64,
	contentType string,
) *errortypes.MultiError {
	return v.validateFileMetadata(filename, size, contentType)
}

func (v *Validator) validateFileMetadata(
	filename string,
	size int64,
	contentType string,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if size > v.maxFileSize {
		multiErr.Add(
			"file",
			errortypes.ErrInvalidLength,
			"File size exceeds maximum allowed size",
		)
	}

	if size == 0 {
		multiErr.Add(
			"file",
			errortypes.ErrRequired,
			"File cannot be empty",
		)
	}

	if slices.Contains(dangerousExtensions, fileExtension(filename)) {
		multiErr.Add(
			"file",
			errortypes.ErrInvalidFormat,
			"File type is not allowed for security reasons",
		)
	}

	if !v.allowsContentType(resolvedContentType(filename, contentType)) {
		multiErr.Add(
			"file",
			errortypes.ErrInvalidFormat,
			"File type is not allowed",
		)
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func fileExtension(filename string) string {
	return strings.ToLower(filepath.Ext(strings.TrimRight(filename, ". ")))
}

func resolvedContentType(filename, declared string) string {
	if strings.TrimSpace(declared) != "" {
		return declared
	}

	return contentTypesByExtension[fileExtension(filename)]
}

func (v *Validator) allowsContentType(contentType string) bool {
	mediaType := fileutils.MediaType(contentType)

	return mediaType != "" && slices.Contains(v.allowedMIMETypes, mediaType)
}

func (v *Validator) ValidateFiles(files []*multipart.FileHeader) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	for i, file := range files {
		if fileErr := v.ValidateFile(file); fileErr != nil {
			for _, err := range fileErr.Errors {
				multiErr.Add(
					err.Field+"["+string(rune('0'+i))+"]",
					err.Code,
					err.Message,
				)
			}
		}
	}

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}
