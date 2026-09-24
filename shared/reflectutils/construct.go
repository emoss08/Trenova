package reflectutils

import (
	"errors"
	"fmt"
	"reflect"
)

var ErrNotConstructor = errors.New("not a constructor")

func Construct(constructor any, supplied map[reflect.Type]reflect.Value) (built any, err error) {
	fn := reflect.ValueOf(constructor)
	if fn.Kind() != reflect.Func || fn.Type().NumOut() == 0 {
		return nil, fmt.Errorf("%T: %w", constructor, ErrNotConstructor)
	}

	fnType := fn.Type()
	args := make([]reflect.Value, fnType.NumIn())
	for idx := range args {
		in := fnType.In(idx)
		if value, ok := supplied[in]; ok {
			args[idx] = value
			continue
		}
		args[idx] = reflect.Zero(in)
	}

	defer func() {
		if recovered := recover(); recovered != nil {
			built = nil
			err = fmt.Errorf("%s panicked when built without dependencies: %v", fnType, recovered)
		}
	}()

	return fn.Call(args)[0].Interface(), nil
}
