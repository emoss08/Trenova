package graphql

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/querycost"
	"github.com/emoss08/trenova/internal/api/helpers"
	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/tenantboundary"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
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

func TestErrorPresenter_UniqueViolationIsADuplicateConflict(t *testing.T) {
	t.Parallel()

	present := newErrorPresenter(presenterTestConfig())
	ctx := gqlctx.WithRequestID(t.Context(), "req-dup")

	presented := present(ctx, fmt.Errorf("insert customer: %w", &pgconn.PgError{
		Code:           pgerrcode.UniqueViolation,
		ConstraintName: "uq_customers_code",
		Message:        "duplicate key value violates unique constraint",
	}))

	require.NotNil(t, presented)
	assert.Equal(t, string(errortypes.ErrDuplicate), presented.Extensions["code"])
	assert.Equal(t,
		"https://api.test/problems/"+string(helpers.ProblemTypeConflict),
		presented.Extensions["type"],
	)
	assert.NotContains(t, presented.Message, "uq_customers_code")
}

func TestErrorPresenter_RowLevelSecurityViolationIsForbidden(t *testing.T) {
	t.Parallel()

	present := newErrorPresenter(presenterTestConfig())
	tracker := tenantboundary.NewTracker()
	ctx := tenantboundary.With(gqlctx.WithRequestID(t.Context(), "req-rls"), tracker)

	presented := present(ctx, &pgconn.PgError{
		Code:    pgerrcode.InsufficientPrivilege,
		Message: `new row violates row-level security policy for table "shipments"`,
	})

	require.NotNil(t, presented)
	assert.Equal(t,
		"https://api.test/problems/"+string(helpers.ProblemTypeAuthorization),
		presented.Extensions["type"],
	)
	assert.Len(t, tracker.Violations(), 1)
}

func TestErrorPresenter_QuotaExceededCarriesItsParams(t *testing.T) {
	t.Parallel()

	present := newErrorPresenter(presenterTestConfig())

	presented := present(context.Background(), fmt.Errorf("create shipment: %w",
		errortypes.NewQuotaExceededError("shipments.total", 12, 12, "free_demo")))
	require.NotNil(t, presented)
	assert.Equal(t, string(errortypes.ErrQuotaExceeded), presented.Extensions["code"])
	assert.Equal(t,
		"https://api.test/problems/"+string(helpers.ProblemTypeQuotaExceeded),
		presented.Extensions["type"],
	)
	assert.Equal(t, map[string]string{
		"meter": "shipments.total",
		"limit": "12",
		"used":  "12",
		"plan":  "free_demo",
	}, presented.Extensions["params"])
	assert.Equal(t, "This organization has reached the limit of its plan", presented.Message)
}

func TestErrorPresenter_PlanRestrictionCarriesItsParams(t *testing.T) {
	t.Parallel()

	present := newErrorPresenter(presenterTestConfig())

	presented := present(context.Background(),
		errortypes.NewPlanRestrictionError("api_keys", "", "free_demo"))
	require.NotNil(t, presented)
	assert.Equal(t, string(errortypes.ErrPlanRestricted), presented.Extensions["code"])
	assert.Equal(t,
		"https://api.test/problems/"+string(helpers.ProblemTypePlanRestricted),
		presented.Extensions["type"],
	)
	assert.Equal(t, map[string]string{
		"capability": "api_keys",
		"reason":     errortypes.PlanRestrictionReasonPlan,
		"plan":       "free_demo",
	}, presented.Extensions["params"])
}
