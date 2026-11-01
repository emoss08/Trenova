package documentresolver

import (
	"strings"

	"github.com/emoss08/trenova/internal/api/actorutil"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqldecimal"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/modules/documents/app/documentservice"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
)

const attachResourceType = "shipment"

func searchQuery(raw *string) string {
	if raw == nil {
		return ""
	}

	return strings.TrimSpace(*raw)
}

func bulkDeleteIDs(raw []string) ([]pulid.ID, error) {
	if len(raw) == 0 {
		return nil, errortypes.NewValidationError(
			"ids",
			errortypes.ErrRequired,
			"Choose at least one document to delete",
		)
	}

	return gqlid.ParseIDListField("ids", raw)
}

func reviewRequest(
	authCtx *authctx.AuthContext,
	rawDocumentID string,
	decision document.ReviewDecision,
	reason string,
) (*documentservice.ReviewRequest, error) {
	documentID, err := gqlid.ParseIDField("documentId", rawDocumentID)
	if err != nil {
		return nil, err
	}

	return &documentservice.ReviewRequest{
		DocumentID: documentID,
		TenantInfo: resolvercore.TenantInfo(authCtx),
		Decision:   decision,
		Reason:     reason,
		Actor:      *actorutil.FromAuthContext(authCtx),
	}, nil
}

func signatureRequest(
	authCtx *authctx.AuthContext,
	input *gqlmodel.SetDocumentSignatureInput,
) (*documentservice.SetSignatureRequest, error) {
	documentID, err := gqlid.ParseIDField("documentId", input.DocumentID)
	if err != nil {
		return nil, err
	}

	return &documentservice.SetSignatureRequest{
		DocumentID: documentID,
		TenantInfo: resolvercore.TenantInfo(authCtx),
		Status:     input.Status,
		SignedAt:   gqlscalar.OptionalInt64(input.SignedAt),
		Version:    int64(input.Version),
		Actor:      *actorutil.FromAuthContext(authCtx),
	}, nil
}

func deliveryFactsRequest(
	authCtx *authctx.AuthContext,
	input *gqlmodel.SetDocumentDeliveryFactsInput,
) (*documentservice.SetDeliveryFactsRequest, error) {
	documentID, err := gqlid.ParseIDField("documentId", input.DocumentID)
	if err != nil {
		return nil, err
	}

	facts, err := documentDeliveryFactsFromInput(input.Facts)
	if err != nil {
		return nil, err
	}

	return &documentservice.SetDeliveryFactsRequest{
		DocumentID: documentID,
		TenantInfo: resolvercore.TenantInfo(authCtx),
		Facts:      facts,
		Version:    int64(input.Version),
		Actor:      *actorutil.FromAuthContext(authCtx),
	}, nil
}

func documentDeliveryFactsFromInput(
	input *gqlmodel.DocumentDeliveryFactsInput,
) (document.DeliveryFacts, error) {
	if input == nil {
		return document.DeliveryFacts{}, nil
	}

	amount, err := gqldecimal.ParseNullDecimalField("facts.amount", input.Amount)
	if err != nil {
		return document.DeliveryFacts{}, err
	}
	chargeID, err := gqlid.OptionalScopedID("facts.chargeId", input.ChargeID)
	if err != nil {
		return document.DeliveryFacts{}, err
	}

	facts := document.DeliveryFacts{
		SignedBy:    gqlscalar.StringValue(input.SignedBy),
		Pieces:      gqlscalar.StringValue(input.Pieces),
		OSDNote:     gqlscalar.StringValue(input.OsdNote),
		Seal:        gqlscalar.StringValue(input.Seal),
		ArrivedAt:   gqlscalar.OptionalInt64(input.ArrivedAt),
		DepartedAt:  gqlscalar.OptionalInt64(input.DepartedAt),
		Amount:      amount,
		Reference:   gqlscalar.StringValue(input.Reference),
		ChargeID:    chargeID,
		ChargeLabel: gqlscalar.StringValue(input.ChargeLabel),
	}
	if input.Osd != nil {
		facts.OSD = *input.Osd
	}

	return facts, nil
}
