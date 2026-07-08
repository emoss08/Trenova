package ediservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/editransport"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
)

func TestTransportModuleWiresAllDeliverableMethods(t *testing.T) {
	t.Parallel()

	var dispatcher services.EDITransportDispatcher
	app := fxtest.New(t, editransport.Module, fx.Populate(&dispatcher))
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
