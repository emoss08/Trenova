package homelayoutservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/homelayout"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/zap"
)

// PresetView is a preset plus the one thing the list needs that the row does
// not carry: how many people it currently reaches.
type PresetView struct {
	Preset            *homelayout.Preset
	AssignedUserCount int
}

type GetPresetRequest struct {
	Request

	PresetID pulid.ID
}

type SavePresetRequest struct {
	Request

	PresetID           pulid.ID
	Name               string
	Description        string
	Layout             *homelayout.Layout
	RoleIDs            []pulid.ID
	CoreResponsibility permission.CoreResponsibility
	IsOrgDefault       bool
	Locked             bool
	Priority           int
	Version            int64
}

type DeletePresetRequest struct {
	Request

	PresetID pulid.ID
}

// PreviewRequest renders a preset the way a member of a given role would see
// it, so an administrator can check an assignment before making it.
type PreviewRequest struct {
	Request

	PresetID pulid.ID
	RoleID   pulid.ID
}

func (s *Service) ListPresets(ctx context.Context, req *Request) ([]*PresetView, error) {
	log := s.l.With(zap.String("operation", "ListPresets"))

	presets, err := s.repo.ListPresets(
		ctx,
		&repositories.ListHomeLayoutPresetsRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		log.Error("failed to list home layout presets", zap.Error(err))
		return nil, err
	}

	views := make([]*PresetView, 0, len(presets))
	for _, preset := range presets {
		count, countErr := s.repo.CountUsersForRoles(ctx, req.TenantInfo, preset.RoleIDs)
		if countErr != nil {
			return nil, countErr
		}
		views = append(views, &PresetView{Preset: preset, AssignedUserCount: count})
	}

	return views, nil
}

func (s *Service) GetPreset(ctx context.Context, req *GetPresetRequest) (*PresetView, error) {
	preset, err := s.repo.GetPreset(ctx, &repositories.GetHomeLayoutPresetRequest{
		TenantInfo: req.TenantInfo,
		PresetID:   req.PresetID,
	})
	if err != nil {
		return nil, err
	}

	count, err := s.repo.CountUsersForRoles(ctx, req.TenantInfo, preset.RoleIDs)
	if err != nil {
		return nil, err
	}

	return &PresetView{Preset: preset, AssignedUserCount: count}, nil
}

func (s *Service) CreatePreset(
	ctx context.Context,
	req *SavePresetRequest,
) (*PresetView, error) {
	log := s.l.With(zap.String("operation", "CreatePreset"), zap.String("name", req.Name))

	entity := &homelayout.Preset{
		OrganizationID:     req.TenantInfo.OrgID,
		BusinessUnitID:     req.TenantInfo.BuID,
		Name:               req.Name,
		Description:        req.Description,
		Layout:             req.Layout,
		RoleIDs:            req.RoleIDs,
		CoreResponsibility: req.CoreResponsibility,
		IsOrgDefault:       req.IsOrgDefault,
		Locked:             req.Locked,
		Priority:           req.Priority,
		CreatedBy:          req.TenantInfo.UserID,
		Version:            1,
	}

	// Validate before normalizing: an author who references a widget that does
	// not exist gets told so, rather than having it silently dropped.
	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	entity.Layout = entity.Layout.Normalize()

	// Demote the incumbent first: the partial unique index would otherwise
	// reject the insert rather than hand the default over.
	if entity.IsOrgDefault {
		if err := s.repo.ClearOrgDefault(ctx, &repositories.ClearHomeLayoutOrgDefaultRequest{
			TenantInfo: req.TenantInfo,
		}); err != nil {
			return nil, err
		}
	}

	saved, err := s.repo.CreatePreset(ctx, entity)
	if err != nil {
		log.Error("failed to create home layout preset", zap.Error(err))
		return nil, err
	}

	count, err := s.repo.CountUsersForRoles(ctx, req.TenantInfo, saved.RoleIDs)
	if err != nil {
		return nil, err
	}

	return &PresetView{Preset: saved, AssignedUserCount: count}, nil
}

func (s *Service) UpdatePreset(
	ctx context.Context,
	req *SavePresetRequest,
) (*PresetView, error) {
	log := s.l.With(
		zap.String("operation", "UpdatePreset"),
		zap.String("presetID", req.PresetID.String()),
	)

	existing, err := s.repo.GetPreset(ctx, &repositories.GetHomeLayoutPresetRequest{
		TenantInfo: req.TenantInfo,
		PresetID:   req.PresetID,
	})
	if err != nil {
		return nil, err
	}

	existing.Name = req.Name
	existing.Description = req.Description
	existing.Layout = req.Layout
	existing.RoleIDs = req.RoleIDs
	existing.CoreResponsibility = req.CoreResponsibility
	existing.IsOrgDefault = req.IsOrgDefault
	existing.Locked = req.Locked
	existing.Priority = req.Priority
	existing.Version = req.Version

	multiErr := errortypes.NewMultiError()
	existing.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}
	existing.Layout = existing.Layout.Normalize()

	if existing.IsOrgDefault {
		if err = s.repo.ClearOrgDefault(ctx, &repositories.ClearHomeLayoutOrgDefaultRequest{
			TenantInfo: req.TenantInfo,
			ExceptID:   existing.ID,
		}); err != nil {
			return nil, err
		}
	}

	saved, err := s.repo.UpdatePreset(ctx, existing)
	if err != nil {
		log.Error("failed to update home layout preset", zap.Error(err))
		return nil, err
	}

	count, err := s.repo.CountUsersForRoles(ctx, req.TenantInfo, saved.RoleIDs)
	if err != nil {
		return nil, err
	}

	return &PresetView{Preset: saved, AssignedUserCount: count}, nil
}

func (s *Service) DeletePreset(ctx context.Context, req *DeletePresetRequest) error {
	return s.repo.DeletePreset(ctx, &repositories.DeleteHomeLayoutPresetRequest{
		TenantInfo: req.TenantInfo,
		PresetID:   req.PresetID,
	})
}

// PreviewPreset renders a preset without saving anything. It narrows against
// the administrator's own permissions rather than the target role's, because
// the permission engine answers for a principal and an administrator must not
// be shown data through a preview they could not open directly.
func (s *Service) PreviewPreset(
	ctx context.Context,
	req *PreviewRequest,
) (*EffectiveLayout, error) {
	layout, name, source, err := s.previewSource(ctx, req)
	if err != nil {
		return nil, err
	}

	narrowed, err := s.narrow(ctx, &req.Request, layout.Normalize())
	if err != nil {
		return nil, err
	}

	return &EffectiveLayout{
		Layout:       narrowed,
		Density:      homelayout.DensityComfortable,
		Source:       source,
		PresetID:     req.PresetID,
		PresetName:   name,
		CanCustomize: false,
	}, nil
}

// previewSource picks what the preview should render: an explicit preset when
// one is named, otherwise whatever a member of the named role would resolve to.
func (s *Service) previewSource(
	ctx context.Context,
	req *PreviewRequest,
) (*homelayout.Layout, string, Source, error) {
	if !req.PresetID.IsNil() {
		preset, err := s.repo.GetPreset(ctx, &repositories.GetHomeLayoutPresetRequest{
			TenantInfo: req.TenantInfo,
			PresetID:   req.PresetID,
		})
		if err != nil {
			return nil, "", SourceBuiltIn, err
		}
		return preset.Layout, preset.Name, SourceRolePreset, nil
	}

	if req.RoleID.IsNil() {
		return nil, "", SourceBuiltIn, errortypes.NewValidationError(
			"roleId",
			errortypes.ErrRequired,
			"Choose a preset or a role to preview",
		)
	}

	presets, err := s.repo.ListPresets(
		ctx,
		&repositories.ListHomeLayoutPresetsRequest{TenantInfo: req.TenantInfo},
	)
	if err != nil {
		return nil, "", SourceBuiltIn, err
	}

	role, err := s.roles.GetByID(ctx, repositories.GetRoleByIDRequest{
		ID:         req.RoleID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, "", SourceBuiltIn, err
	}

	assigned, source := pickPreset(presets, []*permission.Role{role})
	if assigned != nil {
		return assigned.Layout, assigned.Name, source, nil
	}

	return homelayout.BuiltInLayout(role.CoreResponsibility),
		homelayout.BuiltInLayoutName(role.CoreResponsibility),
		SourceBuiltIn,
		nil
}
