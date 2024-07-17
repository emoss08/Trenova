// COPYRIGHT(c) 2024 Trenova
//
// This file is part of Trenova.
//
// The Trenova software is licensed under the Business Source License 1.1. You are granted the right
// to copy, modify, and redistribute the software, but only for non-production use or with a total
// of less than three server instances. Starting from the Change Date (November 16, 2026), the
// software will be made available under version 2 or later of the GNU General Public License.
// If you use the software in violation of this license, your rights under the license will be
// terminated automatically. The software is provided "as is," and the Licensor disclaims all
// warranties and conditions. If you use this license's text or the "Business Source License" name
// and trademark, you must comply with the Licensor's covenants, which include specifying the
// Change License as the GPL Version 2.0 or a compatible license, specifying an Additional Use
// Grant, and not modifying the license in any other way.

package file

import (
	"io"
	"mime/multipart"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

// FileService handles file operations.
type FileService struct {
	logger      *zerolog.Logger
	fileHandler FileHandler
}

// NewFileService creates a new FileService.
func NewFileService(logger *zerolog.Logger, fileHandler FileHandler) *FileService {
	return &FileService{
		logger:      logger,
		fileHandler: fileHandler,
	}
}

// ReadFileData reads data from a multipart file header.
func (fs *FileService) ReadFileData(file *multipart.FileHeader) ([]byte, error) {
	fileData, err := file.Open()
	if err != nil {
		fs.logger.Error().Err(err).Msg("FileService: Error opening file")
		return nil, err
	}
	defer fileData.Close()

	return io.ReadAll(fileData)
}

// RenameFile renames a file based on the object ID.
func (fs *FileService) RenameFile(file *multipart.FileHeader, objectID uuid.UUID) (string, error) {
	randomFileName, err := uuid.NewRandom()
	if err != nil {
		fs.logger.Error().Err(err).Msg("FileService: Error generating random file name")
		return "", err
	}

	fileExt := filepath.Ext(file.Filename)
	if fileExt == "" {
		fs.logger.Error().Msg("FileService: Error getting file extension")
		return "", err
	}

	newFileName := objectID.String() + "_" + randomFileName.String() + fileExt

	return newFileName, nil
}

func (fs *FileService) GetFilePathFromURL(url string) string {
	parts := strings.Split(url, "/")
	return parts[len(parts)-1]
}
