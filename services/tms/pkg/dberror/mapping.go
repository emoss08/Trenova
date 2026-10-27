package dberror

import (
	"errors"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type Mapping struct {
	Entity string
}

func (m Mapping) Map(err error, entityID string) error {
	switch {
	case err == nil:
		return nil
	case IsNotFoundError(err):
		return HandleNotFoundError(err, m.Entity)
	case IsVersionConflict(err):
		return CreateVersionMismatchError(m.Entity, entityID)
	}

	if ce, ok := errors.AsType[*keel.ConstraintError](err); ok {
		return ConstraintViolation(m.Entity, ce)
	}

	return err
}

func IsVersionConflict(err error) bool {
	return errors.Is(err, keel.ErrConflict)
}

func HandleVersionConflict(err error, entityName, entityID string) error {
	if IsVersionConflict(err) {
		return CreateVersionMismatchError(entityName, entityID)
	}

	return err
}

func ConstraintViolation(entity string, ce *keel.ConstraintError) error {
	field := ""
	if len(ce.Fields) > 0 {
		field = ce.Fields[0]
	}
	label := strings.Join(ce.Labels, " and ")

	var verr *errortypes.Error
	switch {
	case ce.Message != "":
		verr = errortypes.NewValidationError(field, violationCode(ce.Kind), ce.Message)
	case ce.Kind == keel.ConstraintUnique && label != "":
		verr = errortypes.NewValidationError(field, errortypes.ErrDuplicate,
			"{0} with this {1} already exists in your organization", entity, label)
	case ce.Kind == keel.ConstraintUnique:
		verr = errortypes.NewValidationError(field, errortypes.ErrDuplicate,
			"{0} already exists in your organization", entity)
	case ce.Kind == keel.ConstraintForeignKey && label != "":
		verr = errortypes.NewValidationError(field, errortypes.ErrInvalidReference,
			"{0} does not exist in your organization", label)
	case ce.Kind == keel.ConstraintForeignKey:
		verr = errortypes.NewValidationError(field, errortypes.ErrInvalidReference,
			"{0} refers to a record that does not exist in your organization", entity)
	case ce.Kind == keel.ConstraintNotNull && label != "":
		verr = errortypes.NewValidationError(field, errortypes.ErrRequired, "{0} is required", label)
	case ce.Kind == keel.ConstraintExclusion:
		verr = errortypes.NewValidationError(field, errortypes.ErrDuplicate,
			"{0} overlaps an existing record in your organization", entity)
	case label != "":
		verr = errortypes.NewValidationError(field, errortypes.ErrInvalid, "{0} is not valid", label)
	default:
		verr = errortypes.NewValidationError(field, errortypes.ErrInvalid, "{0} is not valid", entity)
	}

	return verr.WithInternal(ce)
}

func violationCode(kind keel.ConstraintKind) errortypes.ErrorCode {
	switch kind {
	case keel.ConstraintUnique, keel.ConstraintExclusion:
		return errortypes.ErrDuplicate
	case keel.ConstraintForeignKey:
		return errortypes.ErrInvalidReference
	case keel.ConstraintNotNull:
		return errortypes.ErrRequired
	case keel.ConstraintCheck:
		return errortypes.ErrInvalid
	}

	return errortypes.ErrInvalid
}

func IsTenantBoundaryViolation(err error) bool {
	return errors.Is(err, keel.ErrTenantMismatch) || IsRowLevelSecurityViolation(err)
}
