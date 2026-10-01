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
	var (
		found    Tenant
		resolved bool
	)

	for _, value := range values {
		if value == nil {
			continue
		}

		tenant, ok, conflict := tenantOfValue(reflect.ValueOf(value), 0)
		if conflict {
			return Tenant{}, false
		}
		if !ok {
			continue
		}
		if resolved && !sameTenant(found, tenant) {
			return Tenant{}, false
		}
		if !resolved || found.UserID.IsNil() {
			found = tenant
		}
		resolved = true
	}

	return found, resolved
}

func tenantOfValue(v reflect.Value, depth int) (Tenant, bool, bool) {
	for v.Kind() == reflect.Pointer || v.Kind() == reflect.Interface {
		if v.IsNil() {
			return Tenant{}, false, false
		}
		if v.Type().Implements(tenantScopedType) {
			if scoped, ok := v.Interface().(TenantScoped); ok {
				tenant := scoped.DBTenant()
				return tenant, tenant.Valid(), false
			}
		}
		v = v.Elem()
	}

	if v.Kind() != reflect.Struct {
		return Tenant{}, false, false
	}

	if v.CanInterface() && v.Type().Implements(tenantScopedType) {
		if scoped, ok := v.Interface().(TenantScoped); ok {
			tenant := scoped.DBTenant()
			return tenant, tenant.Valid(), false
		}
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

	var (
		found    Tenant
		resolved bool
	)
	typ := v.Type()
	for i := range v.NumField() {
		field := typ.Field(i)
		if !field.IsExported() || field.Type == pulidType {
			continue
		}

		kind := field.Type.Kind()
		if kind == reflect.Pointer {
			kind = field.Type.Elem().Kind()
		}
		if kind != reflect.Struct {
			continue
		}

		tenant, ok, conflict := tenantOfValue(v.Field(i), depth+1)
		if conflict {
			return Tenant{}, false, true
		}
		if !ok {
			continue
		}
		if resolved && !sameTenant(found, tenant) {
			return Tenant{}, false, true
		}
		if !resolved || found.UserID.IsNil() {
			found = tenant
		}
		resolved = true
	}

	return found, resolved, false
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
