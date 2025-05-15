package fileutils

import (
	"crypto/rand"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/oklog/ulid/v2"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog/log"
)

func ReadFileData(file *multipart.FileHeader) ([]byte, error) {
	fileData, err := file.Open()
	if err != nil {
		return nil, eris.Wrap(err, "open file")
	}
	defer func(fData multipart.File) {
		err = fData.Close()
		if err != nil {
			log.Error().Err(err).Msg("failed to close file")
		}
	}(fileData)

	return io.ReadAll(fileData)
}

func RenameFile(file *multipart.FileHeader, objectID string) (string, error) {
	randomName, err := ulid.New(ulid.Timestamp(time.Now()), ulid.Monotonic(rand.Reader, 0))
	if err != nil {
		return "", eris.Wrap(err, "generate random name")
	}

	fileExt := filepath.Ext(file.Filename)
	if fileExt == "" {
		return "", eris.New("file has no extension")
	}

	newFileName := objectID + "_" + randomName.String() + fileExt

	return newFileName, nil
}

// FindProjectRoot looks for the go.mod file to determine project root
func FindProjectRoot(startPath string) (string, error) {
	currentPath := startPath
	for {
		if _, err := os.Stat(filepath.Join(currentPath, "go.mod")); err == nil {
			return currentPath, nil
		}

		parentPath := filepath.Dir(currentPath)
		if parentPath == currentPath {
			return "", eris.New("could not find project root (no go.mod found)")
		}
		currentPath = parentPath
	}
}

func EnsureDirExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		log.Info().Str("path", path).Msgf("directory %s does not exists, creating...", path)
		return os.MkdirAll(path, 0o750)
	}

	return nil
}

// GetFileSize returns the size of a file in bytes
func GetFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

func GetFileExtensionFromFileName(fileName string) string {
	return filepath.Ext(fileName)
}

func GetFileTypeFromFileName(fileName string) services.FileExtension {
	return services.GetFileTypeFromExtension(GetFileExtensionFromFileName(fileName))
}
