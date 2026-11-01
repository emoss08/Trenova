package customerresolver

import (
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/gqlctx"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvercore"
	"github.com/emoss08/trenova/internal/api/graphql/resolver/resolvertest"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/parties/app/customerservice"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/ports/customerports"
	"github.com/emoss08/trenova/internal/testutil/mocks/customerportsmock"
	"github.com/emoss08/trenova/pkg/authctx"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestShipmentCustomerBillingRules_NeedsShipmentReadNotCustomerRead(t *testing.T) {
	t.Parallel()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	userID := pulid.MustNew("usr_")
	payer := pulid.MustNew("cus_")
	repo := customerportsmock.NewMockCustomerRepository(t)
	repo.EXPECT().
		GetShipmentBillingProfiles(mock.Anything, mock.MatchedBy(func(req customerports.GetShipmentBillingRulesRequest) bool {
			return req.TenantInfo.OrgID == orgID &&
				req.TenantInfo.BuID == buID &&
				assert.ObjectsAreEqual([]pulid.ID{payer}, req.CustomerIDs)
		})).
		Return([]*customer.CustomerBillingProfile{{
			CustomerID:   payer,
			CreditStatus: customer.CreditStatusWarning,
		}}, nil).
		Once()
	engine := &resolvertest.RecordingPermissionEngine{}
	resolver := &QueryResolver{&Deps{
		Core: &resolvercore.Core{PermissionEngine: engine},
		CustomerService: customerservice.New(customerservice.Params{
			Logger: zap.NewNop(),
			Repo:   repo,
		}),
	}}
	ctx := gqlctx.WithAuthContext(t.Context(), &authctx.AuthContext{
		PrincipalType:  authctx.PrincipalTypeUser,
		PrincipalID:    userID,
		UserID:         userID,
		OrganizationID: orgID,
		BusinessUnitID: buID,
	})

	rules, err := resolver.ShipmentCustomerBillingRules(ctx, []string{payer.String()})
	require.NoError(t, err)

	assert.Equal(t, []*customerports.ShipmentBillingRules{{
		CustomerID:   payer,
		CreditStatus: customer.CreditStatusWarning,
	}}, rules)
	require.NotNil(t, engine.Request)
	assert.Equal(t, permission.ResourceShipment.String(), engine.Request.Resource)
	assert.Equal(t, permission.OpRead, engine.Request.Operation)
}
