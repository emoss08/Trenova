package sim

import (
	"errors"
	"fmt"
	"net/http"
)

var badRequestErrors = []error{
	ErrInvalidBody,
	ErrInvalidParameter,
	ErrUniqueConflict,
	ErrReferenceNotFound,
	ErrTagCycle,
	ErrLimitInvalid,
	ErrCursorInvalid,
	ErrStatTypesRequired,
	ErrStatTypeInvalid,
	ErrAssetTypeInvalid,
	ErrTimeRangeRequired,
	ErrRecordIDRequired,
	ErrQueryIDRequired,
	ErrPathIDRequired,
}

func statusForError(err error) int {
	if errors.Is(err, ErrRecordNotFound) {
		return http.StatusNotFound
	}
	for _, target := range badRequestErrors {
		if errors.Is(err, target) {
			return http.StatusBadRequest
		}
	}
	return http.StatusInternalServerError
}

func (s *Server) writeError(writer http.ResponseWriter, err error) {
	s.writeAPIError(writer, statusForError(err), err)
}

func invalidParameter(name, reason string) error {
	return fmt.Errorf("%w %s: %s", ErrInvalidParameter, name, reason)
}

func invalidField(path, reason string) error {
	return fmt.Errorf("%w: %s %s", ErrInvalidBody, path, reason)
}

func notFound(label, ref string) error {
	return fmt.Errorf("%w: no %s matches %q", ErrRecordNotFound, label, ref)
}

func missingReference(label, ref string) error {
	return fmt.Errorf("%w: no %s matches %q", ErrReferenceNotFound, label, ref)
}
