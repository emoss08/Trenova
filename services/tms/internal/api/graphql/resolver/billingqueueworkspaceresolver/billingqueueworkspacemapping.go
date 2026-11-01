package billingqueueworkspaceresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingapprovalrun"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueuefilterpreset"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
)

func positiveInt64(value int64) *int {
	if value <= 0 {
		return nil
	}
	converted := int(value)

	return &converted
}

func runRequest(
	id string,
	authCtx *authctx.AuthContext,
) (*billingqueueports.BillingQueueApprovalRunRequest, error) {
	runID, err := gqlid.ParseIDField("id", id)
	if err != nil {
		return nil, err
	}

	return &billingqueueports.BillingQueueApprovalRunRequest{
		TenantInfo: pagination.FromAuthAsUser(authCtx),
		RunID:      runID,
	}, nil
}

func (r *MutationResolver) requireBulkPermission(
	ctx context.Context,
	authCtx *authctx.AuthContext,
	action billingapprovalrun.RunAction,
) error {
	var (
		resource  permission.Resource
		operation permission.Operation
	)
	switch action {
	case billingapprovalrun.RunActionPost:
		resource, operation = permission.ResourceInvoice, permission.OpPost
	case billingapprovalrun.RunActionAssign:
		resource, operation = permission.ResourceBillingQueue, permission.OpAssign
	case billingapprovalrun.RunActionApprove,
		billingapprovalrun.RunActionHold,
		billingapprovalrun.RunActionCancel,
		billingapprovalrun.RunActionRequestPaperwork,
		billingapprovalrun.RunActionResolveIssue:
		return nil
	default:
		return nil
	}

	if !r.HasPermission(ctx, authCtx, resource, operation) {
		return errortypes.NewAuthorizationError("You don't have permission for this bulk action")
	}

	return nil
}

func startBulkRequest(
	input *gqlmodel.StartBillingQueueBulkRunInput,
	tenantInfo pagination.TenantInfo,
) (*billingqueueports.StartBillingQueueApprovalRequest, error) {
	itemIDs, err := gqlid.ParseIDListField("itemIds", input.ItemIds)
	if err != nil {
		return nil, err
	}
	params, err := bulkParams(input.Params)
	if err != nil {
		return nil, err
	}

	return &billingqueueports.StartBillingQueueApprovalRequest{
		TenantInfo:     tenantInfo,
		ItemIDs:        itemIDs,
		IdempotencyKey: input.IdempotencyKey,
		Action:         input.Action,
		Params:         params,
	}, nil
}

func bulkParams(
	input *gqlmodel.BillingQueueBulkRunParamsInput,
) (billingapprovalrun.RunParams, error) {
	if input == nil {
		return billingapprovalrun.RunParams{}, nil
	}

	billerID, err := gqlid.OptionalID(input.BillerID)
	if err != nil {
		return billingapprovalrun.RunParams{}, gqlid.InvalidIDError(
			"params.billerId",
			"Invalid biller",
		)
	}
	params := billingapprovalrun.RunParams{
		AssignApprover: gqlscalar.BoolValue(input.AssignApprover),
		BillerID:       billerID,
		Reason:         gqlscalar.StringValue(input.Reason),
		IssueCode:      gqlscalar.StringValue(input.IssueCode),
		OptionKey:      gqlscalar.StringValue(input.OptionKey),
	}
	if input.HoldReasonCode != nil {
		params.HoldReasonCode = string(*input.HoldReasonCode)
	}

	return params, nil
}

func filterPresetFromInput(
	input *gqlmodel.BillingQueueFilterPresetInput,
	authCtx *authctx.AuthContext,
) (*billingqueuefilterpreset.BillingQueueFilterPreset, error) {
	entity := &billingqueuefilterpreset.BillingQueueFilterPreset{
		OrganizationID: authCtx.OrganizationID,
		BusinessUnitID: authCtx.BusinessUnitID,
		UserID:         authCtx.UserID,
		Name:           input.Name,
		Filters:        input.Filters,
		IsDefault:      gqlscalar.BoolValue(input.IsDefault),
	}

	multiErr := errortypes.NewMultiError()
	entity.Validate(multiErr)
	if multiErr.HasErrors() {
		return nil, multiErr
	}

	return entity, nil
}
