package authctx

import (
	"reflect"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

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

	presetID := idField(elem)

	if err := c.ShouldBindJSON(req); err != nil {
		return err
	}

	if err := ensureTenantMatches(elem, authCtx); err != nil {
		return err
	}

	if !presetID.IsNil() {
		setPulidField(elem, "ID", presetID)
	}

	AddContextToRequest(authCtx, req)
	setPulidField(elem, "CurrentOrganizationID", authCtx.OrganizationID)

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

func ensureTenantMatches(elem reflect.Value, authCtx *AuthContext) error {
	if mismatched(elem, organizationIDFields, authCtx.OrganizationID) ||
		mismatched(elem, businessUnitIDFields, authCtx.BusinessUnitID) {
		return errortypes.NewAuthorizationError(
			"The request belongs to a different organization than your session",
		)
	}

	return nil
}

func mismatched(elem reflect.Value, names []string, want pulid.ID) bool {
	for _, name := range names {
		field := elem.FieldByName(name)
		if !field.IsValid() || field.Type() != pulidType {
			continue
		}

		got, _ := field.Interface().(pulid.ID)

		return !got.IsNil() && got != want
	}

	return false
}

func idField(elem reflect.Value) pulid.ID {
	field := elem.FieldByName("ID")
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
