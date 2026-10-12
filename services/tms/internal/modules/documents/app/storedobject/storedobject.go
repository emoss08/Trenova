// Package storedobject reads a document's bytes as stored: downloaded and
// decrypted under the associated data that binds them to their tenant and
// path. Every reader of document bytes goes through it, so none can read a
// stored object without the check that it is the tenant's.
package storedobject

import (
	"context"
	"io"

	"github.com/emoss08/trenova/internal/modules/kernel/app/encryptionservice"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

// AAD is the associated data a document object is encrypted under.
func AAD(tenantInfo pagination.TenantInfo, storagePath string) encryptionservice.AAD {
	return encryptionservice.AAD{
		Purpose:        encryptionservice.PurposeDocument,
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ResourceID:     storagePath,
	}
}

// Read downloads the object at storagePath and decrypts it under aad. An
// object stored without encryption is refused rather than returned.
func Read(
	ctx context.Context,
	storage kernelports.Client,
	encryption *encryptionservice.Service,
	storagePath string,
	aad encryptionservice.AAD,
) ([]byte, error) {
	download, err := storage.Download(ctx, storagePath)
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to download document").WithInternal(err)
	}
	if download == nil || download.Body == nil {
		return nil, errortypes.NewDatabaseError("Failed to download document")
	}
	defer download.Body.Close()

	data, err := io.ReadAll(download.Body)
	if err != nil {
		return nil, errortypes.NewDatabaseError("Failed to read document").WithInternal(err)
	}
	if encryption == nil {
		return nil, errortypes.NewBusinessError("Document encryption is not configured")
	}
	if !encryptionservice.IsEnvelope(string(data)) {
		return nil, errortypes.NewBusinessError("Stored document is not encrypted")
	}

	plaintext, err := encryption.DecryptBytesWithAAD(string(data), aad)
	if err != nil {
		return nil, errortypes.NewBusinessError("Failed to decrypt document").WithInternal(err)
	}

	return plaintext, nil
}
