package tms_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/tmstest"
)

const (
	namespace = "trenova"
	secret    = "trenova-secrets"
)

func secretWith(token string) *corev1.Secret {
	data := map[string][]byte{"sessionSecret": []byte("x")}
	if token != "" {
		data[tms.TokenKey] = []byte(token)
	}
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: secret, Namespace: namespace},
		Data:       data,
	}
}

func client(t *testing.T, kube *fake.Clientset, url string) *tms.Client {
	t.Helper()
	c, err := tms.New(tms.Params{
		URL: url,
		Tokens: tms.NewSecretTokens(tms.SecretTokensParams{
			Client: kube, Namespace: namespace, Name: secret,
		}),
	})
	require.NoError(t, err)
	return c
}

func TestReadsAndSendsState(t *testing.T) {
	t.Parallel()
	srv := tmstest.New(t, "token-1")
	hour := time.Date(2026, 11, 5, 13, 0, 0, 0, time.UTC)
	srv.SetReadings([]tms.Reading{{Meter: "shipments.total", Hour: hour, Quantity: 12}})
	c := client(t, fake.NewClientset(secretWith("token-1")), srv.URL+"/")

	readings, err := c.Readings(t.Context(), hour.Add(-2*time.Hour))
	require.NoError(t, err)
	require.Equal(t, []tms.Reading{{Meter: "shipments.total", Hour: hour, Quantity: 12}}, readings)
	require.Equal(t, []time.Time{hour.Add(-2 * time.Hour)}, srv.Since())

	window := tms.Window{Start: hour.Add(-13 * time.Hour), End: hour.Add(11 * time.Hour)}
	state := tms.State{Window: &window, Overage: true, Meters: []tms.Meter{{
		Meter: "shipments.total", Unit: "count", Quantity: 12, Allowance: 100,
	}}}
	require.NoError(t, c.PutState(t.Context(), state))
	require.NoError(t, c.PutState(t.Context(), tms.State{}))
	states := srv.States()
	require.Len(t, states, 2)
	require.Equal(t, state.Meters, states[0].Meters)
	require.True(t, states[0].Window.Start.Equal(window.Start))
	require.Nil(t, states[1].Window, "no window is sent when there is none")
	require.Empty(t, states[1].Meters)
}

func TestRereadsTheTokenWhenRefused(t *testing.T) {
	t.Parallel()
	srv := tmstest.New(t, "old")
	kube := fake.NewClientset(secretWith("old"))
	c := client(t, kube, srv.URL)
	_, err := c.Readings(t.Context(), time.Now().Truncate(time.Hour))
	require.NoError(t, err)

	srv.SetToken("new")
	_, err = kube.CoreV1().
		Secrets(namespace).
		Update(t.Context(), secretWith("new"), metav1.UpdateOptions{})
	require.NoError(t, err)
	_, err = c.Readings(t.Context(), time.Now().Truncate(time.Hour))
	require.NoError(t, err)
	require.Equal(t, 1, srv.Refused(), "one refusal, then the Secret is read again")

	srv.SetToken("rotated-elsewhere")
	_, err = c.Readings(t.Context(), time.Now().Truncate(time.Hour))
	require.ErrorIs(t, err, tms.ErrUnauthorized)
	require.Equal(t, 3, srv.Refused(), "it retries once with the token it read again")
}

func TestErrors(t *testing.T) {
	t.Parallel()
	srv := tmstest.New(t, "token")
	cases := []struct {
		name   string
		secret *corev1.Secret
		since  time.Time
		fail   bool
		want   error
		status int
	}{
		{name: "no Secret", want: tms.ErrNoToken},
		{name: "no key", secret: secretWith(""), want: tms.ErrNoToken},
		{
			name:   "invalid since",
			secret: secretWith("token"),
			since:  time.Date(2026, 11, 5, 13, 30, 0, 0, time.UTC),
			status: 400,
		},
		{
			name:   "unavailable",
			secret: secretWith("token"),
			since:  time.Date(2026, 11, 5, 13, 0, 0, 0, time.UTC),
			fail:   true,
			status: 503,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			kube := fake.NewClientset()
			if tc.secret != nil {
				kube = fake.NewClientset(tc.secret)
			}
			if tc.fail {
				srv.FailNext(1)
			}
			_, err := client(t, kube, srv.URL).Readings(t.Context(), tc.since)
			require.Error(t, err)
			if tc.want != nil {
				require.ErrorIs(t, err, tc.want)
			}
			if tc.status != 0 {
				var se *tms.StatusError
				require.ErrorAs(t, err, &se)
				require.Equal(t, tc.status, se.Status)
				require.NotEmpty(t, se.Title)
			}
		})
	}
}

func TestRefusesABadURL(t *testing.T) {
	t.Parallel()
	tokens := tms.NewSecretTokens(tms.SecretTokensParams{Client: fake.NewClientset()})
	for _, u := range []string{"", "trenova-api:8090", "ftp://trenova-api:8090", "http://"} {
		_, err := tms.New(tms.Params{URL: u, Tokens: tokens})
		require.Error(t, err, u)
	}
	_, err := tms.New(tms.Params{URL: "http://trenova-api.trenova.svc:8090"})
	require.Error(t, err, "a token source is required")
}

const inviteHash = "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"

func TestInstallsTheFirstAdministratorsInvite(t *testing.T) {
	t.Parallel()
	srv := tmstest.New(t, "token")
	kube := fake.NewClientset(secretWith("token"))
	c := client(t, kube, srv.URL)
	expires := time.Date(2026, 11, 14, 10, 0, 0, 0, time.FixedZone("EST", -5*3600))

	admin, err := c.InstallInvite(
		t.Context(),
		tms.Invite{TokenSHA256: inviteHash, ExpiresAt: expires},
	)
	require.NoError(t, err)
	require.Equal(t, tms.Administrator{InviteTokenSHA256: inviteHash}, admin)
	invites := srv.Invites()
	require.Len(t, invites, 1)
	require.Equal(t, inviteHash, invites[0].TokenSHA256)
	require.True(t, invites[0].ExpiresAt.Equal(expires))

	srv.SpoilInvite(inviteHash)
	admin, err = c.InstallInvite(
		t.Context(),
		tms.Invite{TokenSHA256: inviteHash, ExpiresAt: expires},
	)
	require.NoError(t, err)
	require.Equal(t, tms.Administrator{}, admin, "a used invite is no longer reported")

	srv.SetAdministratorSetUp(true)
	admin, err = c.InstallInvite(
		t.Context(),
		tms.Invite{TokenSHA256: inviteHash, ExpiresAt: expires},
	)
	require.NoError(t, err)
	require.Equal(t, tms.Administrator{SetUp: true}, admin)

	srv.FailNext(1)
	_, err = c.InstallInvite(t.Context(), tms.Invite{TokenSHA256: inviteHash, ExpiresAt: expires})
	var se *tms.StatusError
	require.ErrorAs(t, err, &se)
	require.Equal(t, 503, se.Status)

	_, err = c.InstallInvite(t.Context(), tms.Invite{TokenSHA256: "short", ExpiresAt: expires})
	require.ErrorAs(t, err, &se)
	require.Equal(t, 400, se.Status)
}
