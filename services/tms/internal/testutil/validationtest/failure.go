package validationtest

import (
	"errors"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
)

func Failed(t testing.TB, err error) error {
	t.Helper()

	if err == nil {
		t.Fatalf("want a validation that could not complete, got no error")
		return nil
	}
	if multiErr, ok := errortypes.RefusalOf(err); ok {
		t.Fatalf(
			"want a validation that could not complete, got a refusal:\n%s",
			newResult(t, multiErr).describe(),
		)
		return nil
	}

	return err
}

func VersionMismatch(t testing.TB, err error) *errortypes.EditConflictError {
	t.Helper()

	conflict, ok := errors.AsType[*errortypes.EditConflictError](err)
	if !ok {
		t.Fatalf("want a version mismatch, got: %v", err)
		return nil
	}

	return conflict
}
