package ratematrixservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixreview"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrixversion"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

func (s *Service) ListVersions(
	ctx context.Context,
	req *ratematrixports.ListVersionsRequest,
) ([]*ratematrixversion.RateMatrixVersion, error) {
	versions, err := s.versionRepo.List(ctx, req)
	if err != nil {
		return nil, err
	}
	if versions == nil {
		versions = []*ratematrixversion.RateMatrixVersion{}
	}
	return versions, nil
}

func (s *Service) GetVersion(
	ctx context.Context,
	req *ratematrixports.GetVersionRequest,
) (*ratematrixversion.RateMatrixVersion, error) {
	return s.versionRepo.Get(ctx, req)
}

type ListUsagesRequest struct {
	TenantInfo    pagination.TenantInfo
	RateMatrixID  pulid.ID
	VersionNumber *int64
	Limit         int
}

func (s *Service) ListUsages(
	ctx context.Context,
	req *ListUsagesRequest,
) ([]*formulatemplateports.MatrixUsage, error) {
	if _, err := s.repo.GetByID(ctx, &ratematrixports.GetRateMatrixByIDRequest{
		RateMatrixID: req.RateMatrixID,
		TenantInfo:   req.TenantInfo,
	}); err != nil {
		return nil, err
	}
	usages, err := s.usages.ListMatrixUsages(ctx, &formulatemplateports.ListMatrixUsagesRequest{
		TenantInfo:    req.TenantInfo,
		RateMatrixID:  req.RateMatrixID,
		VersionNumber: req.VersionNumber,
		Limit:         req.Limit,
	})
	if err != nil {
		return nil, err
	}
	if usages == nil {
		usages = []*formulatemplateports.MatrixUsage{}
	}
	return usages, nil
}

func (s *Service) ListReviews(
	ctx context.Context,
	req *ratematrixports.ListReviewsRequest,
) ([]*ratematrixreview.RateMatrixReview, error) {
	if _, err := s.repo.GetByID(ctx, &ratematrixports.GetRateMatrixByIDRequest{
		RateMatrixID: req.RateMatrixID,
		TenantInfo:   req.TenantInfo,
	}); err != nil {
		return nil, err
	}
	reviews, err := s.reviewRepo.ListByMatrix(ctx, req)
	if err != nil {
		return nil, err
	}
	if reviews == nil {
		reviews = []*ratematrixreview.RateMatrixReview{}
	}
	return reviews, nil
}

type ArchiveRequest struct {
	TenantInfo   pagination.TenantInfo `json:"-"`
	RateMatrixID pulid.ID              `json:"-"`
	ConfirmInUse bool                  `json:"confirmInUse"`
}

func (s *Service) Archive(
	ctx context.Context,
	req *ArchiveRequest,
) (*ratematrix.RateMatrix, error) {
	var original, archived *ratematrix.RateMatrix
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		matrix, err := s.repo.GetByID(txCtx, &ratematrixports.GetRateMatrixByIDRequest{
			RateMatrixID: req.RateMatrixID,
			TenantInfo:   req.TenantInfo,
			ForUpdate:    true,
		})
		if err != nil {
			return err
		}
		if !matrix.IsPublished() || matrix.Status == formulatemplate.StatusInactive ||
			matrix.Status == formulatemplate.StatusInReview {
			return errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalid,
				"Only a published matrix that is not under review can be archived; this one is {0}",
				matrix.Status,
			)
		}

		usages, err := s.usages.ListMatrixUsages(
			txCtx,
			&formulatemplateports.ListMatrixUsagesRequest{
				TenantInfo:   req.TenantInfo,
				RateMatrixID: req.RateMatrixID,
				Limit:        1,
			},
		)
		if err != nil {
			return err
		}
		if len(usages) > 0 && !req.ConfirmInUse {
			return errortypes.NewValidationError(
				"confirmInUse",
				errortypes.ErrInvalid,
				"Formula template versions pin {0}. They keep reading the version they pinned, but no new template version can read it. Confirm to archive it anyway",
				matrix.Code,
			)
		}

		snapshot := *matrix
		original = &snapshot
		matrix.Status = formulatemplate.StatusInactive
		saved, err := s.repo.UpdateHeader(txCtx, matrix)
		if err != nil {
			return err
		}
		archived = saved
		s.invalidateServed(txCtx, req.TenantInfo)
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.audit(ctx, &auditRecord{
		entity:    archived,
		original:  original,
		operation: permission.OpArchive,
		userID:    req.TenantInfo.UserID,
		comment:   "Rate matrix archived",
		critical:  true,
	})
	return archived, nil
}

type RestoreVersionRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	RateMatrixID  pulid.ID              `json:"-"`
	VersionNumber int64                 `json:"versionNumber"`
}

func (s *Service) RestoreVersion(
	ctx context.Context,
	req *RestoreVersionRequest,
) (*ratematrix.RateMatrix, error) {
	var original, restored *ratematrix.RateMatrix
	err := s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		matrix, err := s.repo.GetByID(txCtx, &ratematrixports.GetRateMatrixByIDRequest{
			RateMatrixID:      req.RateMatrixID,
			TenantInfo:        req.TenantInfo,
			IncludeDimensions: true,
			ForUpdate:         true,
		})
		if err != nil {
			return err
		}
		if matrix.Status == formulatemplate.StatusInReview {
			return errortypes.NewValidationError(
				"status",
				errortypes.ErrInvalid,
				"{0} is under review; ask for changes or reject it before replacing its draft",
				matrix.Code,
			)
		}
		version, err := s.versionRepo.Get(txCtx, &ratematrixports.GetVersionRequest{
			TenantInfo:    req.TenantInfo,
			RateMatrixID:  req.RateMatrixID,
			VersionNumber: req.VersionNumber,
		})
		if err != nil {
			return err
		}

		copied, err := s.repo.CopyVersion(txCtx, &ratematrixports.CopyVersionRequest{
			TenantInfo:   req.TenantInfo,
			RateMatrixID: req.RateMatrixID,
			From:         version.VersionNumber,
			To:           ratematrix.DraftVersion,
		})
		if err != nil {
			return err
		}

		snapshot := *matrix
		original = &snapshot
		applyRestoredDraft(matrix, version, copied, req.TenantInfo.UserID)
		saved, err := s.repo.UpdateHeader(txCtx, matrix)
		if err != nil {
			return err
		}
		restored, err = s.repo.GetByID(txCtx, &ratematrixports.GetRateMatrixByIDRequest{
			RateMatrixID:      saved.ID,
			TenantInfo:        req.TenantInfo,
			IncludeDimensions: true,
		})
		return err
	})
	if err != nil {
		return nil, err
	}

	s.audit(ctx, &auditRecord{
		entity:    restored,
		original:  original,
		operation: permission.OpUpdate,
		userID:    req.TenantInfo.UserID,
		comment:   fmt.Sprintf("Rate matrix draft restored from version %d", req.VersionNumber),
	})
	return restored, nil
}

func applyRestoredDraft(
	matrix *ratematrix.RateMatrix,
	version *ratematrixversion.RateMatrixVersion,
	copied int64,
	userID pulid.ID,
) {
	matrix.ValueType = version.ValueType
	matrix.DraftCellCount = copied
	if version.VersionNumber != matrix.PublishedVersionNumber {
		markDraftChanged(matrix, userID)
		return
	}
	matrix.HasDraftChanges = false
	matrix.DraftEditorIDs = []pulid.ID{}
	if matrix.Status == formulatemplate.StatusDraft {
		matrix.Status = formulatemplate.StatusActive
	}
}
