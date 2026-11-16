package tms_test

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/emoss08/trenova/operator/internal/tms"
	"github.com/emoss08/trenova/operator/internal/tmstest"
)

const bouncedRecipient = "dispatcher@acme.example"

func bounce(id string) tms.EmailEvent {
	return tms.EmailEvent{
		ID:       id,
		Provider: "Postmark",
		Payload:  `{"RecordType":"Bounce","Email":"` + bouncedRecipient + `"}`,
	}
}

func TestRelaysEmailEventsAndReadsWhichWereHandled(t *testing.T) {
	t.Parallel()
	srv := tmstest.New(t, "token")
	c := client(t, fake.NewClientset(secretWith("token")), srv.URL)
	srv.LeaveEmailEventsUnhandled("b")

	handled, err := c.RelayEmailEvents(t.Context(), []tms.EmailEvent{bounce("a"), bounce("b")})

	require.NoError(t, err)
	require.Equal(t, []string{"a"}, handled)
	require.Equal(t, []tms.EmailEvent{bounce("a"), bounce("b")}, srv.EmailEvents())
}

func TestARefusedRelayNamesNoPayload(t *testing.T) {
	t.Parallel()
	srv := tmstest.New(t, "token")
	c := client(t, fake.NewClientset(secretWith("token")), srv.URL)
	srv.FailEmailEventsNext(1)

	_, err := c.RelayEmailEvents(t.Context(), []tms.EmailEvent{bounce("a")})

	var se *tms.StatusError
	require.ErrorAs(t, err, &se)
	require.NotContains(t, err.Error(), bouncedRecipient)
}

func TestAnEmailEventNeverPrintsItsPayload(t *testing.T) {
	t.Parallel()
	e := bounce("a")
	for _, printed := range []string{
		fmt.Sprintf("%v", e),
		fmt.Sprintf("%+v", e),
		fmt.Sprintf("%#v", e),
	} {
		require.NotContains(t, printed, bouncedRecipient)
	}
}
