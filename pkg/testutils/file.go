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

package testutils

import (
	"bytes"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// MockFileHandler is a mock implementation of FileHandler for testing.
type MockFileHandler struct {
	FileData map[string][]byte
}

// NewMockFileHandler creates a new MockFileHandler.
func NewMockFileHandler() *MockFileHandler {
	return &MockFileHandler{
		FileData: make(map[string][]byte),
	}
}

func (mfh *MockFileHandler) Open(name string) (*os.File, error) {
	content, exists := mfh.FileData[name]
	if !exists {
		return nil, os.ErrNotExist
	}

	tmpFile, err := os.CreateTemp("", "mockfile")
	if err != nil {
		return nil, err
	}

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return nil, err
	}

	if _, err := tmpFile.Seek(0, io.SeekStart); err != nil {
		tmpFile.Close()
		return nil, err
	}

	return tmpFile, nil
}

func (mfh *MockFileHandler) Create(name string) (*os.File, error) {
	return os.CreateTemp("", "mockfile")
}

func (mfh *MockFileHandler) Stat(name string) (os.FileInfo, error) {
	content, exists := mfh.FileData[name]
	if !exists {
		return nil, os.ErrNotExist
	}

	tmpFile, err := os.CreateTemp("", "mockfile")
	if err != nil {
		return nil, err
	}

	if _, err := tmpFile.Write(content); err != nil {
		tmpFile.Close()
		return nil, err
	}

	fileInfo, err := tmpFile.Stat()
	if err != nil {
		tmpFile.Close()
		return nil, err
	}

	return fileInfo, nil
}

// CreateTestFile creates a file with the given content and returns its path and a multipart.FileHeader.
func CreateTestFile(dir, fileName, content string) (string, *multipart.FileHeader, error) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", dir)
	if err != nil {
		return "", nil, err
	}

	// Create the file within the temporary directory
	filePath := filepath.Join(tempDir, fileName)
	err = os.WriteFile(filePath, []byte(content), 0o644)
	if err != nil {
		return "", nil, err
	}

	// Create a multipart.FileHeader
	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, err := writer.CreateFormFile("profilePicture", fileName)
	if err != nil {
		return "", nil, err
	}

	file, err := os.Open(filePath)
	if err != nil {
		return "", nil, err
	}
	defer file.Close()

	_, err = io.Copy(part, file)
	if err != nil {
		return "", nil, err
	}
	writer.Close()

	req := multipart.NewReader(&b, writer.Boundary())
	form, err := req.ReadForm(1024)
	if err != nil {
		return "", nil, err
	}

	fileHeader := form.File["profilePicture"][0]
	return filePath, fileHeader, nil
}
