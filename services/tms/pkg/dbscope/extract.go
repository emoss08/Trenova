package dbscope

import (
	"reflect"

	"github.com/emoss08/trenova/shared/pulid"
)

const maxExtractDepth = 3

type TenantScoped interface {
	DBTenant() Tenant
}

var (
	pulidType        = reflect.TypeFor[pulid.ID]()
	tenantScopedType = reflect.TypeFor[TenantScoped]()
	orgFieldNames    = [...]string{"OrganizationID", "OrgID"}
	buFieldNames     = [...]string{"BusinessUnitID", "BuID"}
)

func TenantOf(values ...any) (Tenant, bool) {
	var acc tenantAccumulator

	for _, value := range values {
		if value == nil {
			continue
		}

		tenant, ok, conflict := tenantOfValue(reflect.ValueOf(value), 0)
		if conflict || (ok && !acc.add(tenant)) {
			return Tenant{}, false
		}
	}

	return acc.tenant, acc.resolved
}

type tenantAccumulator struct {
	tenant   Tenant
	resolved bool
}

func (a *tenantAccumulator) add(tenant Tenant) bool {
	if a.resolved && !sameTenant(a.tenant, tenant) {
		return false
	}
	if !a.resolved || a.tenant.UserID.IsNil() {
		a.tenant = tenant
	}
	a.resolved = true

	return true
}

func tenantOfValue(v reflect.Value, depth int) (tenant Tenant, ok, conflict bool) {
	v, tenant, ok, done := unwrapScoped(v)
	if done {
		return tenant, ok, false
	}

	if v.Kind() != reflect.Struct {
		return Tenant{}, false, false
	}

	if tenant, ok = explicitTenant(v); ok {
		return tenant, true, false
	}

	direct := Tenant{
		OrganizationID: firstPulid(v, orgFieldNames[:]),
		BusinessUnitID: firstPulid(v, buFieldNames[:]),
		UserID:         pulidField(v, "UserID"),
	}
	if direct.Valid() {
		return direct, true, false
	}

	if depth >= maxExtractDepth {
		return Tenant{}, false, false
	}

	return nestedTenant(v, depth)
}

func unwrapScoped(v reflect.Value) (unwrapped reflect.Value, tenant Tenant, ok, done bool) {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return v, Tenant{}, false, true
		}
		if tenant, ok = explicitTenant(v); ok {
			return v, tenant, true, true
		}
		v = v.Elem()
	}

	return v, Tenant{}, false, false
}

func explicitTenant(v reflect.Value) (Tenant, bool) {
	if !v.CanInterface() || !v.Type().Implements(tenantScopedType) {
		return Tenant{}, false
	}

	scoped, ok := v.Interface().(TenantScoped)
	if !ok {
		return Tenant{}, false
	}

	tenant := scoped.DBTenant()

	return tenant, tenant.Valid()
}

func nestedTenant(v reflect.Value, depth int) (tenant Tenant, ok, conflict bool) {
	var acc tenantAccumulator
	typ := v.Type()

	for i := range v.NumField() {
		if field := typ.Field(i); !holdsStruct(&field) {
			continue
		}

		found, fieldOK, fieldConflict := tenantOfValue(v.Field(i), depth+1)
		if fieldConflict || (fieldOK && !acc.add(found)) {
			return Tenant{}, false, true
		}
	}

	return acc.tenant, acc.resolved, false
}

func holdsStruct(field *reflect.StructField) bool {
	if !field.IsExported() || field.Type == pulidType {
		return false
	}

	kind := field.Type.Kind()
	if kind == reflect.Pointer {
		kind = field.Type.Elem().Kind()
	}

	return kind == reflect.Struct
}

func sameTenant(a, b Tenant) bool {
	return a.OrganizationID == b.OrganizationID && a.BusinessUnitID == b.BusinessUnitID &&
		(a.UserID.IsNil() || b.UserID.IsNil() || a.UserID == b.UserID)
}

func firstPulid(v reflect.Value, names []string) pulid.ID {
	for _, name := range names {
		if id := pulidField(v, name); !id.IsNil() {
			return id
		}
	}

	return pulid.Nil
}

func pulidField(v reflect.Value, name string) pulid.ID {
	field := v.FieldByName(name)
	if !field.IsValid() || field.Type() != pulidType {
		return pulid.Nil
	}

	id, _ := field.Interface().(pulid.ID)

	return id
}
