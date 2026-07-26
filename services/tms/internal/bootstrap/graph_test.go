package bootstrap

import (
	"testing"

	"go.uber.org/fx"
)

// TestAppGraphResolves catches a service that declares a dependency nothing
// provides — the failure otherwise only surfaces when the API starts.
func TestAppGraphResolves(t *testing.T) {
	if err := fx.ValidateApp(Options(), APIOptions()); err != nil {
		t.Fatalf("fx graph invalid: %v", err)
	}
}
