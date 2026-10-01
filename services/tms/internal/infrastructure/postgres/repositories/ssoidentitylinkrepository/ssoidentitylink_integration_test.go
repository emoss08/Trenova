//go:build integration

package ssoidentitylinkrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSSOIdentityLinks_BindOneSubjectToOneUserPerProvider(t *testing.T) {
	ctx, db, cleanup := seedtest.SetupTestDB(t)
	t.Cleanup(cleanup)

	data := seedtest.SeedFullTestData(t, ctx, db)
	other := seedtest.NewUser(data.Organization.ID, data.BusinessUnit.ID).
		WithUsername("second").
		WithEmail("second@example.com")
	tx, err := db.BeginTx(ctx, nil)
	require.NoError(t, err)
	secondUser := other.Build(t, ctx, tx)
	require.NoError(t, tx.Commit())

	cfg := &tenant.SSOConfig{
		OrganizationID:   data.Organization.ID,
		BusinessUnitID:   data.BusinessUnit.ID,
		Name:             "Okta",
		Provider:         tenant.SSOProviderOkta,
		Protocol:         tenant.SSOProtocolOIDC,
		Enabled:          true,
		OIDCIssuerURL:    "https://idp.example.com",
		OIDCClientID:     "client",
		OIDCClientSecret: "secret",
		OIDCRedirectURL:  "https://app.example.com/callback",
		OIDCScopes:       []string{"openid", "email"},
	}
	_, err = db.NewInsert().Model(cfg).Exec(ctx)
	require.NoError(t, err)

	repo := New(Params{DB: postgres.NewTestConnection(db), Logger: zap.NewNop()})
	link := &tenant.SSOIdentityLink{
		OrganizationID: data.Organization.ID,
		BusinessUnitID: data.BusinessUnit.ID,
		SSOConfigID:    cfg.ID,
		UserID:         data.User.ID,
		Issuer:         cfg.OIDCIssuerURL,
		Subject:        "okta|1",
		EmailAtLink:    data.User.EmailAddress,
		LastLoginAt:    timeutils.NowUnix(),
	}
	require.NoError(t, repo.Create(ctx, link))

	found, err := repo.GetBySubject(ctx, repositories.GetSSOIdentityLinkBySubjectRequest{
		SSOConfigID: cfg.ID, Issuer: cfg.OIDCIssuerURL, Subject: "okta|1",
	})
	require.NoError(t, err)
	assert.Equal(t, data.User.ID, found.UserID)

	sameSubjectOtherUser := *link
	sameSubjectOtherUser.ID = ""
	sameSubjectOtherUser.UserID = secondUser.ID
	err = repo.Create(ctx, &sameSubjectOtherUser)
	require.Error(t, err)
	assert.True(t, dberror.IsUniqueConstraintViolation(err))

	sameUserOtherSubject := *link
	sameUserOtherSubject.ID = ""
	sameUserOtherSubject.Subject = "okta|2"
	err = repo.Create(ctx, &sameUserOtherSubject)
	require.Error(t, err)
	assert.True(t, dberror.IsUniqueConstraintViolation(err))

	require.NoError(t, repo.RecordLogin(ctx, found, timeutils.NowUnix()))

	_, err = repo.GetByUser(ctx, repositories.GetSSOIdentityLinkByUserRequest{
		SSOConfigID: cfg.ID, Issuer: cfg.OIDCIssuerURL, UserID: secondUser.ID,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsNotFoundError(err))
}
