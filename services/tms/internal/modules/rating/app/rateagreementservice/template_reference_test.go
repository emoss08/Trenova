package rateagreementservice

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubTemplateRepo struct {
	formulatemplateports.FormulaTemplateRepository
	templates []*formulatemplate.FormulaTemplate
}

func (s *stubTemplateRepo) GetByIDs(
	_ context.Context,
	_ formulatemplateports.GetFormulaTemplatesByIDsRequest,
) ([]*formulatemplate.FormulaTemplate, error) {
	return s.templates, nil
}

type approvedVersions struct {
	formulatemplateports.FormulaTemplateVersionRepository
	latest *formulatemplate.FormulaTemplateVersion
}

func (a *approvedVersions) GetLatestByStatus(
	_ context.Context,
	_ *formulatemplateports.GetLatestVersionByStatusRequest,
) (*formulatemplate.FormulaTemplateVersion, error) {
	return a.latest, nil
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
			templateRefs{templates: &stubTemplateRepo{}},
			tenantInfo,
			rules,
			multiErr,
		)

		messages := fieldMessages(multiErr, "rules[0].formulaTemplateId")
		require.Len(t, messages, 1)
		assert.Contains(t, messages[0], "does not exist")
	})

	t.Run("a draft never approved", func(t *testing.T) {
		t.Parallel()

		repo := &stubTemplateRepo{templates: []*formulatemplate.FormulaTemplate{
			makeTemplate(formulatemplate.StatusDraft, formulatemplate.TemplateTypeFreightCharge),
		}}

		multiErr := errortypes.NewMultiError()
		validateRuleTemplateReferences(
			t.Context(),
			templateRefs{templates: repo, versions: &approvedVersions{}},
			tenantInfo,
			rules,
			multiErr,
		)

		messages := fieldMessages(multiErr, "rules[0].formulaTemplateId")
		require.Len(t, messages, 1)
		assert.Contains(t, messages[0], "never been approved")
	})

	t.Run("a revision in review keeps its approved content pricing", func(t *testing.T) {
		t.Parallel()

		repo := &stubTemplateRepo{templates: []*formulatemplate.FormulaTemplate{
			makeTemplate(formulatemplate.StatusInReview, formulatemplate.TemplateTypeFreightCharge),
		}}

		multiErr := errortypes.NewMultiError()
		validateRuleTemplateReferences(
			t.Context(),
			templateRefs{
				templates: repo,
				versions: &approvedVersions{
					latest: &formulatemplate.FormulaTemplateVersion{VersionNumber: 4},
				},
			},
			tenantInfo,
			rules,
			multiErr,
		)

		assert.Empty(t, fieldMessages(multiErr, "rules[0].formulaTemplateId"))
	})

	t.Run("an archived template", func(t *testing.T) {
		t.Parallel()

		repo := &stubTemplateRepo{templates: []*formulatemplate.FormulaTemplate{
			makeTemplate(formulatemplate.StatusInactive, formulatemplate.TemplateTypeFreightCharge),
		}}

		multiErr := errortypes.NewMultiError()
		validateRuleTemplateReferences(
			t.Context(),
			templateRefs{
				templates: repo,
				versions: &approvedVersions{
					latest: &formulatemplate.FormulaTemplateVersion{VersionNumber: 4},
				},
			},
			tenantInfo,
			rules,
			multiErr,
		)

		messages := fieldMessages(multiErr, "rules[0].formulaTemplateId")
		require.Len(t, messages, 1)
		assert.Contains(t, messages[0], "archived")
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
		validateRuleTemplateReferences(t.Context(), templateRefs{templates: repo}, tenantInfo, rules, multiErr)

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
		validateRuleTemplateReferences(t.Context(), templateRefs{templates: repo}, tenantInfo, rules, multiErr)

		assert.Empty(t, fieldMessages(multiErr, "rules[0].formulaTemplateId"))
	})

	t.Run("rules without a template reference are skipped", func(t *testing.T) {
		t.Parallel()

		multiErr := errortypes.NewMultiError()
		validateRuleTemplateReferences(
			t.Context(),
			templateRefs{templates: &stubTemplateRepo{}},
			tenantInfo,
			[]*rateagreement.RateAgreementRule{{FormulaTemplateID: nil}},
			multiErr,
		)

		assert.False(t, multiErr.HasErrors())
	})

	t.Run("nil repository is a no-op", func(t *testing.T) {
		t.Parallel()

		multiErr := errortypes.NewMultiError()
		validateRuleTemplateReferences(t.Context(), templateRefs{}, tenantInfo, rules, multiErr)

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
		&rateagreementports.AmendRateAgreementRulesRequest{
			EffectiveFrom: 1_700_000_000,
			Rules:         agreement.Rules,
		},
	)

	require.NotNil(t, multiErr)
	messages := fieldMessages(multiErr, "rules[0].formulaTemplateId")
	require.Len(t, messages, 1)
	assert.Contains(t, messages[0], "does not exist")
}
