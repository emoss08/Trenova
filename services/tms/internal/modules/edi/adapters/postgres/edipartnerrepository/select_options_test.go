package edipartnerrepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestPartnerOptionsQuery_AppliesTheFiltersAndSearch(t *testing.T) {
	t.Parallel()

	q := partnerOptionsQuery(&ediports.EDIPartnerSelectOptionsRequest{
		Kind:               edi.PartnerKindExternal,
		EnabledForOutbound: true,
	}).Where(partnerSearch("acme")).Limit(10)

	sql, args := keeltest.Render(t, q)

	assert.Equal(t, "", sql)
	assert.Equal(t, []any{}, args)
}

func TestReadinessQuery_ChecksEverySetupStepInOneStatement(t *testing.T) {
	t.Parallel()

	q := readinessQuery(&ediports.GetEDIPartnerReadinessRequest{
		PartnerIDs: []pulid.ID{"edip_1", "edip_2"},
	})

	sql, args := keeltest.Render(t, q)

	assert.Equal(t, "", sql)
	assert.Equal(t, []any{}, args)
}
