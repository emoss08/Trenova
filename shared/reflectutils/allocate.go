package reflectutils

import (
	"errors"
	"fmt"
	"reflect"
)

var ErrNotStructPointer = errors.New("not a non-nil pointer to a struct")

func AllocatePointers(target any) error {
	value := reflect.ValueOf(target)
	if value.Kind() != reflect.Pointer || value.IsNil() || value.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("%T: %w", target, ErrNotStructPointer)
	}

	fields := value.Elem()
	for idx := range fields.NumField() {
		field := fields.Field(idx)
		if field.Kind() != reflect.Pointer || !field.IsNil() || !field.CanSet() {
			continue
		}
		field.Set(reflect.New(field.Type().Elem()))
	}

	return nil
}
