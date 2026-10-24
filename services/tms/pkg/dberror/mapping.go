package dberror

import (
	"errors"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
)

type Constraint struct {
	Name    string
	Field   string
	Message string
}

type Mapping struct {
	Entity     string
	Unique     []Constraint
	ForeignKey []Constraint
}

func (m Mapping) Map(err error, entityID string) error {
	switch {
	case err == nil:
		return nil
	case IsNotFoundError(err):
		return HandleNotFoundError(err, m.Entity)
	case IsVersionConflict(err):
		return CreateVersionMismatchError(m.Entity, entityID)
	case IsUniqueConstraintViolation(err):
		return MapUniqueViolation(err, m.Unique...)
	case IsForeignKeyConstraintViolation(err):
		return MapForeignKeyViolation(err, m.ForeignKey...)
	default:
		return err
	}
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

func MapUniqueViolation(err error, constraints ...Constraint) error {
	if !IsUniqueConstraintViolation(err) {
		return err
	}

	return constraintError(err, errortypes.ErrDuplicate, constraints)
}

func MapForeignKeyViolation(err error, constraints ...Constraint) error {
	if !IsForeignKeyConstraintViolation(err) {
		return err
	}

	return constraintError(err, errortypes.ErrInvalidReference, constraints)
}

func constraintError(err error, code errortypes.ErrorCode, constraints []Constraint) error {
	name := ExtractConstraintName(err)
	for _, c := range constraints {
		if c.Name == name {
			return errortypes.NewValidationError(c.Field, code, c.Message)
		}
	}

	return err
}

func IsTenantBoundaryViolation(err error) bool {
	return errors.Is(err, keel.ErrTenantMismatch) || IsRowLevelSecurityViolation(err)
}
