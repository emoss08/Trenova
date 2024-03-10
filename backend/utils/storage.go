package utils

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/google/uuid"
)

type StorageService interface {
	StoreFile(file multipart.File, orgID uuid.UUID, filename string) (string, error)
}

type LocalFileStorage struct {
	BaseDir string
}

// NewLocalFileStorage initializes a new LocalFileStorage with the base directory.
func NewLocalFileStorage() *LocalFileStorage {
	return &LocalFileStorage{
		BaseDir: "./media", // Base directory for all media
	}
}

// StoreFile saves the provided file in an organization-specific subdirectory within the media directory.
func (lfs *LocalFileStorage) StoreFile(file multipart.File, orgID uuid.UUID, filename string) (string, error) {
	// Construct the path to the organization-specific directory
	orgDir := filepath.Join(lfs.BaseDir, orgID.String())

	// Ensure the organization directory exists
	if err := os.MkdirAll(orgDir, os.ModePerm); err != nil {
		return "", err
	}

	// Create a unique filename to avoid overwrites
	newFilename := fmt.Sprintf("logo-%s", filename)
	filePath := filepath.Join(orgDir, newFilename)

	// Create a new file in the organization directory
	dst, err := os.Create(filePath)
	if err != nil {
		return "", err
	}
	defer func(dst *os.File) {
		err := dst.Close()
		if err != nil {
			fmt.Println("Error closing file:", err)
		}
	}(dst)

	// Copy the contents of the uploaded file to the new file
	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}

	// Return the relative path for storing in the database
	return fmt.Sprintf("/media/%s/%s", orgID.String(), newFilename), nil
}
