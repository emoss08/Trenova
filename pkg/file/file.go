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
