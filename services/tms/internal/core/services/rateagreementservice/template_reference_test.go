package rateagreementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubTemplateRepo struct {
	repositories.FormulaTemplateRepository
	templates []*formulatemplate.FormulaTemplate
}

func (s *stubTemplateRepo) GetByIDs(
	_ context.Context,
	_ repositories.GetFormulaTemplatesByIDsRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return s.templates, nil
}

func fieldMessages(multiErr *errortypes.MultiError, field string) []string {
	if multiErr == nil {
		return nil
	}

	messages := make([]string, 0, 1)
	for _, fieldErr := range multiErr.Errors {
		if fieldErr.Field == field {
			messages = append(messages, fieldErr.Message)
		}
	}
	return messages
}

func newAgreementWithRuleTemplate(templateID pulid.ID) *rateagreement.RateAgreement {
	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")

	return &rateagreement.RateAgreement{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		Status:         rateagreement.StatusDraft,
		Rules: []*rateagreement.RateAgreementRule{
			{
				OrganizationID:    orgID,
				BusinessUnitID:    buID,
				FormulaTemplateID: &templateID,
			},
		},
	}
}

func TestValidateRuleTemplateReferences(t *testing.T) {
	t.Parallel()

	tenantInfo := pagination.TenantInfo{
		OrgID: pulid.MustNew("org_"),
		BuID:  pulid.MustNew("bu_"),
	}

	templateID := pulid.MustNew("ft_")

	makeTemplate := func(
		status formulatemplate.Status,
		templateType formulatemplate.TemplateType,
	) *formulatemplate.FormulaTemplate {
		return &formulatemplate.FormulaTemplate{
			ID:     templateID,
			Name:   "Per Mile",
			Status: status,
			Type:   templateType,
		}
	}

	rules := []*rateagreement.RateAgreementRule{{FormulaTemplateID: &templateID}}

	t.Run("missing template", func(t *testing.T) {
		t.Parallel()

		multiErr := errortypes.NewMultiError()
		validateRuleTemplateReferences(
			t.Context(),
			&stubTemplateRepo{},
			tenantInfo,
			rules,
			multiErr,
		)

		messages := fieldMessages(multiErr, "rules[0].formulaTemplateId")
		require.Len(t, messages, 1)
		assert.Contains(t, messages[0], "does not exist")
	})

	t.Run("template not active", func(t *testing.T) {
		t.Parallel()

		repo := &stubTemplateRepo{templates: []*formulatemplate.FormulaTemplate{
			makeTemplate(formulatemplate.StatusDraft, formulatemplate.TemplateTypeFreightCharge),
		}}

		multiErr := errortypes.NewMultiError()
		validateRuleTemplateReferences(t.Context(), repo, tenantInfo, rules, multiErr)

		messages := fieldMessages(multiErr, "rules[0].formulaTemplateId")
		require.Len(t, messages, 1)
		assert.Contains(t, messages[0], "must be Active")
	})

	t.Run("template wrong type", func(t *testing.T) {
		t.Parallel()

		repo := &stubTemplateRepo{templates: []*formulatemplate.FormulaTemplate{
			makeTemplate(
				formulatemplate.StatusActive,
				formulatemplate.TemplateTypeAccessorialCharge,
			),
		}}

		multiErr := errortypes.NewMultiError()
		validateRuleTemplateReferences(t.Context(), repo, tenantInfo, rules, multiErr)

		messages := fieldMessages(multiErr, "rules[0].formulaTemplateId")
		require.Len(t, messages, 1)
		assert.Contains(t, messages[0], "freight charge")
	})

	t.Run("active freight charge template passes", func(t *testing.T) {
		t.Parallel()

		repo := &stubTemplateRepo{templates: []*formulatemplate.FormulaTemplate{
			makeTemplate(formulatemplate.StatusActive, formulatemplate.TemplateTypeFreightCharge),
		}}

		multiErr := errortypes.NewMultiError()
		validateRuleTemplateReferences(t.Context(), repo, tenantInfo, rules, multiErr)

		assert.Empty(t, fieldMessages(multiErr, "rules[0].formulaTemplateId"))
	})

	t.Run("rules without a template reference are skipped", func(t *testing.T) {
		t.Parallel()

		multiErr := errortypes.NewMultiError()
		validateRuleTemplateReferences(
			t.Context(),
			&stubTemplateRepo{},
			tenantInfo,
			[]*rateagreement.RateAgreementRule{{FormulaTemplateID: nil}},
			multiErr,
		)

		assert.False(t, multiErr.HasErrors())
	})

	t.Run("nil repository is a no-op", func(t *testing.T) {
		t.Parallel()

		multiErr := errortypes.NewMultiError()
		validateRuleTemplateReferences(t.Context(), nil, tenantInfo, rules, multiErr)

		assert.False(t, multiErr.HasErrors())
	})
}

func TestValidateAmendmentChecksTemplateReferences(t *testing.T) {
	t.Parallel()

	templateID := pulid.MustNew("ft_")
	validator := NewTestValidatorWithTemplateRepo(&stubTemplateRepo{})
	agreement := newAgreementWithRuleTemplate(templateID)

	multiErr := validator.ValidateAmendment(
		t.Context(),
		agreement,
		&repositories.AmendRateAgreementRulesRequest{
			EffectiveFrom: 1_700_000_000,
			Rules:         agreement.Rules,
		},
	)

	require.NotNil(t, multiErr)
	messages := fieldMessages(multiErr, "rules[0].formulaTemplateId")
	require.Len(t, messages, 1)
	assert.Contains(t, messages[0], "does not exist")
}
