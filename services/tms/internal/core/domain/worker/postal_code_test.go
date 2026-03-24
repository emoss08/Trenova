package worker

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidatePostalCode_ValidFiveDigit(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("12345")
	assert.NoError(t, err)
}

func TestValidatePostalCode_ValidWithExtension(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("12345-6789")
	assert.NoError(t, err)
}

func TestValidatePostalCode_EmptyString(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("")
	assert.NoError(t, err)
}

func TestValidatePostalCode_InvalidAlpha(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("ABCDE")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "valid US postal code")
}

func TestValidatePostalCode_TooShort(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("1234")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "valid US postal code")
}

func TestValidatePostalCode_TooLong(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("123456")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "valid US postal code")
}

func TestValidatePostalCode_InvalidExtension(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("12345-123")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "valid US postal code")
}

func TestValidatePostalCode_InvalidExtensionTooLong(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("12345-12345")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "valid US postal code")
}

func TestValidatePostalCode_DashButNoExtension(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("12345-")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "valid US postal code")
}

func TestValidatePostalCode_DashOnly(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("-")
	assert.Error(t, err)
}

func TestValidatePostalCode_SpecialChars(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("123!5")
	assert.Error(t, err)
}

func TestValidatePostalCode_Spaces(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("123 5")
	assert.Error(t, err)
}

func TestValidatePostalCode_LeadingZeros(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("01234")
	assert.NoError(t, err)
}

func TestValidatePostalCode_AllZeros(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("00000")
	assert.NoError(t, err)
}

func TestValidatePostalCode_AllNines(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("99999")
	assert.NoError(t, err)
}

func TestValidatePostalCode_ExtensionAllZeros(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("12345-0000")
	assert.NoError(t, err)
}

func TestValidatePostalCode_NonStringType(t *testing.T) {
	t.Parallel()

	err := validatePostalCode(12345)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string")
}

func TestValidatePostalCode_NilValue(t *testing.T) {
	t.Parallel()

	err := validatePostalCode(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be a string")
}

func TestValidatePostalCode_MixedAlphaNumeric(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("1234A")
	assert.Error(t, err)
}

func TestValidatePostalCode_DoubleDash(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("12345--6789")
	assert.Error(t, err)
}

func TestValidatePostalCode_CanadianFormat(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("K1A 0B1")
	assert.Error(t, err)
}

func TestValidatePostalCode_UKFormat(t *testing.T) {
	t.Parallel()

	err := validatePostalCode("SW1A 1AA")
	assert.Error(t, err)
}
