package volumegen

import (
	"path"

	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	facNoFile = "Created by the volume seed; no file was stored for this document"
)

// DocumentRow is the metadata row of a file the app would have stored. The
// seed stores no file, so the row says so instead of pretending a preview
// or extracted text exists.
func (g *Generator) DocumentRow(
	resourceType string,
	resourceID pulid.ID,
	fileName, fileType string,
	profile document.ProcessingProfile,
	at int64,
	by pulid.ID,
	size int64,
) *document.Document {
	id := IDAt("doc_", at)
	storagePath := fileutils.GenerateStoragePath(g.Refs.OrgID.String(), resourceType, fileName)
	return &document.Document{
		ID:                  id,
		OrganizationID:      g.Refs.OrgID,
		BusinessUnitID:      g.Refs.BuID,
		LineageID:           id,
		VersionNumber:       1,
		IsCurrentVersion:    true,
		FileName:            path.Base(storagePath),
		OriginalName:        fileName,
		FileSize:            size,
		FileType:            fileType,
		StoragePath:         storagePath,
		Status:              document.StatusActive,
		ResourceID:          resourceID.String(),
		ResourceType:        resourceType,
		ProcessingProfile:   profile,
		UploadedByID:        by,
		PreviewStatus:       document.PreviewStatusUnsupported,
		ContentStatus:       document.ContentStatusFailed,
		ContentError:        facNoFile,
		CreatedAt:           at,
		UpdatedAt:           at,
		CryptoMode:          "blob_v1",
		CryptoVersion:       2,
		ShipmentDraftStatus: document.ShipmentDraftStatusUnavailable,
		SignatureStatus:     document.SignatureStatusUnknown,
	}
}
