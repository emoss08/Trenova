package reconcile_test

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/emoss08/trenova/operator/internal/controlplanetest"
	"github.com/emoss08/trenova/operator/internal/instanceapi"
)

func TestEveryReportSaysWhatThisOperatorCanDo(t *testing.T) {
	t.Parallel()
	spec := controlplanetest.Spec(uuid.New())
	w := newWorldWith(t, spec, nil)

	w.loop.Tick(t.Context())
	w.health.set(true)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())

	invalid := "Not A Host"
	spec.CustomHostname = &invalid
	w.cp.SetSpec(spec)
	w.now = w.now.Add(15 * time.Second)
	w.loop.Tick(t.Context())

	reports := w.cp.Reports()
	require.Len(t, reports, 3, "the first report, the ready one and the refusal")
	require.NotNil(t, reports[2].Refused, "the last is the refused revision's report")
	for i, r := range reports {
		require.NotNil(t, r.Capabilities, "report %d", i)
		require.Equal(t,
			[]instanceapi.StatusReportCapabilities{instanceapi.CapabilityCustomHostname},
			*r.Capabilities, "report %d", i)
	}
}
