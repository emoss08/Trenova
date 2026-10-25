package formulalibraryservice

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulalibrary"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulalibraryports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

var versionDiffIgnoreFields = []string{
	"id",
	"createdAt",
	"createdById",
	"versionNumber",
	"changeMessage",
	"changeSummary",
}

func (s *Service) ListVersions(
	ctx context.Context,
	req *formulalibraryports.ListLibraryVersionsRequest,
) ([]*formulalibrary.FormulaLibraryVersion, error) {
	return s.versionRepo.List(ctx, req)
}

func (s *Service) GetVersion(
	ctx context.Context,
	req *formulalibraryports.GetLibraryVersionRequest,
) (*formulalibrary.FormulaLibraryVersion, error) {
	return s.versionRepo.GetByLibraryAndVersion(ctx, req)
}

type VersionDiff struct {
	FromVersion int64                            `json:"fromVersion"`
	ToVersion   int64                            `json:"toVersion"`
	Changes     map[string]jsonutils.FieldChange `json:"changes"`
	ChangeCount int                              `json:"changeCount"`
}

func (s *Service) CompareVersions(
	ctx context.Context,
	req *formulalibraryports.GetLibraryVersionRangeRequest,
) (*VersionDiff, error) {
	versions, err := s.versionRepo.GetVersionRange(ctx, req)
	if err != nil {
		return nil, err
	}

	var from, to *formulalibrary.FormulaLibraryVersion
	for _, version := range versions {
		switch version.VersionNumber {
		case req.FromVersion:
			from = version
		case req.ToVersion:
			to = version
		}
	}
	if from == nil || to == nil {
		return nil, errortypes.NewValidationError(
			"versions",
			errortypes.ErrNotFound,
			"Both versions must exist for comparison",
		)
	}

	changes, err := jsonutils.JSONDiff(from, to, &jsonutils.DiffOptions{
		IgnoreFields: versionDiffIgnoreFields,
	})
	if err != nil {
		return nil, err
	}
	return &VersionDiff{
		FromVersion: req.FromVersion,
		ToVersion:   req.ToVersion,
		Changes:     changes,
		ChangeCount: len(changes),
	}, nil
}

type RollbackRequest struct {
	TenantInfo    pagination.TenantInfo `json:"-"`
	LibraryID     pulid.ID              `json:"-"`
	TargetVersion int64                 `json:"targetVersion"`
	ChangeMessage string                `json:"changeMessage"`
}

func (s *Service) Rollback(
	ctx context.Context,
	req *RollbackRequest,
) (*formulalibrary.FormulaLibrary, error) {
	target, err := s.versionRepo.GetByLibraryAndVersion(
		ctx,
		&formulalibraryports.GetLibraryVersionRequest{
			TenantInfo:    req.TenantInfo,
			LibraryID:     req.LibraryID,
			VersionNumber: req.TargetVersion,
		},
	)
	if err != nil {
		return nil, err
	}

	current, err := s.repo.GetByID(ctx, &formulalibraryports.GetLibraryByIDRequest{
		TenantInfo: req.TenantInfo,
		LibraryID:  req.LibraryID,
	})
	if err != nil {
		return nil, err
	}

	resolved := current.ApplyVersion(target)
	resolved.Source = renameHeader(resolved.Source, resolved.Path())
	resolved.CurrentVersionNumber = current.CurrentVersionNumber + 1
	if err = s.prepareContent(ctx, resolved); err != nil {
		return nil, err
	}

	changeMessage := req.ChangeMessage
	if changeMessage == "" {
		changeMessage = fmt.Sprintf("Rolled back to version %d", req.TargetVersion)
	}
	comment := changeMessage
	if (current.Status == formulatemplate.StatusActive ||
		current.Status == formulatemplate.StatusInReview) && resolved.HasMaterialChange(current) {
		resolved.Status = formulatemplate.StatusDraft
		clearApprovalFields(resolved)
		comment = changeMessage + "; rollback reverted approval"
	}

	var updated *formulalibrary.FormulaLibrary
	err = s.db.Transact(ctx, kernelports.TxOptions{}, func(txCtx context.Context) error {
		saved, txErr := s.repo.Update(txCtx, resolved)
		if txErr != nil {
			return txErr
		}
		if _, txErr = s.versionRepo.Create(txCtx, formulalibrary.NewVersionFromLibrary(
			saved, saved.CurrentVersionNumber, req.TenantInfo.UserID, changeMessage, nil,
		)); txErr != nil {
			return txErr
		}
		updated = saved
		return nil
	})
	if err != nil {
		return nil, err
	}

	s.logAuditAction(ctx, updated, permission.OpUpdate, req.TenantInfo.UserID, current, comment,
		auditservice.WithCritical())
	return updated, nil
}

type ReviewDiffRequest struct {
	TenantInfo pagination.TenantInfo
	LibraryID  pulid.ID
}

type ReviewDiffResponse struct {
	HasApprovedBase bool                             `json:"hasApprovedBase"`
	BaseVersion     int64                            `json:"baseVersion"`
	CurrentVersion  int64                            `json:"currentVersion"`
	BaseSource      string                           `json:"baseSource"`
	CurrentSource   string                           `json:"currentSource"`
	BaseImports     []tarifftypes.Import             `json:"baseImports"`
	CurrentImports  []tarifftypes.Import             `json:"currentImports"`
	Changes         map[string]jsonutils.FieldChange `json:"changes"`
	ChangeCount     int                              `json:"changeCount"`
}

func (s *Service) ReviewDiff(
	ctx context.Context,
	req *ReviewDiffRequest,
) (*ReviewDiffResponse, error) {
	library, err := s.repo.GetByID(ctx, &formulalibraryports.GetLibraryByIDRequest{
		TenantInfo: req.TenantInfo,
		LibraryID:  req.LibraryID,
	})
	if err != nil {
		return nil, err
	}

	approved, err := s.versionRepo.GetLatestByStatus(
		ctx,
		&formulalibraryports.GetLatestLibraryVersionByStatusRequest{
			TenantInfo: req.TenantInfo,
			LibraryID:  req.LibraryID,
			Status:     formulatemplate.StatusActive,
		},
	)
	if err != nil {
		return nil, err
	}

	current := formulalibrary.NewVersionFromLibrary(
		library, library.CurrentVersionNumber, req.TenantInfo.UserID, "", nil,
	)
	resp := &ReviewDiffResponse{
		CurrentVersion: library.CurrentVersionNumber,
		CurrentSource:  library.Source,
		CurrentImports: orEmpty(library.Imports),
		BaseImports:    []tarifftypes.Import{},
	}
	base := &formulalibrary.FormulaLibraryVersion{}
	if approved != nil {
		base = approved
		resp.HasApprovedBase = true
		resp.BaseVersion = approved.VersionNumber
		resp.BaseSource = approved.Source
		resp.BaseImports = orEmpty(approved.Imports)
	}

	changes, err := jsonutils.JSONDiff(base, current, &jsonutils.DiffOptions{
		IgnoreFields: append(slices.Clone(versionDiffIgnoreFields), "status"),
	})
	if err != nil {
		return nil, err
	}
	resp.Changes = changes
	resp.ChangeCount = len(changes)
	return resp, nil
}

func orEmpty(imports []tarifftypes.Import) []tarifftypes.Import {
	if imports == nil {
		return []tarifftypes.Import{}
	}
	return imports
}
