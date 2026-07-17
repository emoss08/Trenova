package bootstrap

import (
	"testing"

	"go.uber.org/fx"
)

// TestDependencyGraphs validates that the FX dependency graphs for both the
// API and worker processes resolve — every constructor's inputs are provided
// — without instantiating anything.
func TestDependencyGraphs(t *testing.T) {
	tests := []struct {
		name string
		opts fx.Option
	}{
		{name: "worker", opts: fx.Options(Options(), WorkerOptions())},
		{name: "api", opts: fx.Options(Options(), APIOptions())},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := fx.ValidateApp(tt.opts); err != nil {
				t.Fatalf("fx graph does not resolve: %v", err)
			}
		})
	}
}
