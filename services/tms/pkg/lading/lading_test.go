package lading_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/pkg/lading"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func starter(t *testing.T, role string) *lading.Encoded {
	t.Helper()
	built, err := lading.Default().Starter(context.Background(), &lading.StarterRequest{
		Name:  "acme",
		Label: "ACME Retail",
		Role:  role,
		Them:  &[2]string{"ZZ", "ACMERETAIL"},
		Us:    &[2]string{"ZZ", "TRENOVA"},
		Zone:  "America/Chicago",
	})
	require.NoError(t, err)
	require.NotEmpty(t, built.Files)
	require.NotEmpty(t, built.Samples)
	encoded, err := lading.Encode(
		&lading.Project{Files: built.Files, Samples: built.Samples},
		&lading.Context{Org: lading.Org{Name: "Trenova", SCAC: "TRNV", Zone: "America/Chicago"}},
	)
	require.NoError(t, err)
	return encoded
}

func TestStarterProjectsCheckClean(t *testing.T) {
	t.Parallel()

	for _, role := range []string{lading.RoleCarrier, lading.RoleBroker} {
		t.Run(role, func(t *testing.T) {
			t.Parallel()
			check, err := lading.Default().Check(context.Background(), starter(t, role))
			require.NoError(t, err)
			assert.Zero(t, check.Errors, "%+v", check.Files)
			require.NotNil(t, check.Partner)
			assert.Equal(t, "acme", check.Partner.Name)
			require.NotNil(t, check.Partner.Interchange)
			assert.Equal(t, "ACMERETAIL", check.Partner.Interchange.Them.ID)
			assert.NotEmpty(t, check.Maps)
			assert.NotEmpty(t, check.Flows)
		})
	}
}

func TestCarrierStarterReceivesTenders(t *testing.T) {
	t.Parallel()

	project := starter(t, lading.RoleCarrier)
	check, err := lading.Default().Check(context.Background(), project)
	require.NoError(t, err)

	flow, receive := check.Receiving("204")
	require.NotNil(t, flow)
	require.NotNil(t, receive)
	assert.Equal(t, "997", receive.Acknowledge)
	inbound := check.Map(receive.Map)
	require.NotNil(t, inbound)
	assert.Equal(t, lading.DirectionInbound, inbound.Direction)
	assert.Equal(t, "loadTender", inbound.Document)

	_, respond := check.Responding("990")
	require.NotNil(t, respond)
	autoDecline := check.Setting("autoDecline")
	require.NotNil(t, autoDecline)
	assert.JSONEq(t, "false", string(autoDecline.Value))
}

func TestTestReportsReadiness(t *testing.T) {
	t.Parallel()

	report, err := lading.Default().Test(context.Background(), starter(t, lading.RoleCarrier))
	require.NoError(t, err)
	assert.Zero(t, report.Readiness.Errors)
	assert.Zero(t, report.Readiness.Failing)
	assert.Zero(t, report.Readiness.UncoveredFlowClauses)
	assert.True(t, report.Readiness.Ready)
}

func TestFormatLeavesBrokenFilesAlone(t *testing.T) {
	t.Parallel()

	formatted, err := lading.Default().Format(context.Background(), "library common\nconst  a = 1\n")
	require.NoError(t, err)
	assert.Nil(t, formatted.Error)
	assert.Equal(t, "library common\nconst a = 1\n", formatted.Text)

	broken, err := lading.Default().Format(context.Background(), "library common\nconst = \n")
	require.NoError(t, err)
	require.NotNil(t, broken.Error)
}

func TestEncodeBoundsProjects(t *testing.T) {
	t.Parallel()

	files := make([]lading.File, lading.MaxFiles+1)
	_, err := lading.Encode(&lading.Project{Files: files}, nil)
	require.Error(t, err)
}
