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

var displayFilenameReplacer = strings.NewReplacer(
	"/", "-", "\\", "-", ":", "-", "*", "-", "?", "-",
	"\"", "'", "<", "-", ">", "-", "|", "-", "\n", " ", "\r", " ",
)

func HumanizeBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func SanitizeDisplayFilename(name, fallback string, maxLen int) string {
	cleaned := strings.TrimSpace(displayFilenameReplacer.Replace(name))
	if cleaned == "" {
		return fallback
	}
	if maxLen > 0 && len(cleaned) > maxLen {
		cleaned = cleaned[:maxLen]
	}
	return cleaned
}
