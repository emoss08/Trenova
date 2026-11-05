package release_test

import (
	"testing"

	"github.com/sigstore/sigstore-go/pkg/testing/ca"
	"github.com/sigstore/sigstore-go/pkg/verify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"helm.sh/helm/v3/pkg/chart"

	"github.com/emoss08/trenova/operator/internal/release"
)

const (
	actionsIssuer = "https://token.actions.githubusercontent.com"
	releaseSigner = "https://github.com/emoss08/Trenova/.github/workflows/release.yml@refs/tags/v0.11.0"
)

func sigstoreWorld(t *testing.T) (*ca.VirtualSigstore, *release.Sigstore) {
	t.Helper()
	virtual, err := ca.NewVirtualSigstore()
	require.NoError(t, err)
	return virtual, release.NewSigstoreFor(
		virtual,
		verify.WithTransparencyLog(1),
		verify.WithObserverTimestamps(1),
	)
}

func TestAcceptsAChartSignedByTheReleaseWorkflow(t *testing.T) {
	t.Parallel()
	virtual, verifier := sigstoreWorld(t)
	archive := []byte("the chart archive")

	entity, err := virtual.Sign(releaseSigner, actionsIssuer, archive)
	require.NoError(t, err)

	require.NoError(t, verifier.Check(entity, archive))
}

func TestRefusesAChartOtherThanTheOneSigned(t *testing.T) {
	t.Parallel()
	virtual, verifier := sigstoreWorld(t)

	entity, err := virtual.Sign(releaseSigner, actionsIssuer, []byte("the chart archive"))
	require.NoError(t, err)

	require.Error(t, verifier.Check(entity, []byte("a chart moved onto the tag")))
}

func TestRefusesASignatureFromAnyoneButTheReleaseWorkflow(t *testing.T) {
	t.Parallel()
	archive := []byte("the chart archive")
	cases := map[string]struct{ identity, issuer string }{
		"another workflow": {
			"https://github.com/emoss08/Trenova/.github/workflows/operator.yml@refs/tags/v0.11.0",
			actionsIssuer,
		},
		"a branch": {
			"https://github.com/emoss08/Trenova/.github/workflows/release.yml@refs/heads/master",
			actionsIssuer,
		},
		"a fork": {
			"https://github.com/someone/Trenova/.github/workflows/release.yml@refs/tags/v0.11.0",
			actionsIssuer,
		},
		"another issuer": {releaseSigner, "https://accounts.google.com"},
		"a person":       {"someone@example.com", "https://accounts.google.com"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			virtual, verifier := sigstoreWorld(t)
			entity, err := virtual.Sign(tc.identity, tc.issuer, archive)
			require.NoError(t, err)

			require.Error(t, verifier.Check(entity, archive))
		})
	}
}

func TestRefusesAChartWithoutASignature(t *testing.T) {
	t.Parallel()
	_, verifier := sigstoreWorld(t)

	require.ErrorIs(t, verifier.Verify([]byte("chart"), nil), release.ErrChartUnsigned)
}

func TestRefusesASignatureThatIsNotABundle(t *testing.T) {
	t.Parallel()
	_, verifier := sigstoreWorld(t)

	require.Error(t, verifier.Verify([]byte("chart"), []byte("not a bundle")))
}

func TestRefusesAChartThatNamesAnotherRelease(t *testing.T) {
	t.Parallel()
	ch := &chart.Chart{Metadata: &chart.Metadata{Name: "trenova", Version: "0.10.0"}}

	require.NoError(t, release.NamesRelease(ch, "trenova", "0.10.0"))
	assert.ErrorIs(t, release.NamesRelease(ch, "trenova", "0.11.0"), release.ErrChartIdentity)
	assert.ErrorIs(
		t,
		release.NamesRelease(ch, "trenova-operator", "0.10.0"),
		release.ErrChartIdentity,
	)
}
