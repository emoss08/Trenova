package tenant

import (
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
)

func bootstrappedOrganization() *Organization {
	org := validOrganization()
	org.AddressLine1 = ""
	org.City = PlaceholderCity
	org.PostalCode = PlaceholderPostalCode
	org.ScacCode = PlaceholderSCAC
	org.DOTNumber = PlaceholderDOTNumber
	return org
}

func TestSetupOfACompleteOrganizationOwesNothing(t *testing.T) {
	t.Parallel()

	setup := validOrganization().Setup()

	assert.True(t, setup.Complete)
	assert.Empty(t, setup.Missing)
	assert.False(t, setup.CanComplete, "who may complete it is the caller's to say")
}

func TestSetupOfABootstrappedOrganizationNamesEveryPlaceholderAndBlank(t *testing.T) {
	t.Parallel()

	setup := bootstrappedOrganization().Setup()

	assert.False(t, setup.Complete)
	assert.Equal(t, []string{
		SetupFieldAddressLine1,
		SetupFieldCity,
		SetupFieldPostalCode,
		SetupFieldSCACCode,
		SetupFieldDOTNumber,
	}, setup.Missing)
}

func TestSetupTreatsWhitespaceAndAMissingStateAsUnset(t *testing.T) {
	t.Parallel()

	org := validOrganization()
	org.Name = "   "
	org.Timezone = ""
	org.StateID = ""

	assert.Equal(t, []string{
		SetupFieldName,
		SetupFieldTimezone,
		SetupFieldStateID,
	}, org.SetupMissing())
}

func TestSetupReadsAPlaceholderSCACInAnyCase(t *testing.T) {
	t.Parallel()

	org := validOrganization()
	org.ScacCode = "tbdx"

	assert.Equal(t, []string{SetupFieldSCACCode}, org.SetupMissing())
}

func TestEveryFieldSetupNamesIsOneTheValidatorRefuses(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	bootstrappedOrganization().Validate(multiErr)

	assert.True(t, multiErr.HasErrors(), "an incomplete organization cannot be saved as is")
}
