package graphql

import (
	"context"
	"errors"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/querycost"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func init() {
	errcode.RegisterErrorType(querycost.DepthLimitErrorCode, errcode.KindProtocol)
	errcode.RegisterErrorType(querycost.ComplexityLimitErrorCode, errcode.KindProtocol)
	errcode.RegisterErrorType(CostBudgetErrorCode, errcode.KindProtocol)
	errcode.RegisterErrorType(FeatureAccessErrorCode, errcode.KindProtocol)
}

func newErrorPresenter(cfg *config.Config) graphql.ErrorPresenterFunc {
	classifier := helpers.NewDefaultClassifier()
	sanitizer := helpers.NewSanitizer(cfg.App.Debug)
	baseURI := cfg.App.GetProblemTypeBaseURI()

	return func(ctx context.Context, err error) *gqlerror.Error {
		gqlErr := graphql.DefaultErrorPresenter(ctx, err)

		if code, ok := protocolErrorCode(gqlErr); ok {
			gqlErr.Extensions = map[string]any{
				"code":    code,
				"type":    baseURI + string(protocolProblemType(code)),
				"traceId": gqlctx.RequestID(ctx),
			}

			return gqlErr
		}

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

func protocolErrorCode(gqlErr *gqlerror.Error) (string, bool) {
	if gqlErr == nil || gqlErr.Extensions == nil {
		return "", false
	}

	code, ok := gqlErr.Extensions["code"].(string)
	if !ok || code == "" {
		return "", false
	}
	if errcode.GetErrorKind(gqlerror.List{gqlErr}) != errcode.KindProtocol {
		return "", false
	}

	return code, true
}

func protocolProblemType(code string) helpers.ProblemType {
	if code == CostBudgetErrorCode {
		return helpers.ProblemTypeRateLimit
	}

	return helpers.ProblemTypeValidation
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
