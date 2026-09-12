package graphql

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/querycost"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/gqlerror"
)

// The protocol error codes live in a process-global registry that only
// NewServer populates in production. Without this the presenter tests pass or
// fail on test ordering, because every test in the package is parallel and only
// the query-limit tests happened to register them.
func TestMain(m *testing.M) {
	registerQueryLimitErrorCodes()
	errcode.RegisterErrorType(CostBudgetErrorCode, errcode.KindProtocol)
	os.Exit(m.Run())
}

func presenterTestConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Debug:              true,
			ProblemTypeBaseURI: "https://api.test/problems/",
		},
	}
}

func TestProtocolErrorCodesAreRegistered(t *testing.T) {
	t.Parallel()

	for _, code := range []string{
		querycost.DepthLimitErrorCode,
		querycost.ComplexityLimitErrorCode,
		CostBudgetErrorCode,
		FeatureAccessErrorCode,
	} {
		err := &gqlerror.Error{
			Message:    "limit exceeded",
			Extensions: map[string]any{"code": code},
		}
		assert.Equal(t, errcode.KindProtocol, errcode.GetErrorKind(gqlerror.List{err}),
			"%s must map to a 422 rather than a 200, with no constructor having run", code)
	}
}

func TestErrorPresenter_CostBudgetIsRateLimitProblem(t *testing.T) {
	t.Parallel()

	present := newErrorPresenter(presenterTestConfig())
	ctx := gqlctx.WithRequestID(context.Background(), "req-1")

	err := gqlerror.Errorf("budget exhausted")
	errcode.Set(err, CostBudgetErrorCode)

	presented := present(ctx, err)
	require.NotNil(t, presented)
	assert.Equal(t, CostBudgetErrorCode, presented.Extensions["code"])
	assert.Equal(t,
		"https://api.test/problems/"+string(helpers.ProblemTypeRateLimit),
		presented.Extensions["type"],
	)
	assert.Equal(t, "req-1", presented.Extensions["traceId"])
}

func TestErrorPresenter_OtherProtocolErrorsStayValidationProblems(t *testing.T) {
	t.Parallel()

	present := newErrorPresenter(presenterTestConfig())

	err := gqlerror.Errorf("too deep")
	errcode.Set(err, querycost.DepthLimitErrorCode)

	presented := present(context.Background(), err)
	require.NotNil(t, presented)
	assert.Equal(t, querycost.DepthLimitErrorCode, presented.Extensions["code"])
	assert.Equal(t,
		"https://api.test/problems/"+string(helpers.ProblemTypeValidation),
		presented.Extensions["type"],
	)
}

func TestErrorPresenter_DomainErrorsCarryFieldErrors(t *testing.T) {
	t.Parallel()

	present := newErrorPresenter(presenterTestConfig())

	presented := present(
		context.Background(),
		errortypes.NewValidationError("code", errortypes.ErrRequired, "Code is required"),
	)
	require.NotNil(t, presented)
	assert.Equal(t, string(errortypes.ErrRequired), presented.Extensions["code"])
	assert.Equal(t,
		"https://api.test/problems/"+string(helpers.ProblemTypeValidation),
		presented.Extensions["type"],
	)
	assert.NotEmpty(t, presented.Extensions["errors"])
}

func TestErrorPresenter_UnknownErrorsAreSystemErrors(t *testing.T) {
	t.Parallel()

	present := newErrorPresenter(presenterTestConfig())

	presented := present(context.Background(), errors.New("boom"))
	require.NotNil(t, presented)
	assert.Equal(t, string(errortypes.ErrSystemError), presented.Extensions["code"])
}
