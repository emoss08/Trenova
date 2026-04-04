package fileutils

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
)

func GenerateStoragePath(orgID, resourceType, filename string) string {
	ext := filepath.Ext(filename)
	uniqueID := uuid.New().String()
	return fmt.Sprintf("%s/%s/%s%s", orgID, resourceType, uniqueID, strings.ToLower(ext))
}
