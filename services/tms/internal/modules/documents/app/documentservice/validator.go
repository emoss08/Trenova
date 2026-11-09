package documentservice

import (
	"mime/multipart"
	"path/filepath"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config/storageconfig"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/fieldpath"
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

const fileField = "file"

func NewValidator(p ValidatorParams) *Validator {
	cfg := p.StorageConfig
	return &Validator{
		maxFileSize:      cfg.GetMaxFileSize(),
		allowedMIMETypes: cfg.GetAllowedMIMETypes(),
	}
}

func (v *Validator) ValidateFile(file *multipart.FileHeader) error {
	return v.ValidateUploadMetadata(file.Filename, file.Size, file.Header.Get("Content-Type"))
}

func (v *Validator) ValidateUploadMetadata(
	filename string,
	size int64,
	contentType string,
) error {
	multiErr := errortypes.NewMultiError()
	v.checkFile(multiErr, fieldpath.Of(fieldpath.Key(fileField)), filename, size, contentType)

	return multiErr.Err()
}

func (v *Validator) checkFile(
	multiErr *errortypes.MultiError,
	path fieldpath.Path,
	filename string,
	size int64,
	contentType string,
) {
	if size > v.maxFileSize {
		multiErr.AddAt(path, errortypes.ErrInvalidLength, "File size exceeds maximum allowed size")
	}

	if size == 0 {
		multiErr.AddAt(path, errortypes.ErrRequired, "File cannot be empty")
	}

	if slices.Contains(dangerousExtensions, fileExtension(filename)) {
		multiErr.AddAt(
			path,
			errortypes.ErrInvalidFormat,
			"File type is not allowed for security reasons",
		)
	}

	if !v.allowsContentType(resolvedContentType(filename, contentType)) {
		multiErr.AddAt(path, errortypes.ErrInvalidFormat, "File type is not allowed")
	}
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

func (v *Validator) ValidateFiles(files []*multipart.FileHeader) error {
	multiErr := errortypes.NewMultiError()

	for i, file := range files {
		v.checkFile(
			multiErr,
			fieldpath.Of(fieldpath.Key(fileField), fieldpath.Index(i)),
			file.Filename,
			file.Size,
			file.Header.Get("Content-Type"),
		)
	}

	return multiErr.Err()
}
