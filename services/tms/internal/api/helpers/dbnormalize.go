package helpers

import (
	"context"

	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantboundary"
)

func NormalizeDatabaseError(ctx context.Context, classifier *ChainClassifier, err error) error {
	if err == nil {
		return nil
	}

	if dberror.IsRowLevelSecurityViolation(err) {
		tenantboundary.Report(
			ctx,
			tenantboundary.Violation{Source: tenantboundary.SourceDatabasePolicy},
		)
		return errortypes.NewAuthorizationError("insufficient permissions").WithInternal(err)
	}

	if dberror.IsUniqueConstraintViolation(err) && unclassifiedDatabaseFailure(classifier, err) {
		conflict := errortypes.NewConflictError(
			"Another record already uses these values. Refresh and try again.",
		).WithInternal(dberror.DriverError(err))
		conflict.Code = errortypes.ErrDuplicate
		return conflict
	}

	return err
}

func unclassifiedDatabaseFailure(classifier *ChainClassifier, err error) bool {
	problemType := classifier.Classify(err)
	return problemType == ProblemTypeDatabase || problemType == ProblemTypeInternal
}
