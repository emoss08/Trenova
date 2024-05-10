package util

import (
	"errors"
	"io"
	"mime/multipart"
	"path/filepath"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
)

type FileService struct {
	Logger *zerolog.Logger
}

func NewFileService(logger *zerolog.Logger) *FileService {
	return &FileService{
		Logger: logger,
	}
}

// readFileData opens and reads the data from a provided multipart file header representing an uploaded file.
//
// Parameters:
//   - file: The file header containing metadata about the multipart uploaded file.
//
// Returns:
//   - []byte: Byte slice containing the read file data.
//   - error: Error object if an error occurs during file opening or reading.
//
// Errors:
//   - file opening errors: If the file cannot be opened.
//   - file reading errors: If reading the file data fails.
func (s *FileService) ReadFileData(file *multipart.FileHeader) ([]byte, error) {
	fileData, err := file.Open()
	if err != nil {
		s.Logger.Error().Err(err).Msg("failed to open file")
		return nil, err
	}
	defer fileData.Close()

	return io.ReadAll(fileData)
}

// RenameFile generates a new, unique object name for a file based on a random UUID.
//
// Parameters:
//   - file: The file header of the file to be renamed.
//   - objectID: The UUID of the object to be renamed.
//
// Returns:
//   - string: The new object name incorporating the object ID and a random UUID.
//   - error: Error object if generating a random filename fails.
//
// Errors:
//   - random filename generation errors: If generating the UUID fails.
func (s *FileService) RenameFile(file *multipart.FileHeader, objectID uuid.UUID) (string, error) {
	randomFilename, err := uuid.NewRandom()
	if err != nil {
		s.Logger.Error().Err(err).Msg("failed to generate random filename")
		return "", err
	}

	fileExt := filepath.Ext(file.Filename)
	if fileExt == "" {
		s.Logger.Error().Msg("failed to get file extension")
		return "", errors.New("failed to get file extension")
	}

	return objectID.String() + "/" + randomFilename.String() + fileExt, nil
}
