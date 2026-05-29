package fileutils

import (
	"fmt"
	"mime"
	"path"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func GenerateStoragePath(orgID, resourceType, filename string) string {
	ext := filepath.Ext(filename)
	uniqueID := uuid.New().String()
	return fmt.Sprintf("%s/%s/%s%s", orgID, resourceType, uniqueID, strings.ToLower(ext))
}

func ContentDisposition(disposition, filename string) string {
	normalizedDisposition := strings.ToLower(strings.TrimSpace(disposition))
	if normalizedDisposition == "" {
		normalizedDisposition = "attachment"
	}

	safeName := SafeFilename(filename)
	if safeName == "" || safeName == "." || safeName == "/" {
		safeName = "download"
	}

	return mime.FormatMediaType(normalizedDisposition, map[string]string{
		"filename": safeName,
	})
}

func SafeFilename(filename string) string {
	cleaned := strings.NewReplacer("\r", "", "\n", "", "\\", "/").Replace(filename)
	return path.Base(strings.TrimSpace(cleaned))
}
