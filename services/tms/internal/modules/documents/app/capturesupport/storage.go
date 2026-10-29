package capturesupport

import (
	"bytes"
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/documents/app/storedobject"
	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	ThumbnailContentType = "image/jpeg"
	// maxStoredObjectBytes bounds what is read back from storage, above the
	// largest page the upload allows.
	maxStoredObjectBytes = 64 << 20
)

// pageKey names a page's object. The checksum is in the name so a different
// page claiming the same sequence can never overwrite the one already there.
func PageKey(orgID, batchID pulid.ID, sequence int, checksum string) string {
	return fmt.Sprintf("capture/%s/%s/%04d-%s.pdf", orgID, batchID, sequence, checksum[:16])
}

func ThumbnailKey(pageStoragePath string) string {
	return pageStoragePath + ".thumb.jpg"
}

func captureObjectAAD(tenantInfo pagination.TenantInfo, key string) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeCapturePage,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ResourceID:     key,
	}
}

// putObject seals bytes and stores them. Pages are tenant paperwork and are
// sealed at rest exactly like documents, bound to their own key so a copied
// object cannot be opened under another name.
func (s *Service) PutObject(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	key string,
	contentType string,
	plaintext []byte,
) error {
	if _, err := storedobject.Write(ctx, s.storage, s.cipher, &storedobject.WriteRequest{
		Key:         key,
		ContentType: contentType,
		AAD:         captureObjectAAD(tenantInfo, key),
		Body:        bytes.NewReader(plaintext),
		Size:        int64(len(plaintext)),
	}); err != nil {
		return fmt.Errorf("store capture object: %w", err)
	}
	return nil
}

// getObject reads and opens a stored object.
func (s *Service) GetObject(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	key string,
) ([]byte, error) {
	data, err := storedobject.Read(
		ctx,
		s.storage,
		s.cipher,
		key,
		captureObjectAAD(tenantInfo, key),
		maxStoredObjectBytes,
	)
	if err != nil {
		return nil, fmt.Errorf("open capture object: %w", err)
	}
	return data, nil
}

// deleteObject removes an object, tolerating one already gone: a purge that
// failed halfway is run again, and the second run must not stop at the first
// object the first run removed.
func (s *Service) DeleteObject(ctx context.Context, key string) error {
	if key == "" {
		return nil
	}
	exists, err := s.storage.Exists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return nil
	}

	return s.storage.Delete(ctx, key)
}
