package tenderresponse

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/internal/modules/operations/domain/tender"
	"github.com/emoss08/trenova/internal/modules/operations/ports/tenderports"
	"github.com/emoss08/trenova/internal/testutil/plantest"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/emoss08/trenova/shared/tokenutils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type offerTokens struct {
	tenderports.TenderRepository

	tokens      map[string]*tender.TenderOfferToken
	offerReads  int
	offerTenant dbscope.Tenant
}

func (o *offerTokens) GetTokenByHash(
	_ context.Context,
	tokenHash string,
) (*tender.TenderOfferToken, error) {
	return o.tokens[tokenHash], nil
}

func (o *offerTokens) GetOfferByID(
	ctx context.Context,
	req tenderports.GetTenderOfferByIDRequest,
) (*tender.TenderOffer, error) {
	o.offerReads++
	o.offerTenant, _ = dbscope.TenantFrom(ctx)
	return &tender.TenderOffer{ID: req.OfferID}, nil
}

func tokenFor(org pulid.ID, raw string) *tender.TenderOfferToken {
	return &tender.TenderOfferToken{
		ID:             pulid.MustNew("tot_"),
		OrganizationID: org,
		BusinessUnitID: pulid.MustNew("bu_"),
		TenderOfferID:  pulid.MustNew("tof_"),
		TokenHash:      tokenutils.Hash(raw),
		ExpiresAt:      timeutils.NowUnix() + 3_600,
	}
}

func TestATenderOfferLinkAnswersOnlyForAnOrganizationWithBrokerage(t *testing.T) {
	t.Parallel()

	with, without := pulid.MustNew("org_"), pulid.MustNew("org_")
	repo := &offerTokens{tokens: map[string]*tender.TenderOfferToken{}}
	for org, raw := range map[pulid.ID]string{with: "with-brokerage", without: "without-brokerage"} {
		token := tokenFor(org, raw)
		repo.tokens[token.TokenHash] = token
	}
	svc := &Service{
		repo: repo,
		features: plantest.SharedServer().
			Planned(with).
			Without(without, platformcatalog.FeatureCarrierBrokerage),
	}

	_, _, _, err := svc.resolveToken(t.Context(), "without-brokerage")
	require.EqualError(t, err, InvalidTokenError().Error(), "the same answer as an unknown link")
	assert.Zero(t, repo.offerReads, "nothing of the offer is read")

	_, token, offer, err := svc.resolveToken(t.Context(), "with-brokerage")
	require.NoError(t, err)
	require.NotNil(t, offer)
	assert.Equal(t, with, token.OrganizationID)
	assert.Equal(t, with, repo.offerTenant.OrganizationID, "read under the link's own tenant")
}
