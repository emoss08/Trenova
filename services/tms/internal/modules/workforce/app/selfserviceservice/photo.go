package selfserviceservice

import (
	"context"
	"mime/multipart"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/personphoto"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/workforce/app/workerphotoservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
)

type SubmitPhotoChangeRequest struct {
	Worker     *worker.Worker
	File       *multipart.FileHeader
	Note       string
	TenantInfo pagination.TenantInfo
}

func (s *Service) SubmitPhotoChange(
	ctx context.Context,
	req *SubmitPhotoChangeRequest,
) (*profilechange.WorkerProfileChangeRequest, error) {
	pending, err := s.pendingChange(ctx, req.TenantInfo, req.Worker.ID, true)
	if err != nil {
		return nil, err
	}
	if pending != nil {
		return nil, errortypes.NewValidationError(
			"file",
			errortypes.ErrInvalidOperation,
			"You already have a photo waiting on your carrier. Withdraw it to send a different one.",
		)
	}

	staged, err := s.workerPhotos.Stage(ctx, &workerphotoservice.StageRequest{
		TenantInfo: req.TenantInfo,
		WorkerID:   req.Worker.ID,
		File:       req.File,
	})
	if err != nil {
		return nil, err
	}

	entity := &profilechange.WorkerProfileChangeRequest{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		WorkerID:       req.Worker.ID,
		Status:         profilechange.ProfileChangePending,
		Changes:        profilechange.PhotoChange(),
		Note:           req.Note,
		SubmittedAt:    timeutils.NowUnix(),
	}
	entity.ProposePhoto(staged.Key, staged.ThumbnailKey)

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		s.workerPhotos.Discard(ctx, workerRef(entity), staged)
		return nil, multiErr
	}

	created, err := s.repo.CreateChangeRequest(ctx, entity)
	if err != nil {
		s.workerPhotos.Discard(ctx, workerRef(entity), staged)
		return nil, err
	}
	return created, nil
}

func (s *Service) ProposedPhoto(
	ctx context.Context,
	entity *profilechange.WorkerProfileChangeRequest,
) (*personphoto.URLs, error) {
	if entity == nil || !entity.Status.IsOpen() || !entity.HasProposedPhoto() {
		return nil, nil //nolint:nilnil // a request without a staged photo shows none
	}
	return s.workerPhotos.ProposedURLs(ctx, workerRef(entity), &kernelports.StoredPhoto{
		Key:          entity.ProposedPhotoKey,
		ThumbnailKey: entity.ProposedPhotoThumbnailKey,
	})
}

func workerRef(entity *profilechange.WorkerProfileChangeRequest) workerphotoservice.WorkerRef {
	return workerphotoservice.WorkerRef{
		OrganizationID: entity.OrganizationID,
		WorkerID:       entity.WorkerID,
	}
}
