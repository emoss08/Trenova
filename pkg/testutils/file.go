// Copyright (c) 2024 Trenova Technologies, LLC
//
// Licensed under the Business Source License 1.1 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://trenova.app/pricing/
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// Key Terms:
// - Non-production use only
// - Change Date: 2026-11-16
// - Change License: GNU General Public License v2 or later
//
// For full license text, see the LICENSE file in the root directory.

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
