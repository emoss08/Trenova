package helpers

import (
	"context"
	"encoding/json" //nolint:depguard // this is fine
	"errors"
	"io"

	tarifferrors "github.com/emoss08/trenova/internal/modules/rating/app/tariff/errors"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/go-playground/validator/v10"
)

type ErrorClassifier interface {
	Classify(err error) (ProblemType, bool)
}

type ClassifierFunc func(error) (ProblemType, bool)

func (f ClassifierFunc) Classify(err error) (ProblemType, bool) {
	return f(err)
}

type ChainClassifier struct {
	classifiers []ErrorClassifier
}

func NewChainClassifier(classifiers ...ErrorClassifier) *ChainClassifier {
	return &ChainClassifier{classifiers: classifiers}
}

func (c *ChainClassifier) Register(classifier ErrorClassifier) {
	c.classifiers = append(c.classifiers, classifier)
}

func (c *ChainClassifier) Classify(err error) ProblemType {
	for _, classifier := range c.classifiers {
		if problemType, ok := classifier.Classify(err); ok {
			return problemType
		}
	}
	return ProblemTypeInternal
}

func NewDefaultClassifier() *ChainClassifier {
	return NewChainClassifier(
		ClassifierFunc(classifyTimeout),
		ClassifierFunc(classifyRequestTooLarge),
		ClassifierFunc(classifyEditConflict),
		ClassifierFunc(classifyValidation),
		ClassifierFunc(classifyTariff),
		ClassifierFunc(classifyBadRequest),
		ClassifierFunc(classifyBusiness),
		ClassifierFunc(classifyDatabase),
		ClassifierFunc(classifyQuotaExceeded),
		ClassifierFunc(classifyPlanRestricted),
		ClassifierFunc(classifyAuthentication),
		ClassifierFunc(classifyAuthorization),
		ClassifierFunc(classifyNotFound),
		ClassifierFunc(classifyRateLimit),
		ClassifierFunc(classifyConflict),
	)
}

func classifyTimeout(err error) (ProblemType, bool) {
	var timeoutErr *RequestTimeoutError
	if errors.As(err, &timeoutErr) {
		return ProblemTypeTimeout, true
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return ProblemTypeTimeout, true
	}

	return "", false
}

func classifyRequestTooLarge(err error) (ProblemType, bool) {
	if IsRequestTooLargeError(err) {
		return ProblemTypeRequestTooLarge, true
	}
	return "", false
}

func classifyValidation(err error) (ProblemType, bool) {
	if errortypes.IsMultiError(err) || errortypes.IsError(err) {
		return ProblemTypeValidation, true
	}

	if _, ok := errors.AsType[validator.ValidationErrors](err); ok {
		return ProblemTypeValidation, true
	}
	return "", false
}

func classifyTariff(err error) (ProblemType, bool) {
	var (
		diagnostic *tarifferrors.DiagnosticError
		evaluation *tarifferrors.EvaluationError
	)

	if errors.As(err, &diagnostic) || errors.As(err, &evaluation) {
		return ProblemTypeValidation, true
	}

	return "", false
}

func classifyBadRequest(err error) (ProblemType, bool) {
	if errors.Is(err, pulid.ErrInvalidLength) {
		return ProblemTypeValidation, true
	}

	if _, ok := errors.AsType[*json.SyntaxError](err); ok {
		return ProblemTypeValidation, true
	}

	if _, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		return ProblemTypeValidation, true
	}

	if errors.Is(err, io.ErrUnexpectedEOF) || errors.Is(err, io.EOF) {
		return ProblemTypeValidation, true
	}

	return "", false
}

func classifyBusiness(err error) (ProblemType, bool) {
	if errortypes.IsBusinessError(err) {
		return ProblemTypeBusiness, true
	}
	return "", false
}

func classifyDatabase(err error) (ProblemType, bool) {
	if errortypes.IsDatabaseError(err) {
		return ProblemTypeDatabase, true
	}
	return "", false
}

func classifyQuotaExceeded(err error) (ProblemType, bool) {
	if errortypes.IsQuotaExceededError(err) {
		return ProblemTypeQuotaExceeded, true
	}
	return "", false
}

func classifyPlanRestricted(err error) (ProblemType, bool) {
	if errortypes.IsPlanRestrictionError(err) {
		return ProblemTypePlanRestricted, true
	}
	return "", false
}

func classifyAuthentication(err error) (ProblemType, bool) {
	if errortypes.IsAuthenticationError(err) {
		return ProblemTypeAuthentication, true
	}
	return "", false
}

func classifyAuthorization(err error) (ProblemType, bool) {
	if errortypes.IsAuthorizationError(err) {
		return ProblemTypeAuthorization, true
	}
	return "", false
}

func classifyNotFound(err error) (ProblemType, bool) {
	if errortypes.IsNotFoundError(err) {
		return ProblemTypeNotFound, true
	}
	return "", false
}

func classifyRateLimit(err error) (ProblemType, bool) {
	if errortypes.IsRateLimitError(err) {
		return ProblemTypeRateLimit, true
	}
	return "", false
}

func classifyEditConflict(err error) (ProblemType, bool) {
	if errortypes.IsEditConflictError(err) {
		return ProblemTypeConflict, true
	}
	return "", false
}

func classifyConflict(err error) (ProblemType, bool) {
	if errortypes.IsConflictError(err) {
		return ProblemTypeConflict, true
	}
	return "", false
}
