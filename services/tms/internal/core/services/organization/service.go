/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package organization

import (
	"context"
	"mime/multipart"
	"net/http"

	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/fileutils"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator/organizationvalidator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.OrganizationRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	FileService  services.FileService
	Validator    *organizationvalidator.Validator
}

type Service struct {
	repo repositories.OrganizationRepository
	l    *zerolog.Logger
	ps   services.PermissionService
	as   services.AuditService
	fs   services.FileService
	v    *organizationvalidator.Validator
}

//nolint:gocritic // The p parameter is passed using fx.In
func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "organization").
		Logger()

	return &Service{
		repo: p.Repo,
		ps:   p.PermService,
		fs:   p.FileService,
		as:   p.AuditService,
		l:    &log,
		v:    p.Validator,
	}
}

// SelectOptions returns a list of select options for organizations.
func (s *Service) SelectOptions(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, opts)
	if err != nil {
		return nil, eris.Wrap(err, "select organizations")
	}

	// Convert the organizations to select options
	options := make([]*types.SelectOption, 0, len(result.Items))
	for _, org := range result.Items {
		options = append(options, &types.SelectOption{
			Value: org.ID.String(),
			Label: org.Name,
		})
	}

	return options, nil
}

// List returns a list of organizations.
func (s *Service) List(
	ctx context.Context,
	opts *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*organization.Organization], error) {
	log := s.l.With().
		Str("operation", "List").
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.TenantOpts.UserID,
				Resource:       permission.ResourceOrganization,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.TenantOpts.BuID,
			},
		},
	)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError("You do not have permission to read organizations")
	}

	entities, err := s.repo.List(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to list organizations")
		return nil, eris.Wrap(err, "failed to list organizations")
	}

	return &ports.ListResult[*organization.Organization]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

// Get returns an organization by its ID.
func (s *Service) Get(
	ctx context.Context,
	opts repositories.GetOrgByIDOptions,
) (*organization.Organization, error) {
	log := s.l.With().
		Str("operation", "Get").
		Str("orgID", opts.OrgID.String()).
		Logger()

	org, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get organization by id")
		return nil, eris.Wrap(err, "failed to get organization by id")
	}

	return org, nil
}

// Create creates an organization.
func (s *Service) Create(
	ctx context.Context,
	org *organization.Organization,
	userID pulid.ID,
) (*organization.Organization, error) {
	log := s.l.With().Str("operation", "Create").
		Str("orgID", org.ID.String()).
		Str("userID", userID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         userID,
			Resource:       permission.ResourceOrganization,
			Action:         permission.ActionCreate,
			BusinessUnitID: org.BusinessUnitID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to create an organization",
		)
	}

	if err := s.v.Validate(ctx, org); err != nil {
		return nil, err
	}

	createdOrg, err := s.repo.Create(ctx, org)
	if err != nil {
		s.l.Error().Err(err).Interface("org", org).Msg("failed to create organization")
		return nil, eris.Wrap(err, "failed to create organization")
	}

	// Log the creation
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceOrganization,
			ResourceID:     org.ID.String(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(org),
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
		},
		audit.WithComment("Organization created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log organization creation")
	}

	return createdOrg, nil
}

// Update updates an organization.
func (s *Service) Update(
	ctx context.Context,
	org *organization.Organization,
	userID pulid.ID,
) (*organization.Organization, error) {
	log := s.l.With().
		Str("operation", "Update").
		Interface("org", org).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx, []*services.PermissionCheck{
		{
			UserID:         userID,
			Resource:       permission.ResourceOrganization,
			Action:         permission.ActionUpdate,
			BusinessUnitID: org.BusinessUnitID,
			ResourceID:     org.ID,
		},
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "failed to check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this organization",
		)
	}

	if err := s.v.Validate(ctx, org); err != nil {
		return nil, err
	}

	opts := repositories.GetOrgByIDOptions{
		OrgID:        org.ID,
		BuID:         org.BusinessUnitID,
		IncludeState: true,
		// IncludeBu:    true,
	}

	original, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		return nil, eris.Wrap(err, "failed to get original organization")
	}

	updatedOrg, err := s.repo.Update(ctx, org)
	if err != nil {
		log.Error().Err(err).Interface("org", org).Msg("failed to update organization")
		return nil, eris.Wrap(err, "failed to update organization")
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceOrganization,
			ResourceID:     org.ID.String(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(org),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
		},
		audit.WithComment("Organization updated"),
		audit.WithDiff(original, org),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log organization update")
	}

	return updatedOrg, nil
}

func (s *Service) SetLogo(
	ctx context.Context,
	orgID, buID, userID pulid.ID,
	logo *multipart.FileHeader,
) (*organization.Organization, error) {
	result, err := s.ps.HasFieldPermission(ctx, &services.PermissionCheck{
		UserID:     userID,
		Resource:   permission.ResourceOrganization,
		Action:     permission.ActionModifyField,
		ResourceID: orgID,
		Field:      "logo_url",
	})
	if err != nil {
		return nil, eris.Wrap(err, "check field permission")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to set the logo for this organization",
		)
	}

	fileData, err := fileutils.ReadFileData(logo)
	if err != nil {
		return nil, eris.Wrap(err, "read file data")
	}

	fileName, err := fileutils.RenameFile(logo, orgID.String())
	if err != nil {
		return nil, eris.Wrap(err, "rename file")
	}

	s.l.Info().Str("fileName", fileName).Msg("file renamed")

	// Get the organization from the DB
	org, err := s.repo.GetByID(ctx, repositories.GetOrgByIDOptions{
		OrgID: orgID,
		BuID:  buID,
	})
	if err != nil {
		return nil, eris.Wrap(err, "get organization")
	}

	if org.BucketName == "" {
		return nil, ErrOrgBucketNameNotSet
	}

	org.Metadata = &organization.Metadata{
		ObjectID: fileName,
	}

	original, err := s.repo.GetByID(ctx, repositories.GetOrgByIDOptions{
		OrgID: org.ID,
		BuID:  org.BusinessUnitID,
	})
	if err != nil {
		s.l.Error().Err(err).Msg("failed to get original organization")
		return nil, eris.Wrap(err, "get original organization")
	}

	updatedOrg, err := s.uploadLogo(ctx, org, userID, &services.SaveFileRequest{
		File:           fileData,
		FileName:       fileName,
		FileExtension:  fileutils.GetFileTypeFromFileName(fileName),
		Classification: services.ClassificationPublic,
		Category:       services.CategoryBranding,
		Tags:           map[string]string{"organization_id": org.ID.String()},
		OrgID:          org.ID.String(),
		BucketName:     org.BucketName,
		UserID:         userID.String(),
		Metadata: http.Header{
			"organization_id": []string{org.ID.String()},
			"user_id":         []string{userID.String()},
			"file_type":       []string{string(fileutils.GetFileTypeFromFileName(fileName))},
		},
	})
	if err != nil {
		return nil, err
	}

	logErr := s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceOrganization,
			ResourceID:     org.ID.String(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(org),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
		},
		audit.WithComment("Organization logo set"),
		audit.WithDiff(original, org),
	)
	if logErr != nil {
		s.l.Error().Err(logErr).Msg("failed to log organization logo set")
	}

	return updatedOrg, nil
}

func (s *Service) uploadLogo(
	ctx context.Context,
	org *organization.Organization,
	userID pulid.ID,
	params *services.SaveFileRequest,
) (*organization.Organization, error) {
	ui, err := s.fs.SaveFile(ctx, params)
	if err != nil {
		return nil, eris.Wrap(err, "save file")
	}

	// Set the logo URL in the organization
	org.LogoURL = ui.Location

	updatedOrg, err := s.repo.SetLogo(ctx, org)
	if err != nil {
		return nil, eris.Wrap(err, "set logo")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceOrganization,
			ResourceID:     org.ID.String(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedOrg),
			PreviousState:  jsonutils.MustToJSON(org),
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
		},
		audit.WithComment("Organization logo set"),
		audit.WithDiff(org, updatedOrg),
	)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to log organization logo set")
	}

	return updatedOrg, nil
}

func (s *Service) ClearLogo(
	ctx context.Context,
	orgID, buID, userID pulid.ID,
) (*organization.Organization, error) {
	result, err := s.ps.HasFieldPermission(ctx, &services.PermissionCheck{
		UserID:     userID,
		Resource:   permission.ResourceOrganization,
		Action:     permission.ActionModifyField,
		ResourceID: orgID,
		Field:      "logo_url",
	})
	if err != nil {
		return nil, eris.Wrap(err, "check field permission")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to clear the logo for this organization",
		)
	}

	original, err := s.repo.GetByID(ctx, repositories.GetOrgByIDOptions{
		OrgID: orgID,
		BuID:  buID,
	})
	if err != nil {
		s.l.Error().Err(err).Msg("failed to get original organization")
		return nil, eris.Wrap(err, "get original organization")
	}

	org, err := s.repo.GetByID(ctx, repositories.GetOrgByIDOptions{
		OrgID: orgID,
		BuID:  buID,
	})
	if err != nil {
		return nil, eris.Wrap(err, "get organization")
	}

	if org.Metadata != nil && org.Metadata.ObjectID != "" {
		err = s.fs.DeleteFile(ctx, org.BucketName, org.Metadata.ObjectID)
		if err != nil {
			s.l.Warn().
				Err(err).
				Str("orgID", orgID.String()).
				Str("bucketName", org.BucketName).
				Str("objectID", org.Metadata.ObjectID).
				Msg("failed to delete file")
			// ! Non-crticial error, continue with clearing the logo
		}
	}

	updatedOrg, err := s.repo.ClearLogo(ctx, org)
	if err != nil {
		s.l.Error().Err(err).Str("orgID", orgID.String()).Msg("failed to clear logo")
		return nil, eris.Wrap(err, "failed to clear logo")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceOrganization,
			ResourceID:     org.ID.String(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(org),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: org.ID,
			BusinessUnitID: org.BusinessUnitID,
		},
		audit.WithComment("Organization logo set"),
		audit.WithDiff(original, org),
	)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to log organization logo set")
	}

	return updatedOrg, nil
}

func (s *Service) GetUserOrganizations(
	ctx context.Context, opts *ports.LimitOffsetQueryOptions,
) (*ports.ListResult[*organization.Organization], error) {
	result, err := s.repo.GetUserOrganizations(ctx, opts)
	if err != nil {
		return nil, eris.Wrap(err, "get user organizations")
	}

	return &ports.ListResult[*organization.Organization]{
		Items: result.Items,
		Total: result.Total,
	}, nil
}
