package edidelivery

import (
	"testing"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/modules/edi/app/editransport"
	"github.com/emoss08/trenova/internal/modules/edi/ports/editransportports"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestTransportModuleWiresAllDeliverableMethods(t *testing.T) {
	t.Parallel()

	var dispatcher editransportports.EDITransportDispatcher
	app := fxtest.New(
		t,
		fx.Supply(&config.Config{}),
		config.SectionProviders,
		editransport.Module,
		fx.Populate(&dispatcher),
	)
	app.RequireStart()
	t.Cleanup(app.RequireStop)

	require.NotNil(t, dispatcher)
	for _, method := range deliverableMethods {
		require.Truef(
			t,
			dispatcher.Supports(method),
			"no EDI transport is registered in the fx graph for connection method %s",
			method,
		)
	}
}
