package fuelpurchaseservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB   *postgres.Connection
	Repo repositories.FuelPurchaseRepository
}

type CardFinder interface {
	FindCardByLastFour(
		ctx context.Context,
		req *repositories.FindFuelCardByLastFourRequest,
	) (*fuelpurchase.FuelCard, error)
}

type Validator struct {
	cards *validationframework.TenantedValidator[*fuelpurchase.FuelCard]
}

func NewValidator(p ValidatorParams) *Validator {
	return newValidator(
		validationframework.NewBunReferenceCheckerLazy(func() bun.IDB { return p.DB.DB() }),
		p.Repo,
	)
}

func NewValidatorWithDeps(
	references validationframework.ReferenceChecker,
	cards CardFinder,
) *Validator {
	return newValidator(references, cards)
}

func newValidator(
	references validationframework.ReferenceChecker,
	cards CardFinder,
) *Validator {
	builder := validationframework.
		NewTenantedValidatorBuilder[*fuelpurchase.FuelCard]().
		WithModelName("FuelCard").
		WithOptionalReferenceCheck(
			"assignedWorkerId",
			"workers",
			"Assigned worker does not exist in your organization",
			func(c *fuelpurchase.FuelCard) pulid.ID { return derefID(c.AssignedWorkerID) },
		).
		WithOptionalReferenceCheck(
			"assignedTractorId",
			"tractors",
			"Assigned tractor does not exist in your organization",
			func(c *fuelpurchase.FuelCard) pulid.ID { return derefID(c.AssignedTractorID) },
		).
		WithCustomRule(uniqueCardRule(cards))
	if references != nil {
		builder = builder.WithReferenceChecker(references)
	}

	return &Validator{cards: builder.Build()}
}

func derefID(id *pulid.ID) pulid.ID {
	if id == nil {
		return ""
	}
	return *id
}

func uniqueCardRule(cards CardFinder) validationframework.TenantedRule[*fuelpurchase.FuelCard] {
	return validationframework.NewTenantedRule[*fuelpurchase.FuelCard](
		"fuel_card_provider_last_four",
	).
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			card *fuelpurchase.FuelCard,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if cards == nil || card.Status == fuelpurchase.CardStatusCancelled {
				return nil
			}
			existing, err := cards.FindCardByLastFour(
				ctx,
				&repositories.FindFuelCardByLastFourRequest{
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
) *errortypes.MultiError {
	return v.cards.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *fuelpurchase.FuelCard,
) *errortypes.MultiError {
	return v.cards.ValidateUpdate(ctx, entity)
}
