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

package file

import (
	"os"
)

// FileHandler defines methods for file operations.
type FileHandler interface {
	Open(name string) (*os.File, error)
	Create(name string) (*os.File, error)
	Stat(name string) (os.FileInfo, error)
}

// OSFileHandler implements FileHandler using the os package.
type OSFileHandler struct{}

func (OSFileHandler) Open(name string) (*os.File, error) {
	return os.Open(name)
}

func (OSFileHandler) Create(name string) (*os.File, error) {
	return os.Create(name)
}

func (OSFileHandler) Stat(name string) (os.FileInfo, error) {
	return os.Stat(name)
}
