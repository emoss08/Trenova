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

// handlers to test functionality. ONLY DEV.
package handlers

import (
	"fmt"

	"github.com/emoss08/trenova/config"
	"github.com/emoss08/trenova/internal/server"
	"github.com/emoss08/trenova/pkg/file"
	"github.com/emoss08/trenova/pkg/minio"
	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
)

type FileMetadata struct {
	FileName       string `json:"fileName"`
	Classification string `json:"classification"`
	URL            string `json:"url"`
}

type TestHandler struct {
	fileService *file.FileService
	minio       minio.MinioClient
	logger      *config.ServerLogger
}

func NewTestHandler(s *server.Server) *TestHandler {
	return &TestHandler{
		minio:       s.Minio,
		logger:      s.Logger,
		fileService: file.NewFileService(s.Logger, s.FileHandler),
	}
}

func (th *TestHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/upload-multiple-files-with-classification", th.uploadMultipleFilesWithClassificationHandler())
}

func (th *TestHandler) uploadMultipleFilesWithClassificationHandler() fiber.Handler {
	return func(c *fiber.Ctx) error {
		form, err := c.MultipartForm()
		if err != nil {
			th.logger.Error().Err(err).Msg("Failed to parse multipart form")
			return fiber.NewError(fiber.StatusBadRequest, "Failed to parse multipart form")
		}

		files := form.File["files"]
		classifications := form.Value["classifications"]

		// log out the classifications along with the files
		th.logger.Debug().Msgf("Classifications: %v", classifications)
		th.logger.Debug().Msgf("Files: %v", files)

		if len(files) != len(classifications) {
			th.logger.Error().Msg("Number of files and classifications do not match")
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Number of files and classifications do not match",
			})
		}

		const batchSize = 10 // Number of files to process in each batch
		var fileMetadatas []FileMetadata

		for start := 0; start < len(files); start += batchSize {
			end := start + batchSize
			if end > len(files) {
				end = len(files)
			}

			batch := files[start:end]
			for i, file := range batch {
				uniqueID := uuid.New()
				filename := fmt.Sprintf("%s-%s", uniqueID.String(), file.Filename)

				fileData, fErr := th.fileService.ReadFileData(file)
				if fErr != nil {
					th.logger.Error().Err(fErr).Msgf("Failed to read file data for %s", file.Filename)
					return fErr
				}

				th.logger.Debug().Msgf("Uploading file %s with classification %s", filename, classifications[start+i])

				params := minio.SaveFileOptions{
					BucketName:  "test-documents",
					ObjectName:  filename,
					ContentType: file.Header.Get("Content-Type"),
					FileData:    fileData,
				}

				ui, mErr := th.minio.SaveFile(c.UserContext(), params)
				if mErr != nil {
					th.logger.Error().Err(mErr).Msgf("Failed to save file %s to MinIO", file.Filename)
					return mErr
				}

				fileMetadata := FileMetadata{
					FileName:       file.Filename,
					Classification: classifications[start+i],
					URL:            ui,
				}
				fileMetadatas = append(fileMetadatas, fileMetadata)
			}
		}

		return c.JSON(fileMetadatas)
	}
}
