/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package shipmenttype

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/email"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/shipmenttype"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/audit"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/logger"
	"github.com/emoss08/trenova/internal/pkg/utils/jsonutils"
	"github.com/emoss08/trenova/internal/pkg/validator"
	"github.com/emoss08/trenova/internal/pkg/validator/shipmenttypevalidator"
	"github.com/emoss08/trenova/pkg/types"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/rotisserie/eris"
	"github.com/rs/zerolog"
	"go.uber.org/fx"
)

type ServiceParams struct {
	fx.In

	Logger       *logger.Logger
	Repo         repositories.ShipmentTypeRepository
	PermService  services.PermissionService
	AuditService services.AuditService
	Validator    *shipmenttypevalidator.Validator
	EmailService services.EmailService
}

type Service struct {
	l    *zerolog.Logger
	repo repositories.ShipmentTypeRepository
	ps   services.PermissionService
	as   services.AuditService
	v    *shipmenttypevalidator.Validator
	es   services.EmailService
}

func NewService(p ServiceParams) *Service {
	log := p.Logger.With().
		Str("service", "shipmenttype").
		Logger()

	return &Service{
		l:    &log,
		repo: p.Repo,
		ps:   p.PermService,
		as:   p.AuditService,
		v:    p.Validator,
		es:   p.EmailService,
	}
}

func (s *Service) SelectOptions(
	ctx context.Context,
	req *repositories.ListShipmentTypeRequest,
) ([]*types.SelectOption, error) {
	result, err := s.repo.List(ctx, req)
	if err != nil {
		return nil, eris.Wrap(err, "select shipment types")
	}

	options := make([]*types.SelectOption, 0, len(result.Items))
	for _, st := range result.Items {
		options = append(options, &types.SelectOption{
			Value: st.GetID(),
			Label: st.Code,
			Color: st.Color,
		})
	}

	return options, nil
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListShipmentTypeRequest,
) (*ports.ListResult[*shipmenttype.ShipmentType], error) {
	log := s.l.With().Str("operation", "List").Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         req.Filter.TenantOpts.UserID,
				Resource:       permission.ResourceShipmentType,
				Action:         permission.ActionRead,
				BusinessUnitID: req.Filter.TenantOpts.BuID,
				OrganizationID: req.Filter.TenantOpts.OrgID,
			},
		},
	)
	if err != nil {
		s.l.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read shipment types",
		)
	}

	entities, err := s.repo.List(ctx, req)
	if err != nil {
		log.Error().Err(err).Msg("failed to list shipment types")
		return nil, eris.Wrap(err, "list shipment types")
	}

	return &ports.ListResult[*shipmenttype.ShipmentType]{
		Items: entities.Items,
		Total: entities.Total,
	}, nil
}

func (s *Service) Get(
	ctx context.Context,
	opts repositories.GetShipmentTypeByIDOptions,
) (*shipmenttype.ShipmentType, error) {
	log := s.l.With().
		Str("operation", "GetByID").
		Str("shipmentTypeID", opts.ID.String()).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         opts.UserID,
				Resource:       permission.ResourceShipmentType,
				Action:         permission.ActionRead,
				BusinessUnitID: opts.BuID,
				OrganizationID: opts.OrgID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check read shipment type permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to read this shipment type",
		)
	}

	entity, err := s.repo.GetByID(ctx, opts)
	if err != nil {
		log.Error().Err(err).Msg("failed to get shipment type")
		return nil, eris.Wrap(err, "get shipment type")
	}

	return entity, nil
}

func (s *Service) Create(
	ctx context.Context,
	st *shipmenttype.ShipmentType,
	userID pulid.ID,
) (*shipmenttype.ShipmentType, error) {
	log := s.l.With().
		Str("operation", "Create").
		Str("code", st.Code).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceShipmentType,
				Action:         permission.ActionCreate,
				BusinessUnitID: st.BusinessUnitID,
				OrganizationID: st.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check create shipment type permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to create a shipment type",
		)
	}

	valCtx := &validator.ValidationContext{
		IsCreate: true,
		IsUpdate: false,
	}

	if err := s.v.Validate(ctx, valCtx, st); err != nil {
		return nil, err
	}

	createdEntity, err := s.repo.Create(ctx, st)
	if err != nil {
		return nil, eris.Wrap(err, "create shipment type")
	}

	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentType,
			ResourceID:     createdEntity.GetID(),
			Action:         permission.ActionCreate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(createdEntity),
			OrganizationID: createdEntity.OrganizationID,
			BusinessUnitID: createdEntity.BusinessUnitID,
		},
		audit.WithComment("Shipment Type created"),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment type creation")
	}

	_, err = s.es.SendEmail(ctx, &services.SendEmailRequest{
		OrganizationID: createdEntity.OrganizationID,
		BusinessUnitID: createdEntity.BusinessUnitID,
		UserID:         userID,
		Subject:        "Shipment Type Created",
		To:             []string{"admin@trenova.app"},
		HTMLBody:       "Shipment Type Created",
		TextBody:       "Shipment Type Created",
		Priority:       email.PriorityHigh,
	})
	if err != nil {
		log.Error().Err(err).Msg("failed to send email")
	}

	return createdEntity, nil
}

func (s *Service) Update(
	ctx context.Context,
	st *shipmenttype.ShipmentType,
	userID pulid.ID,
) (*shipmenttype.ShipmentType, error) {
	log := s.l.With().
		Str("operation", "Update").
		Str("code", st.Code).
		Logger()

	result, err := s.ps.HasAnyPermissions(ctx,
		[]*services.PermissionCheck{
			{
				UserID:         userID,
				Resource:       permission.ResourceShipmentType,
				Action:         permission.ActionUpdate,
				BusinessUnitID: st.BusinessUnitID,
				OrganizationID: st.OrganizationID,
			},
		},
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to check permissions")
		return nil, eris.Wrap(err, "check update shipment type permissions")
	}

	if !result.Allowed {
		return nil, errors.NewAuthorizationError(
			"You do not have permission to update this shipment type",
		)
	}

	// Validate the shipment type
	valCtx := &validator.ValidationContext{
		IsUpdate: true,
		IsCreate: false,
	}

	if err := s.v.Validate(ctx, valCtx, st); err != nil {
		return nil, err
	}

	original, err := s.repo.GetByID(ctx, repositories.GetShipmentTypeByIDOptions{
		ID:    st.ID,
		OrgID: st.OrganizationID,
		BuID:  st.BusinessUnitID,
	})
	if err != nil {
		return nil, eris.Wrap(err, "get shipment type")
	}

	updatedEntity, err := s.repo.Update(ctx, st)
	if err != nil {
		log.Error().Err(err).Msg("failed to update shipment type")
		return nil, eris.Wrap(err, "update shipment type")
	}

	// Log the update if the insert was successful
	err = s.as.LogAction(
		&services.LogActionParams{
			Resource:       permission.ResourceShipmentType,
			ResourceID:     updatedEntity.GetID(),
			Action:         permission.ActionUpdate,
			UserID:         userID,
			CurrentState:   jsonutils.MustToJSON(updatedEntity),
			PreviousState:  jsonutils.MustToJSON(original),
			OrganizationID: updatedEntity.OrganizationID,
			BusinessUnitID: updatedEntity.BusinessUnitID,
		},
		audit.WithComment("Shipment Type updated"),
		audit.WithDiff(original, updatedEntity),
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to log shipment type update")
	}

	return updatedEntity, nil
}
