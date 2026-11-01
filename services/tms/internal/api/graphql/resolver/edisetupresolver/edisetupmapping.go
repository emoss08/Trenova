package edisetupresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/edi/app/edisetup"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultPendingConnections = 25
	maxPendingConnections     = 100
)

func pendingConnectionLimit(limit *int) int {
	switch {
	case limit == nil || *limit <= 0:
		return defaultPendingConnections
	case *limit > maxPendingConnections:
		return maxPendingConnections
	default:
		return *limit
	}
}

func partnerFromInput(
	input *gqlmodel.EDIPartnerInput,
	defaultEnabled bool,
) (*edi.EDIPartner, error) {
	entity := &edi.EDIPartner{
		Code:               input.Code,
		Name:               input.Name,
		Description:        gqlscalar.StringValue(input.Description),
		Timezone:           gqlscalar.StringValue(input.Timezone),
		Country:            gqlscalar.StringValue(input.Country),
		ContactName:        gqlscalar.StringValue(input.ContactName),
		ContactEmail:       gqlscalar.StringValue(input.ContactEmail),
		ContactPhone:       gqlscalar.StringValue(input.ContactPhone),
		EnabledForInbound:  boolOr(input.EnabledForInbound, defaultEnabled),
		EnabledForOutbound: boolOr(input.EnabledForOutbound, defaultEnabled),
		Settings:           input.Settings,
		Version:            gqlscalar.Int64Value(input.Version),
	}
	if input.Kind != nil {
		entity.Kind = *input.Kind
	}
	if input.Status != nil {
		entity.Status = *input.Status
	}
	ids := []struct {
		field string
		raw   *string
		dest  *pulid.ID
	}{
		{
			field: "internalOrganizationId",
			raw:   input.InternalOrganizationID,
			dest:  &entity.InternalOrganizationID,
		},
		{field: "ediConnectionId", raw: input.EDIConnectionID, dest: &entity.EDIConnectionID},
		{field: "customerId", raw: input.CustomerID, dest: &entity.CustomerID},
		{
			field: "defaultTransportId",
			raw:   input.DefaultTransportID,
			dest:  &entity.DefaultTransportID,
		},
		{
			field: "defaultMappingProfileId",
			raw:   input.DefaultMappingProfileID,
			dest:  &entity.DefaultMappingProfileID,
		},
	}
	for _, id := range ids {
		parsed, err := gqlid.ParseOptionalIDField(id.field, id.raw)
		if err != nil {
			return nil, err
		}
		*id.dest = parsed
	}

	return entity, nil
}

func connectionRequestFromInput(
	input *gqlmodel.CreateEDIConnectionInput,
) (*edisetup.CreateEDIConnectionRequest, error) {
	targetID, err := gqlid.ParseIDField("targetOrganizationId", input.TargetOrganizationID)
	if err != nil {
		return nil, err
	}
	req := &edisetup.CreateEDIConnectionRequest{
		TargetOrganizationID: targetID,
		Method:               input.Method,
		SourcePartnerConfig:  partnerConfigFromInput(input.SourcePartnerConfig),
		TargetPartnerConfig:  partnerConfigFromInput(input.TargetPartnerConfig),
	}
	if input.Capabilities != nil {
		req.Capabilities = edi.ConnectionCapabilities{
			LoadTenderOutbound: input.Capabilities.LoadTenderOutbound,
			LoadTenderInbound:  input.Capabilities.LoadTenderInbound,
			ShipmentStatus:     input.Capabilities.ShipmentStatus,
			Invoice:            input.Capabilities.Invoice,
		}
	}

	return req, nil
}

func partnerConfigFromInput(
	input *gqlmodel.EDIConnectionPartnerConfigInput,
) edi.ConnectionPartnerConfig {
	if input == nil {
		return edi.ConnectionPartnerConfig{}
	}

	return edi.ConnectionPartnerConfig{
		Code:               input.Code,
		Name:               input.Name,
		Description:        gqlscalar.StringValue(input.Description),
		ContactName:        gqlscalar.StringValue(input.ContactName),
		ContactEmail:       gqlscalar.StringValue(input.ContactEmail),
		ContactPhone:       gqlscalar.StringValue(input.ContactPhone),
		EnabledForInbound:  input.EnabledForInbound,
		EnabledForOutbound: input.EnabledForOutbound,
		Settings:           input.Settings,
	}
}

func communicationProfileRequestFromInput(
	input *gqlmodel.EDICommunicationProfileInput,
) (*edisetup.UpsertEDICommunicationProfileRequest, error) {
	connectionID, err := gqlid.ParseOptionalIDField("ediConnectionId", input.EDIConnectionID)
	if err != nil {
		return nil, err
	}
	partnerID, err := gqlid.ParseOptionalIDField("ediPartnerId", input.EDIPartnerID)
	if err != nil {
		return nil, err
	}
	secrets, err := secretsFromInput(input.Secrets)
	if err != nil {
		return nil, err
	}
	req := &edisetup.UpsertEDICommunicationProfileRequest{
		EDIConnectionID: connectionID,
		EDIPartnerID:    partnerID,
		Method:          input.Method,
		Name:            input.Name,
		Description:     gqlscalar.StringValue(input.Description),
		Config:          input.Config,
		Secrets:         secrets,
		Version:         gqlscalar.Int64Value(input.Version),
	}
	if input.Status != nil {
		req.Status = string(*input.Status)
	}

	return req, nil
}

func secretsFromInput(raw map[string]any) (map[string]string, error) {
	if raw == nil {
		return nil, nil
	}
	secrets := make(map[string]string, len(raw))
	for key, value := range raw {
		switch typed := value.(type) {
		case string:
			secrets[key] = typed
		case nil:
			secrets[key] = ""
		default:
			return nil, errortypes.NewValidationError(
				"secrets",
				errortypes.ErrInvalid,
				"Secret values must be text",
			)
		}
	}

	return secrets, nil
}

func boolOr(value *bool, fallback bool) bool {
	if value == nil {
		return fallback
	}

	return *value
}
