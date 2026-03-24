package documentservice

import (
	"mime/multipart"
	"path/filepath"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"go.uber.org/fx"
)

var dangerousExtensions = []string{
	".exe", ".bat", ".cmd", ".sh", ".ps1", ".vbs", ".js", ".jar",
	".msi", ".dll", ".scr", ".com", ".pif", ".application", ".gadget",
	".msp", ".hta", ".cpl", ".msc", ".ws", ".wsf", ".wsc", ".wsh",
	".lnk", ".inf", ".reg", ".vb", ".vbe", ".jse", ".sct", ".pyc",
}

type ValidatorParams struct {
	fx.In

	Config *config.Config
}

type Validator struct {
	maxFileSize      int64
	allowedMIMETypes []string
}

func NewValidator(p ValidatorParams) *Validator {
	cfg := p.Config.GetStorageConfig()
	return &Validator{
		maxFileSize:      cfg.GetMaxFileSize(),
		allowedMIMETypes: cfg.GetAllowedMIMETypes(),
	}
}

func (v *Validator) ValidateFile(file *multipart.FileHeader) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if file.Size > v.maxFileSize {
		multiErr.Add(
			"file",
			errortypes.ErrInvalidLength,
			"File size exceeds maximum allowed size",
		)
	}

	if file.Size == 0 {
		multiErr.Add(
			"file",
			errortypes.ErrRequired,
			"File cannot be empty",
		)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if slices.Contains(dangerousExtensions, ext) {
		multiErr.Add(
			"file",
			errortypes.ErrInvalidFormat,
			"File type is not allowed for security reasons",
		)
	}

	contentType := file.Header.Get("Content-Type")
	if contentType != "" && !slices.Contains(v.allowedMIMETypes, contentType) {
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
