package userservice

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/personphoto"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/sliceutils"
)

func (s *Service) URLsByUser(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	userIDs []pulid.ID,
) (map[pulid.ID]*personphoto.URLs, error) {
	ids := sliceutils.Dedupe(slices.DeleteFunc(slices.Clone(userIDs), pulid.ID.IsNil))
	if len(ids) == 0 {
		return map[pulid.ID]*personphoto.URLs{}, nil
	}

	rows, err := s.repo.GetPhotoKeys(ctx, tenantports.GetUsersByIDsRequest{
		TenantInfo: tenantInfo,
		UserIDs:    ids,
	})
	if err != nil {
		return nil, err
	}

	out := make(map[pulid.ID]*personphoto.URLs, len(rows))
	for _, row := range rows {
		urls, signErr := s.photos.Sign(
			ctx,
			kernelports.UserPhotoKeys(row.UserID, row.Key, row.ThumbnailKey),
		)
		if signErr != nil {
			return nil, signErr
		}
		if urls != nil {
			out[row.UserID] = urls
		}
	}
	return out, nil
}

func (s *Service) SignPhotos(ctx context.Context, users ...*tenant.User) error {
	for _, user := range users {
		if user == nil {
			continue
		}
		urls, err := s.photos.Sign(ctx, profilePictureKeys(user))
		if err != nil {
			return err
		}
		user.Photo = urls
	}
	return nil
}

func (s *Service) withPhoto(ctx context.Context, user *tenant.User) (*tenant.User, error) {
	if err := s.SignPhotos(ctx, user); err != nil {
		return nil, err
	}
	return user, nil
}
