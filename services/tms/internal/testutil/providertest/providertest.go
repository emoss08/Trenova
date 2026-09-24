package providertest

import (
	"reflect"
	"testing"

	"github.com/emoss08/trenova/shared/reflectutils"
)

func Build(t testing.TB, provider any, supplied map[reflect.Type]reflect.Value) any {
	t.Helper()

	built, err := reflectutils.Construct(provider, supplied)
	if err != nil {
		t.Fatalf("%v", err)
	}

	return built
}
