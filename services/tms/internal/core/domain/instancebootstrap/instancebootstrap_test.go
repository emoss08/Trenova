package instancebootstrap

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/onboarding"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func validInputs() *Inputs {
	return &Inputs{
		OrganizationName: "Acme Freight",
		AdminName:        "Dana Whitfield",
		AdminEmail:       "dana@acme.example",
	}
}

func fieldsOf(t *testing.T, inputs *Inputs) []string {
	t.Helper()

	multiErr := errortypes.NewMultiError()
	inputs.Validate(multiErr)

	fields := make([]string, 0, len(multiErr.Errors))
	for _, err := range multiErr.Errors {
		fields = append(fields, err.Field)
	}

	return fields
}

func TestNormalizeAppliesDefaultsAndCanonicalForms(t *testing.T) {
	t.Parallel()

	got := (&Inputs{
		OrganizationName: "  Acme   Freight ",
		AdminName:        " Dana  Whitfield",
		AdminEmail:       " Dana@Acme.Example ",
		State:            " tx ",
		SCAC:             "acme",
	}).Normalize()

	assert.Equal(t, "Acme Freight", got.OrganizationName)
	assert.Equal(t, "Dana Whitfield", got.AdminName)
	assert.Equal(t, "dana@acme.example", got.AdminEmail)
	assert.Equal(t, DefaultTimezone, got.Timezone)
	assert.Equal(t, "TX", got.State)
	assert.Equal(t, "ACME", got.SCAC)
	assert.Equal(t, onboarding.PlaceholderCity, got.City)
	assert.Equal(t, onboarding.PlaceholderPostalCode, got.PostalCode)
	assert.Equal(t, onboarding.PlaceholderDOTNumber, got.DOTNumber)

	assert.Equal(t, DefaultState, validInputs().Normalize().State)
	assert.Equal(t, onboarding.PlaceholderSCAC, validInputs().Normalize().SCAC)
}

func TestValidateAcceptsTheMinimalInputs(t *testing.T) {
	t.Parallel()

	normalized := validInputs().Normalize()
	assert.Empty(t, fieldsOf(t, &normalized))
}

func TestValidateReportsEachInvalidField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		mutate func(*Inputs)
		field  string
	}{
		{"missing organization", func(i *Inputs) { i.OrganizationName = "" }, FieldOrganizationName},
		{
			"long organization",
			func(i *Inputs) { i.OrganizationName = strings.Repeat("a", MaxOrganizationNameLength+1) },
			FieldOrganizationName,
		},
		{"missing admin name", func(i *Inputs) { i.AdminName = "" }, FieldAdminName},
		{"missing email", func(i *Inputs) { i.AdminEmail = "" }, FieldAdminEmail},
		{"bad email", func(i *Inputs) { i.AdminEmail = "not-an-email" }, FieldAdminEmail},
		{"bad timezone", func(i *Inputs) { i.Timezone = "Mars/Olympus" }, FieldTimezone},
		{"bad state", func(i *Inputs) { i.State = "Texas" }, FieldState},
		{"bad postal code", func(i *Inputs) { i.PostalCode = "1234" }, FieldPostalCode},
		{"bad scac", func(i *Inputs) { i.SCAC = "AB1" }, FieldSCAC},
		{"bad dot", func(i *Inputs) { i.DOTNumber = "12a" }, FieldDOTNumber},
		{
			"long address",
			func(i *Inputs) { i.AddressLine1 = strings.Repeat("a", MaxAddressLength+1) },
			FieldAddressLine1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			inputs := validInputs().Normalize()
			tt.mutate(&inputs)
			assert.Equal(t, []string{tt.field}, fieldsOf(t, &inputs))
		})
	}
}

func TestIdentityDifferences(t *testing.T) {
	t.Parallel()

	base := validInputs().Normalize()
	assert.Empty(t, base.IdentityDifferences(&base))

	sameIdentity := base
	sameIdentity.Timezone = "America/Chicago"
	sameIdentity.AdminEmail = "DANA@acme.example"
	assert.Empty(t, base.IdentityDifferences(&sameIdentity))

	changed := base
	changed.OrganizationName = "Other"
	changed.AdminEmail = "someone@acme.example"
	changed.AdminName = "Someone"
	require.Equal(
		t,
		[]string{FieldOrganizationName, FieldAdminEmail, FieldAdminName},
		base.IdentityDifferences(&changed),
	)
}
