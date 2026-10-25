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

	assert.Contains(t, sql, `FROM "edi_partners" ep WHERE`)
	assert.Contains(
		t,
		sql,
		`ep."kind" = 'External' AND ep."enabled_for_outbound" IS TRUE `+
			`AND (ep."name" ILIKE $3 OR ep."code" ILIKE $4) LIMIT 10`,
	)
	assert.Equal(t, []any{"bu_1", "org_1", "%acme%", "%acme%"}, args)
}

func TestReadinessQuery_ChecksEverySetupStepInOneStatement(t *testing.T) {
	t.Parallel()

	q := readinessQuery(&ediports.GetEDIPartnerReadinessRequest{
		PartnerIDs: []pulid.ID{"edip_1", "edip_2"},
	})

	sql, args := keeltest.Render(t, q)

	assert.Contains(t, sql, `EXISTS (SELECT 1 FROM "edi_communication_profiles" ecp`)
	assert.Contains(t, sql, `EXISTS (SELECT 1 FROM "edi_mapping_profile_items" empi`)
	assert.Contains(t, sql, `EXISTS (SELECT 1 FROM "edi_test_cases" etc`)
	assert.Contains(t, sql, `FROM "edi_partners" ep WHERE ep."business_unit_id" = $1 AND ep."organization_id" = $2 AND ep."id" = ANY($3)`)
	assert.Contains(t, sql, `epdp."status" = 'Active' AND epdp."direction" = 'Inbound'`)
	assert.Contains(t, sql, `epdp2."status" = 'Active' AND epdp2."direction" = 'Outbound'`)
	assert.Equal(t, []any{"bu_1", "org_1", []string{"edip_1", "edip_2"}, "", ""}, args)
}
