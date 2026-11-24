package phototest

import (
	"context"
	"sync"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/personphoto"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/shared/pulid"
)

type Fake struct {
	mu sync.Mutex

	StoreErr  error
	Stored    []*kernelports.StorePhotoRequest
	Discarded []string
}

var _ kernelports.Photos = (*Fake)(nil)

func (f *Fake) Store(
	_ context.Context,
	req *kernelports.StorePhotoRequest,
) (*kernelports.StoredPhoto, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.StoreErr != nil {
		return nil, f.StoreErr
	}
	f.Stored = append(f.Stored, req)
	prefix := req.Holder.OrganizationID.String() + "/" + req.Holder.Folder + "/" +
		req.Holder.ID.String() + "/" + pulid.MustNew("tst_").String()
	return &kernelports.StoredPhoto{
		Key:          prefix + ".jpg",
		ThumbnailKey: prefix + "-thumb.jpg",
	}, nil
}

func (f *Fake) Sign(
	_ context.Context,
	keys *kernelports.PhotoKeys,
) (*personphoto.URLs, error) {
	if keys == nil || keys.Key == "" {
		return nil, nil //nolint:nilnil // matches the port: no photo
	}
	thumbnail := keys.ThumbnailKey
	if thumbnail == "" {
		thumbnail = keys.Key
	}
	return &personphoto.URLs{
		URL:          "https://signed.test/" + keys.Key,
		ThumbnailURL: "https://signed.test/" + thumbnail,
	}, nil
}

func (f *Fake) Discard(_ context.Context, keys *kernelports.PhotoKeys) {
	if keys == nil {
		return
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, key := range []string{keys.Key, keys.ThumbnailKey} {
		if key != "" {
			f.Discarded = append(f.Discarded, key)
		}
	}
}

func (f *Fake) DiscardedKeys() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.Discarded...)
}
