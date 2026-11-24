package kernelports

import (
	"context"
	"mime/multipart"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/objectkey"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/personphoto"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const UserPhotoFolder = "user/profile-picture"

type UploadScanRequest struct {
	TenantInfo pagination.TenantInfo
	Field      string
	FileName   string
	Content    []byte
}

type UploadScanner interface {
	ScanUpload(ctx context.Context, req *UploadScanRequest) error
}

type PhotoHolder struct {
	OrganizationID pulid.ID
	Folder         string
	ID             pulid.ID
}

type StorePhotoRequest struct {
	TenantInfo   pagination.TenantInfo
	File         *multipart.FileHeader
	Holder       PhotoHolder
	ResourceType string
}

type StoredPhoto struct {
	Key          string
	ThumbnailKey string
}

type PhotoKeys struct {
	Holder       PhotoHolder
	Key          string
	ThumbnailKey string
}

type Photos interface {
	Store(ctx context.Context, req *StorePhotoRequest) (*StoredPhoto, error)
	Sign(ctx context.Context, keys *PhotoKeys) (*personphoto.URLs, error)
	Discard(ctx context.Context, keys *PhotoKeys)
}

func UserPhotoKeys(userID pulid.ID, key, thumbnailKey string) *PhotoKeys {
	owner, _, _ := objectkey.Owner(key)
	return &PhotoKeys{
		Holder:       PhotoHolder{OrganizationID: owner, Folder: UserPhotoFolder, ID: userID},
		Key:          key,
		ThumbnailKey: thumbnailKey,
	}
}
