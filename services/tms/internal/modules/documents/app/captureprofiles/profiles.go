package captureprofiles

import (
	"context"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/modules/documents/app/capturesupport"
	"github.com/emoss08/trenova/internal/modules/documents/domain/capture"
	"github.com/emoss08/trenova/internal/modules/documents/domain/captureprofile"
	"github.com/emoss08/trenova/internal/modules/documents/ports/captureports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	// maxProfiles is how many profiles a list returns. A tenant keeps a few:
	// one for paperwork, one for photos, one for the odd legal-size form.
	maxProfiles = 100

	profileNameConstraint    = "uq_capture_profiles_name"
	profileDefaultConstraint = "uq_capture_profiles_default"
)

// ProfileSettings is everything a person sets on a scanning profile.
type ProfileSettings struct {
	Name                string                             `json:"name"`
	Description         string                             `json:"description"`
	Status              captureprofile.ProfileStatus       `json:"status"`
	IsDefault           bool                               `json:"isDefault"`
	DPI                 int                                `json:"dpi"`
	PixelType           capture.PixelType                  `json:"pixelType"`
	Duplex              bool                               `json:"duplex"`
	UseFeeder           bool                               `json:"useFeeder"`
	DiscardBlankPages   bool                               `json:"discardBlankPages"`
	JPEGQuality         int                                `json:"jpegQuality"`
	ShowDriverUI        bool                               `json:"showDriverUi"`
	SeparatorStrategies []captureprofile.SeparatorStrategy `json:"separatorStrategies"`
	FixedPageCount      int                                `json:"fixedPageCount"`
}

func (ps *ProfileSettings) apply(profile *captureprofile.CaptureProfile) {
	profile.Name = strings.TrimSpace(ps.Name)
	profile.Description = strings.TrimSpace(ps.Description)
	profile.Status = ps.Status
	profile.IsDefault = ps.IsDefault
	profile.DPI = ps.DPI
	profile.PixelType = ps.PixelType
	profile.Duplex = ps.Duplex
	profile.UseFeeder = ps.UseFeeder
	profile.DiscardBlankPages = ps.DiscardBlankPages
	profile.JPEGQuality = ps.JPEGQuality
	profile.ShowDriverUI = ps.ShowDriverUI
	profile.FixedPageCount = ps.FixedPageCount

	strategies := make([]captureprofile.SeparatorStrategy, 0, len(ps.SeparatorStrategies))
	for _, strategy := range ps.SeparatorStrategies {
		if !slices.Contains(strategies, strategy) {
			strategies = append(strategies, strategy)
		}
	}
	profile.SeparatorStrategies = strategies
	profile.ApplyDefaults()
}

type ListProfilesInput struct {
	TenantInfo pagination.TenantInfo        `json:"-"`
	Status     captureprofile.ProfileStatus `json:"status"`
	Query      string                       `json:"query"`
}

// ListProfiles is every profile, for the people who manage them.
func (s *Service) ListProfiles(
	ctx context.Context,
	in *ListProfilesInput,
) ([]*captureprofile.CaptureProfile, error) {
	if _, err := s.support.Require(
		ctx,
		in.TenantInfo,
		permission.ResourceCaptureProfile,
		permission.OpRead,
	); err != nil {
		return nil, err
	}

	return s.listProfiles(ctx, in.TenantInfo, in.Status, in.Query)
}

// AvailableProfiles is what a person may pick when they start a scan: the
// active profiles, default first. Choosing one needs no more than being able
// to capture.
func (s *Service) AvailableProfiles(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*captureprofile.CaptureProfile, error) {
	if _, err := s.support.Require(
		ctx,
		tenantInfo,
		permission.ResourceCaptureBatch,
		permission.OpCreate,
	); err != nil {
		return nil, err
	}

	return s.listProfiles(ctx, tenantInfo, captureprofile.ProfileActive, "")
}

func (s *Service) listProfiles(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	status captureprofile.ProfileStatus,
	query string,
) ([]*captureprofile.CaptureProfile, error) {
	if status != "" && !status.IsValid() {
		return nil, errortypes.NewValidationError("status", errortypes.ErrInvalid,
			"Status must be Active or Inactive")
	}

	result, err := s.profileRepo.List(ctx, &captureports.ListCaptureProfilesRequest{
		Filter: &pagination.QueryOptions{
			TenantInfo: tenantInfo,
			Pagination: pagination.Info{Limit: maxProfiles},
			Query:      strings.TrimSpace(query),
		},
		Status: status,
	})
	if err != nil {
		return nil, err
	}

	return result.Items, nil
}

// GetProfile is one profile, for editing.
func (s *Service) GetProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	profileID pulid.ID,
) (*captureprofile.CaptureProfile, error) {
	if _, err := s.support.Require(
		ctx,
		tenantInfo,
		permission.ResourceCaptureProfile,
		permission.OpRead,
	); err != nil {
		return nil, err
	}

	return s.profileRepo.GetByID(ctx, captureports.GetCaptureProfileByIDRequest{
		ID:         profileID,
		TenantInfo: tenantInfo,
	})
}

type CreateProfileInput struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	Settings   ProfileSettings       `json:"settings"`
}

// CreateProfile adds a profile. Making it the default takes the flag off
// whichever profile had it.
func (s *Service) CreateProfile(
	ctx context.Context,
	in *CreateProfileInput,
) (*captureprofile.CaptureProfile, error) {
	if _, err := s.support.Require(
		ctx,
		in.TenantInfo,
		permission.ResourceCaptureProfile,
		permission.OpCreate,
	); err != nil {
		return nil, err
	}

	profile := &captureprofile.CaptureProfile{
		ID:             pulid.MustNew("cprf_"),
		OrganizationID: in.TenantInfo.OrgID,
		BusinessUnitID: in.TenantInfo.BuID,
	}
	in.Settings.apply(profile)
	if err := validateProfile(profile); err != nil {
		return nil, err
	}

	created, err := s.profileRepo.Create(ctx, profile)
	if err != nil {
		return nil, profileConstraintError(err)
	}

	s.profileChanged(ctx, in.TenantInfo, created, permission.OpCreate, capturesupport.ActionCreated,
		"Created a scanning profile")

	return created, nil
}

type UpdateProfileInput struct {
	TenantInfo pagination.TenantInfo `json:"-"`
	ProfileID  pulid.ID              `json:"-"`
	// Version is the version the person was editing. A save over somebody
	// else's change is refused rather than overwriting it.
	Version  int64           `json:"version"`
	Settings ProfileSettings `json:"settings"`
}

// UpdateProfile changes a profile. It changes only scans started from now on:
// a batch keeps the settings it was captured with.
func (s *Service) UpdateProfile(
	ctx context.Context,
	in *UpdateProfileInput,
) (*captureprofile.CaptureProfile, error) {
	if _, err := s.support.Require(
		ctx,
		in.TenantInfo,
		permission.ResourceCaptureProfile,
		permission.OpUpdate,
	); err != nil {
		return nil, err
	}

	profile, err := s.profileRepo.GetByID(ctx, captureports.GetCaptureProfileByIDRequest{
		ID:         in.ProfileID,
		TenantInfo: in.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if profile.Version != in.Version {
		return nil, errortypes.NewConflictError(
			"Somebody else changed this profile; reload it and try again")
	}

	in.Settings.apply(profile)
	if err = validateProfile(profile); err != nil {
		return nil, err
	}

	updated, err := s.profileRepo.Update(ctx, profile)
	if err != nil {
		return nil, profileConstraintError(err)
	}

	s.profileChanged(ctx, in.TenantInfo, updated, permission.OpUpdate, capturesupport.ActionUpdated,
		"Changed a scanning profile")

	return updated, nil
}

// DeleteProfile removes a profile. Requests waiting on it fall back to the
// default when they start, and batches already captured keep the settings
// they were captured with, so nothing in flight depends on the row.
func (s *Service) DeleteProfile(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	profileID pulid.ID,
) error {
	if _, err := s.support.Require(
		ctx,
		tenantInfo,
		permission.ResourceCaptureProfile,
		permission.OpDelete,
	); err != nil {
		return err
	}

	profile, err := s.profileRepo.GetByID(ctx, captureports.GetCaptureProfileByIDRequest{
		ID:         profileID,
		TenantInfo: tenantInfo,
	})
	if err != nil {
		return err
	}

	if err = s.profileRepo.Delete(ctx, captureports.DeleteCaptureProfileRequest{
		ID:         profileID,
		TenantInfo: tenantInfo,
	}); err != nil {
		return err
	}

	s.profileChanged(ctx, tenantInfo, profile, permission.OpDelete, capturesupport.ActionDeleted,
		"Deleted a scanning profile")

	return nil
}

func validateProfile(profile *captureprofile.CaptureProfile) error {
	multiErr := errortypes.NewMultiError()
	profile.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// profileConstraintError turns the name and default indexes into field errors
// a form can show. The default index is only hit when two saves race to take
// the flag, since the repository clears it inside the same transaction.
func profileConstraintError(err error) error {
	if !dberror.IsUniqueConstraintViolation(err) {
		return err
	}

	switch dberror.ExtractConstraintName(err) {
	case profileNameConstraint:
		return errortypes.NewValidationError("name", errortypes.ErrDuplicate,
			"A profile with this name already exists")
	case profileDefaultConstraint:
		return errortypes.NewConflictError(
			"Another profile was made the default at the same time; try again")
	default:
		return err
	}
}

func (s *Service) profileChanged(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	profile *captureprofile.CaptureProfile,
	op permission.Operation,
	action string,
	comment string,
) {
	s.support.Publish(
		ctx,
		tenantInfo,
		permission.ResourceCaptureProfile,
		profile.ID,
		action,
		pulid.Nil,
		nil,
	)
	s.support.LogAudit(&auditports.LogActionParams{
		Resource:       permission.ResourceCaptureProfile,
		ResourceID:     profile.ID.String(),
		Operation:      op,
		UserID:         tenantInfo.UserID,
		PrincipalType:  permissionports.PrincipalTypeUser,
		PrincipalID:    tenantInfo.UserID,
		CurrentState:   jsonutils.MustToJSON(profile),
		OrganizationID: profile.OrganizationID,
		BusinessUnitID: profile.BusinessUnitID,
	}, comment)
}
