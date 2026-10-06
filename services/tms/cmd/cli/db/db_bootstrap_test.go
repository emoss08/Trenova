package db

import (
	"errors"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/instancebootstrap"
	"github.com/emoss08/trenova/internal/core/services/instancebootstrapservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadPasswordLine(t *testing.T) {
	t.Parallel()

	got, err := readPasswordLine(strings.NewReader("Tr4ck-the-l0ads-north!\r\nignored\n"))
	require.NoError(t, err)
	assert.Equal(t, "Tr4ck-the-l0ads-north!", got)

	got, err = readPasswordLine(strings.NewReader("  spaced secret  "))
	require.NoError(t, err)
	assert.Equal(t, "  spaced secret  ", got)

	_, err = readPasswordLine(strings.NewReader("\n"))
	require.ErrorIs(t, err, errNoBootstrapPassword)

	_, err = readPasswordLine(strings.NewReader(""))
	require.ErrorIs(t, err, errNoBootstrapPassword)
}

func TestDescribeBootstrapErrorNamesTheFlagAndVariable(t *testing.T) {
	t.Parallel()

	multiErr := errortypes.NewMultiError()
	multiErr.Add(instancebootstrap.FieldAdminEmail, errortypes.ErrRequired, "Administrator email is required")
	multiErr.Add("password", errortypes.ErrInvalid, "Password must be at least 12 characters")

	message := describeBootstrapError(multiErr).Error()
	assert.Contains(t, message, "Administrator email is required (--admin-email or TRENOVA_BOOTSTRAP_ADMIN_EMAIL)")
	assert.Contains(t, message, "(TRENOVA_BOOTSTRAP_ADMIN_PASSWORD or --password-stdin)")
}

func TestDescribeBootstrapErrorKeepsRefusalsWrapped(t *testing.T) {
	t.Parallel()

	err := describeBootstrapError(instancebootstrapservice.ErrAlreadyInitialized)
	require.ErrorIs(t, err, instancebootstrapservice.ErrAlreadyInitialized)
	assert.True(t, strings.HasPrefix(err.Error(), "bootstrap refused"))
	assert.False(t, errors.Is(err, instancebootstrapservice.ErrInputsDiffer))
}

func TestResolveBootstrapInputsPrefersFlagsOverEnvironment(t *testing.T) {
	t.Setenv("TRENOVA_BOOTSTRAP_ORG_NAME", "From Env")
	t.Setenv("TRENOVA_BOOTSTRAP_ADMIN_EMAIL", "env@acme.example")
	t.Setenv("TRENOVA_BOOTSTRAP_TIMEZONE", "America/Chicago")

	cmd := &cobra.Command{}
	cmd.Flags().AddFlagSet(dbBootstrapCmd.Flags())
	t.Cleanup(func() {
		for _, input := range bootstrapInputs {
			input.value = ""
		}
	})
	require.NoError(t, cmd.Flags().Set("org-name", "From Flag"))

	inputs := resolveBootstrapInputs(cmd)
	assert.Equal(t, "From Flag", inputs.OrganizationName)
	assert.Equal(t, "env@acme.example", inputs.AdminEmail)
	assert.Equal(t, "America/Chicago", inputs.Timezone)
	assert.Empty(t, inputs.AdminName)
}

func TestEveryBootstrapInputHasAFlagAndVariable(t *testing.T) {
	t.Parallel()

	seen := make(map[string]struct{}, len(bootstrapInputs))
	for _, input := range bootstrapInputs {
		assert.NotNil(t, dbBootstrapCmd.Flags().Lookup(input.flag), input.flag)
		assert.True(t, strings.HasPrefix(input.env, "TRENOVA_BOOTSTRAP_"), input.env)
		seen[input.field] = struct{}{}
	}
	assert.Len(t, seen, 10)
	assert.Nil(t, dbBootstrapCmd.Flags().Lookup("password"))
}
