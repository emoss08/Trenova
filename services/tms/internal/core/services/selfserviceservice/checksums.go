package selfserviceservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type documentChecksums struct {
	repo repositories.DocumentRepository
}

// NewDocumentChecksums reads the checksum of a stored document, which is all
// an acknowledgement needs from the document store.
func NewDocumentChecksums(repo repositories.DocumentRepository) DocumentChecksums {
	return &documentChecksums{repo: repo}
}

func (d *documentChecksums) Checksum(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	documentID pulid.ID,
) (string, error) {
	doc, err := d.repo.GetByID(ctx, repositories.GetDocumentByIDRequest{
		ID:         documentID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return "", err
	}
	return doc.ChecksumSHA256, nil
}
