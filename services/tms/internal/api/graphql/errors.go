package graphql

import (
	"context"
	"errors"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func newErrorPresenter(cfg *config.Config) graphql.ErrorPresenterFunc {
	classifier := helpers.NewDefaultClassifier()
	sanitizer := helpers.NewSanitizer(cfg.App.Debug)
	baseURI := cfg.App.GetProblemTypeBaseURI()

	return func(ctx context.Context, err error) *gqlerror.Error {
		gqlErr := graphql.DefaultErrorPresenter(ctx, err)
		problemType := classifier.Classify(err)
		gqlErr.Message = sanitizer.SanitizeMessage(err, problemType)
		gqlErr.Extensions = map[string]any{
			"code":    string(errorCode(err, problemType)),
			"type":    baseURI + string(problemType),
			"traceId": gqlctx.RequestID(ctx),
		}

		if params := sanitizer.ExtractParams(err); len(params) > 0 {
			gqlErr.Extensions["params"] = params
		}
		if validationErrors := sanitizer.ExtractErrors(err); len(validationErrors) > 0 {
			gqlErr.Extensions["errors"] = validationErrors
		}

		return gqlErr
	}
}

func errorCode(err error, problemType helpers.ProblemType) errortypes.ErrorCode {
	var errorable errortypes.Errorable
	if errors.As(err, &errorable) {
		return errorable.GetCode()
	}

	switch problemType {
	case helpers.ProblemTypeAuthentication:
		return errortypes.ErrUnauthorized
	case helpers.ProblemTypeAuthorization:
		return errortypes.ErrForbidden
	case helpers.ProblemTypeNotFound:
		return errortypes.ErrNotFound
	case helpers.ProblemTypeRateLimit:
		return errortypes.ErrTooManyRequests
	case helpers.ProblemTypeConflict:
		return errortypes.ErrResourceInUse
	case helpers.ProblemTypeValidation, helpers.ProblemTypeBusiness:
		return errortypes.ErrInvalid
	default:
		return errortypes.ErrSystemError
	}
}

func recoverFunc(ctx context.Context, err any) error {
	return errortypes.NewDatabaseError("GraphQL request failed").
		WithInternal(fmt.Errorf("panic recovered: %v", err)).
		WithContext(errortypes.NewErrorContext().WithTraceID(gqlctx.RequestID(ctx)))
}
