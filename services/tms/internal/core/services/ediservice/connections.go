//nolint:gocritic // EDI service value params are kept stable across handler and test contracts.
package ediservice

import (
	"context"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

func (s *Service) ListConnections(
	ctx context.Context,
	req *repositories.ListEDIConnectionsRequest,
) (*pagination.ListResult[*edi.EDIConnection], error) {
	return s.connectionRepo.ListConnections(ctx, req)
}

func (s *Service) GetConnection(
	ctx context.Context,
	req repositories.GetEDIConnectionByIDRequest,
) (*edi.EDIConnection, error) {
	return s.connectionRepo.GetConnectionByID(ctx, req)
}

func (s *Service) CreateConnection(
	ctx context.Context,
	req *CreateEDIConnectionRequest,
	actor *services.RequestActor,
) (*edi.EDIConnection, error) {
	if req == nil {
		return nil, errortypes.NewValidationError(
			"",
			errortypes.ErrRequired,
			"EDI connection request is required",
		)
	}

	method := req.Method
	if method == "" {
		method = edi.ConnectionMethodInternal
	}
	entity := &edi.EDIConnection{
		BusinessUnitID:       req.TenantInfo.BuID,
		SourceOrganizationID: req.TenantInfo.OrgID,
		TargetOrganizationID: req.TargetOrganizationID,
		Method:               method,
		Status:               edi.ConnectionStatusPendingAcceptance,
		Capabilities:         normalizeConnectionCapabilities(req.Capabilities),
		SourcePartnerConfig:  normalizePartnerConfig(req.SourcePartnerConfig),
		TargetPartnerConfig:  normalizePartnerConfig(req.TargetPartnerConfig),
		RequestedByID:        req.TenantInfo.UserID,
	}
	if multiErr := s.validator.ValidateConnection(entity); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.connectionRepo.CreateConnection(ctx, entity)
	if err != nil {
		return nil, mapEDIConnectionConstraint(err)
	}

	s.logAction(created, actor, permission.OpCreate, nil, created, "EDI connection requested")
	return created, nil
}

func (s *Service) AcceptConnection(
	ctx context.Context,
	req *EDIConnectionActionRequest,
	actor *services.RequestActor,
) (*edi.EDIConnection, error) {
	connection, err := s.connectionRepo.GetConnectionForUpdate(
		ctx,
		repositories.GetEDIConnectionForUpdateRequest{
			ID:         req.ConnectionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}
	if connection.TargetOrganizationID != req.TenantInfo.OrgID {
		return nil, errortypes.NewValidationError(
			"connectionId",
			errortypes.ErrInvalidOperation,
			"Only the target organization can accept this EDI connection",
		)
	}
	if connection.Status != edi.ConnectionStatusPendingAcceptance {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalidOperation,
			"Only pending EDI connections can be accepted",
		)
	}
	if connection.Method != edi.ConnectionMethodInternal {
		return nil, errortypes.NewBusinessError(
			"external EDI connections can be configured but are not accepted through invite flow",
		)
	}

	now := timeutils.NowUnix()
	original := *connection
	connection.Status = edi.ConnectionStatusActive
	connection.AcceptedByID = actor.UserID
	connection.AcceptedAt = &now

	sourcePartner := buildConnectionPartner(
		connection,
		connection.SourceOrganizationID,
		connection.TargetOrganizationID,
		connection.SourcePartnerConfig,
	)
	targetPartner := buildConnectionPartner(
		connection,
		connection.TargetOrganizationID,
		connection.SourceOrganizationID,
		connection.TargetPartnerConfig,
	)
	sourceProfile := buildInternalProfile(connection, sourcePartner.OrganizationID, sourcePartner)
	targetProfile := buildInternalProfile(connection, targetPartner.OrganizationID, targetPartner)

	accepted, err := s.connectionRepo.AcceptInternalConnection(
		ctx,
		&repositories.CreateInternalEDIConnectionAcceptanceRequest{
			Connection:    connection,
			SourcePartner: sourcePartner,
			TargetPartner: targetPartner,
			SourceProfile: sourceProfile,
			TargetProfile: targetProfile,
			TenantInfo:    req.TenantInfo,
		},
	)
	if err != nil {
		return nil, mapEDIPartnerConstraint(mapEDIConnectionConstraint(err))
	}

	s.logAction(
		accepted,
		actor,
		permission.OpUpdate,
		&original,
		accepted,
		"EDI connection accepted",
	)
	return accepted, nil
}

func (s *Service) RejectConnection(
	ctx context.Context,
	req *EDIConnectionActionRequest,
	actor *services.RequestActor,
) (*edi.EDIConnection, error) {
	reason := strings.TrimSpace(req.Reason)
	if reason == "" {
		return nil, errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Rejection reason is required",
		)
	}

	return s.transitionConnection(
		ctx,
		req,
		actor,
		edi.ConnectionStatusRejected,
		func(connection *edi.EDIConnection, now int64) error {
			if connection.TargetOrganizationID != req.TenantInfo.OrgID {
				return errortypes.NewValidationError(
					"connectionId",
					errortypes.ErrInvalidOperation,
					"Only the target organization can reject this EDI connection",
				)
			}
			if connection.Status != edi.ConnectionStatusPendingAcceptance {
				return errortypes.NewValidationError(
					"status",
					errortypes.ErrInvalidOperation,
					"Only pending EDI connections can be rejected",
				)
			}
			connection.RejectedByID = actor.UserID
			connection.RejectedAt = &now
			connection.RejectionReason = reason
			return nil
		},
		"EDI connection rejected",
	)
}

func (s *Service) SuspendConnection(
	ctx context.Context,
	req *EDIConnectionActionRequest,
	actor *services.RequestActor,
) (*edi.EDIConnection, error) {
	return s.transitionConnection(
		ctx,
		req,
		actor,
		edi.ConnectionStatusSuspended,
		func(connection *edi.EDIConnection, now int64) error {
			if connection.Status != edi.ConnectionStatusActive {
				return errortypes.NewValidationError(
					"status",
					errortypes.ErrInvalidOperation,
					"Only active EDI connections can be suspended",
				)
			}
			connection.SuspendedByID = actor.UserID
			connection.SuspendedAt = &now
			return nil
		},
		"EDI connection suspended",
	)
}

func (s *Service) RevokeConnection(
	ctx context.Context,
	req *EDIConnectionActionRequest,
	actor *services.RequestActor,
) (*edi.EDIConnection, error) {
	return s.transitionConnection(
		ctx,
		req,
		actor,
		edi.ConnectionStatusRevoked,
		func(connection *edi.EDIConnection, now int64) error {
			if connection.Status == edi.ConnectionStatusRejected ||
				connection.Status == edi.ConnectionStatusRevoked {
				return errortypes.NewValidationError(
					"status",
					errortypes.ErrInvalidOperation,
					"EDI connection cannot be revoked from its current status",
				)
			}
			connection.RevokedByID = actor.UserID
			connection.RevokedAt = &now
			return nil
		},
		"EDI connection revoked",
	)
}

func (s *Service) transitionConnection(
	ctx context.Context,
	req *EDIConnectionActionRequest,
	actor *services.RequestActor,
	status edi.ConnectionStatus,
	mutate func(connection *edi.EDIConnection, now int64) error,
	comment string,
) (*edi.EDIConnection, error) {
	connection, err := s.connectionRepo.GetConnectionForUpdate(
		ctx,
		repositories.GetEDIConnectionForUpdateRequest{
			ID:         req.ConnectionID,
			TenantInfo: req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	now := timeutils.NowUnix()
	original := *connection
	if err = mutate(connection, now); err != nil {
		return nil, err
	}
	connection.Status = status

	updated, err := s.connectionRepo.UpdateConnection(ctx, connection)
	if err != nil {
		return nil, mapEDIConnectionConstraint(err)
	}

	s.logAction(updated, actor, permission.OpUpdate, &original, updated, comment)
	return updated, nil
}

func (s *Service) CreateInternalPartnerPairViaConnection(
	ctx context.Context,
	req *CreateInternalPartnerPairRequest,
	actor *services.RequestActor,
) (*edi.InternalPartnerPair, error) {
	connection, err := s.CreateConnection(
		ctx,
		&CreateEDIConnectionRequest{
			TenantInfo:           req.TenantInfo,
			TargetOrganizationID: req.TargetOrganizationID,
			Method:               edi.ConnectionMethodInternal,
			Capabilities: edi.ConnectionCapabilities{
				LoadTenderOutbound: true,
				LoadTenderInbound:  true,
			},
			SourcePartnerConfig: edi.ConnectionPartnerConfig{
				Code:               req.SourceCode,
				Name:               req.SourceName,
				Description:        req.SourceDescription,
				ContactName:        req.SourceContactName,
				ContactEmail:       req.SourceContactEmail,
				ContactPhone:       req.SourceContactPhone,
				EnabledForInbound:  req.SourceEnabledInbound,
				EnabledForOutbound: req.SourceEnabledOutbound,
				Settings:           req.SourceSettings,
			},
			TargetPartnerConfig: edi.ConnectionPartnerConfig{
				Code:               req.TargetCode,
				Name:               req.TargetName,
				Description:        req.TargetDescription,
				ContactName:        req.TargetContactName,
				ContactEmail:       req.TargetContactEmail,
				ContactPhone:       req.TargetContactPhone,
				EnabledForInbound:  req.TargetEnabledInbound,
				EnabledForOutbound: req.TargetEnabledOutbound,
				Settings:           req.TargetSettings,
			},
		},
		actor,
	)
	if err != nil {
		return nil, err
	}

	accepted, err := s.AcceptConnection(
		ctx,
		&EDIConnectionActionRequest{
			TenantInfo: pagination.TenantInfo{
				OrgID:  connection.TargetOrganizationID,
				BuID:   connection.BusinessUnitID,
				UserID: actor.UserID,
			},
			ConnectionID: connection.ID,
		},
		actor,
	)
	if err != nil {
		return nil, err
	}

	return &edi.InternalPartnerPair{
		SourcePartner: accepted.SourcePartner,
		TargetPartner: accepted.TargetPartner,
	}, nil
}

func normalizeConnectionCapabilities(
	capabilities edi.ConnectionCapabilities,
) edi.ConnectionCapabilities {
	if capabilities == (edi.ConnectionCapabilities{}) {
		return edi.ConnectionCapabilities{
			LoadTenderOutbound: true,
			LoadTenderInbound:  true,
		}
	}
	return capabilities
}

func normalizePartnerConfig(config edi.ConnectionPartnerConfig) edi.ConnectionPartnerConfig {
	config.Code = strings.TrimSpace(config.Code)
	config.Name = strings.TrimSpace(config.Name)
	config.Description = strings.TrimSpace(config.Description)
	config.ContactName = strings.TrimSpace(config.ContactName)
	config.ContactEmail = strings.TrimSpace(config.ContactEmail)
	config.ContactPhone = strings.TrimSpace(config.ContactPhone)
	if !config.EnabledForInbound && !config.EnabledForOutbound {
		config.EnabledForInbound = true
		config.EnabledForOutbound = true
	}
	if config.Settings == nil {
		config.Settings = map[string]any{}
	}
	return config
}

func buildConnectionPartner(
	connection *edi.EDIConnection,
	organizationID pulid.ID,
	internalOrganizationID pulid.ID,
	config edi.ConnectionPartnerConfig,
) *edi.EDIPartner {
	return &edi.EDIPartner{
		BusinessUnitID:         connection.BusinessUnitID,
		OrganizationID:         organizationID,
		Kind:                   edi.PartnerKindInternal,
		Status:                 domaintypes.StatusActive,
		Code:                   config.Code,
		Name:                   config.Name,
		Description:            config.Description,
		InternalOrganizationID: internalOrganizationID,
		EDIConnectionID:        connection.ID,
		Country:                "US",
		ContactName:            config.ContactName,
		ContactEmail:           config.ContactEmail,
		ContactPhone:           config.ContactPhone,
		EnabledForInbound:      config.EnabledForInbound,
		EnabledForOutbound:     config.EnabledForOutbound,
		Settings:               config.Settings,
	}
}

func buildInternalProfile(
	connection *edi.EDIConnection,
	organizationID pulid.ID,
	partner *edi.EDIPartner,
) *edi.EDICommunicationProfile {
	return &edi.EDICommunicationProfile{
		BusinessUnitID:  connection.BusinessUnitID,
		OrganizationID:  organizationID,
		EDIConnectionID: connection.ID,
		EDIPartnerID:    partner.ID,
		Method:          edi.ConnectionMethodInternal,
		Status:          domaintypes.StatusActive,
		Name:            partner.Name + " Internal EDI",
		Config: map[string]any{
			"connectedOrganizationId": partner.InternalOrganizationID.String(),
		},
		EncryptedSecrets: map[string]string{},
	}
}

func profileWithSecretState(
	profile *edi.EDICommunicationProfile,
) *edi.EDICommunicationProfile {
	if profile == nil {
		return nil
	}

	keys := make([]string, 0, len(profile.EncryptedSecrets))
	for key, value := range profile.EncryptedSecrets {
		if strings.TrimSpace(value) != "" {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)

	profile.SecretState = make([]edi.CommunicationProfileSecretState, 0, len(keys))
	for _, key := range keys {
		profile.SecretState = append(
			profile.SecretState,
			edi.CommunicationProfileSecretState{Key: key, HasValue: true},
		)
	}
	profile.EncryptedSecrets = nil
	return profile
}
