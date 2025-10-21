package fileutils

import (
	"crypto/rand"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"time"

	"github.com/oklog/ulid/v2"
)

func ReadFileData(file *multipart.FileHeader) ([]byte, error) {
	fileData, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}
	defer func(fData multipart.File) {
		err = fData.Close()
		if err != nil {
			fmt.Printf("failed to close file: %v", err)
			return
		}
	}(fileData)

	return io.ReadAll(fileData)
}

func RenameFile(file *multipart.FileHeader, objectID string) (string, error) {
	randomName, err := ulid.New(ulid.Timestamp(time.Now()), ulid.Monotonic(rand.Reader, 0))
	if err != nil {
		return "", fmt.Errorf("generate random name: %w", err)
	}

	fileExt := filepath.Ext(file.Filename)
	if fileExt == "" {
		return "", ErrFileHasNoExtension
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
			return "", ErrCouldNotFindProjectRoot
		}
		currentPath = parentPath
	}
}

func EnsureDirExists(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		fmt.Printf("directory %s does not exists, creating...", path)
		return os.MkdirAll(path, 0o750)
	}

	return nil
}
