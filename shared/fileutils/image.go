package fileutils

import (
	"path/filepath"
	"slices"
	"strings"
)

var AllowedImageMIMETypes = []string{"image/jpeg", "image/png", "image/webp"}

var AllowedImageExtensions = []string{".jpg", ".jpeg", ".png", ".webp"}

func IsSupportedImageContentType(contentType string) bool {
	normalized := strings.ToLower(strings.TrimSpace(contentType))
	if normalized == "" || normalized == "application/octet-stream" {
		return true
	}

	return slices.Contains(AllowedImageMIMETypes, normalized)
}

func HasSupportedImageExtension(filename string) bool {
	return slices.Contains(
		AllowedImageExtensions,
		strings.ToLower(filepath.Ext(strings.TrimSpace(filename))),
	)
}

func IsExternalURL(value string) bool {
	normalized := strings.ToLower(strings.TrimSpace(value))
	return strings.HasPrefix(normalized, "http://") || strings.HasPrefix(normalized, "https://")
}
