package gqlext

import (
	"context"
	"fmt"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/querycost"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config/appconfig"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

func init() {
	errcode.RegisterErrorType(querycost.DepthLimitErrorCode, errcode.KindProtocol)
	errcode.RegisterErrorType(querycost.ComplexityLimitErrorCode, errcode.KindProtocol)
	errcode.RegisterErrorType(CostBudgetErrorCode, errcode.KindProtocol)
}

func newErrorPresenter(cfg *appconfig.Config) graphql.ErrorPresenterFunc {
	classifier := helpers.NewDefaultClassifier()
	sanitizer := helpers.NewSanitizer(cfg.Debug)
	baseURI := cfg.GetProblemTypeBaseURI()

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

		err = helpers.NormalizeDatabaseError(ctx, classifier, err)
		locale := i18n.FromContext(ctx)
		problemType := classifier.Classify(err)
		gqlErr.Message = sanitizer.SanitizeMessage(err, problemType).Localize(locale)
		gqlErr.Extensions = map[string]any{
			"code":    string(helpers.ErrorCodeOf(err, problemType)),
			"type":    baseURI + string(problemType),
			"traceId": gqlctx.RequestID(ctx),
		}

		if params := sanitizer.ExtractParams(err); len(params) > 0 {
			gqlErr.Extensions["params"] = params
		}
		if conflict := sanitizer.ExtractConflict(err); conflict != nil {
			gqlErr.Extensions["conflict"] = conflict
		}
		if validationErrors := sanitizer.ExtractErrors(err); len(validationErrors) > 0 {
			gqlErr.Extensions["errors"] = helpers.LocalizeErrors(locale, validationErrors)
		}
		if advisories := sanitizer.ExtractAdvisories(err); len(advisories) > 0 {
			gqlErr.Extensions["advisories"] = helpers.LocalizeAdvisories(locale, advisories)
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

func recoverFunc(ctx context.Context, err any) error {
	return errortypes.NewDatabaseError("GraphQL request failed").
		WithInternal(fmt.Errorf("panic recovered: %v", err)).
		WithContext(errortypes.NewErrorContext().WithTraceID(gqlctx.RequestID(ctx)))
}
