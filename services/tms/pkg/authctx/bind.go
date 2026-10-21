package authctx

import (
	"errors"
	"reflect"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantboundary"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/gin-gonic/gin"
)

const maxTenantDepth = 32

var (
	pulidType            = reflect.TypeFor[pulid.ID]()
	organizationIDFields = []string{"OrganizationID", "OrgID", "CurrentOrganizationID"}
	businessUnitIDFields = []string{"BusinessUnitID", "BuID"}

	errTenantTooDeep = errors.New("request body nests records too deeply to check their tenant")
	errStopWalk      = errors.New("stop walking the request body")
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

	violation, mismatch, err := tenantMismatch(elem, authCtx)
	if errors.Is(err, errTenantTooDeep) {
		return errortypes.NewBusinessError(
			"The request nests records too deeply to be accepted",
		).WithInternal(err)
	}
	if mismatch {
		tenantboundary.Report(c, violation)
		return errortypes.NewAuthorizationError(
			"The request belongs to a different organization than your session",
		)
	}

	if !presetID.IsNil() {
		setPulidField(elem, "ID", presetID)
	}

	if err = stampTenant(elem, authCtx); err != nil {
		return errortypes.NewBusinessError(
			"The request nests records too deeply to be accepted",
		).WithInternal(err)
	}
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

func tenantMismatch(
	elem reflect.Value,
	authCtx *AuthContext,
) (tenantboundary.Violation, bool, error) {
	var violation tenantboundary.Violation
	found := false
	err := eachStruct(elem, func(record reflect.Value) error {
		field, got, ok := fieldMismatch(record, organizationIDFields, authCtx.OrganizationID)
		if ok {
			violation = tenantboundary.Violation{
				Source:         tenantboundary.SourceRequestBody,
				Field:          field,
				OrganizationID: got,
			}
			found = true
			return errStopWalk
		}
		field, got, ok = fieldMismatch(record, businessUnitIDFields, authCtx.BusinessUnitID)
		if ok {
			violation = tenantboundary.Violation{
				Source:         tenantboundary.SourceRequestBody,
				Field:          field,
				BusinessUnitID: got,
			}
			found = true
			return errStopWalk
		}
		return nil
	})
	if errors.Is(err, errStopWalk) {
		err = nil
	}

	return violation, found, err
}

func stampTenant(elem reflect.Value, authCtx *AuthContext) error {
	return eachStruct(elem, func(record reflect.Value) error {
		for _, name := range organizationIDFields {
			setPulidField(record, name, authCtx.OrganizationID)
		}
		for _, name := range businessUnitIDFields {
			setPulidField(record, name, authCtx.BusinessUnitID)
		}
		if tenant := record.FieldByName("TenantInfo"); tenant.IsValid() &&
			tenant.Kind() == reflect.Struct {
			setPulidField(tenant, "UserID", authCtx.UserID)
		}
		return nil
	})
}

// eachStruct calls visit on every struct the request body reaches: nested
// structs, pointers, slices, arrays, map values and interfaces, however a
// record is wrapped. A map value or interface holding a struct is visited
// on a copy that is written back, so a stamp reaches it too. Only those
// containers can nest without bound, so they are what is counted, and a
// body nested through more than maxTenantDepth of them is refused rather
// than left partly unchecked.
func eachStruct(root reflect.Value, visit func(reflect.Value) error) error {
	walker := &structWalker{visit: visit, seen: make(map[uintptr]struct{})}
	return walker.walk(root, 0)
}

type structWalker struct {
	visit func(reflect.Value) error
	seen  map[uintptr]struct{}
}

func (w *structWalker) walk(value reflect.Value, depth int) error {
	if depth > maxTenantDepth {
		return errTenantTooDeep
	}

	kind := value.Kind()
	if kind == reflect.Struct {
		return w.walkStruct(value, depth)
	}
	if kind == reflect.Pointer {
		return w.walkPointer(value, depth)
	}
	if kind == reflect.Interface {
		return w.walkInterface(value, depth)
	}
	if kind == reflect.Map {
		return w.walkMap(value, depth)
	}
	if kind == reflect.Slice || kind == reflect.Array {
		return w.walkSequence(value, depth)
	}
	return nil
}

func (w *structWalker) walkPointer(value reflect.Value, depth int) error {
	if value.IsNil() {
		return nil
	}
	address := value.Pointer()
	if _, ok := w.seen[address]; ok {
		return nil
	}
	w.seen[address] = struct{}{}
	return w.walk(value.Elem(), depth+1)
}

func (w *structWalker) walkSequence(value reflect.Value, depth int) error {
	if value.Kind() == reflect.Slice && value.IsNil() || !holdsStruct(value.Type().Elem()) {
		return nil
	}
	for i := range value.Len() {
		if err := w.walk(value.Index(i), depth+1); err != nil {
			return err
		}
	}
	return nil
}

func (w *structWalker) walkStruct(value reflect.Value, depth int) error {
	if value.Type() == pulidType {
		return nil
	}
	if err := w.visit(value); err != nil {
		return err
	}

	typ := value.Type()
	for i := range value.NumField() {
		if !typ.Field(i).IsExported() {
			continue
		}
		if err := w.walk(value.Field(i), depth); err != nil {
			return err
		}
	}
	return nil
}

func (w *structWalker) walkInterface(value reflect.Value, depth int) error {
	if value.IsNil() {
		return nil
	}
	held := value.Elem()
	if held.Kind() != reflect.Struct {
		return w.walk(held, depth+1)
	}

	copied := reflect.New(held.Type()).Elem()
	copied.Set(held)
	if err := w.walk(copied, depth+1); err != nil {
		return err
	}
	if value.CanSet() {
		value.Set(copied)
	}
	return nil
}

func (w *structWalker) walkMap(value reflect.Value, depth int) error {
	if value.IsNil() {
		return nil
	}
	if !holdsStruct(value.Type().Elem()) {
		return nil
	}
	iter := value.MapRange()
	for iter.Next() {
		entry := iter.Value()
		copied := reflect.New(entry.Type()).Elem()
		copied.Set(entry)
		if err := w.walk(copied, depth+1); err != nil {
			return err
		}
		value.SetMapIndex(iter.Key(), copied)
	}
	return nil
}

func holdsStruct(typ reflect.Type) bool {
	for {
		kind := typ.Kind()
		if kind == reflect.Struct {
			return typ != pulidType
		}
		if kind == reflect.Interface {
			return true
		}
		if kind != reflect.Pointer && kind != reflect.Slice && kind != reflect.Array &&
			kind != reflect.Map {
			return false
		}
		typ = typ.Elem()
	}
}

func fieldMismatch(elem reflect.Value, names []string, want pulid.ID) (string, pulid.ID, bool) {
	for _, name := range names {
		got := pulidField(elem, name)
		if !got.IsNil() && got != want {
			return name, got, true
		}
	}

	return "", pulid.Nil, false
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
