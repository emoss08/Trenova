package entitlementtest

import (
	"context"
	"reflect"
	"sync"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/testsuite"
)

const fillDepth = 3

var (
	pulidType   = reflect.TypeFor[pulid.ID]()
	contextType = reflect.TypeFor[context.Context]()
	errorType   = reflect.TypeFor[error]()
	orgFields   = [...]string{"OrganizationID", "OrgID"}
	buFields    = [...]string{"BusinessUnitID", "BuID"}
)

type fakeActivities struct {
	world    *World
	mu       sync.Mutex
	answered map[string]bool
}

func (f *fakeActivities) register(env *testsuite.TestWorkflowEnvironment, acts any) {
	value := reflect.ValueOf(acts)
	typ := value.Type()
	for i := range typ.NumMethod() {
		method := typ.Method(i)
		if !method.IsExported() {
			continue
		}
		bound := value.Method(i).Type()
		if bound.NumIn() == 0 || bound.In(0) != contextType || !returnsError(bound) {
			continue
		}
		name := method.Name
		fake := reflect.MakeFunc(bound, func(args []reflect.Value) []reflect.Value {
			return f.answer(name, bound, args)
		})
		env.RegisterActivityWithOptions(fake.Interface(), activity.RegisterOptions{
			Name:                          name,
			DisableAlreadyRegisteredCheck: true,
		})
	}
}

func returnsError(fn reflect.Type) bool {
	return fn.NumOut() > 0 && fn.Out(fn.NumOut()-1) == errorType
}

func (f *fakeActivities) answer(
	name string,
	fn reflect.Type,
	args []reflect.Value,
) []reflect.Value {
	values := make([]any, 0, len(args))
	for _, arg := range args[1:] {
		if arg.CanInterface() {
			values = append(values, arg.Interface())
		}
	}
	_, scoped := dbscope.TenantOf(values...)
	list := !scoped && f.firstAnswer(name)

	out := make([]reflect.Value, fn.NumOut())
	for i := range fn.NumOut() {
		typ := fn.Out(i)
		if typ == errorType {
			out[i] = reflect.Zero(typ)
			continue
		}
		out[i] = f.fill(typ, list, 0)
	}
	return out
}

func (f *fakeActivities) firstAnswer(name string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.answered[name] {
		return false
	}
	f.answered[name] = true
	return true
}

func (f *fakeActivities) fill(typ reflect.Type, list bool, depth int) reflect.Value {
	switch typ.Kind() {
	case reflect.Pointer:
		ptr := reflect.New(typ.Elem())
		ptr.Elem().Set(f.fill(typ.Elem(), list, depth))
		return ptr
	case reflect.Slice:
		if list && tenantBearing(typ.Elem()) {
			return f.tenantSlice(typ)
		}
		return reflect.Zero(typ)
	case reflect.Struct:
		out := reflect.New(typ).Elem()
		if depth >= fillDepth || typ == pulidType {
			return out
		}
		for i := range typ.NumField() {
			field := typ.Field(i)
			if !field.IsExported() {
				continue
			}
			switch field.Type.Kind() {
			case reflect.Slice, reflect.Struct, reflect.Pointer:
				out.Field(i).Set(f.fill(field.Type, list, depth+1))
			default:
			}
		}
		return out
	default:
		return reflect.Zero(typ)
	}
}

func (f *fakeActivities) tenantSlice(typ reflect.Type) reflect.Value {
	tenants := f.world.Tenants()
	out := reflect.MakeSlice(typ, 0, len(tenants))
	for _, tenant := range tenants {
		out = reflect.Append(out, tenantItem(typ.Elem(), tenant))
	}
	return out
}

func tenantItem(typ reflect.Type, tenant dbscope.Tenant) reflect.Value {
	if typ.Kind() == reflect.Pointer {
		ptr := reflect.New(typ.Elem())
		ptr.Elem().Set(tenantItem(typ.Elem(), tenant))
		return ptr
	}
	item := reflect.New(typ).Elem()
	setTenant(item, tenant, 0)
	return item
}

func setTenant(v reflect.Value, tenant dbscope.Tenant, depth int) {
	typ := v.Type()
	for i := range typ.NumField() {
		field := typ.Field(i)
		if !field.IsExported() {
			continue
		}
		target := v.Field(i)
		switch {
		case field.Type == pulidType && isOneOf(field.Name, orgFields[:]):
			target.Set(reflect.ValueOf(tenant.OrganizationID))
		case field.Type == pulidType && isOneOf(field.Name, buFields[:]):
			target.Set(reflect.ValueOf(tenant.BusinessUnitID))
		case field.Type == pulidType:
			target.Set(reflect.ValueOf(pulid.MustNew("tst_")))
		case field.Type.Kind() == reflect.Struct && depth < fillDepth && hasTenantFields(field.Type):
			setTenant(target, tenant, depth+1)
		default:
		}
	}
}

func tenantBearing(typ reflect.Type) bool {
	if typ.Kind() == reflect.Pointer {
		typ = typ.Elem()
	}
	if typ.Kind() != reflect.Struct || typ == pulidType {
		return false
	}
	if hasTenantFields(typ) {
		return true
	}
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.IsExported() && field.Type.Kind() == reflect.Struct &&
			hasTenantFields(field.Type) {
			return true
		}
	}
	return false
}

func hasTenantFields(typ reflect.Type) bool {
	if typ.Kind() != reflect.Struct {
		return false
	}
	return hasPulid(typ, orgFields[:]) && hasPulid(typ, buFields[:])
}

func hasPulid(typ reflect.Type, names []string) bool {
	for _, name := range names {
		if field, ok := typ.FieldByName(name); ok && field.Type == pulidType {
			return true
		}
	}
	return false
}

func isOneOf(name string, names []string) bool {
	for _, n := range names {
		if n == name {
			return true
		}
	}
	return false
}

type nopLogger struct{}

func (nopLogger) Debug(string, ...any) {}
func (nopLogger) Info(string, ...any)  {}
func (nopLogger) Warn(string, ...any)  {}
func (nopLogger) Error(string, ...any) {}
