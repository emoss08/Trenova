package rateagreementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/rateagreement"
	"github.com/emoss08/trenova/internal/modules/rating/ports/rateagreementports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/tarifftest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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

func literalInputs(t *testing.T, values map[string]string) tarifftypes.Inputs {
	t.Helper()
	inputs, err := tarifftypes.InputsFromLiterals(values)
	require.NoError(t, err)
	return inputs
}

func ruleWith(templateID pulid.ID, inputs tarifftypes.Inputs) *rateagreement.RateAgreementRule {
	return &rateagreement.RateAgreementRule{FormulaTemplateID: templateID, Inputs: inputs}
}

func agreementWith(rules ...*rateagreement.RateAgreementRule) *rateagreement.RateAgreement {
	return &rateagreement.RateAgreement{
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		Status:         rateagreement.StatusDraft,
		Rules:          rules,
	}
}

func checkRules(
	t *testing.T,
	tariffs *tariffInputs,
	agreement *rateagreement.RateAgreement,
) *errortypes.MultiError {
	t.Helper()
	multiErr := errortypes.NewMultiError()
	require.NoError(t, tariffs.checkRules(t.Context(), agreement, agreement.Rules, multiErr))
	require.NoError(t, tariffs.checkConditions(t.Context(), agreement, multiErr))
	return multiErr
}

func TestTariffInputsCheckRules(t *testing.T) {
	t.Parallel()

	perMile := tarifftest.PerMile().TemplateID

	t.Run("a template with no approved version", func(t *testing.T) {
		t.Parallel()

		multiErr := checkRules(t, testValidator().tariffs, agreementWith(
			ruleWith(pulid.MustNew("ft_"), literalInputs(t, map[string]string{"rate": "$2/mi"})),
		))

		messages := fieldMessages(multiErr, "rules[0].formulaTemplateId")
		require.Len(t, messages, 1)
		assert.Contains(t, messages[0], "no approved version")
	})

	t.Run("inputs that fit the template pass", func(t *testing.T) {
		t.Parallel()

		multiErr := checkRules(t, testValidator().tariffs, agreementWith(
			ruleWith(perMile, literalInputs(t, map[string]string{
				"rate":          "$2.35/mi",
				"minimumCharge": "$450",
			})),
		))

		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})

	t.Run("an input of the wrong type is reported on its field", func(t *testing.T) {
		t.Parallel()

		multiErr := checkRules(t, testValidator().tariffs, agreementWith(
			ruleWith(perMile, literalInputs(t, map[string]string{"rate": "$450"})),
		))

		assert.Len(t, fieldMessages(multiErr, "rules[0].inputs.rate"), 1)
		assert.Empty(t, fieldMessages(multiErr, "rules[0].formulaTemplateId"))
	})

	t.Run("a missing required input is reported on its field", func(t *testing.T) {
		t.Parallel()

		multiErr := checkRules(t, testValidator().tariffs, agreementWith(ruleWith(perMile, nil)))

		assert.Len(t, fieldMessages(multiErr, "rules[0].inputs.rate"), 1)
		assert.Empty(t, fieldMessages(multiErr, "rules[0].formulaTemplateId"))
	})

	t.Run("an agreement input is reported once for every rule it breaks", func(t *testing.T) {
		t.Parallel()

		agreement := agreementWith(
			ruleWith(perMile, literalInputs(t, map[string]string{"rate": "$2/mi"})),
			ruleWith(perMile, literalInputs(t, map[string]string{"rate": "$3/mi"})),
		)
		agreement.Inputs = literalInputs(t, map[string]string{"minimumCharge": "12%"})

		multiErr := checkRules(t, testValidator().tariffs, agreement)

		assert.Len(t, fieldMessages(multiErr, "inputs.minimumCharge"), 1)
		assert.Empty(t, fieldMessages(multiErr, "rules[0].inputs.minimumCharge"))
	})

	t.Run("agreement inputs the template does not declare are not checked", func(t *testing.T) {
		t.Parallel()

		agreement := agreementWith(
			ruleWith(perMile, literalInputs(t, map[string]string{"rate": "$2/mi"})),
		)
		agreement.Inputs = literalInputs(t, map[string]string{"fuelBase": "12%"})

		assert.False(t, checkRules(t, testValidator().tariffs, agreement).HasErrors())
	})

	t.Run("a template that does not check is reported on the template", func(t *testing.T) {
		t.Parallel()

		broken := tarifftest.PerMile()
		broken.TemplateID = pulid.MustNew("ft_")
		broken.Source = "tariff \"Broken\"\n\ncharge \"Freight\" = missing\n"
		tariffs := NewTestValidator(tarifftest.NewTemplates(broken), &tarifftest.Checker{}).tariffs

		multiErr := checkRules(t, tariffs, agreementWith(ruleWith(broken.TemplateID, nil)))

		messages := fieldMessages(multiErr, "rules[0].formulaTemplateId")
		require.NotEmpty(t, messages)
		assert.Contains(t, messages[0], "does not check")
	})

	t.Run("rules that share a template resolve it once", func(t *testing.T) {
		t.Parallel()

		templates := tarifftest.NewTemplates(tarifftest.PerMile())
		checker := &tarifftest.Checker{}
		tariffs := NewTestValidator(templates, checker).tariffs

		checkRules(t, tariffs, agreementWith(
			ruleWith(perMile, literalInputs(t, map[string]string{"rate": "$2/mi"})),
			ruleWith(perMile, literalInputs(t, map[string]string{"rate": "$3/mi"})),
		))

		assert.EqualValues(t, 1, templates.Resolves())
		assert.EqualValues(t, 2, checker.Calls())
	})

	t.Run("rules without a template are not checked", func(t *testing.T) {
		t.Parallel()

		checker := &tarifftest.Checker{}
		tariffs := NewTestValidator(tarifftest.NewTemplates(), checker).tariffs

		multiErr := checkRules(t, tariffs, agreementWith(ruleWith(pulid.Nil, nil)))

		assert.False(t, multiErr.HasErrors())
		assert.Zero(t, checker.Calls())
	})
}

func TestTariffInputsCheckConditions(t *testing.T) {
	t.Parallel()

	withCondition := func(condition string) *rateagreement.RateAgreement {
		agreement := agreementWith()
		agreement.Accessorials = []*rateagreement.RateAgreementAccessorial{
			{AccessorialChargeID: pulid.MustNew("acc_"), AutoApply: true, Condition: condition},
		}
		return agreement
	}

	t.Run("a condition that checks passes", func(t *testing.T) {
		t.Parallel()

		multiErr := checkRules(
			t,
			testValidator().tariffs,
			withCondition("condition \"Multi stop\"\n\nshipment.stops.count() > 2\n"),
		)

		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})

	t.Run("a condition that does not check is reported on it", func(t *testing.T) {
		t.Parallel()

		multiErr := checkRules(
			t,
			testValidator().tariffs,
			withCondition("condition \"Broken\"\n\nshipment.nothing > 2\n"),
		)

		messages := fieldMessages(multiErr, "accessorials[0].condition")
		require.Len(t, messages, 1)
		assert.Contains(t, messages[0], "does not check")
	})

	t.Run("an empty condition is not checked", func(t *testing.T) {
		t.Parallel()

		checker := &tarifftest.Checker{}
		tariffs := NewTestValidator(tarifftest.NewTemplates(), checker).tariffs

		assert.False(t, checkRules(t, tariffs, withCondition("  ")).HasErrors())
		assert.Zero(t, checker.Calls())
	})
}

func TestValidateAmendmentChecksTariffs(t *testing.T) {
	t.Parallel()

	validator := testValidator()
	agreement := agreementWith(
		ruleWith(pulid.MustNew("ft_"), nil),
		ruleWith(
			tarifftest.PerMile().TemplateID,
			literalInputs(t, map[string]string{"rate": "$1"}),
		),
	)

	multiErr := validator.ValidateAmendment(
		t.Context(),
		agreement,
		&rateagreementports.AmendRateAgreementRulesRequest{
			EffectiveFrom: 1_700_000_000,
			Rules:         agreement.Rules,
		},
	)

	require.NotNil(t, multiErr)
	assert.Len(t, fieldMessages(multiErr, "rules[0].formulaTemplateId"), 1)
	assert.Len(t, fieldMessages(multiErr, "rules[1].inputs.rate"), 1)
}

func TestValidateCreateChecksTariffs(t *testing.T) {
	t.Parallel()

	agreement := agreementWith(
		ruleWith(
			tarifftest.PerMile().TemplateID,
			literalInputs(t, map[string]string{"rate": "$1"}),
		),
	)
	agreement.Accessorials = []*rateagreement.RateAgreementAccessorial{
		{
			AccessorialChargeID: pulid.MustNew("acc_"),
			AutoApply:           true,
			Condition:           "condition \"X\"\n\n1 +\n",
		},
	}

	multiErr := testValidator().ValidateCreate(t.Context(), agreement)

	require.NotNil(t, multiErr)
	assert.Len(t, fieldMessages(multiErr, "rules[0].inputs.rate"), 1)
	assert.Len(t, fieldMessages(multiErr, "accessorials[0].condition"), 1)
}
