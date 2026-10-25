package edicommunicationprofilerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/internal/modules/edi/ports/ediports"
	"github.com/emoss08/trenova/internal/testutil/keeltest"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
)

func TestProfileOptionsQuery_AppliesTheFiltersAndSearch(t *testing.T) {
	t.Parallel()

	partnerID := pulid.ID("edip_1")
	q := profileOptionsQuery(&ediports.EDICommunicationProfileSelectOptionsRequest{
		Status:    domaintypes.StatusActive,
		Method:    edi.ConnectionMethodSFTP,
		PartnerID: partnerID,
	}).Where(profileOptionsSearch("car_rier")).Limit(10)

	sql, args := keeltest.Render(t, q)

	assert.Equal(
		t,
		`SELECT ecp."id", ecp."business_unit_id", ecp."organization_id", ecp."edi_connection_id", `+
			`ecp."edi_partner_id", ecp."method", ecp."status", ecp."name", ecp."description" `+
			`FROM "edi_communication_profiles" ecp `+
			`WHERE ecp."business_unit_id" = $1 AND ecp."organization_id" = $2 AND ecp."status" = $3 `+
			`AND ecp."method" = $4 AND ecp."edi_partner_id" = $5 `+
			`AND (ecp."name" ILIKE $6 OR ecp."description" ILIKE $7 OR (CAST(ecp."method" AS text) ILIKE $8)) `+
			`ORDER BY ecp."name" ASC LIMIT 10`,
		sql,
	)
	assert.Equal(
		t,
		[]any{"bu_1", "org_1", "Active", "SFTP", "edip_1", `%car\_rier%`, `%car\_rier%`, `%car\_rier%`},
		args,
	)
}
