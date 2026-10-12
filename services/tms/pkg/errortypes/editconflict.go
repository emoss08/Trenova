package errortypes

import "errors"

// EditConflict tells a person whose save lost a race what the other save
// changed, who made it and when, so they can load it or keep their own.
type EditConflict struct {
	Version       int64                `json:"version"`
	UpdatedByID   string               `json:"updatedById,omitempty"`
	UpdatedByName string               `json:"updatedByName,omitempty"`
	UpdatedAt     int64                `json:"updatedAt"`
	Changes       []EditConflictChange `json:"changes"`
}

type EditConflictChange struct {
	Field string `json:"field"`
	Label string `json:"label"`
}

type EditConflictError struct {
	BaseError
	Edit *EditConflict `json:"conflict"`
}

func NewEditConflictError(edit *EditConflict) *EditConflictError {
	return &EditConflictError{
		BaseError: BaseError{
			Code:    ErrVersionMismatch,
			Message: "Someone else saved changes while you were editing",
		},
		Edit: edit,
	}
}

func (e *EditConflictError) WithInternal(err error) *EditConflictError {
	e.Internal = err
	return e
}

func (e *EditConflictError) LogFields() LogFields {
	fields := e.BaseError.LogFields()
	if e.Edit != nil {
		fields["conflict_version"] = e.Edit.Version
		fields["conflict_changes"] = len(e.Edit.Changes)
	}
	return fields
}

func IsEditConflictError(err error) bool {
	_, ok := errors.AsType[*EditConflictError](err)
	return ok
}

func EditConflictOf(err error) (*EditConflict, bool) {
	conflict, ok := errors.AsType[*EditConflictError](err)
	if !ok || conflict.Edit == nil {
		return nil, false
	}
	return conflict.Edit, true
}
