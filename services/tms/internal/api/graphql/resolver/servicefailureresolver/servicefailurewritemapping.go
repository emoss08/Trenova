package servicefailureresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/ports/servicefailureports"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (r *MutationResolver) currentFailure(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	id string,
) (*servicefailure.ServiceFailure, error) {
	failureID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	return r.ServiceFailureSvc.GetByID(ctx, &servicefailureports.GetServiceFailureByIDRequest{
		ID:         failureID,
		TenantInfo: resolvercore.TenantInfo(authCtx),
	})
}

func (r *MutationResolver) lifecycleRequest(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	id string,
	input *gqlmodel.ServiceFailureLifecycleInput,
) (*servicefailureports.ServiceFailureLifecycleRequest, error) {
	current, err := r.currentFailure(ctx, authCtx, id)
	if err != nil {
		return nil, err
	}

	return lifecycleRequestFromInput(current, input, resolvercore.TenantInfo(authCtx))
}

func lifecycleRequestFromInput(
	current *servicefailure.ServiceFailure,
	input *gqlmodel.ServiceFailureLifecycleInput,
	tenantInfo pagination.TenantInfo,
) (*servicefailureports.ServiceFailureLifecycleRequest, error) {
	reasonCodeID, err := gqlid.ParseOptionalIDField("reasonCodeId", input.ReasonCodeID)
	if err != nil {
		return nil, err
	}

	return &servicefailureports.ServiceFailureLifecycleRequest{
		TenantInfo:   tenantInfo,
		ID:           current.ID,
		ShipmentID:   current.ShipmentID,
		ReasonCodeID: reasonCodeID,
		Notes:        gqlscalar.StringValue(input.Notes),
		Version:      int64(input.Version),
	}, nil
}

func updateRequestFromInput(
	current *servicefailure.ServiceFailure,
	input *gqlmodel.UpdateServiceFailureInput,
	tenantInfo pagination.TenantInfo,
) (*servicefailureports.UpdateServiceFailureRequest, error) {
	reasonCodeID, err := gqlid.ParseOptionalIDField("reasonCodeId", input.ReasonCodeID)
	if err != nil {
		return nil, err
	}

	return &servicefailureports.UpdateServiceFailureRequest{
		TenantInfo:            tenantInfo,
		ID:                    current.ID,
		ShipmentID:            current.ShipmentID,
		ReasonCodeID:          reasonCodeID,
		ClearReasonCode:       gqlscalar.BoolValue(input.ClearReasonCode),
		Notes:                 gqlscalar.StringValue(input.Notes),
		InternalNotes:         gqlscalar.StringValue(input.InternalNotes),
		X12StatusCodeOverride: gqlscalar.StringValue(input.X12StatusCodeOverride),
		X12ReasonCodeOverride: gqlscalar.StringValue(input.X12ReasonCodeOverride),
		X12ExceptionCode:      gqlscalar.StringValue(input.X12ExceptionCode),
		Version:               int64(input.Version),
	}, nil
}

func edi214ReadinessTrigger(
	trigger *servicefailureports.ServiceFailureEDITrigger,
	status servicefailure.Status,
) (servicefailureports.ServiceFailureEDITrigger, servicefailure.Status) {
	switch {
	case trigger != nil && *trigger == servicefailureports.ServiceFailureEDITriggerResolved:
		return servicefailureports.ServiceFailureEDITriggerResolved, servicefailure.StatusResolved
	case trigger != nil && *trigger == servicefailureports.ServiceFailureEDITriggerReviewed:
		return servicefailureports.ServiceFailureEDITriggerReviewed, servicefailure.StatusReviewed
	case status == servicefailure.StatusOpen:
		return servicefailureports.ServiceFailureEDITriggerReviewed, servicefailure.StatusReviewed
	default:
		return servicefailureports.ServiceFailureEDITriggerResolved, servicefailure.StatusResolved
	}
}
