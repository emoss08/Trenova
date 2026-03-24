package helpers

import (
	"encoding/json" //nolint:depguard // this is fine
	"errors"
	"io"

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
		ClassifierFunc(classifyValidation),
		ClassifierFunc(classifyBadRequest),
		ClassifierFunc(classifyBusiness),
		ClassifierFunc(classifyDatabase),
		ClassifierFunc(classifyAuthentication),
		ClassifierFunc(classifyAuthorization),
		ClassifierFunc(classifyNotFound),
		ClassifierFunc(classifyRateLimit),
		ClassifierFunc(classifyConflict),
	)
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

func classifyConflict(err error) (ProblemType, bool) {
	if errortypes.IsConflictError(err) {
		return ProblemTypeConflict, true
	}
	return "", false
}
