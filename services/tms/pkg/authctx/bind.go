package authctx

import (
	"reflect"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

const maxTenantDepth = 4

var (
	pulidType            = reflect.TypeFor[pulid.ID]()
	organizationIDFields = []string{"OrganizationID", "OrgID", "CurrentOrganizationID"}
	businessUnitIDFields = []string{"BusinessUnitID", "BuID"}
)

func BindJSON(c *gin.Context, authCtx *AuthContext, req any) error {
	elem, ok := structElem(req)
	if !ok {
		return c.ShouldBindJSON(req)
	}

	presetID := pulidField(elem, "ID")

	if err := c.ShouldBindJSON(req); err != nil {
		return err
	}

	if tenantMismatch(elem, authCtx, 0) {
		return errortypes.NewAuthorizationError(
			"The request belongs to a different organization than your session",
		)
	}

	if !presetID.IsNil() {
		setPulidField(elem, "ID", presetID)
	}

	stampTenant(elem, authCtx, 0)
	setPulidField(elem, "UserID", authCtx.UserID)

	return nil
}

func structElem(req any) (reflect.Value, bool) {
	val := reflect.ValueOf(req)
	if val.Kind() != reflect.Pointer || val.IsNil() {
		return reflect.Value{}, false
	}

	elem := val.Elem()
	if elem.Kind() != reflect.Struct {
		return reflect.Value{}, false
	}

	return elem, true
}

func tenantMismatch(elem reflect.Value, authCtx *AuthContext, depth int) bool {
	if fieldMismatch(elem, organizationIDFields, authCtx.OrganizationID) ||
		fieldMismatch(elem, businessUnitIDFields, authCtx.BusinessUnitID) {
		return true
	}

	if depth >= maxTenantDepth {
		return false
	}

	for _, nested := range nestedStructs(elem) {
		if tenantMismatch(nested, authCtx, depth+1) {
			return true
		}
	}

	return false
}

func stampTenant(elem reflect.Value, authCtx *AuthContext, depth int) {
	for _, name := range organizationIDFields {
		setPulidField(elem, name, authCtx.OrganizationID)
	}
	for _, name := range businessUnitIDFields {
		setPulidField(elem, name, authCtx.BusinessUnitID)
	}

	if depth >= maxTenantDepth {
		return
	}

	for _, nested := range nestedStructs(elem) {
		stampTenant(nested, authCtx, depth+1)
	}

	if tenant := elem.FieldByName(
		"TenantInfo",
	); tenant.IsValid() &&
		tenant.Kind() == reflect.Struct {
		setPulidField(tenant, "UserID", authCtx.UserID)
	}
}

func nestedStructs(elem reflect.Value) []reflect.Value {
	nested := make([]reflect.Value, 0)
	typ := elem.Type()

	for i := range elem.NumField() {
		if !typ.Field(i).IsExported() {
			continue
		}

		field := elem.Field(i)
		if field.Kind() == reflect.Struct && field.Type() != pulidType {
			nested = append(nested, field)
			continue
		}
		if field.Kind() == reflect.Pointer && !field.IsNil() &&
			field.Elem().Kind() == reflect.Struct {
			nested = append(nested, field.Elem())
		}
	}

	return nested
}

func fieldMismatch(elem reflect.Value, names []string, want pulid.ID) bool {
	for _, name := range names {
		got := pulidField(elem, name)
		if !got.IsNil() && got != want {
			return true
		}
	}

	return false
}

func pulidField(elem reflect.Value, name string) pulid.ID {
	field := elem.FieldByName(name)
	if !field.IsValid() || field.Type() != pulidType {
		return pulid.Nil
	}

	id, _ := field.Interface().(pulid.ID)

	return id
}

func setPulidField(elem reflect.Value, name string, id pulid.ID) {
	field := elem.FieldByName(name)
	if field.IsValid() && field.CanSet() && field.Type() == pulidType {
		field.Set(reflect.ValueOf(id))
	}
}
