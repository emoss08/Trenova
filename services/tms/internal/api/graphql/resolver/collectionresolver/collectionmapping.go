package collectionresolver

import (
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlid"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/gqlscalar"
	"github.com/emoss08/trenova/internal/modules/billing/ports/collectionports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/shopspring/decimal"
)

func activityCursorFromInput(
	input *gqlmodel.CollectionActivityCursorInput,
) (*collectionports.ActivityCursor, error) {
	if input == nil {
		return nil, nil //nolint:nilnil // no cursor reads the newest page
	}

	id, err := gqlid.RequiredID("before.id", input.ID)
	if err != nil {
		return nil, err
	}

	return &collectionports.ActivityCursor{CreatedAt: int64(input.CreatedAt), ID: id}, nil
}

func logActivityRequestFromInput(
	input *gqlmodel.LogCollectionActivityInput,
) (*collectionports.LogActivityRequest, error) {
	invoiceID, err := gqlid.OptionalScopedID("invoiceId", input.InvoiceID)
	if err != nil {
		return nil, err
	}

	promisedAmount, err := promisedAmountFromInput(input.PromisedAmount)
	if err != nil {
		return nil, err
	}

	return &collectionports.LogActivityRequest{
		Kind:           input.Kind,
		Summary:        input.Summary,
		ContactName:    input.ContactName,
		InvoiceID:      invoiceID,
		PromisedAmount: promisedAmount,
		PromisedOn:     gqlscalar.OptionalInt64(input.PromisedOn),
		FollowUpOn:     gqlscalar.OptionalInt64(input.FollowUpOn),
	}, nil
}

func promisedAmountFromInput(raw *string) (decimal.NullDecimal, error) {
	if raw == nil || *raw == "" {
		return decimal.NullDecimal{}, nil
	}

	amount, err := decimal.NewFromString(*raw)
	if err != nil {
		return decimal.NullDecimal{}, errortypes.NewValidationError(
			"promisedAmount",
			errortypes.ErrInvalid,
			"Promised amount must be a number",
		)
	}

	return decimal.NewNullDecimal(amount), nil
}
