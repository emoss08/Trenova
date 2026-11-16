package tms_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/tmstest"
)

const supportX = "11qYAYKxCrfVS_7TyWQHOg7hcvPapiMlrwIaaPcHURo"

func TestSendsAndReadsTheSupportAccessKeys(t *testing.T) {
	t.Parallel()
	srv := tmstest.New(t, "token")
	c := client(t, fake.NewClientset(secretWith("token")), srv.URL+"/")

	held, err := c.SupportKeys(t.Context())
	require.NoError(t, err)
	assert.Empty(t, held.InstanceID)
	assert.Equal(t, []string{}, held.KeyIDs, "an instance holding none answers an empty list")

	require.NoError(t, c.PutSupportKeys(t.Context(), tms.SupportKeys{
		InstanceID:  "inst-1",
		StaffDomain: "trenova.app",
		Keys: []tms.SupportKey{
			{Kid: "cp-1", Kty: "OKP", Crv: "Ed25519", Alg: "EdDSA", Use: "sig", X: supportX},
		},
	}))

	held, err = c.SupportKeys(t.Context())
	require.NoError(t, err)
	assert.Equal(t, "inst-1", held.InstanceID)
	assert.Equal(t, "trenova.app", held.StaffDomain)
	assert.Equal(t, []string{"cp-1"}, held.KeyIDs)
	assert.NotEmpty(t, held.ReceivedAt)
}

func TestAnEmptySupportKeySetIsRefused(t *testing.T) {
	t.Parallel()
	srv := tmstest.New(t, "token")
	c := client(t, fake.NewClientset(secretWith("token")), srv.URL+"/")

	err := c.PutSupportKeys(t.Context(), tms.SupportKeys{InstanceID: "inst-1", StaffDomain: "trenova.app"})

	require.Error(t, err)
	assert.Nil(t, srv.SupportKeys())
}

func TestReadsTheLiveSupportGrants(t *testing.T) {
	t.Parallel()
	srv := tmstest.New(t, "token")
	c := client(t, fake.NewClientset(secretWith("token")), srv.URL+"/")

	grants, err := c.SupportGrants(t.Context())
	require.NoError(t, err)
	assert.Equal(t, []tms.SupportGrant{}, grants)

	ends := time.Date(2027, 1, 15, 8, 0, 0, 0, time.UTC)
	srv.SetSupportGrants([]tms.SupportGrant{{
		ID:               "sag_01",
		OrganizationID:   "org_01JDZ8K3M4N5P6Q7R8S9T0V1W2",
		OrganizationName: "Acme Freight",
		StaffEmail:       "dana@trenova.app",
		Scope:            "read_only",
		ExpiresAt:        ends,
	}})
	grants, err = c.SupportGrants(t.Context())
	require.NoError(t, err)
	require.Len(t, grants, 1)
	assert.Equal(t, "Acme Freight", grants[0].OrganizationName)
	assert.Equal(t, "read_only", grants[0].Scope)
	assert.True(t, grants[0].ExpiresAt.Equal(ends))

	srv.FailSupportGrants(true)
	_, err = c.SupportGrants(t.Context())
	require.Error(t, err)
}
