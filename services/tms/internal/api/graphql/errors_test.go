package graphql

import (
	"context"
	"errors"
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

func presenterTestConfig() *config.Config {
	return &config.Config{
		App: config.AppConfig{
			Debug:              true,
			ProblemTypeBaseURI: "https://api.test/problems/",
		},
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
