package carrierintelservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/carrierintel"
	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func eventTestInput() *ingestInput {
	return &ingestInput{
		tenant: pagination.TenantInfo{OrgID: pulid.MustNew("org_"), BuID: pulid.MustNew("bu_")},
		subject: repositories.CarrierIntelSubject{
			SubjectType: carrierintel.SubjectTypeCarrier,
			SubjectID:   pulid.MustNew("car_").String(),
			Name:        "FedEx Ground",
			DOTNumber:   "265752",
		},
		bound:  &boundProvider{provider: integration.TypeCarrierOK},
		source: carrierintel.EventSourceSnapshotDiff,
	}
}

func TestBuildEventsFormatsChangeSummaries(t *testing.T) {
	t.Parallel()

	in := eventTestInput()
	snapshot := &carrierintel.CarrierIntelSnapshot{DOTNumber: "265752"}
	changes := []carrierintel.FieldChange{
		{
			Path:     "safety.ratingDate",
			Section:  carrierintel.SectionSafety,
			Label:    "Safety rating date",
			Severity: carrierintel.SeverityLow,
			Current:  float64(843609600),
		},
		{
			Path:     "insurance.bipdRequired",
			Section:  carrierintel.SectionInsurance,
			Label:    "BIPD coverage required",
			Severity: carrierintel.SeverityMedium,
			Prior:    "750000",
			Current:  "5000000",
		},
		{
			Path:     "safety.outOfServiceOrder",
			Section:  carrierintel.SectionSafety,
			Label:    "Out-of-service order",
			Severity: carrierintel.SeverityCritical,
			Prior:    false,
			Current:  true,
		},
	}

	events := (&Service{}).buildEvents(in, snapshot, changes, nil, nil, testNow)
	require.Len(t, events, 3)
	assert.Equal(t, "Safety rating date changed from none to Sep 25, 1996", events[0].Summary)
	assert.Equal(
		t,
		"BIPD coverage required changed from $750,000 to $5,000,000",
		events[1].Summary,
	)
	assert.Equal(t, "Out-of-service order changed from No to Yes", events[2].Summary)
	assert.Equal(t, "5000000", events[1].CurrentValue)
}

func TestBuildEventsRuleEventsCarryTheMessageOnlyInTheSummary(t *testing.T) {
	t.Parallel()

	in := eventTestInput()
	snapshot := &carrierintel.CarrierIntelSnapshot{DOTNumber: "265752"}
	finding := carrierintel.Finding{
		Code:     carrierintel.RuleInsuranceBIPDBelow,
		Category: carrierintel.SectionInsurance,
		Action:   carrierintel.RuleActionBlock,
		Severity: carrierintel.SeverityCritical,
		Message:  "BIPD liability coverage on file ($0) is below the required $5,000,000",
	}
	other := finding
	other.Message = "BIPD liability coverage on file ($0) is below the required $750,000"

	svc := &Service{}
	events := svc.buildEvents(in, snapshot, nil, []carrierintel.Finding{finding}, nil, testNow)
	require.Len(t, events, 1)
	assert.Equal(t, finding.Message, events[0].Summary)
	assert.Nil(t, events[0].CurrentValue)
	assert.Nil(t, events[0].PriorValue)

	again := svc.buildEvents(in, snapshot, nil, []carrierintel.Finding{finding}, nil, testNow)
	assert.Equal(t, events[0].Fingerprint, again[0].Fingerprint)

	changed := svc.buildEvents(in, snapshot, nil, []carrierintel.Finding{other}, nil, testNow)
	assert.NotEqual(t, events[0].Fingerprint, changed[0].Fingerprint)
}
