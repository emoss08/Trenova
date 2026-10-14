package factoringresolver

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/loaders"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/base"
	"github.com/emoss08/trenova/internal/modules/billing/domain/factoring"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

func (d *Deps) factorFor(
	ctx context.Context,
	id pulid.ID,
) (*factoringcompany.FactoringCompany, error) {
	if id.IsNil() || !loaders.Available(ctx) {
		return nil, nil //nolint:nilnil // an unset factor resolves to null
	}
	authCtx, err := d.RequireAuth(ctx)
	if err != nil {
		return nil, err
	}
	if !d.HasPermission(ctx, authCtx, permission.ResourceFactoringCompany, permission.OpRead) {
		return nil, nil //nolint:nilnil // a factor the caller cannot read resolves to null
	}
	return loaders.FactoringCompanyByID(ctx, id.String())
}

func agreementInputFromModel(
	in *gqlmodel.FactoringAgreementInput,
) (*factoringports.AgreementInput, error) {
	if in == nil {
		return nil, errortypes.NewValidationError(
			"agreement",
			errortypes.ErrRequired,
			"Agreement terms are required",
		)
	}
	factorID, err := parseID(in.FactoringCompanyID, "factoringCompanyId")
	if err != nil {
		return nil, err
	}
	typeIDs, err := parseIDs(in.DocumentTypeIds, "documentTypeIds")
	if err != nil {
		return nil, err
	}
	tiers := make([]*factoring.FeeTier, 0, len(in.FeeTiers))
	for _, tier := range in.FeeTiers {
		if tier == nil {
			continue
		}
		tiers = append(tiers, &factoring.FeeTier{
			DaysFrom:  tier.DaysFrom,
			DaysTo:    tier.DaysTo,
			RateBP:    tier.RateBp,
			FlatMinor: int64(intOrZero(tier.FlatMinor)),
		})
	}
	return &factoringports.AgreementInput{
		FactoringCompanyID:   factorID,
		EffectiveFrom:        int64(in.EffectiveFrom),
		EffectiveTo:          int64Ptr(in.EffectiveTo),
		AdvanceRateBP:        in.AdvanceRateBp,
		ReserveRateBP:        in.ReserveRateBp,
		MinimumFeeMinor:      int64(intOrZero(in.MinimumFeeMinor)),
		ChargebackWindowDays: in.ChargebackWindowDays,
		Recourse:             in.Recourse,
		AccountingTreatment:  in.AccountingTreatment,
		FundingDays:          in.FundingDays,
		SubmissionTime:       in.SubmissionTime,
		Notes:                stringutils.FromPtr(in.Notes),
		FeeTiers:             tiers,
		DocumentTypeIDs:      typeIDs,
	}, nil
}

func statusChangeFromModel(
	in *gqlmodel.ChangeFactoringAgreementStatusInput,
	authCtx *authctx.AuthContext,
) (*factoringports.ChangeAgreementStatusRequest, error) {
	id, err := parseID(in.ID, "id")
	if err != nil {
		return nil, err
	}
	return &factoringports.ChangeAgreementStatusRequest{
		TenantInfo: base.TenantInfo(authCtx),
		ID:         id,
		Version:    int64(in.Version),
		EndAt:      int64Ptr(in.EndAt),
	}, nil
}

func noaConnectionToModel(
	result *pagination.CursorListResult[*factoring.Noa],
) (*gqlmodel.FactoringNoaConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *factoring.Noa, cursor string) *gqlmodel.FactoringNoaEdge {
			return &gqlmodel.FactoringNoaEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.FactoringNoaEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.FactoringNoaConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func invoiceStateConnectionToModel(
	result *pagination.CursorListResult[*factoring.InvoiceState],
) (*gqlmodel.FactoringInvoiceStateConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *factoring.InvoiceState, cursor string) *gqlmodel.FactoringInvoiceStateEdge {
			return &gqlmodel.FactoringInvoiceStateEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.FactoringInvoiceStateEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.FactoringInvoiceStateConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func submissionConnectionToModel(
	result *pagination.CursorListResult[*factoring.Submission],
) (*gqlmodel.FactoringSubmissionConnection, error) {
	page, err := base.EntityCursorConnection(
		result,
		func(node *factoring.Submission, cursor string) *gqlmodel.FactoringSubmissionEdge {
			return &gqlmodel.FactoringSubmissionEdge{Node: node, Cursor: cursor}
		},
		func(edge *gqlmodel.FactoringSubmissionEdge) string { return edge.Cursor },
	)
	if err != nil {
		return nil, err
	}
	return &gqlmodel.FactoringSubmissionConnection{
		Edges:      page.Edges,
		PageInfo:   page.PageInfo,
		TotalCount: page.TotalCount,
	}, nil
}

func planRequestFromModel(
	in *gqlmodel.FactoringSubmissionPlanInput,
	authCtx *authctx.AuthContext,
) (*factoringports.PlanSubmissionRequest, error) {
	factorID, err := parseID(in.FactoringCompanyID, "factoringCompanyId")
	if err != nil {
		return nil, err
	}
	invoiceIDs, err := parseIDs(in.InvoiceIds, "invoiceIds")
	if err != nil {
		return nil, err
	}
	return &factoringports.PlanSubmissionRequest{
		TenantInfo:         base.TenantInfo(authCtx),
		FactoringCompanyID: factorID,
		InvoiceIDs:         invoiceIDs,
	}, nil
}

func transportRequestFromModel(
	in *gqlmodel.SaveFactoringTransportInput,
	authCtx *authctx.AuthContext,
) (*factoringports.SaveTransportRequest, error) {
	factorID, err := parseID(in.FactoringCompanyID, "factoringCompanyId")
	if err != nil {
		return nil, err
	}
	req := &factoringports.SaveTransportRequest{
		TenantInfo:         base.TenantInfo(authCtx),
		FactoringCompanyID: factorID,
		Version:            int64Ptr(in.Version),
		Input: factoringports.TransportInput{
			Method:           in.Method,
			EmailRecipients:  in.EmailRecipients,
			AttachPackets:    in.AttachPackets == nil || *in.AttachPackets,
			SftpHost:         stringutils.FromPtr(in.SftpHost),
			SftpPort:         intOrZero(in.SftpPort),
			SftpUsername:     stringutils.FromPtr(in.SftpUsername),
			SftpSecret:       in.SftpSecret,
			SftpKnownHostKey: stringutils.FromPtr(in.SftpKnownHostKey),
			SftpDirectory:    stringutils.FromPtr(in.SftpDirectory),
		},
	}
	if in.SftpAuthMethod != nil {
		req.Input.SftpAuthMethod = *in.SftpAuthMethod
	}
	return req, nil
}

func parseID(value, field string) (pulid.ID, error) {
	id, err := pulid.MustParse(value)
	if err != nil || id.IsNil() {
		return pulid.Nil, errortypes.NewValidationError(field, errortypes.ErrInvalid, "Invalid ID")
	}
	return id, nil
}

func parseOptionalID(value *string, field string) (pulid.ID, error) {
	if value == nil || *value == "" {
		return pulid.Nil, nil
	}
	return parseID(*value, field)
}

func parseIDs(values []string, field string) ([]pulid.ID, error) {
	ids := make([]pulid.ID, 0, len(values))
	for _, value := range values {
		id, err := parseID(value, field)
		if err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, nil
}

func idOrNil(id pulid.ID) *string {
	if id.IsNil() {
		return nil
	}
	value := id.String()
	return &value
}

func int64Ptr(value *int) *int64 {
	if value == nil {
		return nil
	}
	out := int64(*value)
	return &out
}

func intOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func permissionDenied(resource permission.Resource, op permission.Operation) error {
	return errortypes.NewAuthorizationError(
		"You don't have permission to perform this action: {0} {1}", resource, op,
	)
}
