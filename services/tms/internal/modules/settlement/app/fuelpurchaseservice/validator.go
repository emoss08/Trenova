package fuelpurchaseservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/tenantedvalidation"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	Integrity validationframework.IntegrityChecker
	Repo      fuelpurchaseports.FuelPurchaseRepository
}

type CardFinder interface {
	FindCardByLastFour(
		ctx context.Context,
		req *fuelpurchaseports.FindFuelCardByLastFourRequest,
	) (*fuelpurchase.FuelCard, error)
}

type Validator struct {
	cards tenantedvalidation.EntityValidator[*fuelpurchase.FuelCard]
}

func NewValidator(p ValidatorParams) *Validator {
	cards := tenantedvalidation.NewBuilder[*fuelpurchase.FuelCard]().
		WithModelName("FuelCard").
		WithIntegrityChecker(p.Integrity).
		WithOptionalReferenceCheck(
			"assignedWorkerId",
			"workers",
			"Assigned worker does not exist in your organization",
			func(c *fuelpurchase.FuelCard) pulid.ID { return typeutils.DerefID(c.AssignedWorkerID) },
		).
		WithOptionalReferenceCheck(
			"assignedTractorId",
			"tractors",
			"Assigned tractor does not exist in your organization",
			func(c *fuelpurchase.FuelCard) pulid.ID { return typeutils.DerefID(c.AssignedTractorID) },
		).
		WithCustomRule(uniqueCardRule(p.Repo)).
		Build()

	return &Validator{cards: cards}
}

func uniqueCardRule(cards CardFinder) validationframework.Rule[*fuelpurchase.FuelCard] {
	return validationframework.NewRule[*fuelpurchase.FuelCard](
		"fuel_card_provider_last_four",
	).
		OnBoth().
		WithStage(validationframework.StageBusinessRules).
		WithValidation(func(
			ctx context.Context,
			card *fuelpurchase.FuelCard,
			_ *validationframework.Context,
			multiErr *errortypes.MultiError,
		) error {
			if card.Status == fuelpurchase.CardStatusCancelled {
				return nil
			}
			existing, err := cards.FindCardByLastFour(
				ctx,
				&fuelpurchaseports.FindFuelCardByLastFourRequest{
					TenantInfo: pagination.TenantInfo{
						OrgID: card.OrganizationID,
						BuID:  card.BusinessUnitID,
					},
					Provider: card.Provider,
					LastFour: card.LastFour,
				},
			)
			if err != nil {
				return err
			}
			if existing != nil && existing.ID != card.ID {
				multiErr.Add(
					"lastFour",
					errortypes.ErrDuplicate,
					"A card from this provider with these last four digits already exists",
				)
			}
			return nil
		})
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *fuelpurchase.FuelCard,
) error {
	return v.cards.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *fuelpurchase.FuelCard,
) error {
	return v.cards.ValidateUpdate(ctx, entity)
}
