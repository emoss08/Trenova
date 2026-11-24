package driverportalservice

import (
	"context"
	"mime/multipart"
	"strings"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/personphoto"

	"github.com/emoss08/trenova/internal/modules/identity/domain/dashcontrol"
	"github.com/emoss08/trenova/internal/modules/workforce/app/selfserviceservice"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/profilechange"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workerpage"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type PortalPhotoChange struct {
	ID            pulid.ID                          `json:"id"`
	Status        profilechange.ProfileChangeStatus `json:"status"`
	SubmittedAt   int64                             `json:"submittedAt"`
	ProposedPhoto *personphoto.URLs                 `json:"proposedPhoto"`
}

type SubmitMyPhotoRequest struct {
	TenantInfo pagination.TenantInfo
	File       *multipart.FileHeader
	Note       string
}

func (s *Service) SubmitMyPhoto(
	ctx context.Context,
	req *SubmitMyPhotoRequest,
) (*PortalPhotoChange, error) {
	wrk, err := s.ResolveWorker(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}
	if _, err = s.RequireFeature(ctx, req.TenantInfo,
		func(control *dashcontrol.DashControl) bool { return control.AllowContactInfoEdit },
		"Your carrier manages your profile in the office. Ask them to update your photo.",
	); err != nil {
		return nil, err
	}

	submitted, err := s.selfService.SubmitPhotoChange(
		ctx,
		&selfserviceservice.SubmitPhotoChangeRequest{
			Worker:     wrk,
			File:       req.File,
			Note:       strings.TrimSpace(req.Note),
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	s.NotifyDispatch(
		ctx,
		req.TenantInfo,
		"profile_change_requested",
		"Photo change requested",
		wrk.FirstName+" "+wrk.LastName+
			" sent a new photo for their record. Review it in their file under Files, Driver portal.",
		workerpage.Path(
			wrk.ID,
			workerpage.AtView(workerpage.TabFiles, workerpage.ViewDriverPortal).
				WithRecord(workerpage.RecordRequest, submitted.ID),
		),
		map[string]any{"workerId": wrk.ID.String()},
	)

	proposed, err := s.selfService.ProposedPhoto(ctx, submitted)
	if err != nil {
		return nil, err
	}
	return &PortalPhotoChange{
		ID:            submitted.ID,
		Status:        submitted.Status,
		SubmittedAt:   submitted.SubmittedAt,
		ProposedPhoto: proposed,
	}, nil
}
