package rateresolver

import (
	"context"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/rating/app/rateagreementservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratematrixservice"
	"github.com/emoss08/trenova/internal/modules/rating/app/ratequoteservice"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const defaultVersionPage = 20

func boundedFirst(first *int, fallback int) int {
	if first == nil {
		return fallback
	}

	return min(max(*first, 1), pagination.MaxLimit)
}

func fieldRequested(ctx context.Context, path string) bool {
	if graphql.GetFieldContext(ctx) == nil || !graphql.HasOperationContext(ctx) {
		return false
	}

	return graphql.FieldRequested(ctx, path)
}

type agreementReview struct {
	operation permission.Operation
	action    rateagreementservice.ReviewAction
	id        string
	comment   *string
}

func (d *Deps) reviewRateAgreement(
	ctx context.Context,
	review *agreementReview,
) (*gqlmodel.RateAgreement, error) {
	authCtx, err := d.RequirePermission(ctx, permission.ResourceRateAgreement, review.operation)
	if err != nil {
		return nil, err
	}

	agreementID, err := gqlid.ParseIDField("id", review.id)
	if err != nil {
		return nil, err
	}

	updated, err := review.action(ctx, &rateagreementservice.ApprovalActionRequest{
		TenantInfo: resolvercore.TenantInfo(authCtx),
		EntityID:   agreementID,
		Comment:    gqlscalar.StringValue(review.comment),
	})
	if err != nil {
		return nil, err
	}

	return rateAgreementCardToModel(updated, nil, false)
}

type matrixDecision struct {
	authCtx       *authctx.AuthContext
	action        func(context.Context, *ratematrixservice.ApprovalActionRequest) (*ratematrix.RateMatrix, error)
	id            string
	comment       *string
	mayApproveOwn bool
}

func (d *Deps) decideRateMatrix(
	ctx context.Context,
	decision *matrixDecision,
) (*gqlmodel.RateMatrix, error) {
	matrixID, err := gqlid.ParseIDField("id", decision.id)
	if err != nil {
		return nil, err
	}

	updated, err := decision.action(ctx, &ratematrixservice.ApprovalActionRequest{
		TenantInfo:    resolvercore.TenantInfo(decision.authCtx),
		EntityID:      matrixID,
		Comment:       gqlscalar.StringValue(decision.comment),
		MayApproveOwn: decision.mayApproveOwn,
	})
	if err != nil {
		return nil, err
	}

	return rateMatrixCardToModel(updated), nil
}

func submitRateMatrixRequest(
	matrixID pulid.ID,
	input *gqlmodel.SubmitRateMatrixInput,
	tenant pagination.TenantInfo,
) (*ratematrixservice.SubmitRequest, error) {
	req := &ratematrixservice.SubmitRequest{
		ApprovalActionRequest: ratematrixservice.ApprovalActionRequest{
			TenantInfo: tenant,
			EntityID:   matrixID,
		},
		ReviewerID: pulid.Nil,
	}
	if input == nil {
		return req, nil
	}
	req.Comment = gqlscalar.StringValue(input.Comment)
	req.ReviewerNote = gqlscalar.StringValue(input.ReviewerNote)
	if raw := strings.TrimSpace(gqlscalar.StringValue(input.ReviewerID)); raw != "" {
		reviewerID, err := pulid.Parse(raw)
		if err != nil {
			return nil, errortypes.NewValidationError(
				"reviewerId",
				errortypes.ErrInvalid,
				"Choose an active reviewer, other than you, who can publish rate matrices",
			)
		}
		req.ReviewerID = reviewerID
	}

	return req, nil
}

func rateIncreaseRequestFromInput(
	input *gqlmodel.RateIncreaseInput,
	tenant pagination.TenantInfo,
) (*rateagreementservice.RateIncreaseRequest, error) {
	agreementIDs, err := idList("agreementIds", input.AgreementIds)
	if err != nil {
		return nil, err
	}
	customerID, err := gqlid.PulidPtrFromOptionalString(input.CustomerID)
	if err != nil {
		return nil, err
	}
	carrierID, err := gqlid.PulidPtrFromOptionalString(input.CarrierID)
	if err != nil {
		return nil, err
	}
	var percent *string
	if input.Adjustment != nil {
		percent = input.Adjustment.PercentChange
	}
	percentChange, err := gqldecimal.ParseNullDecimalField("adjustment.percentChange", percent)
	if err != nil {
		return nil, err
	}

	req := &rateagreementservice.RateIncreaseRequest{
		TenantInfo:    tenant,
		AgreementIDs:  agreementIDs,
		CustomerID:    customerID,
		CarrierID:     carrierID,
		Adjustment:    rateagreementservice.RateAdjustment{PercentChange: percentChange},
		EffectiveFrom: int64(input.EffectiveFrom),
	}
	if input.PartyType != nil {
		req.PartyType = rateagreement.PartyType(*input.PartyType)
	}

	return req, nil
}

func shopRequestFromInput(
	shipmentID string,
	input *gqlmodel.ShopShipmentCarriersInput,
	tenant pagination.TenantInfo,
) (*ratequoteservice.ShopRequest, error) {
	id, err := gqlid.ParseIDField("shipmentId", shipmentID)
	if err != nil {
		return nil, err
	}
	req := &ratequoteservice.ShopRequest{TenantInfo: tenant, ShipmentID: id}
	if input == nil {
		return req, nil
	}
	carrierIDs, err := idList("carrierIds", input.CarrierIds)
	if err != nil {
		return nil, err
	}
	if input.Strategy != nil {
		req.Strategy = *input.Strategy
	}
	req.CarrierIDs = carrierIDs
	req.AsOf = gqlscalar.Int64Value(input.AsOf)
	req.Limit = gqlscalar.IntValue(input.Limit)

	return req, nil
}
