// Package editconflict records each save of a setting people edit together
// and turns a save that lost a race into what the person needs to settle it:
// who saved in between, when, and which settings they changed since the
// version this person loaded.
package editconflict

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/editchange"
	"github.com/emoss08/trenova/internal/core/domain/settingversion"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/zap"
)

// Detail is what the other save did.
type Detail struct {
	Version   int64
	UpdatedAt int64
	Author    *tenant.User
	Changes   []editchange.Change
}

// Error reports the conflict, carrying the detail and the version-mismatch
// it explains.
func Error(detail *Detail, cause error) error {
	edit := &errortypes.EditConflict{
		Version:   detail.Version,
		UpdatedAt: detail.UpdatedAt,
		Changes:   make([]errortypes.EditConflictChange, 0, len(detail.Changes)),
	}
	if detail.Author != nil {
		edit.UpdatedByID = detail.Author.ID.String()
		edit.UpdatedByName = detail.Author.Name
	}
	for _, change := range detail.Changes {
		edit.Changes = append(edit.Changes, errortypes.EditConflictChange{
			Field: change.Field,
			Label: change.Label,
		})
	}

	return errortypes.NewEditConflictError(edit).WithInternal(cause)
}

type RecordRequest struct {
	TenantInfo pagination.TenantInfo
	Kind       settingversion.Kind
	SubjectID  pulid.ID
	Version    int64
	Snapshot   any
	AuthorID   pulid.ID
}

// Record keeps a save as the version it became. It runs in the caller's
// transaction, so a save and its version commit together.
func Record(
	ctx context.Context,
	versions repositories.SettingVersionRepository,
	req *RecordRequest,
) error {
	snapshot := make(map[string]any)
	if err := jsonutils.Convert(req.Snapshot, &snapshot); err != nil {
		return err
	}

	return versions.Create(ctx, &settingversion.SettingVersion{
		BusinessUnitID: req.TenantInfo.BuID,
		OrganizationID: req.TenantInfo.OrgID,
		Kind:           req.Kind,
		SubjectID:      req.SubjectID,
		Version:        req.Version,
		Snapshot:       snapshot,
		AuthorID:       typeutils.IDPtr(req.AuthorID),
	})
}

type ExplainRequest[T any] struct {
	TenantInfo     pagination.TenantInfo
	Kind           settingversion.Kind
	SubjectID      pulid.ID
	Loaded         int64
	Current        *T
	CurrentVersion int64
	UpdatedAt      int64
	Rules          []editchange.Rule[T]
	Cause          error
}

// Explain turns a version mismatch into an edit conflict; any other failure
// passes through unchanged. Detail it cannot find is left out rather than
// hiding the conflict.
func Explain[T any](
	ctx context.Context,
	versions repositories.SettingVersionRepository,
	logger *zap.Logger,
	req *ExplainRequest[T],
) error {
	if !errortypes.IsVersionMismatchError(req.Cause) {
		return req.Cause
	}

	detail := &Detail{Version: req.CurrentVersion, UpdatedAt: req.UpdatedAt}

	latest, err := versions.LatestAt(ctx, &repositories.GetSettingVersionAtRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		SubjectID:  req.SubjectID,
		Version:    req.CurrentVersion,
	})
	if err != nil {
		logger.Warn("could not read who last saved the setting", zap.Error(err))
	}
	if latest != nil {
		detail.Author = latest.Author
	}

	loaded, err := versions.LatestAt(ctx, &repositories.GetSettingVersionAtRequest{
		TenantInfo: req.TenantInfo,
		Kind:       req.Kind,
		SubjectID:  req.SubjectID,
		Version:    req.Loaded,
	})
	if err != nil {
		logger.Warn("could not read the version the person loaded", zap.Error(err))
	}
	if loaded != nil {
		before := new(T)
		if convErr := jsonutils.Convert(loaded.Snapshot, before); convErr != nil {
			logger.Warn("could not read the loaded version", zap.Error(convErr))
		} else {
			detail.Changes = editchange.Detect(req.Rules, before, req.Current)
		}
	}

	return Error(detail, req.Cause)
}
