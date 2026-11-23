package aiusageresolver

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/gqlmodel"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type countingUsage struct {
	calls int
}

func (c *countingUsage) Summary(
	context.Context,
	pagination.TenantInfo,
	int64,
) (*aiusageports.AIUsageSummary, error) {
	c.calls++
	return &aiusageports.AIUsageSummary{}, nil
}

func (c *countingUsage) Daily(
	context.Context,
	*aiusageports.AIUsageDailyRequest,
) ([]aiusageports.AIUsageDay, error) {
	c.calls++
	return nil, nil
}

func usageResolver(granted ...string) (*QueryResolver, *countingUsage) {
	engine := &resolvertest.GrantingPermissionEngine{Granted: make(map[string]bool)}
	for _, grant := range granted {
		engine.Granted[grant] = true
	}
	usage := &countingUsage{}
	return &QueryResolver{&Deps{
		Core:           &resolvercore.Core{PermissionEngine: engine},
		AiUsageService: usage,
	}}, usage
}

func personCtx(t *testing.T) context.Context {
	t.Helper()
	userID := pulid.MustNew("usr_")
	return gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
	})
}

func readUsage(ctx context.Context, q *QueryResolver) []error {
	_, summaryErr := q.AiUsageSummary(ctx, nil)
	_, dailyErr := q.AiUsageDaily(ctx, nil, nil, nil)
	_, featuresErr := q.AiUsageFeatures(ctx, gqlmodel.DataTableConnectionInput{}, nil)
	return []error{summaryErr, dailyErr, featuresErr}
}

func TestUsageReads_AIUsageViewAloneReadsEveryUsageFigure(t *testing.T) {
	t.Parallel()

	q, usage := usageResolver(permission.ResourceAIUsage.String() + "|" + string(permission.OpRead))
	ctx := personCtx(t)
	_, err := q.AiUsageSummary(ctx, nil)
	require.NoError(t, err)
	_, err = q.AiUsageDaily(ctx, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 2, usage.calls)
}

func TestUsageReads_AIProviderViewAloneSeesNoUsage(t *testing.T) {
	t.Parallel()

	q, usage := usageResolver(
		permission.ResourceAIProvider.String() + "|" + string(permission.OpRead),
	)
	for _, err := range readUsage(personCtx(t), q) {
		var authz *errortypes.AuthorizationError
		require.ErrorAs(t, err, &authz)
	}
	assert.Zero(t, usage.calls, "nothing was read for someone who cannot see usage")
}
