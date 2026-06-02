package bootstrap

import (
	"testing"

	"go.uber.org/fx"
)

func TestAPIOptionsValidate(t *testing.T) {
	t.Parallel()

	if err := fx.ValidateApp(Options(), APIOptions(), fx.NopLogger); err != nil {
		t.Fatal(err)
	}
}
