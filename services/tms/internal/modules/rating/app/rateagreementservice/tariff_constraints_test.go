package rateagreementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/ports/tariffports"
	"github.com/emoss08/trenova/internal/modules/rating/ratingtest/tarifftest"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/rules"
	"github.com/emoss08/trenova/pkg/tarifftypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const boundedSource = `tariff "Bounded"

/// The rate per mile.
input rate: Money/mi where $0.50/mi..=$15/mi
/// The service the contract buys.
input service: Text = "Standard" where ["Standard", "Guaranteed"]
/// A cap per stop, first stop first.
input caps: List<Money> = [] where $0..=$500

charge "Linehaul" (quantity: shipment.distance, rate: rate)
if service == "Guaranteed" {
  charge "Guarantee" = $50
}
for cap in caps {
  charge "Cap" = cap
}
`

func boundedTemplate(t *testing.T) *tariffports.Template {
	t.Helper()
	check, err := rules.CheckTariff(t.Context(), &rules.TariffCheckRequest{
		Source:  boundedSource,
		Kind:    rules.TariffKindTariff,
		Context: &rules.TariffContext{},
	})
	require.NoError(t, err)
	require.True(t, check.OK, "%+v", check.Diagnostics)
	return &tariffports.Template{
		TemplateID:    pulid.ID("ft_bounded_stub"),
		VersionNumber: 1,
		Name:          "Bounded",
		Source:        boundedSource,
		Inputs:        tarifftypes.InputDeclsFrom(check.Inputs),
	}
}

func renderedMessages(multiErr *errortypes.MultiError, field string) []string {
	out := make([]string, 0, 1)
	for _, fieldErr := range multiErr.Errors {
		if fieldErr.Field == field {
			out = append(out, fieldErr.Error())
		}
	}
	return out
}

func boundedTariffs(t *testing.T) *tariffInputs {
	t.Helper()
	return NewTestValidator(
		tarifftest.NewTemplates(boundedTemplate(t)),
		&tarifftest.Checker{},
	).tariffs
}

func TestTariffInputsRefuseValuesOutsideTheirConstraint(t *testing.T) {
	t.Parallel()

	templateID := pulid.ID("ft_bounded_stub")

	t.Run("a rate within its bounds saves", func(t *testing.T) {
		t.Parallel()

		multiErr := checkRules(t, boundedTariffs(t), agreementWith(
			ruleWith(templateID, literalInputs(t, map[string]string{"rate": "$15/mi"})),
		))

		assert.False(t, multiErr.HasErrors(), multiErr.Error())
	})

	t.Run("a rate outside its bounds is refused on its field, naming the bounds", func(t *testing.T) {
		t.Parallel()

		multiErr := checkRules(t, boundedTariffs(t), agreementWith(
			ruleWith(templateID, literalInputs(t, map[string]string{"rate": "$235/mi"})),
		))

		assert.Equal(t,
			[]string{"`rate` must be from $0.50/mi to $15.00/mi; $235.00/mi is outside it"},
			renderedMessages(multiErr, "rules[0].inputs.rate"),
		)
		assert.Empty(t, fieldMessages(multiErr, "rules[0].formulaTemplateId"))
	})

	t.Run("a value missing from its list is refused on its field", func(t *testing.T) {
		t.Parallel()

		inputs := literalInputs(t, map[string]string{"rate": "$2/mi"})
		inputs["service"] = rules.TextValue("Overnight")
		multiErr := checkRules(t, boundedTariffs(t), agreementWith(ruleWith(templateID, inputs)))

		assert.Equal(t,
			[]string{"`service` must be one of [Standard, Guaranteed]; Overnight isn't"},
			renderedMessages(multiErr, "rules[0].inputs.service"),
		)
	})

	t.Run("an agreement value outside its list is refused once on the agreement", func(t *testing.T) {
		t.Parallel()

		agreement := agreementWith(
			ruleWith(templateID, literalInputs(t, map[string]string{"rate": "$2/mi"})),
			ruleWith(templateID, literalInputs(t, map[string]string{"rate": "$3/mi"})),
		)
		agreement.Inputs = tarifftypes.Inputs{"service": rules.TextValue("Overnight")}

		multiErr := checkRules(t, boundedTariffs(t), agreement)

		assert.Len(t, renderedMessages(multiErr, "inputs.service"), 1)
		assert.Empty(t, fieldMessages(multiErr, "rules[0].inputs.service"))
	})

	t.Run("a list item outside its bounds is refused at its index", func(t *testing.T) {
		t.Parallel()

		inputs := literalInputs(t, map[string]string{"rate": "$2/mi"})
		inputs["caps"] = rules.RawJSON(`[{"$q":"100","u":"$"},{"$q":"900","u":"$"}]`)
		multiErr := checkRules(t, boundedTariffs(t), agreementWith(ruleWith(templateID, inputs)))

		messages := renderedMessages(multiErr, "rules[0].inputs.caps[1]")
		require.Len(t, messages, 1, multiErr.Error())
		assert.Contains(t, messages[0], "$900.00")
		assert.Empty(t, fieldMessages(multiErr, "rules[0].inputs.caps[0]"))
	})
}
