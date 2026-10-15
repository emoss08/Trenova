package permissionports

import (
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type DelegationViolationKind int

const (
	DelegationResourceNotHeld DelegationViolationKind = iota + 1
	DelegationOperationNotHeld
	DelegationScopeTooWide
)

type DelegationViolation struct {
	Kind      DelegationViolationKind
	Resource  string
	Operation permission.Operation
	Scope     permission.DataScope
}

func (e *EffectivePermissions) DelegationViolation(
	resource string,
	operations []permission.Operation,
	scope permission.DataScope,
) (DelegationViolation, bool) {
	held, ok := e.Resources[resource]
	if !ok {
		return DelegationViolation{Kind: DelegationResourceNotHeld, Resource: resource}, true
	}

	heldOps := permission.NewOperationSet(held.Operations...)
	for _, op := range operations {
		if !heldOps.Has(op) {
			return DelegationViolation{
				Kind:      DelegationOperationNotHeld,
				Resource:  resource,
				Operation: op,
			}, true
		}
	}

	if scope.IsMorePermissive(held.DataScope) {
		return DelegationViolation{
			Kind:     DelegationScopeTooWide,
			Resource: resource,
			Scope:    scope,
		}, true
	}

	return DelegationViolation{}, false
}

func (v DelegationViolation) FieldError(field string) *errortypes.MultiError {
	me := errortypes.NewMultiError()
	switch v.Kind {
	case DelegationResourceNotHeld:
		me.Add(
			field,
			errortypes.ErrForbidden,
			"cannot grant permissions for resource '{0}' that you don't have access to",
			v.Resource,
		)
	case DelegationOperationNotHeld:
		me.Add(
			field,
			errortypes.ErrForbidden,
			"cannot grant '{0}' operation on '{1}' that you don't have",
			string(v.Operation),
			v.Resource,
		)
	case DelegationScopeTooWide:
		me.Add(
			field,
			errortypes.ErrForbidden,
			"cannot grant data scope '{0}' on '{1}' that is more permissive than your own",
			string(v.Scope),
			v.Resource,
		)
	}
	return me
}
