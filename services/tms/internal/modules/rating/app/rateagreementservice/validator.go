package rateagreementservice

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/typeutils"
	"go.uber.org/fx"
)

type ValidatorParams struct {
	fx.In

	DB                         *postgres.Connection
	FormulaTemplateRepo        formulatemplateports.FormulaTemplateRepository
	FormulaTemplateVersionRepo formulatemplateports.FormulaTemplateVersionRepository `optional:"true"`
}

type templateRefs struct {
	templates formulatemplateports.FormulaTemplateRepository
	versions  formulatemplateports.FormulaTemplateVersionRepository
}

type Validator struct {
	validator validationframework.TenantedEntityValidator[*rateagreement.RateAgreement]
	refs      templateRefs
}

func NewValidator(p ValidatorParams) *Validator {
	refs := templateRefs{templates: p.FormulaTemplateRepo, versions: p.FormulaTemplateVersionRepo}
	return &Validator{
		validator: newBuilder(p.DB, refs).Build(),
		refs:      refs,
	}
}

func newBuilder(
	db *postgres.Connection,
	refs templateRefs,
) *validationframework.TenantedValidatorBuilder[*rateagreement.RateAgreement] {
	builder := validationframework.
		NewTenantedValidatorBuilder[*rateagreement.RateAgreement]().
		WithModelName("Rate Agreement").
		WithCustomRule(activationReadinessRule()).
		WithCustomRule(templateReferenceRule(refs))

	if db == nil {
		return builder
	}

	return builder.
		WithUniquenessChecker(
			validationframework.NewBunUniquenessCheckerScoped(db),
		).
		WithUniqueField(
			"code",
			"code",
			"A rate agreement with this code already exists in your organization",
			func(a *rateagreement.RateAgreement) any { return a.Code },
		).
		WithReferenceChecker(
			validationframework.NewBunReferenceCheckerScoped(db),
		).
		WithOptionalReferenceCheck(
			"customerId",
			"customers",
			"Customer does not exist in your organization",
			func(a *rateagreement.RateAgreement) pulid.ID { return typeutils.DerefID(a.CustomerID) },
		).
		WithOptionalReferenceCheck(
			"carrierId",
			"carriers",
			"Carrier does not exist in your organization",
			func(a *rateagreement.RateAgreement) pulid.ID { return typeutils.DerefID(a.CarrierID) },
		).
		WithOptionalReferenceCheck(
			"billToCustomerId",
			"customers",
			"Bill-to customer does not exist in your organization",
			func(a *rateagreement.RateAgreement) pulid.ID { return typeutils.DerefID(a.BillToCustomerID) },
		)
}

func (v *Validator) ValidateCreate(
	ctx context.Context,
	entity *rateagreement.RateAgreement,
) *errortypes.MultiError {
	return v.validator.ValidateCreate(ctx, entity)
}

func (v *Validator) ValidateUpdate(
	ctx context.Context,
	entity *rateagreement.RateAgreement,
) *errortypes.MultiError {
	return v.validator.ValidateUpdate(ctx, entity)
}

// ValidateAmendment checks a rule change before it is written.
//
// Amendments are the one write that bypasses the entity validator, because they
// do not carry a whole agreement — so the rules the validator would have run
// have to run here instead.
func (v *Validator) ValidateAmendment(
	ctx context.Context,
	agreement *rateagreement.RateAgreement,
	req *rateagreementports.AmendRateAgreementRulesRequest,
) *errortypes.MultiError {
	multiErr := errortypes.NewMultiError()

	if req.EffectiveFrom <= 0 {
		multiErr.Add(
			"effectiveFrom",
			errortypes.ErrRequired,
			"An amendment needs the date its rates take effect",
		)
	}

	if len(req.SupersededIDs) == 0 && len(req.Rules) == 0 {
		multiErr.Add(
			"rules",
			errortypes.ErrRequired,
			"An amendment must add or replace at least one rate",
		)
	}

	if agreement.Status == rateagreement.StatusArchived {
		multiErr.Add(
			"status",
			errortypes.ErrInvalidOperation,
			"An archived rate agreement cannot be amended",
		)
	}

	for index, rule := range req.Rules {
		if rule == nil {
			continue
		}

		// The rule inherits the agreement's tenancy and party before it is
		// checked, because its own validation reads both — and the repository
		// will stamp them again on insert regardless.
		rule.RateAgreementID = agreement.ID
		rule.OrganizationID = agreement.OrganizationID
		rule.BusinessUnitID = agreement.BusinessUnitID
		rule.PartyType = agreement.PartyType
		rule.PartyID = agreement.PartyID()

		rule.Validate(multiErr.WithIndex("rules", index))
	}

	validateRuleTemplateReferences(
		ctx,
		v.refs,
		pagination.TenantInfo{
			OrgID: agreement.OrganizationID,
			BuID:  agreement.BusinessUnitID,
		},
		req.Rules,
		multiErr,
	)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// templateReferenceRule refuses a rule that points at a formula template the
// rating engine could never price with — one that does not exist in the
// tenant, is not Active, or is not a freight-charge template. Without it the
// failure surfaces only at rating time, where it blocks the shipment save.
func templateReferenceRule(
	refs templateRefs,
) validationframework.TenantedRule[*rateagreement.RateAgreement] {
	return validationframework.
		NewTenantedRule[*rateagreement.RateAgreement]("rate_agreement_rule_template_reference").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			ctx context.Context,
			entity *rateagreement.RateAgreement,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			validateRuleTemplateReferences(
				ctx,
				refs,
				pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				},
				entity.Rules,
				multiErr,
			)

			return nil
		})
}

func validateRuleTemplateReferences(
	ctx context.Context,
	refs templateRefs,
	tenantInfo pagination.TenantInfo,
	rules []*rateagreement.RateAgreementRule,
	multiErr *errortypes.MultiError,
) {
	if refs.templates == nil {
		return
	}

	templateIDs := make([]pulid.ID, 0, len(rules))
	seen := make(map[pulid.ID]struct{}, len(rules))
	for _, rule := range rules {
		if rule == nil || rule.FormulaTemplateID == nil || rule.FormulaTemplateID.IsNil() {
			continue
		}
		if _, ok := seen[*rule.FormulaTemplateID]; ok {
			continue
		}
		seen[*rule.FormulaTemplateID] = struct{}{}
		templateIDs = append(templateIDs, *rule.FormulaTemplateID)
	}

	if len(templateIDs) == 0 {
		return
	}

	templates, err := refs.templates.GetByIDs(
		ctx,
		formulatemplateports.GetFormulaTemplatesByIDsRequest{
			TenantInfo:  tenantInfo,
			TemplateIDs: templateIDs,
		},
	)
	if err != nil {
		multiErr.Add(
			"rules",
			errortypes.ErrInvalid,
			"Unable to verify the referenced formula templates",
		)
		return
	}

	byID := make(map[pulid.ID]*formulatemplate.FormulaTemplate, len(templates))
	for _, template := range templates {
		byID[template.ID] = template
	}

	for index, rule := range rules {
		if rule == nil || rule.FormulaTemplateID == nil || rule.FormulaTemplateID.IsNil() {
			continue
		}

		indexed := multiErr.WithIndex("rules", index)
		template, ok := byID[*rule.FormulaTemplateID]
		switch {
		case !ok:
			indexed.Add(
				"formulaTemplateId",
				errortypes.ErrInvalid,
				"Formula template does not exist in your organization",
			)
		case template.Status == formulatemplate.StatusInactive:
			indexed.Add(
				"formulaTemplateId",
				errortypes.ErrInvalid,
				"Formula template is archived and no longer prices shipments",
			)
		case !refs.bills(ctx, template, tenantInfo):
			indexed.Add(
				"formulaTemplateId",
				errortypes.ErrInvalid,
				"Formula template has never been approved, so it cannot price shipments yet",
			)
		case template.Type != formulatemplate.TemplateTypeFreightCharge:
			indexed.Add(
				"formulaTemplateId",
				errortypes.ErrInvalid,
				"Formula template must be a freight charge template",
			)
		}
	}
}

// activationReadinessRule stops an agreement from being saved in a state it
// could never price anything from.
//
// An active agreement with no rules is the failure that looks like it worked:
// shipments resolve nothing, fall through to whatever the organization's
// unrated disposition says, and nobody notices until the invoices go out.
func activationReadinessRule() validationframework.TenantedRule[*rateagreement.RateAgreement] {
	return validationframework.
		NewTenantedRule[*rateagreement.RateAgreement]("rate_agreement_activation_readiness").
		OnBoth().
		WithStage(validationframework.ValidationStageBusinessRules).
		WithPriority(validationframework.ValidationPriorityHigh).
		WithValidation(func(
			_ context.Context,
			entity *rateagreement.RateAgreement,
			_ *validationframework.TenantedValidationContext,
			multiErr *errortypes.MultiError,
		) error {
			if entity.Status != rateagreement.StatusActive || len(entity.Rules) > 0 {
				return nil
			}

			multiErr.Add(
				"rules",
				errortypes.ErrInvalidOperation,
				"An active rate agreement needs at least one rate, "+
					"otherwise no shipment can price against it",
			)

			return nil
		})
}

func (r templateRefs) bills(
	ctx context.Context,
	template *formulatemplate.FormulaTemplate,
	tenantInfo pagination.TenantInfo,
) bool {
	if template.Status == formulatemplate.StatusActive {
		return true
	}
	if r.versions == nil {
		return false
	}

	approved, err := r.versions.GetLatestByStatus(
		ctx,
		&formulatemplateports.GetLatestVersionByStatusRequest{
			TenantInfo: tenantInfo,
			TemplateID: template.ID,
			Status:     formulatemplate.StatusActive,
		},
	)
	return err == nil && approved != nil
}
