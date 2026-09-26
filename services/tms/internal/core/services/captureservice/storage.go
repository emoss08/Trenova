package captureservice

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/emoss08/trenova/internal/core/ports/storage"
	"github.com/emoss08/trenova/internal/core/services/encryptionservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	thumbnailContentType = "image/jpeg"
	// maxStoredObjectBytes bounds what is read back from storage. An envelope
	// is larger than its plaintext by its base64 and header, so this sits well
	// above the largest page the upload allows.
	maxStoredObjectBytes = 64 << 20
)

// pageKey names a page's object. The checksum is in the name so a different
// page claiming the same sequence can never overwrite the one already there.
func pageKey(orgID, batchID pulid.ID, sequence int, checksum string) string {
	return fmt.Sprintf("capture/%s/%s/%04d-%s.pdf", orgID, batchID, sequence, checksum[:16])
}

func thumbnailKey(pageStoragePath string) string {
	return pageStoragePath + ".thumb.jpg"
}

func (s *Service) aad(tenantInfo pagination.TenantInfo, key string) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeCapturePage,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ResourceID:     key,
	}
}

// putObject seals bytes and stores them. Pages are tenant paperwork and are
// encrypted at rest exactly like documents, bound to their own key so a copied
// object cannot be opened under another name.
func (s *Service) putObject(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	key string,
	contentType string,
	plaintext []byte,
) error {
	sealed, err := s.cipher.EncryptBytesWithAAD(plaintext, s.aad(tenantInfo, key))
	if err != nil {
		return fmt.Errorf("seal capture object: %w", err)
	}

	body := []byte(sealed)
	_, err = s.storage.Upload(ctx, &storage.UploadParams{
		Key:         key,
		ContentType: contentType,
		Size:        int64(len(body)),
		Body:        bytes.NewReader(body),
	})
	if err != nil {
		return fmt.Errorf("store capture object: %w", err)
	}

	return nil
}

// getObject reads and opens a stored object.
func (s *Service) getObject(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	key string,
) ([]byte, error) {
	download, err := s.storage.Download(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("read capture object: %w", err)
	}
	defer download.Body.Close()

	sealed, err := io.ReadAll(io.LimitReader(download.Body, maxStoredObjectBytes))
	if err != nil {
		return nil, fmt.Errorf("read capture object: %w", err)
	}

	plaintext, err := s.cipher.DecryptBytesWithAAD(string(sealed), s.aad(tenantInfo, key))
	if err != nil {
		return nil, fmt.Errorf("open capture object: %w", err)
	}

	return plaintext, nil
}

// deleteObject removes an object, tolerating one already gone: a purge that
// failed halfway is run again, and the second run must not stop at the first
// object the first run removed.
func (s *Service) deleteObject(ctx context.Context, key string) error {
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
