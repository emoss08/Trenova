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

package file_test

import (
	"bytes"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/require"
)

func createMultipartFile(t *testing.T, filename, content string) *multipart.FileHeader {
	t.Helper()

	var b bytes.Buffer
	writer := multipart.NewWriter(&b)
	part, err := writer.CreateFormFile("file", filename)
	require.NoError(t, err)

	_, err = part.Write([]byte(content))
	require.NoError(t, err)
	writer.Close()

	req := multipart.NewReader(&b, writer.Boundary())
	form, err := req.ReadForm(1024)
	require.NoError(t, err)

	return form.File["file"][0]
}

// func TestReadFileData(t *testing.T) {
// 	logger := zerolog.New(log.Logger)
// 	mockFileHandler := testutil.NewMockFileHandler()
// 	fs := file.NewFileService(&logger, mockFileHandler)

// 	// Create a test file
// 	filename := "test.txt"
// 	content := "This is a test file"
// 	fileHeader := createMultipartFile(t, filename, content)

// 	// Add the file content to the mock handler
// 	mockFileHandler.FileData[filename] = []byte(content)

// 	// Test ReadFileData
// 	fileData, err := fs.ReadFileData(fileHeader)
// 	require.NoError(t, err)
// 	assert.Equal(t, content, string(fileData))
// }

// func TestRenameFile(t *testing.T) {
// 	logger := zerolog.New(log.Logger)
// 	mockFileHandler := testutil.NewMockFileHandler()
// 	fs := file.NewFileService(&logger, mockFileHandler)

// 	// Create a test file
// 	filename := "test.txt"
// 	content := "This is a test file"
// 	fileHeader := createMultipartFile(t, filename, content)

// 	// Create a new object ID
// 	objectID := pulid.MustNew("test")

// 	// Test RenameFile
// 	newFileName, err := fs.RenameFile(fileHeader, objectID)
// 	require.NoError(t, err)

// 	fileExt := filepath.Ext(filename)
// 	assert.NotEmpty(t, newFileName)
// 	assert.Equal(t, filepath.Ext(newFileName), fileExt)
// }
