package providertest

import (
	"reflect"
	"testing"
)

func Build(t testing.TB, provider any, supplied map[reflect.Type]reflect.Value) any {
	t.Helper()

	fn := reflect.ValueOf(provider)
	if fn.Kind() != reflect.Func || fn.Type().NumOut() == 0 {
		t.Fatalf("%T is not a constructor", provider)
	}

	args := make([]reflect.Value, fn.Type().NumIn())
	for idx := range args {
		in := fn.Type().In(idx)
		if value, ok := supplied[in]; ok {
			args[idx] = value
			continue
		}
		args[idx] = reflect.Zero(in)
	}

	var out []reflect.Value
	func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				t.Fatalf("%s panicked when built without dependencies: %v", fn.Type(), recovered)
			}
		}()
		out = fn.Call(args)
	}()

	return out[0].Interface()
}
